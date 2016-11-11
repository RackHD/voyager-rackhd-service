package server

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/RackHD/voyager-utilities/amqp"
	"github.com/RackHD/voyager-utilities/models"
	"github.com/RackHD/voyager-utilities/random"

	samqp "github.com/streadway/amqp"
)

// Server defines amqp server
type Server struct {
	MQ        *amqp.Client
	RackHDURI string
}

// NewServer creates new amqp server
func NewServer(amqpServer string, rackhdURI string) *Server {
	server := Server{}
	server.RackHDURI = rackhdURI
	server.MQ = amqp.NewClient(amqpServer)
	if server.MQ == nil {
		log.Fatalf("Could not connect to RabbitMQ server: %s\n", amqpServer)
	}
	return &server
}

// Start starts the server
func (s *Server) Start() error {
	rackhdQueueName := random.RandQueue()
	_, deliveries, err := s.MQ.Listen(models.RackHDExchange, models.RackHDExchangeType, rackhdQueueName, models.RackHDBindingKey, "")
	if err != nil {
		return err
	}

	go func() {
		for m := range deliveries {
			log.Printf("got %dB delivery on exchange %s: [%v] %s", len(m.Body), m.Exchange, m.DeliveryTag, m.Body)
			m.Ack(true)
			go s.ProcessMessage(&m)
		}
	}()

	return nil
}

// ProcessMessage processes a message
func (s *Server) ProcessMessage(m *samqp.Delivery) error {
	switch m.Exchange {
	case models.RackHDExchange:
		log.Printf("A message to RackHD service! %s\n", string(m.Body))
		return s.processRackHDService(m)
	default:
		err := fmt.Errorf("Unknown exchange name: %s\n", m.Exchange)
		log.Printf("Error: %s", err)
		return err

	}
}

func (s *Server) processRackHDService(d *samqp.Delivery) error {
	// Unmarshal the message (assumes message is valid json)
	var request models.RackHDReq
	response := models.RackHDResp{}

	err := json.Unmarshal(d.Body, &request)
	if err != nil {
		log.Printf("Error: %s\n", err)
		response.Failed = true
		response.Error = err.Error()
	}
	var funcToCall func(d *samqp.Delivery) (string, error)
	switch request.Action {
	case models.UploadTemplateAction:
		funcToCall = s.rackhdUploadConfig
	case models.UploadWorkflowAction:
		funcToCall = s.rackhdUploadWorkload
	case models.RunWorkflowAction:
		funcToCall = s.rackhdRunWorkflow
	case models.ListenWorkflowAction:
		funcToCall = s.rackhdListenWorkflow
	case models.DeleteTemplateAction:
		return fmt.Errorf("DeleteTemplateAction is not implemented")
	default:
		err = fmt.Errorf("Unknown Action\n")
		log.Printf("Error: %s", err)
		return err
	}

	serverResponse, err := funcToCall(d)
	if err != nil {
		response.Failed = true
		response.Error = err.Error()
	}
	response.ServerResponse = serverResponse

	bodyBytes, err := json.Marshal(response)
	if err != nil {
		return err
	}

	err = s.MQ.Send(d.Exchange, models.RackHDExchangeType, d.ReplyTo, string(bodyBytes), d.CorrelationId, "")
	if err != nil {
		return fmt.Errorf("Failed to send resp to %s due to %s", d.ReplyTo, err)
	}

	return nil
}

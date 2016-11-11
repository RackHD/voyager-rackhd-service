package server_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/RackHD/voyager-rackhd-service/server"
	"github.com/RackHD/voyager-utilities/models"
	"github.com/RackHD/voyager-utilities/random"
	"github.com/streadway/amqp"
)

var _ = Describe("INTEGRATION", func() {

	Describe("AMQP Message Handling", func() {
		var rabbitMQURL string
		var rackHDURL string
		var rackHDEndpoint string

		var s *server.Server
		var testExchange string
		var testExchangeType string
		var testQueueName string
		var testBindingKey string
		var testConsumerTag string
		var testReplyTo string
		var testCorID string
		var index = 0

		BeforeEach(func() {
			rabbitMQURL = os.Getenv("RABBITMQ_URL")
			// Can't close the server so creating a new one for each test.
			rackHDEndpoint = fmt.Sprintf("localhost:%d", 8888+index)
			index += 1
			rackHDURL = fmt.Sprintf("http://%s", rackHDEndpoint)
			s = server.NewServer(rabbitMQURL, rackHDURL)
			testExchange = models.RackHDExchange
			testExchangeType = models.RackHDExchangeType
			testBindingKey = models.RackHDBindingKey
			testConsumerTag = models.RackHDConsumerTag
			testCorID = random.RandQueue()
			testReplyTo = "Replies"
			testQueueName = random.RandQueue()

			Expect(s.MQ).ToNot(Equal(nil))

			err := s.Start()
			Expect(err).ToNot(HaveOccurred())

		})
		AfterEach(func() {
			s.MQ.Close()
		})
		Context("When a message comes in to voyager-rackhd-service", func() {

			var deliveries <-chan amqp.Delivery

			It("INTEGRATION should do a PUT template request to RackHD", func() {
				configBody := "Hello I'm a template"
				configFile := "TestConfig"

				type Fake struct {
					FakeResponse string `json:"fakeResponse"`
				}
				fakeResponse := Fake{
					FakeResponse: "fake rackhd response",
				}
				expectedFakeResponseBytes, err := json.Marshal(fakeResponse)
				Expect(err).ToNot(HaveOccurred())
				expectedFakeResponse := fmt.Sprintf("%s\n", string(expectedFakeResponseBytes))

				router := mux.NewRouter().StrictSlash(true)
				url := fmt.Sprintf("%s{configName}", models.RackHDTemplateURI)
				router.HandleFunc(url, func(w http.ResponseWriter, r *http.Request) {
					vars := mux.Vars(r)
					configName := vars["configName"]
					Expect(configName).To(Equal(configFile))

					requestBody, err := ioutil.ReadAll(r.Body)
					Expect(err).ToNot(HaveOccurred())
					Expect(string(requestBody)).To(Equal(configBody))

					w.WriteHeader(http.StatusCreated)
					json.NewEncoder(w).Encode(fakeResponse)
				})

				go func() {
					fmt.Printf("Listenning on: %s\n", rackHDEndpoint)
					http.ListenAndServe(rackHDEndpoint, router)
				}()

				_, deliveries, err = s.MQ.Listen(testExchange, testExchangeType, testQueueName, testReplyTo, testConsumerTag)
				Expect(err).ToNot(HaveOccurred())

				testReq := models.RackHDConfigReq{
					Name:   configFile,
					Config: configBody,
				}

				testReq.Action = models.UploadTemplateAction
				testMessage, err := json.Marshal(testReq)
				Expect(err).ToNot(HaveOccurred())

				err = s.MQ.Send(testExchange, testExchangeType, testBindingKey, string(testMessage), testCorID, testReplyTo)
				Expect(err).ToNot(HaveOccurred())

				d := <-deliveries
				d.Ack(false)
				Expect(d.CorrelationId).To(Equal(testCorID))

				log.Printf("d.Body is %+s\n", d.Body)

				var response models.RackHDResp
				err = json.Unmarshal(d.Body, &response)
				Expect(err).ToNot(HaveOccurred())
				Expect(response.Failed).To(BeFalse())
				Expect(response.Error).To(BeEmpty())
				Expect(response.ServerResponse).To(Equal(expectedFakeResponse))

			})

			It("INTEGRATION should do a PUT workflow request to RackHD", func() {
				workflowBody := "Hello I'm a workflow"

				type Fake struct {
					FakeResponse string `json:"fakeResponse"`
				}
				fakeResponse := Fake{
					FakeResponse: "fake rackhd response",
				}
				expectedFakeResponseBytes, err := json.Marshal(fakeResponse)
				Expect(err).ToNot(HaveOccurred())
				expectedFakeResponse := fmt.Sprintf("%s\n", string(expectedFakeResponseBytes))

				router := mux.NewRouter().StrictSlash(true)
				url := fmt.Sprintf("%s", models.RackHDGraphURI)
				router.HandleFunc(url, func(w http.ResponseWriter, r *http.Request) {
					requestBody, err := ioutil.ReadAll(r.Body)
					Expect(err).ToNot(HaveOccurred())
					Expect(string(requestBody)).To(Equal(workflowBody))

					w.WriteHeader(http.StatusCreated)
					json.NewEncoder(w).Encode(fakeResponse)
				})

				go func() {
					fmt.Printf("Listenning on: %s\n", rackHDEndpoint)
					http.ListenAndServe(rackHDEndpoint, router)
				}()

				_, deliveries, err = s.MQ.Listen(testExchange, testExchangeType, testQueueName, testReplyTo, testConsumerTag)
				Expect(err).ToNot(HaveOccurred())

				testReq := models.RackHDWorkflowReq{
					Workflow: workflowBody,
				}

				testReq.Action = models.UploadWorkflowAction
				testMessage, err := json.Marshal(testReq)
				Expect(err).ToNot(HaveOccurred())

				err = s.MQ.Send(testExchange, testExchangeType, testBindingKey, string(testMessage), testCorID, testReplyTo)
				Expect(err).ToNot(HaveOccurred())

				d := <-deliveries
				d.Ack(false)
				Expect(d.CorrelationId).To(Equal(testCorID))

				var response models.RackHDResp
				err = json.Unmarshal(d.Body, &response)
				Expect(err).ToNot(HaveOccurred())
				Expect(response.Failed).To(BeFalse())
				Expect(response.Error).To(BeEmpty())
				Expect(response.ServerResponse).To(Equal(expectedFakeResponse))

			})

			It("INTEGRATION should do a POST workflow request to a node in RackHD", func() {
				nodeID := random.RandQueue()
				testReq := models.RackHDRunWorkflowReq{}
				testReq.Action = models.RunWorkflowAction
				testReq.NodeID = nodeID
				testReq.Name = models.DefaultCiscoInjectableName
				testReq.Options.DeployConfigAndImages.BootImage = models.DefaultCiscoBootImage
				testReq.Options.DeployConfigAndImages.StartupConfig = "config-file.txt"

				workflowBody, err := json.Marshal(testReq)
				Expect(err).ToNot(HaveOccurred())

				type Fake struct {
					FakeResponse string `json:"fakeResponse"`
				}
				fakeResponse := Fake{
					FakeResponse: "fake rackhd response",
				}
				expectedFakeResponseBytes, err := json.Marshal(fakeResponse)
				Expect(err).ToNot(HaveOccurred())
				expectedFakeResponse := fmt.Sprintf("%s\n", string(expectedFakeResponseBytes))

				router := mux.NewRouter().StrictSlash(true)

				url := fmt.Sprintf("%snodes/%s/workflows/", models.RackHDRootURI, nodeID)
				router.HandleFunc(url, func(w http.ResponseWriter, r *http.Request) {
					requestBody, err := ioutil.ReadAll(r.Body)
					Expect(err).ToNot(HaveOccurred())
					Expect(string(requestBody)).To(Equal(string(workflowBody)))
					name := r.URL.Query().Get("name")
					Expect(name).To(Equal(models.DefaultCiscoInjectableName))

					w.WriteHeader(http.StatusCreated)
					json.NewEncoder(w).Encode(fakeResponse)
				})

				go func() {
					fmt.Printf("Listenning on: %s\n", rackHDEndpoint)
					http.ListenAndServe(rackHDEndpoint, router)
				}()

				_, deliveries, err = s.MQ.Listen(testExchange, testExchangeType, testQueueName, testReplyTo, testConsumerTag)
				Expect(err).ToNot(HaveOccurred())

				testMessage, err := json.Marshal(testReq)
				Expect(err).ToNot(HaveOccurred())

				err = s.MQ.Send(testExchange, testExchangeType, testBindingKey, string(testMessage), testCorID, testReplyTo)
				Expect(err).ToNot(HaveOccurred())

				d := <-deliveries
				d.Ack(false)
				Expect(d.CorrelationId).To(Equal(testCorID))

				var response models.RackHDResp
				err = json.Unmarshal(d.Body, &response)
				Expect(err).ToNot(HaveOccurred())
				Expect(response.Failed).To(BeFalse())
				Expect(response.Error).To(BeEmpty())
				Expect(response.ServerResponse).To(Equal(expectedFakeResponse))
			})

			It("INTEGRATION should listen on requested exchange and routing key after receiving listenActionRequest", func() {
				_, deliveries, err := s.MQ.Listen(testExchange, testExchangeType, testQueueName, testReplyTo, testConsumerTag)
				Expect(err).ToNot(HaveOccurred())
				graphUUID := "fake-graph-uuid"
				routingKey := fmt.Sprintf("%s.%s", models.GraphFinishedRoutingKey, graphUUID)
				requestMessage := models.RackHDListenReq{}
				requestMessage.Action = models.ListenWorkflowAction
				requestMessage.RoutingKey = routingKey
				requestMessage.Exchange = models.OnEventsExchange
				requestMessage.ExchangeType = models.OnEventsExchangeType

				testMessage, err := json.Marshal(requestMessage)
				Expect(err).ToNot(HaveOccurred())

				err = s.MQ.Send(testExchange, testExchangeType, testBindingKey, string(testMessage), testCorID, testReplyTo)
				Expect(err).ToNot(HaveOccurred())

				fakeRackHDSuccessMessage := `{"status":"succeeded"}`
				go func() {
					time.Sleep(5 * time.Second)
					err := s.MQ.Send(models.OnEventsExchange, models.OnEventsExchangeType, routingKey, fakeRackHDSuccessMessage, "", "")
					Expect(err).ToNot(HaveOccurred())
				}()

				actualMsg := <-deliveries
				actualMsg.Ack(true)

				rackHDResponse := models.RackHDResp{}
				err = json.Unmarshal(actualMsg.Body, &rackHDResponse)
				Expect(err).ToNot(HaveOccurred())
				Expect(rackHDResponse).To(Equal(models.RackHDResp{
					ServerResponse: fakeRackHDSuccessMessage,
				}))
			})
		})

	})

})

package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/RackHD/voyager-utilities/models"
	"github.com/RackHD/voyager-utilities/random"

	samqp "github.com/streadway/amqp"
)

func (s *Server) rackhdUploadConfig(d *samqp.Delivery) (string, error) {
	var request models.RackHDConfigReq
	err := json.Unmarshal(d.Body, &request)
	if err != nil {
		log.Printf("Error: %s\n", err)
		return "", err
	}

	configFile := bytes.NewBuffer([]byte(request.Config))

	url := fmt.Sprintf("%s%s%s", s.RackHDURI, models.RackHDTemplateURI, request.Name)
	response, err := s.rackhdUploadFile(url, configFile, int64(configFile.Len()))
	if err != nil {
		return "", err
	}
	log.Printf("uploaded config template: %s to server", request.Name)

	return response, nil
}

func (s *Server) rackhdUploadWorkload(d *samqp.Delivery) (string, error) {
	var request models.RackHDWorkflowReq
	err := json.Unmarshal(d.Body, &request)
	if err != nil {
		log.Printf("Error: %s\n", err)
		return "", err
	}

	configFile := bytes.NewBuffer([]byte(request.Workflow))

	url := fmt.Sprintf("%s%s", s.RackHDURI, models.RackHDGraphURI)
	response, err := s.rackhdUploadWorkflow(url, configFile)
	if err != nil {
		return "", err
	}
	log.Printf("Uploaded workflows: %s to server", url)

	return response, nil
}

func (s *Server) rackhdListenWorkflow(d *samqp.Delivery) (string, error) {
	var request models.RackHDListenReq
	err := json.Unmarshal(d.Body, &request)
	if err != nil {
		log.Printf("Error: %s\n", err)
		return "", err
	}
	queueName := random.RandQueue()
	_, deliveries, err := s.MQ.Listen(request.Exchange, request.ExchangeType, queueName, request.RoutingKey, models.RackHDConsumerTag)
	if err != nil {
		log.Printf("error: %s\n", err)
		return "", err
	}
	rackHDWorkflowStatus := <-deliveries
	rackHDWorkflowStatus.Ack(true)

	return string(rackHDWorkflowStatus.Body), nil
}

func (s *Server) rackhdUploadFile(serverURL string, r io.Reader, contentLength int64) (string, error) {
	body := ioutil.NopCloser(r)
	request, err := http.NewRequest("PUT", serverURL, body)
	if err != nil {
		return "", fmt.Errorf("Error building request to api server: %s", err)
	}
	request.ContentLength = contentLength
	request.Header.Add("Content-Type", "text/plain")
	request.Header.Add("Accept", "application/json")

	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		return "", fmt.Errorf("Error making request to api server: %s", err)
	}

	defer resp.Body.Close()
	if resp.StatusCode != 201 {
		return "", fmt.Errorf("Failed uploading %s with status: %s\nresponse: %+v", serverURL, resp.Status, resp)
	}
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(bodyBytes), nil
}

func (s *Server) rackhdUploadWorkflow(serverURL string, r io.Reader) (string, error) {
	body := ioutil.NopCloser(r)
	request, err := http.NewRequest("PUT", serverURL, body)
	if err != nil {
		return "", fmt.Errorf("Error building request to api server: %s", err)
	}
	request.Header.Add("Content-Type", "application/json")
	request.Header.Add("Accept", "application/json")

	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		return "", fmt.Errorf("Error making request to api server: %s", err)
	}

	defer resp.Body.Close()
	if resp.StatusCode != 201 {
		return "", fmt.Errorf("Failed uploading %s with status: %s\nresponse: %+v", serverURL, resp.Status, resp)
	}
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(bodyBytes), nil
}

func (s *Server) rackhdRunWorkflow(d *samqp.Delivery) (string, error) {
	var request models.RackHDRunWorkflowReq
	err := json.Unmarshal(d.Body, &request)
	if err != nil {
		log.Printf("Error: %s\n", err)
		return "", err
	}
	workflowConfig := bytes.NewBuffer(d.Body)

	url := fmt.Sprintf("%s%snodes/%s/workflows/?name=%s", s.RackHDURI, models.RackHDRootURI, request.NodeID, models.DefaultCiscoInjectableName)

	response, err := s.sendHTTPRequest(url, workflowConfig, int64(workflowConfig.Len()))
	if err != nil {
		log.Printf("Error: %s\n", err)
		return "", err
	}
	log.Printf("Ran workflow: %s to server", url)

	return response, nil
}

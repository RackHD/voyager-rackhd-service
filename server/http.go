package server

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
)

func (s *Server) sendHTTPRequest(serverURL string, body io.Reader, contentLength int64) (string, error) {

	request, err := http.NewRequest("POST", serverURL, body)
	if err != nil {
		return "", fmt.Errorf("Error building request to api server: %s", err)

	}
	request.ContentLength = contentLength
	request.Header.Add("Content-Type", "application/json")
	request.Header.Add("Accept", "application/json")

	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		return "", fmt.Errorf("Error making request to api server: %s", err)
	}

	defer resp.Body.Close()
	if resp.StatusCode != 201 {
		return "", fmt.Errorf("Failed to run workflow %s with status: %s\nresponse: %+v", serverURL, resp.Status, resp)
	}
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(bodyBytes), nil
}

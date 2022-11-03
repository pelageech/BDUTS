package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"
)

type serverJSON struct {
	URL string
}

type configJSON struct {
	Port            int
	NumberOfRetries int
}

var servers []*url.URL
var currentIndex int

var port int
var numberOfRetries int

func readConfig() {
	/*
		read servers config
	*/
	serversFile, err := os.Open("resources/servers.json")
	if err != nil {
		log.Fatal("Failed to open servers config: ", err)
	}
	defer serversFile.Close()

	serversFileByte, err := io.ReadAll(serversFile)
	if err != nil {
		log.Fatal("Failed to read servers config file: ", err)
	}

	var serversJSON []serverJSON
	err = json.Unmarshal(serversFileByte, &serversJSON)
	if err != nil {
		log.Fatal("Failed to unmarshal servers config: ", err)
	}

	for _, server := range serversJSON {
		serverURL, err := url.Parse(server.URL)
		if err != nil {
			log.Fatal("Failed to parse server URL", err)
		}

		servers = append(servers, serverURL)
	}

	/*
		read load balancer config
	*/
	lbConfigFile, err := os.Open("resources/config.json")
	if err != nil {
		log.Fatal("Failed to open load balancer config file: ", err)
	}
	defer lbConfigFile.Close()

	lbConfigFileByte, err := io.ReadAll(lbConfigFile)
	if err != nil {
		log.Fatal("Failed to read load balancer config file: ", err)
	}

	var lbConfig configJSON
	err = json.Unmarshal(lbConfigFileByte, &lbConfig)
	if err != nil {
		log.Fatal("Failed to unmarshal load balancer config: ", err)
	}

	port = lbConfig.Port
	numberOfRetries = lbConfig.NumberOfRetries
}

func makeRequest(rw http.ResponseWriter, req *http.Request, url *url.URL) error {
	fmt.Printf("[reverse proxy server] received request at: %s\n", time.Now())

	// set req Host, URL and Request URI to forward a request to the origin server
	req.Host = url.Host
	req.URL.Host = url.Host
	req.URL.Scheme = url.Scheme
	req.RequestURI = "" // https://go.dev/src/net/http/client.go:217

	// save the response from the origin server
	originServerResponse, err := http.DefaultClient.Do(req)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		_, _ = fmt.Print(rw, err)
		return err
	}

	for key, values := range originServerResponse.Header {
		for _, value := range values {
			rw.Header().Add(key, value)
		}
	}
	// return response to the client
	rw.WriteHeader(http.StatusOK)
	io.Copy(rw, originServerResponse.Body)

	return nil
}

func loadBalancer(rw http.ResponseWriter, req *http.Request) {
out:
	for i := 0; i < len(servers); i++ {
		url := servers[currentIndex]
		currentIndex++
		for j := 0; j < numberOfRetries; j++ {
			err := makeRequest(rw, req, url)
			if err == nil {
				break out
			}
		}

		if currentIndex == len(servers) {
			currentIndex = 0
		}
	}
}

func main() {
	readConfig()

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), http.HandlerFunc(loadBalancer)))
}

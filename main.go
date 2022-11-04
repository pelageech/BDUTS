package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"
)

type serverJSON struct {
	URL string
}

type configJSON struct {
	Port            int
	NumberOfRetries int
}

type index struct {
	currentIndex int
	mu           sync.Mutex
}

var servers []*url.URL
var serverCounter index

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

	serversFileByte, err := io.ReadAll(serversFile)
	if err != nil {
		log.Fatal("Failed to read servers config file: ", err)
	}

	err = serversFile.Close()
	if err != nil {
		log.Print("Failed to close servers config: ", err)
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

	lbConfigFileByte, err := io.ReadAll(lbConfigFile)
	if err != nil {
		log.Fatal("Failed to read load balancer config file: ", err)
	}

	err = lbConfigFile.Close()
	if err != nil {
		log.Print("Failed to close load balancer config file: ", err)
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

	// https://go.dev/src/net/http/client.go:217
	req.RequestURI = ""

	// save the response from the origin server
	originServerResponse, err := http.DefaultClient.Do(req)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		_, _ = fmt.Print(rw, err)
		return err
	}

	// copy headers from origin server response to our response
	for key, values := range originServerResponse.Header {
		for _, value := range values {
			rw.Header().Add(key, value)
		}
	}
	// return response to the client
	rw.WriteHeader(http.StatusOK)
	_, err = io.Copy(rw, originServerResponse.Body)

	return err
}

func calculateNextIndex() {
	serverCounter.mu.Lock()
	serverCounter.currentIndex++
	if serverCounter.currentIndex == len(servers) {
		serverCounter.currentIndex = 0
	}
	serverCounter.mu.Unlock()
}

func loadBalancer(rw http.ResponseWriter, req *http.Request) {
out:
	for i := 0; i < len(servers); i++ {
		server := servers[serverCounter.currentIndex]

		calculateNextIndex()

		for j := 0; j < numberOfRetries; j++ {
			err := makeRequest(rw, req, server)
			if err == nil {
				break out
			}
		}
	}
}

func main() {
	readConfig()

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), http.HandlerFunc(loadBalancer)))
}

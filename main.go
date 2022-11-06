package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"sync"
	"time"
)

type index struct {
	currentIndex int
	mu           sync.Mutex
}

var servers []*url.URL
var serverCounter index

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

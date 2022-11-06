package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"
	"time"
)

type Backend struct {
	URL   *url.URL
	mux   sync.Mutex
	alive bool
}

type ServerPool struct {
	servers []*Backend
	current int32
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

func (serverPool *ServerPool) GetNextPeer() (*Backend, error) {

	serverList := serverPool.servers

	current := atomic.AddInt32(&serverPool.current, 1)
	index := current % int32(len(serverList))

	for i := current; i < current+int32(len(serverList)); i++ {

		index = i % int32(len(serverList))
		if serverList[index].alive {
			if index != current {
				atomic.StoreInt32(&serverPool.current, index)
			}
			return serverList[index], nil
		}
	}

	return nil, errors.New("all the backends are turned down")
}

func loadBalancer(rw http.ResponseWriter, req *http.Request) {
	server, err := serverPool.GetNextPeer()

	if err != nil {
		http.Error(rw, err.Error(), http.StatusServiceUnavailable)
	}

	for j := 0; j < numberOfRetries; j++ {
		err := makeRequest(rw, req, server.URL)
		if err == nil {
			break
		}
	}
}

func main() {
	readConfig()
	serverPool.current = -1

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), http.HandlerFunc(loadBalancer)))
}

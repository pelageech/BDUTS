package main

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"
	"time"
)

type Backend struct {
	URL                   *url.URL
	healthCheckTcpTimeout time.Duration
	mux                   sync.Mutex
	alive                 bool
}

func (server *Backend) setAlive(b bool) {
	server.mux.Lock()
	server.alive = b
	server.mux.Unlock()
}

type ServerPool struct {
	servers []*Backend
	current int32
}

func makeRequest(rw http.ResponseWriter, req *http.Request, url *url.URL) error {
	log.Printf("[%s] received a request\n", url)

	// set req Host, URL and Request URI to forward a request to the origin server
	req.Host = url.Host
	req.URL.Host = url.Host
	req.URL.Scheme = url.Scheme

	// https://go.dev/src/net/http/client.go:217
	req.RequestURI = ""

	// save the response from the origin server
	originServerResponse, err := http.DefaultClient.Do(req)

	if err != nil {
		// rw.WriteHeader(http.StatusInternalServerError)
		// при переподключении http жалуется

		log.Println(err)
		//	fmt.Println(rw, err)
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

	log.Printf("[%s] returned %s\n", url, originServerResponse.Status)

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

	return nil, errors.New("all backends are turned down")
}

func faviconHandler(rw http.ResponseWriter, _ *http.Request) {
	http.Error(rw, "Not Found", http.StatusNotFound)
}

func loadBalancer(rw http.ResponseWriter, req *http.Request) {

	for attempt := 0; attempt < attempts; attempt++ {
		server, err := serverPool.GetNextPeer()
		if err != nil {
			http.Error(rw, "Service not available", http.StatusServiceUnavailable)
			log.Println(err.Error())
			return
		}

		for j := 0; j < retries; j++ {
			err := makeRequest(rw, req, server.URL)

			if err == nil {
				return
			}

			if uerr, ok := err.(*url.Error); ok {
				if uerr.Err == context.Canceled {
					return
				}
			}
		}

		server.setAlive(false)

	}

	http.Error(rw, "Cannot resolve a request", http.StatusInternalServerError)
}

func HealthChecker() {
	ticker := time.NewTicker(healthCheckPeriod)

	for {
		select {
		case <-ticker.C:
			log.Println("Health Check has been started!")
			healthCheck()
			log.Println("All the checks has been completed!")
		}
	}
}

func healthCheck() {
	for _, server := range serverPool.servers {
		alive := server.IsAlive()
		server.setAlive(alive)
		if alive {
			log.Printf("[%s] is alive.\n", server.URL.Host)
		} else {
			log.Printf("[%s] is down.\n", server.URL.Host)
		}
	}
}

func (server *Backend) IsAlive() bool {
	conn, err := net.DialTimeout("tcp", server.URL.Host, server.healthCheckTcpTimeout)
	if err != nil {
		log.Println("Connection problem: ", err)
		return false
	}
	defer conn.Close()
	return true
}

func main() {
	readConfig()

	// Config TLS: setting a pair crt-key
	Crt, _ := tls.LoadX509KeyPair("MyCertificate.crt", "MyKey.key")
	tlsConfig := &tls.Config{Certificates: []tls.Certificate{Crt}}

	// Start listening
	ln, err := tls.Listen("tcp", fmt.Sprintf(":%d", port), tlsConfig)
	if err != nil {
		log.Fatal("There's problem with listening")
	}

	// current is -1, it's automatically will turn into 0
	serverPool.current = -1

	// Serving
	http.HandleFunc("/", loadBalancer)
	http.HandleFunc("/favicon.ico", faviconHandler)

	// Firstly, identify the working servers
	log.Println("Configured! Now setting up the first health check...")
	healthCheck()
	log.Println("Ready!")

	// go health check
	go HealthChecker()

	log.Printf("Load Balancer started at :%d\n", port)
	if err := http.Serve(ln, nil); err != nil {
		log.Fatal(err)
	}
}

package main

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httptrace"
	"net/http/httputil"
	"net/url"
	"sync/atomic"
	"time"
)

// LoadBalancerConfig is parse from `config.json` file.
// It contains all the necessary information of the load balancer.
type LoadBalancerConfig struct {
	hostname          string
	port              int
	healthCheckPeriod time.Duration
}

type Backend struct {
	URL                   *url.URL
	healthCheckTcpTimeout time.Duration
	server                *httputil.ReverseProxy
	alive                 atomic.Bool
}

func (server *Backend) setAlive(b bool) {
	server.alive.Store(b)
}

type ServerPool struct {
	servers []*Backend
	current int32
}

type BackendResources struct {
	url      *url.URL
	connect  time.Duration
	response time.Duration
	transfer time.Duration
}

func makeRequestTimeTracker(req *http.Request, resources *BackendResources) {
	var start, connect, wroteReq time.Time

	trace := &httptrace.ClientTrace{

		ConnectStart: func(network, addr string) {
			connect = time.Now()
		},
		ConnectDone: func(network, addr string, err error) {
			resources.connect = time.Since(connect)
		},

		WroteRequest: func(_ httptrace.WroteRequestInfo) {
			wroteReq = time.Now()
		},

		GotFirstResponseByte: func() {
			now := time.Now()
			resources.transfer = now.Sub(start)
			resources.response = now.Sub(wroteReq)
		},
	}

	*req = *req.WithContext(httptrace.WithClientTrace(req.Context(), trace))
	start = time.Now()
}

func (serverPool *ServerPool) GetNextPeer() (*Backend, error) {

	serverList := serverPool.servers

	current := atomic.AddInt32(&serverPool.current, 1)
	index := current % int32(len(serverList))

	for i := current; i < current+int32(len(serverList)); i++ {

		index = i % int32(len(serverList))
		if serverList[index].alive.Load() {
			if index != current {
				atomic.StoreInt32(&serverPool.current, index)
			}
			return serverList[index], nil
		}
	}

	return nil, errors.New("all backends are turned down")
}

func loadBalancer(rw http.ResponseWriter, req *http.Request) {

	if maj, min, ok := http.ParseHTTPVersion(req.Proto); ok {
		if !(maj == 1 && min == 1) {
			http.Error(rw, "HTTP/1.1 required", http.StatusHTTPVersionNotSupported)
			log.Println(req.URL, "HTTP/1.1 required but was", req.Proto)
		}
	}

	// get next server to send a request
	backend, err := serverPool.GetNextPeer()
	if err != nil {
		http.Error(rw, "Service not available", http.StatusServiceUnavailable)
		log.Println(err.Error())
		return
	}

	resources := &BackendResources{}
	req = req.WithContext(context.Background())
	req = req.WithContext(context.WithValue(req.Context(), "resource", resources))
	log.Printf("[%s] received a request\n", backend.URL)
	makeRequestTimeTracker(req, resources)
	backend.server.ServeHTTP(rw, req)
}

func HealthChecker() {
	ticker := time.NewTicker(loadBalancerConfig.healthCheckPeriod)

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
	ln, err := tls.Listen("tcp", fmt.Sprintf(":%d", loadBalancerConfig.port), tlsConfig)
	if err != nil {
		log.Fatal("There's problem with listening")
	}

	// current is -1, it's automatically will turn into 0
	serverPool.current = -1

	// Serving
	http.HandleFunc("/", loadBalancer)
	http.HandleFunc("/favicon.ico", http.NotFound)

	// Firstly, identify the working servers
	log.Println("Configured! Now setting up the first health check...")
	healthCheck()

	log.Println("Ready!")

	// set up health check
	go HealthChecker()

	log.Printf("Load Balancer started at :%d\n", loadBalancerConfig.port)
	if err := http.Serve(ln, nil); err != nil {
		log.Fatal(err)
	}
}

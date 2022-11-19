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
	alive                 *atomic.Bool
}

func (server *Backend) setAlive(b bool) {
	server.alive.Store(b)
}

type ServerPool struct {
	servers []*Backend
	current int32
}

type ResponseError struct {
	request    *http.Request
	statusCode int
	err        error
}

func makeRequestTimeTracker(url *url.URL, req *http.Request) (*http.Request, *time.Duration) {
	var start, connect, dns time.Time
	var finish time.Duration

	trace := &httptrace.ClientTrace{
		DNSStart: func(dsi httptrace.DNSStartInfo) { dns = time.Now() },
		DNSDone: func(ddi httptrace.DNSDoneInfo) {
			fmt.Printf("[%s] DNS Done: %v\n", url, time.Since(dns))
		},

		ConnectStart: func(network, addr string) { connect = time.Now() },
		ConnectDone: func(network, addr string, err error) {
			fmt.Printf("[%s] Connect time: %v\n", url, time.Since(connect))
		},

		GotFirstResponseByte: func() {
			finish = time.Since(start)
			fmt.Printf("[%s] Time from start to first byte: %v\n", url, finish)
		},
	}

	req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))
	start = time.Now()

	return req, &finish
}

func (server *Backend) MakeRequest(req *http.Request) (*http.Response, *ResponseError) {
	respError := &ResponseError{request: req}
	serverUrl := server.URL

	// set req Host, URL and Request URI to forward a request to the origin server
	req.Host = serverUrl.Host
	req.URL.Host = serverUrl.Host
	req.URL.Scheme = serverUrl.Scheme

	// https://go.dev/src/net/http/client.go:217
	req.RequestURI = ""

	// save the response from the origin server
	req, _ = makeRequestTimeTracker(server.URL, req)
	originServerResponse, err := http.DefaultClient.Do(req)

	// error handler
	if err != nil {
		if uerr, ok := err.(*url.Error); ok {
			respError.err = uerr.Err

			if uerr.Err == context.Canceled {
				respError.statusCode = -1
			} else { // server error
				respError.statusCode = http.StatusInternalServerError
			}
		}
		return nil, respError
	}

	return originServerResponse, nil
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
	for {
		// get next server to send a request
		server, err := serverPool.GetNextPeer()
		if err != nil {
			http.Error(rw, "Service not available", http.StatusServiceUnavailable)
			log.Println(err.Error())
			return
		}

		// send it to the backend
		log.Printf("[%s] received a request\n", server.URL)
		resp, respError := server.MakeRequest(req)

		if respError != nil {
			// on cancellation
			if respError.err == context.Canceled {
				//	cancel()
				log.Printf("[%s] %s", server.URL, respError.err)
				return
			}

			server.setAlive(false) // СДЕЛАТЬ СЧЁТЧИК ИЛИ ПОЧИТАТЬ КАК У НДЖИНКС
			log.Println(respError.err)
			continue
		}

		// if resp != nil
		log.Printf("[%s] returned %s\n", server.URL, resp.Status)
		for key, values := range resp.Header {
			for _, value := range values {
				rw.Header().Add(key, value)
			}
		}

		if _, err := io.Copy(rw, resp.Body); err != nil {
			log.Panic(err)
		}

		return
	}
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

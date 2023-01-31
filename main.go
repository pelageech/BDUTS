package main

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/boltdb/bolt"
	"github.com/pelageech/BDUTS/cache"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"sync"
	"sync/atomic"
	"time"
)

// LoadBalancerConfig is parse from `config.json` file.
// It contains all the necessary information of the load balancer.
type LoadBalancerConfig struct {
	port              int
	healthCheckPeriod time.Duration
}

type Backend struct {
	URL                   *url.URL
	healthCheckTcpTimeout time.Duration
	mux                   sync.Mutex
	alive                 bool
	currentRequests       int32
	maximalRequests       int32
}

func (server *Backend) setAlive(b bool) {
	server.mux.Lock()
	server.alive = b
	server.mux.Unlock()
}

type ServerPool struct {
	mux     sync.Mutex
	servers []*Backend
	current int32
}

type ResponseError struct {
	request    *http.Request
	statusCode int
	err        error
}

var db *bolt.DB

func makeRequestTimeTracker(req *http.Request) (*http.Request, *time.Duration) {
	var start, connStart time.Time
	var finish, finishBackend time.Duration

	trace := &httptrace.ClientTrace{

		GotConn: func(_ httptrace.GotConnInfo) {
			connStart = time.Now()
		},

		GotFirstResponseByte: func() {
			finishBackend = time.Since(connStart)
			finish = time.Since(start)
			fmt.Printf("[%s] Time from start to first bytes: full trip: %v, backend: %v\n", req.URL, finish, finishBackend)
		},
	}

	req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))
	start = time.Now()

	return req, &finish
}

func (server *Backend) makeRequest(req *http.Request) (*http.Response, *ResponseError) {
	respError := &ResponseError{request: req}
	serverUrl := server.URL

	// set req Host, URL and Request URI to forward a request to the origin server
	req.Host = serverUrl.Host
	req.URL.Host = serverUrl.Host
	req.URL.Scheme = serverUrl.Scheme

	// https://go.dev/src/net/http/client.go:217
	req.RequestURI = ""

	// save the response from the origin server
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
	status := originServerResponse.StatusCode
	if status >= 500 && status < 600 &&
		status != http.StatusHTTPVersionNotSupported &&
		status != http.StatusNotImplemented {
		respError.statusCode = status
		return nil, respError
	}
	return originServerResponse, nil
}

func (serverPool *ServerPool) getNextPeer() (*Backend, error) {
	serverList := serverPool.servers

	serverPool.mux.Lock()
	defer serverPool.mux.Unlock()

	for i := 0; i < len(serverList); i++ {
		serverPool.current++
		if serverPool.current == int32(len(serverList)) {
			serverPool.current = 0
		}
		if serverList[serverPool.current].alive {
			return serverList[serverPool.current], nil
		}
	}

	return nil, errors.New("all backends are turned down")
}

func isHTTPVersionSupported(req *http.Request) bool {
	if maj, min, ok := http.ParseHTTPVersion(req.Proto); ok {
		if maj == 1 && min == 1 {
			return true
		}
	}
	return false
}

func loadBalancer(rw http.ResponseWriter, req *http.Request) {
	if !isHTTPVersionSupported(req) {
		http.Error(rw, "Expected HTTP/1.1", http.StatusHTTPVersionNotSupported)
	}

	req, _ = makeRequestTimeTracker(req)

	cache.GetCacheIfExists(nil, req)

	// on cache miss make request to backend
	for {
		// get next server to send a request
		var server *Backend
		var err error
		for {
			server, err = serverPool.getNextPeer()
			if server.currentRequests+1 <= server.maximalRequests {
				atomic.AddInt32(&server.currentRequests, int32(1))
				break
			}
		}

		if err != nil {
			http.Error(rw, "Service not available", http.StatusServiceUnavailable)
			log.Println(err.Error())
			return
		}

		// send it to the backend
		log.Printf("[%s] received a request\n", server.URL)
		resp, respError := server.makeRequest(req)

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
			http.Error(rw, "Internal Server Error", http.StatusInternalServerError)
		}

		atomic.AddInt32(&server.currentRequests, int32(-1))

		return
	}
}

func healthChecker() {
	ticker := time.NewTicker(loadBalancerConfig.healthCheckPeriod)

	for {
		<-ticker.C
		log.Println("Health Check has been started!")
		healthCheck()
		log.Println("All the checks has been completed!")
	}
}

func healthCheck() {
	for _, server := range serverPool.servers {
		alive := server.isAlive()
		server.setAlive(alive)
		if alive {
			log.Printf("[%s] is alive.\n", server.URL.Host)
		} else {
			log.Printf("[%s] is down.\n", server.URL.Host)
		}
	}
}

func (server *Backend) isAlive() bool {
	conn, err := net.DialTimeout("tcp", server.URL.Host, server.healthCheckTcpTimeout)

	if err != nil {
		log.Println("Connection problem: ", err)
		return false
	}

	defer func(conn net.Conn) {
		err := conn.Close()
		if err != nil {
			log.Println("Failed to close connection: ", err)
		}
	}(conn)
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
	go healthChecker()

	// opening db
	log.Println("Opening cache database")
	db, err = cache.OpenDatabase("./cache-data/database.db")
	if err != nil {
		log.Fatalln("DB error: ", err)
	}
	defer cache.CloseDatabase(db)

	log.Printf("Load Balancer started at :%d\n", loadBalancerConfig.port)
	if err := http.Serve(ln, nil); err != nil {
		log.Fatal(err)
	}
}

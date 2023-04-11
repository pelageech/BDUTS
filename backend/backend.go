package backend

import (
	"context"
	"github.com/pelageech/BDUTS/config"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"
)

type Backend struct {
	url                   *url.URL
	healthCheckTcpTimeout time.Duration
	mux                   sync.Mutex
	alive                 bool
	requestChan           chan bool
}

func NewBackend(url *url.URL, healthCheckTimeout time.Duration, maxRequests int32) *Backend {
	c := make(chan bool, maxRequests)
	return &Backend{
		url:                   url,
		healthCheckTcpTimeout: healthCheckTimeout,
		mux:                   sync.Mutex{},
		alive:                 false,
		requestChan:           c,
	}
}

func NewBackendConfig(server config.ServerConfig) *Backend {
	parsed, err := url.Parse(server.URL)
	if err != nil {
		log.Printf("Failed to parse server URL: %s\n", err)
		return nil
	}

	u := parsed
	h := time.Duration(server.HealthCheckTcpTimeout) * time.Millisecond
	max := server.MaximalRequests
	return NewBackend(u, h, max)
}

func (b *Backend) URL() *url.URL {
	return b.url
}

func (b *Backend) HealthCheckTcpTimeout() time.Duration {
	return b.healthCheckTcpTimeout
}

func (b *Backend) Lock() {
	b.mux.Lock()
}

func (b *Backend) Unlock() {
	b.mux.Unlock()
}

func (b *Backend) Alive() bool {
	return b.alive
}

// AssignRequest returns true if backend's channel is not full.
// Apart from that, channel gets a new item.
func (b *Backend) AssignRequest() bool {
	select {
	case b.requestChan <- true:
		return true
	default:
		return false
	}
}

func (b *Backend) Free() bool {
	select {
	case <-b.requestChan:
		return true
	default:
		return false
	}
}

type responseError struct {
	request    *http.Request
	statusCode int
	err        error
}

func (b *Backend) SetAlive(alive bool) {
	b.Lock()
	b.alive = alive
	b.Unlock()
}

func (b *Backend) CheckIfAlive() bool {
	conn, err := net.DialTimeout("tcp", b.URL().Host, b.HealthCheckTcpTimeout())
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

// SendRequestToBackend returns error if there is an error on backend side.
func (b *Backend) SendRequestToBackend(req *http.Request) (*http.Response, error) {
	log.Printf("[%s] received a request\n", b.URL())

	// send it to the backend
	r := b.prepareRequest(req)
	resp, respError := b.makeRequest(r)

	if respError != nil {
		return nil, respError.err
	}

	log.Printf("[%s] returned %s\n", b.URL(), resp.Status)

	return resp, nil
}

func WriteBodyAndReturn(rw http.ResponseWriter, resp *http.Response) ([]byte, error) {
	for key, values := range resp.Header {
		for _, value := range values {
			rw.Header().Add(key, value)
		}
	}

	byteArray, err := io.ReadAll(resp.Body)
	if err != nil && err != io.EOF {
		http.Error(rw, "Internal Server Error", http.StatusInternalServerError)
		return nil, err
	}
	resp.Body.Close()

	_, err = rw.Write(byteArray)
	if err != nil {
		http.Error(rw, "Internal Server Error", http.StatusInternalServerError)
	}
	return byteArray, nil
}

func (b *Backend) prepareRequest(r *http.Request) *http.Request {
	newReq := *r
	req := &newReq
	serverUrl := b.URL()

	// set req Host, URL and Request URI to forward a request to the origin b
	req.Host = serverUrl.Host
	req.URL.Host = serverUrl.Host
	req.URL.Scheme = serverUrl.Scheme

	// https://go.dev/src/net/http/client.go:217
	req.RequestURI = ""
	return req
}

func (b *Backend) makeRequest(req *http.Request) (*http.Response, *responseError) {
	respError := &responseError{request: req}

	// save the response from the origin b
	originServerResponse, err := http.DefaultClient.Do(req)

	// error handler
	if err != nil {
		if uerr, ok := err.(*url.Error); ok {
			respError.err = uerr.Err

			if uerr.Err == context.Canceled {
				respError.statusCode = -1
			} else { // b error
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

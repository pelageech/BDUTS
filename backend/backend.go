package backend

import (
	"context"
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/pelageech/BDUTS/config"
)

// Backend is a struct that contains all the configuration
// of the backend server.
type Backend struct {
	url                   *url.URL
	healthCheckTcpTimeout time.Duration
	mux                   sync.Mutex
	alive                 bool
	requestChan           chan bool
}

// NewBackend creates a new Backend.
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

// NewBackendConfig creates a new Backend from config.ServerConfig.
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

// URL returns the URL of the backend.
func (b *Backend) URL() *url.URL {
	return b.url
}

// HealthCheckTcpTimeout returns the timeout of the health check.
func (b *Backend) HealthCheckTcpTimeout() time.Duration {
	return b.healthCheckTcpTimeout
}

// Lock are used to lock the backend.
func (b *Backend) Lock() {
	b.mux.Lock()
}

// Unlock are used to unlock the backend.
func (b *Backend) Unlock() {
	b.mux.Unlock()
}

// Alive is used to check if the backend is alive.
func (b *Backend) Alive() bool {
	return b.alive
}

// AssignRequest returns true if backend's channel is not full.
// Apart from that, channel gets a new item.
func (b *Backend) AssignRequest() bool {
	select {
	case b.requestChan <- true:
		return true
	case <-time.After(100 * time.Millisecond):
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

// SetAlive sets the backend to alive or not alive.
func (b *Backend) SetAlive(alive bool) {
	b.Lock()
	b.alive = alive
	b.Unlock()
}

// CheckIfAlive checks if the backend is alive.
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

	return resp, nil
}

// WriteBodyAndReturn writes response to the client and returns the response body.
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
		respError.err = errors.New("Error: " + strconv.Itoa(originServerResponse.StatusCode))
		originServerResponse.Body.Close()

		return nil, respError
	}
	return originServerResponse, nil
}

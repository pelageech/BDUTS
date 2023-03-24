package backend

import (
	"context"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"
)

type Backend struct {
	URL                   *url.URL
	HealthCheckTcpTimeout time.Duration
	Mux                   sync.Mutex
	Alive                 bool
	RequestChan           chan bool
}

type responseError struct {
	request    *http.Request
	statusCode int
	err        error
}

func (b *Backend) SetAlive(alive bool) {
	b.Mux.Lock()
	b.Alive = alive
	b.Mux.Unlock()
}

func (b *Backend) IsAlive() bool {
	conn, err := net.DialTimeout("tcp", b.URL.Host, b.HealthCheckTcpTimeout)
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
	log.Printf("[%s] received a request\n", b.URL)

	// send it to the backend
	resp, respError := b.makeRequest(req)
	<-b.RequestChan

	if respError != nil {
		// on cancellation
		if respError.err == context.Canceled {
			//	cancel()
			log.Printf("[%s] %s", b.URL, respError.err)
			return nil, respError.err
		}

		b.SetAlive(false) // СДЕЛАТЬ СЧЁТЧИК ИЛИ ПОЧИТАТЬ КАК У НДЖИНКС
		return nil, respError.err
	}

	log.Printf("[%s] returned %s\n", b.URL, resp.Status)

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
	serverUrl := b.URL

	// set req Host, URL and Request URI to forward a request to the origin b
	req.Host = serverUrl.Host
	req.URL.Host = serverUrl.Host
	req.URL.Scheme = serverUrl.Scheme

	// https://go.dev/src/net/http/client.go:217
	req.RequestURI = ""
	return req
}

func (b *Backend) makeRequest(r *http.Request) (*http.Response, *responseError) {
	req := b.prepareRequest(r)
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

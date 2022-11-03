package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"
)

type server struct {
	url string
}

type config struct {
	port string
}

func main() {
	// define origin server URL
	originServerURL, err := url.Parse("http://127.0.0.1:3031")
	if err != nil {
		log.Fatal("invalid origin server URL")
	}

	loadBalancer := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		fmt.Printf("[reverse proxy server] received request at: %s\n", time.Now())

		// set req Host, URL and Request URI to forward a request to the origin server
		req.Host = originServerURL.Host
		req.URL.Host = originServerURL.Host
		req.URL.Scheme = originServerURL.Scheme
		req.RequestURI = "" // https://go.dev/src/net/http/client.go:217

		// save the response from the origin server
		originServerResponse, err := http.DefaultClient.Do(req)
		if err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			_, _ = fmt.Print(rw, err)
			return
		}

		for key, values := range originServerResponse.Header {
			for _, value := range values {
				rw.Header().Add(key, value)
			}
		}
		// return response to the client
		rw.WriteHeader(http.StatusOK)
		io.Copy(rw, originServerResponse.Body)
	})

	log.Fatal(http.ListenAndServe(":8080", loadBalancer))
}

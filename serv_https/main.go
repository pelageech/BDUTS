package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"time"
)

const maxClients = 2

var semaphore = make(chan struct{}, maxClients)

func hello(w http.ResponseWriter, req *http.Request) {
	requestReceived := time.Now()

	semaphore <- struct{}{}
	defer func() { <-semaphore }()

	time.Sleep(time.Second * 5)

	requestProcessed := time.Now()

	if _, err := fmt.Fprintln(w, "hello from 33"); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
	if _, err := fmt.Fprintf(w, "Request received: %s\n", requestReceived.String()); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
	if _, err := fmt.Fprintf(w, "Request processed: %s\n", requestProcessed.String()); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
	if _, err := fmt.Fprintf(w, "Request processing time: %s\n", requestProcessed.Sub(requestReceived).String()); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func main() {

	http.HandleFunc("/", hello)

	// config pairs
	Crt, _ := tls.LoadX509KeyPair("MyCertificate.crt", "MyKey.key")

	// create a var-config
	tlsconf := &tls.Config{Certificates: []tls.Certificate{Crt}}

	// start accept reqs
	ln, err := tls.Listen("tcp", ":3033", tlsconf)
	if err != nil {
		log.Fatal("There's problem on the lb")
	}

	// deal with reqs
	err = http.Serve(ln, nil)
	if err != nil {
		log.Fatal(err)
	}
}

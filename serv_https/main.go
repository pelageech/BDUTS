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
	semaphore <- struct{}{}
	defer func() { <-semaphore }()

	time.Sleep(time.Second * 5)
	fmt.Fprintf(w, "hello from 33\n")
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
	http.Serve(ln, nil)
}

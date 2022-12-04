package main

import (
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

	select {
	case <-time.After(5 * time.Second):
		requestProcessed := time.Now()

		if _, err := fmt.Fprintln(w, "hello from 3032"); err != nil {
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
	case <-req.Context().Done():
		log.Println("canceled")
	}

}

func main() {

	http.HandleFunc("/hello", hello)

	err := http.ListenAndServe(":3032", nil)
	if err != nil {
		log.Println(err)
		return
	}
}

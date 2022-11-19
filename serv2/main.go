package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"
)

func hello(w http.ResponseWriter, req *http.Request) {
	select {
	case <-time.After(5 * time.Second):
		fmt.Fprintf(w, "hello from 32\n")
	case <-req.Context().Done():
		log.Println(context.Canceled)
	}
}

func main() {

	http.HandleFunc("/hello", hello)

	http.ListenAndServe(":3032", nil)
}

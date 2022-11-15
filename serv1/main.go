package main

import (
	"fmt"
	"log"
	"net/http"
	"time"
)

func hello(w http.ResponseWriter, req *http.Request) {
	select {
	case <-time.After(5 * time.Second):
		w.Header().Add("server-name", "PORT_32_SERVER")
		w.Header().Add("header_test", "hahaha")
		w.Write([]byte("hello from 3031"))
	case <-req.Context().Done():
		err := req.Context().Err()
		fmt.Println("server:", err)
		internalError := http.StatusBadRequest
		http.Error(w, err.Error(), internalError)
	}
}

func main() {

	http.HandleFunc("/hello", hello)

	err := http.ListenAndServe(":3031", nil)
	if err != nil {
		log.Println(err)
		return
	}
}

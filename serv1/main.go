package main

import (
	"log"
	"net/http"
	"time"
)

func hello(w http.ResponseWriter, req *http.Request) {
	select {
	case <-time.After(10 * time.Second):
		w.Header().Add("server-name", "PORT_32_SERVER")
		w.Header().Add("header_test", "hahaha")
		if _, err := w.Write([]byte("hello from 3031")); err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
	case <-req.Context().Done():
		log.Println("canceled")
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

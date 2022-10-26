package main

import (
	"fmt"
	"net/http"
	"time"
)

func hello(w http.ResponseWriter, req *http.Request) {
	select {
	case <-time.After(5 * time.Second):
		fmt.Fprintf(w, "hello from 31\n")
	case <-req.Context().Done():
		err := req.Context().Err()
		fmt.Println("server:", err)
		internalError := http.StatusInternalServerError
		http.Error(w, err.Error(), internalError)
	}
}

func main() {

	http.HandleFunc("/", hello)

	http.ListenAndServe(":3031", nil)
}

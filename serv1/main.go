package main

import (
	"fmt"
	"net/http"
	"time"
)

func hello(w http.ResponseWriter, req *http.Request) {
	time.Sleep(time.Second * 5)
	fmt.Fprintf(w, "hello from 31\n")
}

func main() {

	http.HandleFunc("/", hello)

	http.ListenAndServe(":3031", nil)
}

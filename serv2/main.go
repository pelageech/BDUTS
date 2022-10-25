package main

import (
	"fmt"
	"net/http"
	"time"
)

func hello(w http.ResponseWriter, req *http.Request) {
	time.Sleep(5 * time.Second)
	fmt.Fprintf(w, "hello from 32\n")
}

func main() {

	http.HandleFunc("/", hello)

	http.ListenAndServe(":3032", nil)
}

package main

import (
	"log"
	"net/http"
	"time"
)

func hello(w http.ResponseWriter, req *http.Request) {
	select {
	case <-time.After(5 * time.Second):
		if _, err := w.Write([]byte("hello from 3032")); err != nil {
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

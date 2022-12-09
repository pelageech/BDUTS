package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
)

func hello(w http.ResponseWriter, req *http.Request) {
	log.Println("conn")
	e := exec.Command("Graphics-exe.exe", "Sin X" /*, "--width", "5000", "--height", "3000"*/)
	if err := e.Run(); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}

	str, err := os.ReadFile("./output.svg")
	if err == nil {
		if _, err := fmt.Fprintln(w, string(str)); err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
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

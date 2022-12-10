package main

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"os/exec"
)

func hello(w http.ResponseWriter, req *http.Request) {
	var stdout bytes.Buffer
	log.Println("conn")

	e := exec.Command("Graphics-exe.exe", "Sin X", "--width", "5000", "--height", "3000")
	e.Stdout = &stdout

	if err := e.Start(); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
	go func() {
		<-req.Context().Done()
		_ = e.Process.Kill()
	}()

	if err := e.Wait(); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}

	if _, err := fmt.Fprintln(w, stdout.String()); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
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

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
	f := req.FormValue("func")
	if f == "" {
		f = "Sin (Div (Num 1) X)"
	}

	e := exec.Command("./Graphics-exe.exe", f, "--width", "2000", "--height", "1500")
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

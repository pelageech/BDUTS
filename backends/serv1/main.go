package main

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"os/exec"
)

func index(w http.ResponseWriter, req *http.Request) {
	http.ServeFile(w, req, "backends/serv1/index.html")
}

func draw(w http.ResponseWriter, req *http.Request) {
	var stdout, stderr bytes.Buffer
	f := req.FormValue("func")
	if f == "" {
		f = "Sin ( Div (Num 1) X)"
	}

	e := exec.Command("./Graphics-exe.exe", f, "--width", "3200", "--height", "2000")
	e.Stdout = &stdout
	e.Stderr = &stderr

	if err := e.Start(); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	go func() {
		<-req.Context().Done()
		_ = e.Process.Kill()
	}()

	if err := e.Wait(); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	if _, err := fmt.Fprintln(w, stdout.String()); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

func main() {

	http.HandleFunc("/draw", draw)
	http.HandleFunc("/", index)

	err := http.ListenAndServe(":3031", nil)
	if err != nil {
		log.Println(err)
		return
	}
}

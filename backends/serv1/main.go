package main

import (
	"bytes"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os/exec"
)

func index(w http.ResponseWriter, req *http.Request) {
	http.ServeFile(w, req, "backends/serv1/index.html")
}

func drawRandom(w http.ResponseWriter, req *http.Request) {
	w.Header().Add("cache-control", "no-store")

	t := rand.Intn(20) - 10
	str := fmt.Sprintf("Sin (Mul X (Num (%d)))", t)
	fmt.Println(str)
	q := req.URL.Query()
	q.Set("func", str)
	req.URL.RawQuery = q.Encode()
	draw(w, req)
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
	http.HandleFunc("/random", drawRandom)
	http.HandleFunc("/", index)

	err := http.ListenAndServe(":3031", nil)
	if err != nil {
		log.Println(err)
		return
	}
}

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
	w.Header().Add("cache-control", "max-age=0;public")
	http.ServeFile(w, req, "backends/graphics_server/main.html")
}

func about(w http.ResponseWriter, req *http.Request) {
	w.Header().Add("cache-control", "max-age=0;public")
	http.ServeFile(w, req, "backends/graphics_server/about.html")
}

func drawRandom(w http.ResponseWriter, req *http.Request) {
	w.Header().Add("cache-control", "no-store")

	t := rand.Intn(20)
	str := fmt.Sprintf("Mul (Sin (Mul X (Num (%d)))) (Div (Num %d) (Num 10))", t, t)
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

	if _, err := fmt.Fprintln(w, `<a href="/">Return</a><br>`); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if _, err := fmt.Fprintln(w, stdout.String()); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

func main() {

	http.HandleFunc("/about", about)
	http.HandleFunc("/draw", draw)
	http.HandleFunc("/random", drawRandom)
	http.HandleFunc("/", index)
	http.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir("backends/graphics_server/assets"))))
	err := http.ListenAndServe(":3031", nil)
	if err != nil {
		log.Println(err)
		return
	}
}

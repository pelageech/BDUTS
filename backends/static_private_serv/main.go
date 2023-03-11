package main

import (
	"log"
	"net/http"
)

func main() {
	// Set the "Cache-Control" header to "private" with a max-age of 1 minute.
	cacheControl := "private, max-age=10"
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", cacheControl)
		http.ServeFile(w, r, "./content.mp4")
	})

	log.Println("Server listening on port 3037")
	// Start the server.
	err := http.ListenAndServe(":3037", nil)
	if err != nil {
		log.Fatal(err)
	}
}

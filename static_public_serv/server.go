package main

import (
	"log"
	"net/http"
	"time"
)

func main() {
	// Create a new http server with a custom handler.
	srv := &http.Server{
		Addr:         ":3036",
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Set the Cache-Control header to "public" to allow caching by intermediate proxies.
			w.Header().Set("Cache-Control", "public, max-age=3600") // 1 hour

			// Serve the requested file using the default file server.
			http.FileServer(http.Dir(".")).ServeHTTP(w, r)
		}),
	}

	// Start the server.
	log.Println("Server listening on port 3036...")
	if err := srv.ListenAndServe(); err != nil {
		log.Printf("Server error: %s\n", err)
	}
}

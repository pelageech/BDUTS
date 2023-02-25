package main

import (
	"fmt"
	"log"
	"net/http"
	"time"
)

func main() {
	// Create a new http server with a custom handler.
	srv := &http.Server{
		Addr:         ":3035",
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Disable caching by setting Cache-Control and Pragma headers.
			w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
			w.Header().Set("Pragma", "no-cache")

			// Set the content type header to indicate that we're returning plain text.
			w.Header().Set("Content-Type", "image/svg+xml")

			// Get the current time.
			t := time.Now()

			// Define the size of the clock.
			width := 200
			height := 100

			// Define the font size for the clock display.
			fontSize := 48

			// Generate the SVG for the clock.
			fmt.Fprintf(w, "<svg xmlns='http://www.w3.org/2000/svg' width='%d' height='%d'>\n", width, height)
			fmt.Fprintf(w, "  <rect width='100%%' height='100%%' fill='white' />\n")
			fmt.Fprintf(w, "  <text x='%d' y='%d' font-size='%d' text-anchor='middle'>%02d:%02d:%02d</text>\n", width/2, height-fontSize/2, fontSize, t.Hour(), t.Minute(), t.Second())
			fmt.Fprintf(w, "</svg>\n")
		}),
	}

	// Start the server.
	log.Println("Server listening on port 3035...")
	if err := srv.ListenAndServe(); err != nil {
		log.Printf("Server error: %s\n", err)
	}
}

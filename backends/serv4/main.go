package main

import (
	"fmt"
	"image"
	"image/color"
	"image/gif"
	"math/rand"
	"net/http"
	"strconv"
	"time"
)

func drawFractal(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Generating animated fractal...")

	const size = 1024
	const frames = 64
	const delay = 8

	q := r.URL.Query()
	seedStr := q.Get("seed")
	var seed int64
	if seedStr == "" {
		seed = time.Now().UnixNano()
	} else {
		var err error
		seed, err = strconv.ParseInt(seedStr, 10, 64)
		if err != nil {
			http.Error(w, "Invalid seed", http.StatusBadRequest)
			return
		}
	}

	src := rand.NewSource(seed)
	rng := rand.New(src)

	palette := make([]color.Color, 0, frames)
	for i := 0; i < frames; i++ {
		palette = append(palette, color.RGBA{R: uint8(rng.Intn(255)), G: uint8(rng.Intn(255)), B: uint8(rng.Intn(255)), A: 255})
	}

	anim := gif.GIF{LoopCount: frames}
	for i := 0; i < frames; i++ {
		img := image.NewRGBA(image.Rect(0, 0, size, size))
		for x := 0; x < size; x++ {
			for y := 0; y < size; y++ {
				// Perform time-consuming calculation here to determine color
				// for each pixel in the image
				img.Set(x, y, palette[rng.Intn(frames)])
			}
		}
		pImg := image.NewPaletted(image.Rect(0, 0, size, size), palette)
		for i := 0; i < size*size; i++ {
			pImg.Pix[i] = uint8(rng.Intn(frames))
		}
		anim.Image = append(anim.Image, pImg)
		anim.Delay = append(anim.Delay, delay)
	}

	err := gif.EncodeAll(w, &anim)
	if err != nil {
		http.Error(w, "Error encoding image", http.StatusInternalServerError)
		return
	}
}

func main() {
	fmt.Println("Starting server...")

	http.HandleFunc("/fractal.gif", drawFractal)

	err := http.ListenAndServe(":3034", nil)
	if err != nil {
		fmt.Println("Error starting server:", err)
	}
}

package main

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
	"golang.org/x/image/draw"
)

func resizeImage(data []byte, result chan<- string) {
	outputDir := "/data"
	origImage, err := jpeg.Decode(bytes.NewReader(data))
	if err != nil {
		log.Printf("Unable to decode image: %v", err)
		return
	}

	jobId := uuid.NewString()[:6]
	result <- jobId

	origX := origImage.Bounds().Max.X
	origY := origImage.Bounds().Max.Y
	newX := origX / 2
	newY := origY / 2

	log.Printf("Resizing image: (%d, %d) -> (%d, %d)\n", origX, origY, newX, newY)
	time.Sleep(10 * time.Second)
	newImage := image.NewRGBA(image.Rect(0, 0, newX, newY))
	draw.CatmullRom.Scale(newImage, newImage.Rect, origImage, origImage.Bounds(), draw.Over, nil)

	output, err := os.Create(fmt.Sprintf("%s/%s.jpg", outputDir, jobId))
	if err != nil {
		log.Printf("Unable to write image: %v", err)
		return
	}
	defer output.Close()

	if err := jpeg.Encode(output, newImage, nil); err != nil {
		log.Printf("could not encode image: %v", err)
		return
	}
	log.Println("Finished resizing image")
}

func uploadImage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST is supported", http.StatusMethodNotAllowed)
		return
	}
	log.Println("uploading an image for processing")

	data, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "could not read request body", http.StatusBadRequest)
		return
	}

	ch := make(chan string)
	go resizeImage(data, ch)
	job_id := <-ch

	fmt.Fprintf(w, "{job_id: %s}", job_id)
}

func main() {
	fmt.Println("starting up")
	http.HandleFunc("/upload", uploadImage)

	log.Fatal(http.ListenAndServe(":8080", nil))
}

package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

func main() {

	// Prepare client
	client := http.Client{
		Timeout: 5 * time.Second,
	}

	// Start timer
	start := time.Now()

	// Send request
	res, err := client.Get("http://www.google.com/robots.txt")

	// Stop timer
	latency := time.Since(start)

	// Handle error of request
	if err != nil {
		log.Fatalf("Request failed: %v", err)
	}
	// Close connection right after checking for error
	defer res.Body.Close()

	fmt.Printf("Latency is : %s\n", latency)

	// Read body and error if any
	body, err := io.ReadAll(res.Body)

	// Handle status code
	if res.StatusCode > 299 && body != nil {
		log.Printf("Response failed with status code: %d and\nbody: %s\n", res.StatusCode, body)
	}

	// Handle error body
	if err != nil {
		log.Print(err)
	}

	fmt.Printf("Response body : %s", body)
}

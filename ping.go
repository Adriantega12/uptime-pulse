package main

import (
	"io"
	"log"
	"net/http"
	"sync"
	"time"
)

type Result struct {
	URL        string
	StatusCode int
	Error      error
	Latency    time.Duration
	Timestamp  time.Time
}

func getLatency(url string) Result {
	// Prepare client
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	// Start timer
	start := time.Now()

	// Prepare and send request
	req, _ := http.NewRequest("GET", url, nil) // Ignoring error of request malformation for now
	res, err := client.Do(req)

	// Stop timer right after response or error is obtained
	latency := time.Since(start)

	// Handle error of request
	if err != nil {
		return Result{
			url,
			-1,
			err,
			latency,
			start,
		}
	}

	// Close connection right after checking for error
	defer res.Body.Close()

	// Read body and error if any
	body, err := io.ReadAll(res.Body)

	// Handle status code
	statusCode := res.StatusCode
	if statusCode > 299 && body != nil {
		log.Printf("Response failed with status code: %d and\nbody: %s\n", res.StatusCode, body)
	}

	// Handle error body
	if err != nil {
		log.Print(err)
	}

	return Result{
		url,
		statusCode,
		nil,
		latency,
		start,
	}
}

func main() {
	urlList := []string{
		"http://google.com/robots.txt",
		"https://fake.com/myfile.txt",
		"https://github.com",
	}

	// Create a channel to handle concurrency
	resultsChannel := make(chan Result, len(urlList))

	// Setup multi threaded execution of getLatency by using wg.Go
	var wg sync.WaitGroup
	for _, url := range urlList {
		wg.Go(func() {
			resultsChannel <- getLatency(url)
		})
	}

	// Wait for threads to wrap up
	wg.Wait()

	// Close channel
	close(resultsChannel)

	// Process results
	for result := range resultsChannel {
		log.Printf(
			"URL : %s, Status Code : %d, Error : %s, Latency : %s, Timestamp : %s\n",
			result.URL,
			result.StatusCode,
			result.Error,
			result.Latency,
			result.Timestamp,
		)
	}
}

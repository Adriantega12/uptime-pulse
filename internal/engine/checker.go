package engine

import (
	"io"
	"log"
	"net/http"
	"time"
)

type Result struct {
	URL        string
	StatusCode int
	Error      error
	Latency    time.Duration
	Timestamp  time.Time
}

func GetLatency(url string) Result {
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

	// Handle error of request, return as nothing else to do here
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

	// Read body and error, if any
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

type TargetPingsView struct {
	ID         int
	URL        string
	StatusCode int
	Error      string
	LatencyMs  int
	Timestamp  string
}

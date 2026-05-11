package main

import (
	"database/sql"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	_ "modernc.org/sqlite"
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

func initializeDatabase() *sql.DB {
	dir := "db"
	err := os.MkdirAll(dir, 0755)
	dbFilePath := filepath.Join(dir, "test.sqlite")
	db, err := sql.Open("sqlite", dbFilePath)
	if err != nil {
		log.Fatalf("Error while opening DB : %s", err)
	}

	err = db.Ping()
	if err != nil {
		log.Fatalf("Error while pinging DB : %s", err)
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS targets (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			url	TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE IF NOT EXISTS pings (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			target_id INTEGER NOT NULL,
			status_code INTEGER,
			latency_ms INTEGER,
			timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(target_id) REFERENCES targets(id)
		);
	`)
	if err != nil {
		log.Fatalf("Error while creating tables in DB : %s", err)
	}
	return db
}

func selectPingByUrl(db *sql.DB, url string) {
	stmt, err := db.Prepare("SELECT id FROM targets WHERE id=?")
	if err != nil {
		log.Fatal(err)
	}
	rows, err := stmt.Query(url)
	// sqlResult, err := stmt.Exec(url)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	log.Print(rows.Next())
}

func savePing(db *sql.DB, result Result) {

}

func saveResult(db *sql.DB, result Result) {
	stmt, err := db.Prepare("INSERT INTO targets(url) VALUES(?)")
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	if _, err := stmt.Exec(result.URL); err != nil {
		log.Fatal(err)
	}

}

func main() {
	db := initializeDatabase()
	urlList := []string{
		"http://google.com/robots.txt",
		"https://fakeurlthisdoesnotexists.com/myfile.txt",
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
		// log.Printf(
		// 	"URL : %s, Status Code : %d, Error : %s, Latency : %s, Timestamp : %s\n",
		// 	result.URL,
		// 	result.StatusCode,
		// 	result.Error,
		// 	result.Latency,
		// 	result.Timestamp,
		// )
		saveResult(db, result)
		selectPingByUrl(db, result.URL)
	}
}

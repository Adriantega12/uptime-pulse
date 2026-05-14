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
			url	TEXT UNIQUE NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE IF NOT EXISTS pings (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			target_id INTEGER NOT NULL,
			status_code INTEGER,
			error TEXT,
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

func selectTargetByUrl(db *sql.DB, url string) int {
	stmt, err := db.Prepare("SELECT * FROM targets WHERE url=?")
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	rows, err := stmt.Query(url)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	id := -1
	var targetURL string
	var createdAt string
	for rows.Next() {
		if err := rows.Scan(&id, &targetURL, &createdAt); err != nil {
			log.Printf("Did not find target due to %s", err)
			return id
		}
		log.Printf("Found Target in DB - ID: %d, URL: %s", id, targetURL)
	}

	return id
}

func saveTarget(db *sql.DB, url string) int {
	stmt, err := db.Prepare("INSERT INTO targets(url) VALUES(?)")
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	res, err := stmt.Exec(url)
	if err != nil {
		log.Fatal(err)
	}
	newId, _ := res.LastInsertId()
	return int(newId)
}

func saveResult(db *sql.DB, result Result) {
	// Check if URL has been pinged before
	url := result.URL
	targetId := selectTargetByUrl(db, url)
	if targetId == -1 {
		targetId = saveTarget(db, url)
	}

	stmt, err := db.Prepare("INSERT INTO pings(target_id, status_code, error, latency_ms) VALUES(?, ?, ?, ?)")
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	statusCode := result.StatusCode
	latencyMs := result.Latency
	var errMessage string
	if statusCode == -1 {
		errMessage = result.Error.Error()

	}
	if _, err := stmt.Exec(targetId, statusCode, errMessage, latencyMs); err != nil {
		log.Fatal(err)
	}
}

func main() {
	// debug.PrintStack()

	db := initializeDatabase()
	urlList := []string{
		"http://google.com/robots.txt",
		"https://fakeurlthisdoesnotexists.com/myfile.txt",
		"https://github.com",
	}

	// Create a channel to handle concurrency
	resultsChannel := make(chan Result, len(urlList))

	// Worker routine to write into DB
	var dbWg sync.WaitGroup
	dbWg.Go(func() {
		for result := range resultsChannel {
			saveResult(db, result)
		}
	})

	// Setup multi threaded execution of getLatency by using pingWg.Go
	var pingWg sync.WaitGroup
	for _, url := range urlList {
		pingWg.Go(func() {
			resultsChannel <- getLatency(url)
		})
	}

	// Faster procedure ends first
	// Wait for ping threads to wrap up
	pingWg.Wait()
	// Close channel
	close(resultsChannel)

	// Slower Disk I/O ends later because it depends on pingWg
	// Wait for worker threads to wrap up
	dbWg.Wait()
}

package main

import (
	"database/sql"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
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

type UptimeEngine struct {
	DB                        *sql.DB
	GetTargetByUrlStatement   *sql.Stmt
	SaveTargetStatement       *sql.Stmt
	SavePingStatement         *sql.Stmt
	GetAllTargetUrlsStatement *sql.Stmt
}

func NewUptimeEngine() *UptimeEngine {

	// 1. Handle all storage path setup
	dir := "db"
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		log.Fatalf("Error while trying to create directory : %s", err)
	}
	dbFilePath := filepath.Join(dir, "test.sqlite")
	db, err := sql.Open("sqlite", dbFilePath)
	if err != nil {
		log.Fatalf("Error while opening DB : %s", err)
	}

	err = db.Ping()
	if err != nil {
		log.Fatalf("Error while pinging DB : %s", err)
	}

	// 2. Create database schemas
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

	// 3. Pre-compile all prepared statements
	getTargetByUrlStatement, err := db.Prepare("SELECT id FROM targets WHERE url=?")
	if err != nil {
		log.Fatal(err)
	}
	saveTargetStatement, err := db.Prepare("INSERT INTO targets(url) VALUES(?)")
	if err != nil {
		log.Fatal(err)
	}
	savePingStatement, err := db.Prepare("INSERT INTO pings(target_id, status_code, error, latency_ms) VALUES(?, ?, ?, ?)")
	if err != nil {
		log.Fatal(err)
	}
	getAllTargetUrlsStatement, err := db.Prepare("SELECT url FROM targets")
	if err != nil {
		log.Fatal(err)
	}

	return &UptimeEngine{
		DB:                        db,
		GetTargetByUrlStatement:   getTargetByUrlStatement,
		SaveTargetStatement:       saveTargetStatement,
		SavePingStatement:         savePingStatement,
		GetAllTargetUrlsStatement: getAllTargetUrlsStatement,
	}
}

func (e *UptimeEngine) CloseUptimeEngineStatements() {
	e.GetTargetByUrlStatement.Close()
	e.SaveTargetStatement.Close()
	e.SavePingStatement.Close()
	e.GetAllTargetUrlsStatement.Close()
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

func (e *UptimeEngine) getTargetByUrl(url string) int {
	rows, err := e.GetTargetByUrlStatement.Query(url)
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
		log.Printf("Found Target in DB - ID: %d, URL: %s, created at: %s", id, targetURL, createdAt)
	}

	return id
}

func (e *UptimeEngine) saveTarget(url string) int {
	res, err := e.SaveTargetStatement.Exec(url)
	if err != nil {
		log.Fatal(err)
	}
	newId, _ := res.LastInsertId()
	return int(newId)
}

func (e *UptimeEngine) saveResult(result Result) {
	// Check if URL has been pinged before
	url := result.URL
	targetId := e.getTargetByUrl(url)
	if targetId == -1 {
		targetId = e.saveTarget(url)
	}

	statusCode := result.StatusCode
	latencyMs := result.Latency
	var errMessage string
	if statusCode == -1 {
		errMessage = result.Error.Error()

	}
	if _, err := e.SavePingStatement.Exec(targetId, statusCode, errMessage, latencyMs); err != nil {
		log.Fatal(err)
	}
}

func (e *UptimeEngine) getAllTargetUrls() []string {
	urlList := []string{}
	rows, err := e.GetAllTargetUrlsStatement.Query()
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	for rows.Next() {
		var url string
		if err := rows.Scan(&url); err != nil {
			log.Printf("Could not process URL due to %s", err)
		}
		urlList = append(urlList, url)
	}

	return urlList
}

func (e *UptimeEngine) doGetLatencyCallback() {
	// Load URL list of targets
	urlList := e.getAllTargetUrls()

	// Create a channel to handle concurrency
	resultsChannel := make(chan Result, len(urlList))

	// Worker routine to write into DB
	var dbWg sync.WaitGroup
	dbWg.Go(func() {
		for result := range resultsChannel {
			e.saveResult(result)
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

func main() {
	// debug.PrintStack()

	// Initialize DB
	uptimeEngine := NewUptimeEngine()

	// Channel to capture operating system termination interrupts (e.g., Ctrl+C, systemd stop)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	ticker := time.NewTicker(60 * time.Second)
	done := make(chan bool)
	defer ticker.Stop()

	go func() {
		for {
			select {
			case <-sigChan:
				fmt.Println("Done! Running cleanup...")
				uptimeEngine.CloseUptimeEngineStatements()
				done <- true
				return
			case t := <-ticker.C:
				fmt.Println("Current time: ", t)
				uptimeEngine.doGetLatencyCallback()
			}
		}
	}()

	<-done
	fmt.Println("Ticker stopped")
}

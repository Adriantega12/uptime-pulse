package engine

import (
	"database/sql"
	"log"
	"os"
	"path/filepath"
	"sync"
)

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
	getTargetByUrlStatement, err := db.Prepare("SELECT id, url, created_at FROM targets WHERE url=?")
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

func (e *UptimeEngine) DoGetLatencyCallback() {
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
			resultsChannel <- GetLatency(url)
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

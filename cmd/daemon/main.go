package main

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	"uptime-pulse/internal/engine"
	"uptime-pulse/internal/web"

	_ "modernc.org/sqlite"
)

func main() {
	// debug.PrintStack()

	// Initialize engines
	uptimeEngine := engine.NewUptimeEngine()
	webServer := web.NewServer(uptimeEngine)

	// Web server logic
	http.HandleFunc("/", webServer.HandleDashboard)
	http.HandleFunc("/api/pings", webServer.HandlePingsAPI)

	go func() {
		http.ListenAndServe(":8080", nil)
	}()

	// Uptime engine logic
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
				done <- true
				return
			case t := <-ticker.C:
				fmt.Println("Current time: ", t)
				uptimeEngine.DoGetLatencyCallback()
			}
		}
	}()

	<-done

	// Cleanup sequence
	uptimeEngine.CloseUptimeEngineStatements()
	fmt.Println("Ticker stopped")
}

package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"
	"uptime-pulse/internal/engine"

	_ "modernc.org/sqlite"
)

func main() {
	// debug.PrintStack()

	// Initialize DB
	uptimeEngine := engine.NewUptimeEngine()

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
				uptimeEngine.DoGetLatencyCallback()
			}
		}
	}()

	<-done
	fmt.Println("Ticker stopped")
}

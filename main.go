package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	// Create application
	app, err := NewApplication("8080")
	if err != nil {
		log.Fatalf("Failed to create application: %v", err)
	}
	defer app.Stop()

	// Start application
	if err := app.Start(); err != nil {
		log.Fatalf("Failed to start application: %v", err)
	}

	// Wait for interrupt signal
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig

	fmt.Println("Exiting...")
} 
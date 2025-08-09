package main

import (
	"os"
	"os/signal"
	"syscall"
)

func main() {
	// Create logger (stdout + rotating file as example)
	logger := NewStdoutAndFileLogger(10, 5, 7, false)

	// Create application
	app, err := NewApplication("8080", logger)
	if err != nil {
		logger.Errorf("Failed to create application: %v", err)
		os.Exit(1)
	}
	defer app.Stop()

	// Start application
	if err := app.Start(); err != nil {
		logger.Errorf("Failed to start application: %v", err)
		os.Exit(1)
	}

	// Wait for interrupt signal
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig

	logger.Infof("Exiting...")
} 
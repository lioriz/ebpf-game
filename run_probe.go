package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
)

// Application manages both eBPF monitoring and API server
type Application struct {
	ebpfMonitor *EBpfMonitor
	apiServer   *APIServer
}

// NewApplication creates a new application instance
func NewApplication(apiPort string) (*Application, error) {
	// Initialize eBPF monitor
	ebpfMonitor, err := NewEBpfMonitor()
	if err != nil {
		return nil, fmt.Errorf("failed to create eBPF monitor: %v", err)
	}

	// Initialize API server
	apiServer := NewAPIServer(apiPort, ebpfMonitor)

	return &Application{
		ebpfMonitor: ebpfMonitor,
		apiServer:   apiServer,
	}, nil
}

// Start begins both eBPF monitoring and API server
func (app *Application) Start() error {
	// Start eBPF monitoring
	app.ebpfMonitor.Start()

	// Start API server in a goroutine
	go func() {
		if err := app.apiServer.Start(); err != nil {
			log.Printf("API Server error: %v", err)
		}
	}()

	return nil
}

// Stop cleans up all resources
func (app *Application) Stop() {
	if app.ebpfMonitor != nil {
		app.ebpfMonitor.Stop()
	}
}

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
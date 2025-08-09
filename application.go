package main

import (
	"fmt"
	"log"
)

// Application manages eBPF monitoring, the command-processing app, and the API server.
type Application struct {
	ebpfMonitor *EBpfMonitor
	app        *ReadWriteMonitorApp
	cmdCh      chan MonitorCommand
	apiServer  *APIServer
}

// NewApplication creates a new application instance
func NewApplication(apiPort string) (*Application, error) {
	// Initialize eBPF monitor
	ebpfMonitor, err := NewEBpfMonitor()
	if err != nil {
		return nil, fmt.Errorf("failed to create eBPF monitor: %v", err)
	}

	// Shared command queue
	cmdCh := make(chan MonitorCommand, 256)

	// Initialize app (reads from queue and controls eBPF)
	appCore := NewReadWriteMonitorApp(ebpfMonitor, cmdCh)

	// Initialize API server (enqueues to queue, queries via app)
	apiServer := NewAPIServer(apiPort, cmdCh, appCore)

	return &Application{
		ebpfMonitor: ebpfMonitor,
		app:        appCore,
		cmdCh:      cmdCh,
		apiServer:  apiServer,
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
	if app.app != nil {
		app.app.Stop()
	}
	if app.ebpfMonitor != nil {
		app.ebpfMonitor.Stop()
	}
}
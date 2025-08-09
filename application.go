package main

import (
	"errors"
)

// Application manages eBPF monitoring, the command-processing app, and the API server.
type Application struct {
	logger         Logger
	ebpfProbe      *EBpfProbe
	ebpfController *EBpfController
	cmdCh          chan MonitorCommand
	apiServer      *APIServer
}

// NewApplication creates a new application instance
func NewApplication(apiPort string, logger Logger) (*Application, error) {
	// Initialize eBPF monitor
	ebpfProbe, err := NewEBpfProbe(logger)
	if err != nil {
		logger.Errorf("failed to create eBPF monitor: %v", err)
		return nil, errors.New("failed to create eBPF monitor: " + err.Error())
	}

	// Shared command queue
	cmdCh := make(chan MonitorCommand, 256)

	// Initialize controller (reads from queue and controls eBPF)
	ebpfController := NewEBpfController(logger, ebpfProbe, cmdCh)

	// Initialize API server (enqueues to queue, queries via controller)
	apiServer := NewAPIServer(apiPort, logger, cmdCh, ebpfController)

	return &Application{
		logger:         logger,
		ebpfProbe:      ebpfProbe,
		ebpfController: ebpfController,
		cmdCh:          cmdCh,
		apiServer:      apiServer,
	}, nil
}

// Start begins both eBPF monitoring and API server
func (app *Application) Start() error {
	// Start eBPF monitoring
	app.ebpfProbe.Start()

	// Start API server in a goroutine
	go func() {
		if err := app.apiServer.Start(); err != nil {
			app.logger.Errorf("API Server error: %v", err)
		}
	}()

	return nil
}

// Stop cleans up all resources
func (app *Application) Stop() {
	if app.ebpfController != nil {
		app.ebpfController.Stop()
	}
	if app.ebpfProbe != nil {
		app.ebpfProbe.Stop()
	}
}
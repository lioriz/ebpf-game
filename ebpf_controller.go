package main

import (
	"fmt"
)

type CommandKind int

const (
	CommandAddPID CommandKind = iota
	CommandClearPIDs
	CommandSetPrintAll
)

type MonitorCommand struct {
	Kind     CommandKind
	PID      uint32
	PrintAll bool
}

// EBpfController decouples API requests from the EBpfProbe via a command queue
// It owns a worker goroutine which processes commands sequentially
// to ensure consistent state updates.
type EBpfController struct {
	logger    Logger
	ebpfProbe *EBpfProbe
	cmdCh     chan MonitorCommand
	stopCh    chan struct{}
}

// NewEBpfController constructs the app given an ebpf monitor and a shared command queue
func NewEBpfController(logger Logger, ebpf *EBpfProbe, cmdCh chan MonitorCommand) *EBpfController {
	app := &EBpfController{
		logger:   logger,
		ebpfProbe: ebpf,
		cmdCh:     cmdCh,
		stopCh:    make(chan struct{}),
	}
	go app.run()
	return app
}

func (r *EBpfController) run() {
	for {
		select {
		case cmd := <-r.cmdCh:
			r.handle(cmd)
		case <-r.stopCh:
			return
		}
	}
}

func (r *EBpfController) handle(cmd MonitorCommand) {
	switch cmd.Kind {
	case CommandAddPID:
		if err := r.ebpfProbe.AddTargetPID(cmd.PID); err != nil {
			r.logger.Errorf("Failed to add PID %d: %v", cmd.PID, err)
		}
		if err := r.ebpfProbe.SetPrintAll(false); err != nil {
			r.logger.Errorf("Failed to set print_all false after add: %v", err)
		}
	case CommandClearPIDs:
		if err := r.ebpfProbe.ClearTargetPIDs(); err != nil {
			r.logger.Errorf("Failed to clear PIDs: %v", err)
		}
		if err := r.ebpfProbe.SetPrintAll(false); err != nil {
			r.logger.Errorf("Failed to set print_all false after clear: %v", err)
		}
	case CommandSetPrintAll:
		if err := r.ebpfProbe.SetPrintAll(cmd.PrintAll); err != nil {
			r.logger.Errorf("Failed to set print_all=%v: %v", cmd.PrintAll, err)
		}
	default:
		r.logger.Warnf("Unknown command kind: %v", cmd.Kind)
	}
}

// Query helpers pass-through to EBpfController for current state
func (r *EBpfController) GetTargetPIDs() ([]uint32, error) {
	return r.ebpfProbe.GetTargetPIDs()
}

func (r *EBpfController) GetPrintAllState() (bool, error) {
	return r.ebpfProbe.GetPrintAllState()
}

func (r *EBpfController) Stop() error {
	select {
	case <-r.stopCh:
		return nil
	default:
		close(r.stopCh)
	}
	return nil
}

func (r *EBpfController) String() string {
	return fmt.Sprintf("EBpfController(queue=%d)", len(r.cmdCh))
}
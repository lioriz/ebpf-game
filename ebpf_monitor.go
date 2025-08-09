package main

import (
	"fmt"
	"log"
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

// ReadWriteMonitorApp decouples API requests from the EBpfMonitor via a command queue
// It owns a worker goroutine which processes commands sequentially
// to ensure consistent state updates.
type ReadWriteMonitorApp struct {
	ebpfMonitor *EBpfMonitor
	cmdCh       chan MonitorCommand
	stopCh      chan struct{}
}

// NewReadWriteMonitorApp constructs the app given an ebpf monitor and a shared command queue
func NewReadWriteMonitorApp(ebpf *EBpfMonitor, cmdCh chan MonitorCommand) *ReadWriteMonitorApp {
	app := &ReadWriteMonitorApp{
		ebpfMonitor: ebpf,
		cmdCh:       cmdCh,
		stopCh:      make(chan struct{}),
	}
	go app.run()
	return app
}

func (r *ReadWriteMonitorApp) run() {
	for {
		select {
		case cmd := <-r.cmdCh:
			r.handle(cmd)
		case <-r.stopCh:
			return
		}
	}
}

func (r *ReadWriteMonitorApp) handle(cmd MonitorCommand) {
	switch cmd.Kind {
	case CommandAddPID:
		if err := r.ebpfMonitor.AddTargetPID(cmd.PID); err != nil {
			log.Printf("ReadWriteMonitorApp: failed to add PID %d: %v", cmd.PID, err)
		}
		// Per API contract: adding a PID sets print_all to false
		if err := r.ebpfMonitor.SetPrintAll(false); err != nil {
			log.Printf("ReadWriteMonitorApp: failed to set print_all false after add: %v", err)
		}
	case CommandClearPIDs:
		if err := r.ebpfMonitor.ClearTargetPIDs(); err != nil {
			log.Printf("ReadWriteMonitorApp: failed to clear PIDs: %v", err)
		}
		// Per API contract: clearing sets print_all to false
		if err := r.ebpfMonitor.SetPrintAll(false); err != nil {
			log.Printf("ReadWriteMonitorApp: failed to set print_all false after clear: %v", err)
		}
	case CommandSetPrintAll:
		if err := r.ebpfMonitor.SetPrintAll(cmd.PrintAll); err != nil {
			log.Printf("ReadWriteMonitorApp: failed to set print_all=%v: %v", cmd.PrintAll, err)
		}
	default:
		log.Printf("ReadWriteMonitorApp: unknown command kind: %v", cmd.Kind)
	}
}

// Query helpers pass-through to EBpfMonitor for current state
func (r *ReadWriteMonitorApp) GetTargetPIDs() ([]uint32, error) {
	return r.ebpfMonitor.GetTargetPIDs()
}

func (r *ReadWriteMonitorApp) GetPrintAllState() (bool, error) {
	return r.ebpfMonitor.GetPrintAllState()
}

func (r *ReadWriteMonitorApp) Stop() error {
	select {
	case <-r.stopCh:
		return nil
	default:
		close(r.stopCh)
	}
	return nil
}

func (r *ReadWriteMonitorApp) String() string {
	return fmt.Sprintf("ReadWriteMonitorApp(queue=%d)", len(r.cmdCh))
}
package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"os"
	"strings"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/link"
	"github.com/cilium/ebpf/perf"
)

// Data structure matching the C struct
// struct data_t { u32 pid; u32 event_type; } in ebpf_probe.c
type Data struct {
	Pid       uint32
	EventType uint32
}

const (
	evtRead  = 1
	evtWrite = 2
)

// EBpfProbe handles eBPF monitoring
type EBpfProbe struct {
	objs      *ebpf_probeObjects
	readLink  link.Link
	writeLink link.Link
	rd        *perf.Reader
	logger    Logger
	stopCh    chan struct{}
}

func findSyscallSymbol(base string, logger Logger) (string, error) {
    candidates := []string{
        "__x64_sys_" + base,
        "__arm64_sys_" + base,
        "__arm_sys_" + base,
        "sys_" + base,
    }
    data, err := os.ReadFile("/proc/kallsyms")
    if err != nil {
        return "", err
    }
    for _, cand := range candidates {
        if bytes.Contains(data, []byte(cand)) {
			logger.Infof("Found syscall symbol: %s", cand)
            return cand, nil
        }
    }
    return "", errors.New("no matching syscall symbol found for " + base)
}

// NewEBpfProbe creates a new eBPF monitor instance
func NewEBpfProbe(logger Logger) (*EBpfProbe, error) {
	// Load the eBPF program
	objs := ebpf_probeObjects{}
	if err := loadEbpf_probeObjects(&objs, nil); err != nil {
		logger.Errorf("failed to load eBPF objects: %v", err)
		return nil, errors.New("failed to load eBPF objects: " + err.Error())
	}

	// Skip this PID
	pid := uint32(os.Getpid())
	err := objs.SkipPid.Update(&pid, &pid, ebpf.UpdateAny)
	if err != nil {
		objs.Close()
		logger.Errorf("failed to set skip PID: %v", err)
		return nil, errors.New("failed to set skip PID: " + err.Error())
	}

	// Initialize print_all flag to 0 (disabled)
	flagKey := uint32(0)
	flagValue := uint32(0)
	err = objs.PrintAllFlag.Update(&flagKey, &flagValue, ebpf.UpdateAny)
	if err != nil {
		objs.Close()
		logger.Errorf("failed to initialize print_all flag: %v", err)
		return nil, errors.New("failed to initialize print_all flag: " + err.Error())
	}

	logger.Infof("Loading eBPF program")
	logger.Infof("Monitoring sys_read and sys_write calls...")
	logger.Infof("Host PID: %d", pid)
	logger.Infof("Skipping self PID: %d", pid)
	logger.Infof("Initial state: No PIDs in target list, print_all disabled")

	readSym, _ := findSyscallSymbol("read", logger)
	if readSym == "" {
		return nil, errors.New("no read syscall symbol found")
	}
	writeSym, _ := findSyscallSymbol("write", logger)
	if writeSym == "" {
		return nil, errors.New("no write syscall symbol found")
	}

	// Attach kprobes
	readLink, err := link.Kprobe(readSym, objs.SysReadCall, nil)
	if err != nil {
		objs.Close()
		logger.Errorf("failed to attach sys_read kprobe: %v", err)
		return nil, errors.New("failed to attach sys_read kprobe: " + err.Error())
	}

	writeLink, err := link.Kprobe(writeSym, objs.SysWriteCall, nil)
	if err != nil {
		readLink.Close()
		objs.Close()
		logger.Errorf("failed to attach sys_write kprobe: %v", err)
		return nil, errors.New("failed to attach sys_write kprobe: " + err.Error())
	}

	// Set up perf buffer (increase size to reduce drops)
	rd, err := perf.NewReader(objs.Events, 1<<18) // 256KB
	if err != nil {
		readLink.Close()
		writeLink.Close()
		objs.Close()
		logger.Errorf("failed to create perf reader: %v", err)
		return nil, errors.New("failed to create perf reader: " + err.Error())
	}

	return &EBpfProbe{
		objs:      &objs,
		readLink:  readLink,
		writeLink: writeLink,
		rd:        rd,
		logger:    logger,
		stopCh:    make(chan struct{}),
	}, nil
}

// Start begins monitoring
func (em *EBpfProbe) Start() {
	// Handle events
	go func() {
		for {
			select {
			case <-em.stopCh:
				return
			default:
			}

			record, err := em.rd.Read()
			if err != nil {
				// During shutdown, the reader may return various "file already closed" errors.
				if errors.Is(err, perf.ErrClosed) || errors.Is(err, os.ErrClosed) || strings.Contains(err.Error(), "file already closed") {
					return
				}
				em.logger.Errorf("Error reading perf event: %v", err)
				continue
			}

			if record.LostSamples != 0 {
				em.logger.Warnf("Lost %d samples", record.LostSamples)
				continue
			}

			if len(record.RawSample) >= 8 { // sizeof(Data)
				var event Data
				event.Pid = binary.LittleEndian.Uint32(record.RawSample[0:4])
				event.EventType = binary.LittleEndian.Uint32(record.RawSample[4:8])

				switch event.EventType {
				case evtRead:
					em.logger.Infof("%d - hello sys_read was called", event.Pid)
				case evtWrite:
					em.logger.Infof("%d - hello sys_write was called", event.Pid)
				default:
					em.logger.Infof("%d - unknown event %d", event.Pid, event.EventType)
				}
			}
		}
	}()
}

// Stop cleans up resources
func (em *EBpfProbe) Stop() {
	// Signal loop to stop before closing underlying resources
	select {
	case <-em.stopCh:
		// already closed
	default:
		close(em.stopCh)
	}
	if em.rd != nil {
		em.rd.Close()
	}
	if em.readLink != nil {
		em.readLink.Close()
	}
	if em.writeLink != nil {
		em.writeLink.Close()
	}
	if em.objs != nil {
		em.objs.Close()
	}
}

// AddTargetPID adds a PID to the target list
func (em *EBpfProbe) AddTargetPID(pid uint32) error {
	if em.objs == nil || em.objs.TargetPids == nil {
		em.logger.Errorf("eBPF objects not initialized")
		return errors.New("eBPF objects not initialized")
	}
	return em.objs.TargetPids.Update(&pid, &pid, ebpf.UpdateAny)
}

// RemoveTargetPID removes a PID from the target list
func (em *EBpfProbe) RemoveTargetPID(pid uint32) error {
	if em.objs == nil || em.objs.TargetPids == nil {
		em.logger.Errorf("eBPF objects not initialized")
		return errors.New("eBPF objects not initialized")
	}
	return em.objs.TargetPids.Delete(&pid)
}

// ClearTargetPIDs clears all target PIDs
func (em *EBpfProbe) ClearTargetPIDs() error {
	if em.objs == nil || em.objs.TargetPids == nil {
		em.logger.Errorf("eBPF objects not initialized")
		return errors.New("eBPF objects not initialized")
	}

	// Iterate and delete all entries
	iter := em.objs.TargetPids.Iterate()
	var key uint32
	var value uint32
	for iter.Next(&key, &value) {
		em.objs.TargetPids.Delete(&key)
	}

	if iter.Err() != nil {
		em.logger.Errorf("error clearing target PIDs: %v", iter.Err())
		return errors.New("error clearing target PIDs: " + iter.Err().Error())
	}

	return nil
}

// SetPrintAll sets the print_all flag
func (em *EBpfProbe) SetPrintAll(enabled bool) error {
	if em.objs == nil || em.objs.PrintAllFlag == nil {
		em.logger.Errorf("eBPF objects not initialized")
		return errors.New("eBPF objects not initialized")
	}

	flagKey := uint32(0)
	var flagValue uint32
	if enabled {
		flagValue = 1
	} else {
		flagValue = 0
	}
	return em.objs.PrintAllFlag.Update(&flagKey, &flagValue, ebpf.UpdateAny)
}

// GetTargetPIDs returns all target PIDs
func (em *EBpfProbe) GetTargetPIDs() ([]uint32, error) {
	if em.objs == nil || em.objs.TargetPids == nil {
		em.logger.Errorf("eBPF objects not initialized")
		return []uint32{}, errors.New("eBPF objects not initialized")
	}

	pids := make([]uint32, 0)
	iter := em.objs.TargetPids.Iterate()
	var key uint32
	var value uint32
	for iter.Next(&key, &value) {
		pids = append(pids, key)
	}

	if iter.Err() != nil {
		em.logger.Errorf("error iterating target PIDs: %v", iter.Err())
		return pids, errors.New("error iterating target PIDs: " + iter.Err().Error())
	}

	return pids, nil
}

// GetPrintAllState returns the current print_all flag state
func (em *EBpfProbe) GetPrintAllState() (bool, error) {
	if em.objs == nil || em.objs.PrintAllFlag == nil {
		em.logger.Errorf("eBPF objects not initialized")
		return false, errors.New("eBPF objects not initialized")
	}

	flagKey := uint32(0)
	var printAllFlag uint32
	if err := em.objs.PrintAllFlag.Lookup(&flagKey, &printAllFlag); err != nil {
		em.logger.Errorf("failed to lookup print_all flag: %v", err)
		return false, errors.New("failed to lookup print_all flag: " + err.Error())
	}

	return printAllFlag == 1, nil
}
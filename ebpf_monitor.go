package main

import (
	"fmt"
	"log"
	"os"
	"unsafe"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/link"
	"github.com/cilium/ebpf/perf"
)

// Data structure matching the C struct
type Data struct {
	Pid uint32
	Msg [32]byte
}

// EBpfMonitor handles eBPF monitoring
type EBpfMonitor struct {
	objs      *ebpf_probeObjects
	readLink  link.Link
	writeLink link.Link
	rd        *perf.Reader
}

// NewEBpfMonitor creates a new eBPF monitor instance
func NewEBpfMonitor() (*EBpfMonitor, error) {
	// Load the eBPF program
	objs := ebpf_probeObjects{}
	if err := loadEbpf_probeObjects(&objs, nil); err != nil {
		return nil, fmt.Errorf("failed to load eBPF objects: %v", err)
	}

	// Skip this PID
	pid := uint32(os.Getpid())
	err := objs.SkipPid.Update(&pid, &pid, ebpf.UpdateAny)
	if err != nil {
		objs.Close()
		return nil, fmt.Errorf("failed to set skip PID: %v", err)
	}

	// Initialize print_all flag to 0 (disabled)
	flagKey := uint32(0)
	flagValue := uint32(0)
	err = objs.PrintAllFlag.Update(&flagKey, &flagValue, ebpf.UpdateAny)
	if err != nil {
		objs.Close()
		return nil, fmt.Errorf("failed to initialize print_all flag: %v", err)
	}

	fmt.Println("Loading eBPF program")
	fmt.Println("Monitoring sys_read and sys_write calls...")
	fmt.Printf("Host PID: %d\n", pid)
	fmt.Printf("Skipping self PID: %d\n", pid)
	fmt.Println("Initial state: No PIDs in target list, print_all disabled")

	// Attach kprobes
	readLink, err := link.Kprobe("__x64_sys_read", objs.SysReadCall, nil)
	if err != nil {
		objs.Close()
		return nil, fmt.Errorf("failed to attach sys_read kprobe: %v", err)
	}

	writeLink, err := link.Kprobe("__x64_sys_write", objs.SysWriteCall, nil)
	if err != nil {
		readLink.Close()
		objs.Close()
		return nil, fmt.Errorf("failed to attach sys_write kprobe: %v", err)
	}

	// Set up perf buffer
	rd, err := perf.NewReader(objs.Events, 4096)
	if err != nil {
		readLink.Close()
		writeLink.Close()
		objs.Close()
		return nil, fmt.Errorf("failed to create perf reader: %v", err)
	}

	return &EBpfMonitor{
		objs:      &objs,
		readLink:  readLink,
		writeLink: writeLink,
		rd:        rd,
	}, nil
}

// Start begins monitoring
func (em *EBpfMonitor) Start() {
	// Handle events
	go func() {
		for {
			record, err := em.rd.Read()
			if err != nil {
				if err == perf.ErrClosed {
					return
				}
				log.Printf("Error reading perf event: %v", err)
				continue
			}

			if record.LostSamples != 0 {
				log.Printf("Lost %d samples", record.LostSamples)
				continue
			}

			var event Data
			if len(record.RawSample) >= 36 { // sizeof(Data)
				copy((*[36]byte)(unsafe.Pointer(&event))[:], record.RawSample[:36])
				fmt.Printf("%d - %s\n", event.Pid, string(event.Msg[:]))
			}
		}
	}()
}

// Stop cleans up resources
func (em *EBpfMonitor) Stop() {
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
func (em *EBpfMonitor) AddTargetPID(pid uint32) error {
	if em.objs == nil || em.objs.TargetPids == nil {
		return fmt.Errorf("eBPF objects not initialized")
	}
	return em.objs.TargetPids.Update(&pid, &pid, ebpf.UpdateAny)
}

// RemoveTargetPID removes a PID from the target list
func (em *EBpfMonitor) RemoveTargetPID(pid uint32) error {
	if em.objs == nil || em.objs.TargetPids == nil {
		return fmt.Errorf("eBPF objects not initialized")
	}
	return em.objs.TargetPids.Delete(&pid)
}

// ClearTargetPIDs clears all target PIDs
func (em *EBpfMonitor) ClearTargetPIDs() error {
	if em.objs == nil || em.objs.TargetPids == nil {
		return fmt.Errorf("eBPF objects not initialized")
	}

	// Iterate and delete all entries
	iter := em.objs.TargetPids.Iterate()
	var key uint32
	var value uint32
	for iter.Next(&key, &value) {
		em.objs.TargetPids.Delete(&key)
	}

	if iter.Err() != nil {
		return fmt.Errorf("error clearing target PIDs: %v", iter.Err())
	}

	return nil
}

// SetPrintAll sets the print_all flag
func (em *EBpfMonitor) SetPrintAll(enabled bool) error {
	if em.objs == nil || em.objs.PrintAllFlag == nil {
		return fmt.Errorf("eBPF objects not initialized")
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
func (em *EBpfMonitor) GetTargetPIDs() ([]uint32, error) {
	if em.objs == nil || em.objs.TargetPids == nil {
		return []uint32{}, fmt.Errorf("eBPF objects not initialized")
	}

	pids := make([]uint32, 0)
	iter := em.objs.TargetPids.Iterate()
	var key uint32
	var value uint32
	for iter.Next(&key, &value) {
		pids = append(pids, key)
	}

	if iter.Err() != nil {
		return pids, fmt.Errorf("error iterating target PIDs: %v", iter.Err())
	}

	return pids, nil
}

// GetPrintAllState returns the current print_all flag state
func (em *EBpfMonitor) GetPrintAllState() (bool, error) {
	if em.objs == nil || em.objs.PrintAllFlag == nil {
		return false, fmt.Errorf("eBPF objects not initialized")
	}

	flagKey := uint32(0)
	var printAllFlag uint32
	err := em.objs.PrintAllFlag.Lookup(&flagKey, &printAllFlag)
	if err != nil {
		return false, fmt.Errorf("failed to lookup print_all flag: %v", err)
	}

	return printAllFlag == 1, nil
}
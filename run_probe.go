package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
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

	fmt.Println("Loading eBPF program")
	fmt.Println("Monitoring sys_read and sys_write calls...")
	fmt.Printf("Host PID: %d\n", pid)
	fmt.Printf("Skipping self PID: %d\n", pid)

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
	apiServer := NewAPIServer(apiPort)

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
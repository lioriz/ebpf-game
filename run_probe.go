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

func main() {
	// Load the eBPF program
	objs := ebpf_probeObjects{}
	if err := loadEbpf_probeObjects(&objs, nil); err != nil {
		log.Fatalf("Failed to load eBPF objects: %v", err)
	}
	defer objs.Close()

	// Skip this PID
	pid := uint32(os.Getpid())
	err := objs.SkipPid.Update(&pid, &pid, ebpf.UpdateAny)
	if err != nil {
		log.Fatalf("Failed to set skip PID: %v", err)
	}

	fmt.Println("Loading eBPF program")
	fmt.Println("Monitoring sys_read and sys_write calls...")
	fmt.Printf("Host PID: %d\n", pid)
	fmt.Printf("Skipping self PID: %d\n", pid)

	// Attach kprobes
	readLink, err := link.Kprobe("__x64_sys_read", objs.SysReadCall, nil)
	if err != nil {
		log.Fatalf("Failed to attach sys_read kprobe: %v", err)
	}
	defer readLink.Close()

	writeLink, err := link.Kprobe("__x64_sys_write", objs.SysWriteCall, nil)
	if err != nil {
		log.Fatalf("Failed to attach sys_write kprobe: %v", err)
	}
	defer writeLink.Close()

	// Set up perf buffer
	rd, err := perf.NewReader(objs.Events, 4096)
	if err != nil {
		log.Fatalf("Failed to create perf reader: %v", err)
	}
	defer rd.Close()

	// Handle events
	go func() {
		for {
			record, err := rd.Read()
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

	// Wait for interrupt signal
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig

	fmt.Println("Exiting...")
} 
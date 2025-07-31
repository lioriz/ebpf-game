package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/iovisor/gobpf/bcc"
)

const source string = `
#include <uapi/linux/ptrace.h>

BPF_PERF_OUTPUT(events);

int sys_read_probe(struct pt_regs *ctx) {
    char msg[] = "hello sys_read was called";
    events.perf_submit(ctx, &msg, sizeof(msg));
    return 0;
}

int sys_write_probe(struct pt_regs *ctx) {
    char msg[] = "hello sys_write was called";
    events.perf_submit(ctx, &msg, sizeof(msg));
    return 0;
}
`

func main() {
	// Create BPF module
	m := bcc.NewModule(source, []string{})
	defer m.Close()

	// Load kprobes
	sysReadKprobe, err := m.LoadKprobe("sys_read_probe")
	if err != nil {
		log.Fatalf("Failed to load sys_read kprobe: %v", err)
	}

	sysWriteKprobe, err := m.LoadKprobe("sys_write_probe")
	if err != nil {
		log.Fatalf("Failed to load sys_write kprobe: %v", err)
	}

	// Attach kprobes
	err = m.AttachKprobe("sys_read", sysReadKprobe, -1)
	if err != nil {
		log.Fatalf("Failed to attach sys_read kprobe: %v", err)
	}

	err = m.AttachKprobe("sys_write", sysWriteKprobe, -1)
	if err != nil {
		log.Fatalf("Failed to attach sys_write kprobe: %v", err)
	}

	// Open perf buffer
	table := bcc.NewTable(m.TableId("events"), m)
	channel := make(chan []byte)
	perfMap, err := bcc.InitPerfMap(table, channel, nil)
	if err != nil {
		log.Fatalf("Failed to init perf map: %v", err)
	}

	perfMap.Start()
	defer perfMap.Stop()

	fmt.Println("eBPF program loaded successfully!")
	fmt.Println("Monitoring sys_read and sys_write calls...")
	fmt.Println("Press Ctrl+C to exit")

	// Handle events
	go func() {
		for {
			data := <-channel
			msg := string(data)
			fmt.Println(msg)
		}
	}()

	// Wait for interrupt signal
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	fmt.Println("\nExiting...")
} 
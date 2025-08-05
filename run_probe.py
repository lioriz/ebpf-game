#!/usr/bin/env python3
from bcc import BPF

# Load the eBPF program
b = BPF(src_file="ebpf_probe.c")

# Attach kprobes
b.attach_kprobe(event="sys_read", fn_name="sys_read_enter")
b.attach_kprobe(event="sys_write", fn_name="sys_write_enter")

print("eBPF program loaded. Press Ctrl+C to exit.")
print("Monitoring sys_read and sys_write calls...")

try:
    b.trace_print()
except KeyboardInterrupt:
    print("\nExiting...") 
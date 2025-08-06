from bcc import BPF
import ctypes
import sys
import time
import os


# Clear old traces
os.system("echo > /sys/kernel/debug/tracing/trace")

# Load the eBPF program
b = BPF(src_file="ebpf_probe.c")

# Map to skip self process
skip_map = b["skip_pid"]
pid = os.getpid()
c_pid = ctypes.c_uint(pid)
skip_map[c_pid] = c_pid

print("Loding eBPF program")
print("Monitoring sys_read and sys_write calls...")
print(f"Host PID: {pid}")
print(f"Skipping self PID: {pid}")

# Attach kprobes
b.attach_kprobe(event="__x64_sys_read", fn_name="sys_read_enter")
b.attach_kprobe(event="__x64_sys_write", fn_name="sys_write_enter")

try:
    b.trace_print()
except KeyboardInterrupt:
    print("Exiting...")

from bcc import BPF
import ctypes
import sys
import time
import os

# Load the eBPF program
b = BPF(src_file="ebpf_probe.c")
# b["events"].open_perf_buffer(print_event, page_cnt=256)

print("Loding eBPF program. Press Ctrl+C to exit.")
print("Monitoring sys_read and sys_write calls...")

# Attach kprobes
b.attach_kprobe(event="__x64_sys_read", fn_name="sys_read_enter")
b.attach_kprobe(event="__x64_sys_write", fn_name="sys_write_enter")

# Map to skip self process
skip_map = b["skip_pid"]

pid = os.getpid()
print(f"Host PID: {pid}")
print(f"Skipping self PID: {pid}")
c_pid = ctypes.c_uint(pid)
skip_map[c_pid] = c_pid  # BPF maps require ctypes

try:
    b.trace_print()
except KeyboardInterrupt:
    print("Exiting...")

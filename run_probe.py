from bcc import BPF
import ctypes
import os

# Clear old traces
os.system("echo > /sys/kernel/debug/tracing/trace")

# Load the eBPF program
b = BPF(src_file="ebpf_probe.c")

# Skip this PID
skip_map = b["skip_pid"]
pid = os.getpid()
skip_map[ctypes.c_uint(pid)] = ctypes.c_uint(pid)

print("Loading eBPF program")
print("Monitoring sys_read and sys_write calls...")
print(f"Host PID: {pid}")
print(f"Skipping self PID: {pid}")

# Attach kprobes
b.attach_kprobe(event="__x64_sys_read", fn_name="sys_read_call")
b.attach_kprobe(event="__x64_sys_write", fn_name="sys_write_call")

# Define event data structure
class Data(ctypes.Structure):
    _fields_ = [("pid", ctypes.c_uint),
    ("msg", ctypes.c_char * 32)]

def print_event(cpu, data, size):
    event = ctypes.cast(data, ctypes.POINTER(Data)).contents
    print(f"{event.pid} - {event.msg.decode('utf-8', 'replace')}")

# Set up perf buffer
b["events"].open_perf_buffer(print_event)

# Poll for events
try:
    while True:
        b.perf_buffer_poll()
except KeyboardInterrupt:
    print("Exiting...")

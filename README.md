# eBPF System Call Monitor

This project demonstrates real-time monitoring of system calls using eBPF (Extended Berkeley Packet Filter) technology. It tracks `sys_read` and `sys_write` system calls across all processes on the system, providing insights into file I/O activity.

## Prerequisites

### System Requirements
- **Linux machine or VM** (Ubuntu 24.04 recommended)
- Docker and Docker Compose installed

## Quick Start

### Build and Run

From the project root dir

```bash
# Build the Docker image
docker-compose build

# Run the eBPF monitor
docker-compose up

# Run in background (detached mode)
docker-compose up -d
```

### Stop the Monitor
```bash
# Stop the running container
docker-compose down

# Stop and remove containers, networks, and images
docker-compose down --rmi all
```

## How It Works

This project uses eBPF to attach kernel probes to system call entry points:

1. **Kernel Probes**: Attaches to `__x64_sys_read` and `__x64_sys_write` kernel functions
2. **Event Filtering**: Skips monitoring the monitoring process itself to avoid infinite loops
3. **Real-time Output**: Uses perf events to stream monitoring data to userspace
4. **Process Tracking**: Captures the PID of each process making system calls

### What You'll See

When running, you'll see output like:
```
Loading eBPF program
Monitoring sys_read and sys_write calls...
Host PID: 12345
Skipping self PID: 12345
45678 - hello sys_read was called
78901 - hello sys_write was called
```

Each line shows:
- **PID**: The process ID making the system call
- **Message**: Indicates which system call was intercepted

## Project Structure

```
ebpf-game/
├── docker-compose.yml    # Container orchestration
├── Dockerfile           # Container definition
├── ebpf_probe.c        # eBPF C program
├── run_probe.py        # Python loader and event handler
└── README.md           # This file
```

## Technical Details

- **eBPF Program**: Written in C, compiled to eBPF bytecode
- **Kernel Probes**: Uses kprobes to attach to kernel functions
- **Event Handling**: Uses perf events for efficient data transfer
- **Containerization**: Runs in privileged Docker container for kernel access

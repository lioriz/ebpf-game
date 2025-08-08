# eBPF System Call Monitor with API Server

This project demonstrates real-time monitoring of system calls using eBPF (Extended Berkeley Packet Filter) technology, combined with a REST API server for managing IP lists. It tracks `sys_read` and `sys_write` system calls across all processes on the system, providing insights into file I/O activity.

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

# Run the eBPF monitor with API server
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
5. **API Server**: Provides REST API endpoints for IP list management

### What You'll See

When running, you'll see output like:
```
Loading eBPF program
Monitoring sys_read and sys_write calls...
Host PID: 12345
Skipping self PID: 12345
API Server starting on localhost:8080
45678 - hello sys_read was called
78901 - hello sys_write was called
```

Each line shows:
- **PID**: The process ID making the system call
- **Message**: Indicates which system call was intercepted

## API Endpoints

The application includes a REST API server running on port 8080:

### GET /apis
Get all available API endpoints and usage information.

### POST /add_ips
Add IPs to the list.
```json
{"add_ips": [1,2,3]}
```

### POST /clear_ip_list
Clear all IPs from the list.
```json
{"clear_ip_list": true}
```

### POST /print_all_ips
Get all IPs in the list.
```json
{"print_all_ips": true}
```

## Testing the API

Use the provided test script:
```bash
chmod +x test_api.sh
./test_api.sh
```

Or test manually with curl:
```bash
# Get available APIs
curl http://localhost:8080/apis

# Add IPs
curl -X POST http://localhost:8080/add_ips \
  -H "Content-Type: application/json" \
  -d '{"add_ips": [1,2,3]}'

# Print all IPs
curl -X POST http://localhost:8080/print_all_ips \
  -H "Content-Type: application/json" \
  -d '{"print_all_ips": true}'
```

## Project Structure

```
ebpf-game/
├── docker-compose.yml    # Container orchestration
├── Dockerfile           # Multi-stage container definition
├── ebpf_probe.c        # eBPF C program
├── run_probe.go        # Main application (eBPF + API)
├── api_server.go       # API server implementation
├── go.mod              # Go module definition
├── test_api.sh         # API testing script
└── README.md           # This file
```

## Technical Details

- **eBPF Program**: Written in C, compiled to eBPF bytecode using bpf2go
- **Go Implementation**: Uses `github.com/cilium/ebpf` library for modern eBPF interaction
- **API Server**: Uses Gin framework for REST API endpoints
- **Build Process**: Multi-stage Docker build with bpf2go compilation
- **Kernel Probes**: Uses kprobes to attach to kernel functions
- **Event Handling**: Uses perf events for efficient data transfer
- **Containerization**: Runs in minimal Alpine container for efficiency
- **Object-Oriented Design**: Clean separation between eBPF monitoring and API server

## Development

### Local Development (if Go is installed)

```bash
# Install bpf2go tool
go install github.com/cilium/ebpf/cmd/bpf2go@latest

# Generate Go code from C
bpf2go -cc clang -cflags "-g -O2 -Wall" -target bpf -go-package main -output-stem ebpf_probe ebpf_probe ebpf_probe.c

# Install dependencies
go mod tidy

# Build the application
go build -o run_probe .

# Run (requires root privileges)
sudo ./run_probe
```

## Architecture

The application follows an object-oriented design:

- **EBpfMonitor**: Handles eBPF program loading, kprobe attachment, and event processing
- **IPManager**: Manages IP list operations with thread-safe access
- **APIServer**: Provides REST API endpoints using Gin framework
- **Application**: Orchestrates both eBPF monitoring and API server

Both components run concurrently, with the eBPF monitor capturing system calls in real-time while the API server handles HTTP requests for IP list management.


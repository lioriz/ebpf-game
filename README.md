# eBPF Game - Dynamic PID Monitoring

A Go-based eBPF application that monitors system calls (`sys_read` and `sys_write`) with dynamic PID filtering capabilities.

## Features

- **Dynamic PID Filtering**: Monitor specific PIDs or all PIDs except the monitor's own
- **Real-time System Call Monitoring**: Track `sys_read` and `sys_write` calls
- **REST API**: Manage target PIDs and monitoring settings via HTTP endpoints
- **Object-Oriented Design**: Clean separation between eBPF monitoring and API server

## Architecture

### Components

1. **eBPF Monitor** (`run_probe.go`):
   - Loads and manages eBPF programs
   - Handles system call monitoring
   - Manages target PID list and print_all flag

2. **API Server** (`api_server.go`):
   - REST API for PID management
   - Thread-safe PID operations
   - Integration with eBPF monitor

3. **eBPF Program** (`ebpf_probe.c`):
   - Kernel-level system call monitoring
   - Dynamic PID filtering logic
   - Perf event output for userspace communication

## API Endpoints

### GET `/apis`
List all available API endpoints and usage examples.
```bash
curl http://localhost:8080/apis
```

### POST `/add_pid`
Add a PID to the target monitoring list and set print_all flag to false.
```bash
curl -X POST http://localhost:8080/add_pid \
  -H "Content-Type: application/json" \
  -d '{"pid": 1234}'
```

### POST `/clear_pid_list`
Clear all target PIDs and set print_all flag to false.
```bash
curl -X POST http://localhost:8080/clear_pid_list
```

### POST `/set_print_all`
Set print_all flag to true (monitor all PIDs except the monitor's own PID).
```bash
curl -X POST http://localhost:8080/set_print_all
```

### GET `/target_pids`
Get current target PIDs and print_all flag state.
```bash
curl http://localhost:8080/target_pids
```

## Monitoring Modes

### 1. Target List Mode (Default)
- Only monitors PIDs in the target list
- Initial state: empty list = no monitoring
- Use `/add_pid` to add specific PIDs
- Automatically sets print_all flag to false

### 2. Print All Mode
- Monitors all PIDs except the monitor's own PID
- Use `/set_print_all` to enable this mode
- Automatically sets print_all flag to true

## Building and Running

### Prerequisites
- Docker and Docker Compose
- Linux kernel with eBPF support
- Privileged container access

## Technical Details

### eBPF Maps
- `skip_pid`: Contains the monitor's own PID (always skipped)
- `target_pids`: Hash map of PIDs to monitor in target list mode
- `print_all_flag`: Single entry flag for print_all mode
- `events`: Perf event array for userspace communication

### Build Process
1. **Multi-stage Docker build**:
   - Stage 1: Compile eBPF C code to Go using `bpf2go`
   - Stage 2: Build Go application with embedded eBPF bytecode
   - Stage 3: Create minimal runtime image

2. **eBPF Compilation**:
   - Uses `clang` and `llvm` for C compilation
   - `bpf2go` generates Go bindings for eBPF maps and programs
   - Modern `libbpf` syntax with proper map definitions

### Concurrency
- eBPF monitoring runs in background goroutine
- API server runs in separate goroutine
- Thread-safe PID management with `sync.RWMutex`

## Development

### File Structure
```
ebpf-game/
├── ebpf_probe.c          # eBPF C program
├── run_probe.go          # Main Go application
├── api_server.go         # REST API server
├── go.mod               # Go module definition
├── Dockerfile           # Multi-stage build
├── docker-compose.yml   # Container orchestration
├── test_api.sh          # API testing script
└── README.md           # This file
```

### Key Dependencies
- `github.com/cilium/ebpf`: Modern eBPF library
- `github.com/gin-gonic/gin`: HTTP framework
- `bpf2go`: eBPF C to Go compiler

## Troubleshooting

### Common Issues
1. **Permission denied**: Ensure container runs with `privileged: true`
2. **eBPF program load failed**: Check kernel version and eBPF support
3. **API connection refused**: Verify port 8080 is accessible

### Debug Mode
The application logs:
- eBPF program loading status
- API server startup
- Received API requests
- System call events (when PIDs match criteria)

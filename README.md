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

### POST `/add_pids`
Add PIDs to the target monitoring list.
```json
{
  "add_pids": [1234, 5678, 9999]
}
```

### POST `/clear_pid_list`
Clear all target PIDs from the monitoring list.
```json
{
  "clear_pid_list": true
}
```

### POST `/print_all_pids`
Get all currently monitored PIDs.
```json
{
  "print_all_pids": true
}
```

### POST `/set_print_all`
Enable/disable monitoring of all PIDs (except the monitor's own PID).
```json
{
  "print_all": true
}
```

### GET `/target_pids`
Get current target PIDs from the eBPF map.

## Monitoring Modes

### 1. Target List Mode (Default)
- Only monitors PIDs in the target list
- Initial state: empty list = no monitoring
- Use `/add_pids` to add specific PIDs

### 2. Print All Mode
- Monitors all PIDs except the monitor's own PID
- Use `/set_print_all` with `{"print_all": true}`
- Disable with `{"print_all": false}`

## Building and Running

### Prerequisites
- Docker and Docker Compose
- Linux kernel with eBPF support
- Privileged container access

### Quick Start
```bash
# Build and run
docker-compose up --build

# Test the API
./test_api.sh
```

### Manual Testing
```bash
# Add PIDs to monitor
curl -X POST http://localhost:8080/add_pids \
  -H "Content-Type: application/json" \
  -d '{"add_pids": [1234, 5678]}'

# Enable print_all mode
curl -X POST http://localhost:8080/set_print_all \
  -H "Content-Type: application/json" \
  -d '{"print_all": true}'

# Get current target PIDs
curl http://localhost:8080/target_pids
```

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

## License

This project uses the GPL license for eBPF components as required by the Linux kernel.


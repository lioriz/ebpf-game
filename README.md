# eBPF Game - Dynamic PID Monitoring

A Go-based eBPF application that monitors system calls (`sys_read` and `sys_write`) with dynamic PID filtering capabilities.

## Features

- **Dynamic PID Filtering**: Monitor specific PIDs or all PIDs except the monitor's own
- **Real-time System Call Monitoring**: Track `sys_read` and `sys_write` calls
- **REST API**: Manage target PIDs and monitoring settings via HTTP endpoints
- **Object-Oriented Design**: Clean separation between eBPF probe, controller, API server, and wiring
- **Polymorphic logging**: stdout, rotating file, or both (lumberjack), with timestamps and file:line

## Architecture

### Components

1. **eBPF Probe** (`ebpf_probe.go`):
   - Loads and manages eBPF programs, attaches kprobes
   - Maps enum event types from the kernel (read/write) to log messages
   - Resolves syscall symbol per-arch from `/proc/kallsyms` (e.g., `__x64_sys_read`, `__arm64_sys_read`, ...)
   - Clean shutdown of perf reader to avoid "file already closed" spam

2. **Controller** (`ebpf_controller.go`):
   - Reads commands from a queue and updates the eBPF probe
   - Ensures ordered, single-writer updates to eBPF maps
   - Exposes helpers to query current state

3. **API Server** (`api_server.go`):
   - Enqueues commands into the queue on POST endpoints
   - Reads current state via the controller for GET endpoints

4. **Application Wiring** (`application.go`):
   - Creates the shared command queue
   - Wires `EBpfProbe`, `EBpfController`, and `APIServer`
   - Starts/stops components and injects the Logger

5. **Logger** (`logger.go`):
   - Polymorphic interface (stdout, rotating file, combined)
   - Includes timestamp, microseconds, and short file:line
   - Supports Infof, Warnf, Errorf, Debugf

6. **Entrypoint** (`main.go`):
   - Minimal main: boot, start, wait for signal, shutdown

7. **eBPF Program** (`ebpf_probe.c`):
   - Kernel-level system call monitoring
   - Dynamic PID filtering logic
   - Perf event output sending enum `event_type` instead of strings

### File Structure
```
ebpf-game/
├── main.go                  # Entrypoint
├── application.go           # App wiring (probe + controller + API + logger)
├── ebpf_probe.go            # eBPF probe (load, attach, maps, perf, enum mapping)
├── ebpf_controller.go       # Queue-driven controller calling eBPF
├── api_server.go            # REST API (enqueues; reads state via controller)
├── logger.go                # Polymorphic logger (stdout/file/both)
├── ebpf_probe.c             # eBPF C program
├── go.mod                   # Go module definition
├── Dockerfile               # Multi-stage build
├── docker-compose.yml       # Container orchestration
└── README.md
```

Notes:
- Go bindings for the eBPF program are generated at build time by `bpf2go`.
- The server does not mutate eBPF directly; it only enqueues commands.
- The controller is the single writer to eBPF maps, preventing races.
- Kernel → userspace payload is compact (PID + enum), the Go side formats messages.
- Syscall symbol is resolved dynamically to support multiple architectures.

## API Endpoints

### GET `/apis`
List all available API endpoints and usage examples.
```bash
curl http://localhost:8080/apis
```

### POST `/add_pids`
Add PIDs to the target monitoring list and set print_all flag to false.
```bash
curl -X POST http://localhost:8080/add_pids \
  -H "Content-Type: application/json" \
  -d '{"pids": [1234, 5678]}'
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
- Use `/add_pids` to add specific PIDs
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

### Logs persistence
- By default, the app logs to `/var/log/ebpf-game/ebpf-game.log` inside the container
- `docker-compose.yml` mounts `/var/log/ebpf-game/` so logs persist on the host as `/var/log/ebpf-game/ebpf-game.log`
- You can change the logger in `main.go` (stdout only, rotating file only, or both)
- In production service the logger kind would be set by configuration

## Technical Details

### eBPF Maps
- `skip_pid`: Contains the monitor's own PID (always skipped)
- `target_pids`: Hash map of PIDs to monitor in target list mode
- `print_all_flag`: Single entry flag for print_all mode
- `events`: Perf event array for userspace communication

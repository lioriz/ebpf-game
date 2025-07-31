# eBPF Game - sys_read/sys_write Monitor

This project demonstrates an eBPF program that monitors `sys_read` and `sys_write` kernel function calls using kprobes and the BCC framework.

## Overview

The eBPF program uses kprobes to attach to the `sys_read` and `sys_write` kernel functions. Every time these system calls are made on the Linux machine, the program prints a message indicating which system call was called.

## Files

- `ebpf_probe.c` - The eBPF program written in C
- `main.go` - Go user program that loads and runs the eBPF program
- `go.mod` - Go module file with dependencies
- `Dockerfile` - Docker configuration for containerized deployment
- `docker-compose.yml` - Docker Compose configuration

## Prerequisites

### For Docker:
1. **Docker**: Docker and Docker Compose installed
2. **Linux Host**: Docker must run on a Linux host (eBPF doesn't work on Windows/macOS)
3. **Kernel Version**: 4.4+ with eBPF support

## Building and Running

### Docker (Recommended)

1. **Build the Docker image:**
   ```bash
   docker-compose build
   ```

2. **Run the container:**
   ```bash
   docker-compose up
   ```

3. **Run in detached mode (background):**
   ```bash
   docker-compose up -d
   ```

4. **Stop the container:**
   ```bash
   docker-compose down
   ```

5. **View logs (if running in detached mode):**
   ```bash
   docker-compose logs -f
   ```

## Expected Output

When the program is running, you should see output like:

```
eBPF program loaded successfully!
Monitoring sys_read and sys_write calls...
Press Ctrl+C to exit
hello sys_read was called
hello sys_write was called
hello sys_read was called
```

## Testing the Program

To test that the program is working, you can run some commands in another terminal:

```bash
# This will trigger sys_read calls
cat /etc/passwd

# This will trigger sys_write calls
echo "test" > /tmp/testfile

# This will trigger both sys_read and sys_write
cat /tmp/testfile
```

## How It Works

1. **eBPF Program (`ebpf_probe.c`)**:
   - Defines two kprobe functions: `sys_read_probe` and `sys_write_probe`
   - Each function sends a simple message to user space
   - Uses `BPF_PERF_OUTPUT` to send data to user space

2. **Go User Program (`main.go`)**:
   - Loads the eBPF program using the BCC framework
   - Attaches kprobes to `sys_read` and `sys_write` kernel functions
   - Sets up a perf buffer to receive events from the eBPF program
   - Processes events and prints the required messages

3. **Docker Setup**:
   - Provides all necessary dependencies in a containerized environment
   - Uses privileged mode to access the host kernel
   - Mounts system directories needed for eBPF operation

## Security Note

This program requires root privileges because eBPF programs need elevated permissions to load into the kernel. The Docker version runs with `--privileged` access, which gives it full access to the host system. Always be cautious when running programs with root privileges.

## Troubleshooting

### Docker:
- **Permission denied errors**: Make sure Docker has the necessary permissions
- **Kernel headers not found**: The container mounts `/boot` to access kernel headers
- **eBPF not supported**: Ensure your host kernel supports eBPF (version 4.4+)
- **Container won't start**: Check if Docker has privileged access capabilities

## Cleanup

### Docker:
```bash
# Stop and remove containers
docker-compose down

# Remove the built image
docker-compose down --rmi all

# Clean up all related resources
docker-compose down --volumes --remove-orphans
```

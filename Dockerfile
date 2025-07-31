FROM ubuntu:22.04

# Install system dependencies
RUN apt-get update && apt-get install -y \
    golang-go \
    bpfcc-tools \
    linux-headers-generic \
    build-essential \
    git \
    && rm -rf /var/lib/apt/lists/*

# Set working directory
WORKDIR /app

# Copy go mod files first for better caching
COPY go.mod go.sum* ./

# Download Go dependencies
RUN go mod download

# Copy source code
COPY ebpf_probe.c .
COPY main.go .

# Build the application
RUN go build -o ebpf-monitor main.go

# Create a non-root user
RUN useradd -m -u 1000 ebpfuser

# Change ownership of the app directory
RUN chown -R ebpfuser:ebpfuser /app

# Switch to non-root user
USER ebpfuser

# Expose any ports if needed (though this app doesn't use network ports)
EXPOSE 8080

# Set the entrypoint
ENTRYPOINT ["./ebpf-monitor"] 
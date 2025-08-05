FROM ubuntu:24.04

# Install dependencies
RUN apt-get update && apt-get install -y \
    python3 \
    python3-pip \
    bpfcc-tools \
    linux-headers-$(uname -r) \
    && rm -rf /var/lib/apt/lists/*

# Install Python BCC
RUN pip3 install bcc

WORKDIR /app

# Copy files
COPY ebpf_probe.c .
COPY run_probe.py .

# Run as root (required for eBPF)
USER root

CMD ["python3", "run_probe.py"]

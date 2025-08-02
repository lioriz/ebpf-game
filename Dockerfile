FROM ghcr.io/iovisor/bcc:ubuntu-24.04

RUN sed -i '/llvm-toolchain-noble-15/d' /etc/apt/sources.list /etc/apt/sources.list.d/* || true

# Optional: Install Go, build tools, etc.
RUN apt-get update && apt-get install -y \
    build-essential \
    git \
    clang-17 \
    llvm-17 \
    llvm-17-dev \
    llvm-17-runtime \
    make \
    curl \
    libbpf-dev \
    ca-certificates \
    tzdata && \
    ln -fs /usr/share/zoneinfo/$TZ /etc/localtime && \
    dpkg-reconfigure -f noninteractive tzdata && \
    apt-get clean && rm -rf /var/lib/apt/lists/*

# Install Go manually
ENV GO_VERSION=1.21.5
RUN wget https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz && \
    tar -C /usr/local -xzf go${GO_VERSION}.linux-amd64.tar.gz && \
    rm go${GO_VERSION}.linux-amd64.tar.gz

ENV PATH="/usr/local/go/bin:$PATH"

# Clone latest tagged release of BCC
WORKDIR /tmp
RUN git clone --depth 1 --branch $(git ls-remote --tags https://github.com/iovisor/bcc.git | \
     grep -o 'refs/tags/v[0-9.]*$' | sort -V | tail -n1 | sed 's/refs\/tags\///') https://github.com/iovisor/bcc.git

# Build BCC (only what we need)
WORKDIR /tmp/bcc/build
RUN cmake .. \
    -DCMAKE_INSTALL_PREFIX=/usr && \
    make -j$(nproc) && \
    make install/fast

WORKDIR /app

COPY go.mod .
COPY main.go .

RUN go mod tidy && go mod download

RUN ls -la /usr/include
RUN ls -la /usr/include/bcc

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

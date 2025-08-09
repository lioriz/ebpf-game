FROM golang:1.23 AS builder

WORKDIR /app

RUN apt-get update && apt-get install -y \
    clang \
    llvm \
    libelf-dev \
    libbpf-dev \
    linux-headers-generic \
    && rm -rf /var/lib/apt/lists/*

COPY ebpf_probe.c .

RUN go install github.com/cilium/ebpf/cmd/bpf2go@latest
RUN bpf2go -cc clang -cflags "-g -O2 -Wall -I/usr/include -I/usr/include/x86_64-linux-gnu" -target bpf -go-package main -output-stem ebpf_probe ebpf_probe ebpf_probe.c

COPY go.mod ./
# Copy all Go sources to ensure types like Logger and controllers are included
COPY *.go ./

RUN go mod tidy

RUN CGO_ENABLED=0 GOOS=linux go build -o main .

FROM debian:bullseye-slim

RUN apt-get update && apt-get install -y \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=builder /app/main .

EXPOSE 8080

CMD ["./main"]
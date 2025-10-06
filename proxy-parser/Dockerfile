# Build stage
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN go build -o codex-proxy ./cmd/proxy

# Runtime stage
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache ca-certificates

# Create non-root user
RUN addgroup -g 1000 proxy && \
    adduser -u 1000 -G proxy -s /bin/sh -D proxy

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/codex-proxy /app/codex-proxy

# Copy configuration file
COPY config.yaml /app/config.yaml

# Create directory for output
RUN mkdir -p /data && chown -R proxy:proxy /data

# Switch to non-root user
USER proxy

# Expose proxy port
EXPOSE 8080

# Volume for captured traffic
VOLUME ["/data"]

# Default command
CMD ["/app/codex-proxy", "start", "--config", "/app/config.yaml", "--output", "/data/traffic.json"]
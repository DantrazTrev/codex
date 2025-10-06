# Multi-stage build for Codex Proxy Parser
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bin/codex-proxy ./cmd/proxy

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates tzdata

# Create non-root user
RUN adduser -D -s /bin/sh codex-proxy

# Set working directory
WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /app/bin/codex-proxy ./bin/

# Copy configuration files
COPY --from=builder /app/config.yaml ./
COPY --from=builder /app/examples/ ./examples/

# Create logs directory
RUN mkdir -p logs && chown -R codex-proxy:codex-proxy /app

# Switch to non-root user
USER codex-proxy

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Default command
CMD ["./bin/codex-proxy", "--config", "config.yaml"]
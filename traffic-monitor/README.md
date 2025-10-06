# Codex CLI Traffic Monitor & Proxy

This directory contains Go-based tools for monitoring and proxying traffic from the Codex CLI to GenAI/vibe coding tool providers.

## Components

1. **Traffic Parser** (`parser/`) - Parses and analyzes HTTP traffic logs
2. **Proxy Server** (`proxy/`) - HTTP/HTTPS proxy server for routing Codex CLI traffic
3. **Configuration** (`config/`) - Configuration files and examples

## Quick Start

### 1. Build the tools
```bash
cd traffic-monitor
go mod init codex-traffic-monitor
go mod tidy
go build -o bin/proxy ./cmd/proxy
go build -o bin/parser ./cmd/parser
```

### 2. Start the proxy server
```bash
./bin/proxy -port 8080 -log-file traffic.log
```

### 3. Configure Codex CLI to use proxy
```bash
# Set environment variables
export HTTP_PROXY=http://localhost:8080
export HTTPS_PROXY=http://localhost:8080

# Or use system proxy settings
export http_proxy=http://localhost:8080
export https_proxy=http://localhost:8080
```

### 4. Run Codex CLI normally
The CLI will now route through your proxy and log all traffic.

### 5. Parse the traffic logs
```bash
./bin/parser -input traffic.log -output analysis.json
```

## Features

- **Traffic Logging**: Captures all HTTP/HTTPS requests and responses
- **GenAI Detection**: Identifies requests to AI providers (OpenAI, Anthropic, etc.)
- **Request Analysis**: Parses request payloads, headers, and response data
- **Metrics Collection**: Tracks token usage, response times, error rates
- **Filtering**: Focus on specific domains or request types
- **Export Formats**: JSON, CSV, and custom formats

## Configuration

See `config/proxy.yaml` for proxy configuration options and `config/parser.yaml` for parser settings.
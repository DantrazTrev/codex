# Codex CLI Proxy Setup Guide

This guide provides detailed instructions for setting up the Codex CLI traffic proxy and parser.

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Installation](#installation)
3. [Configuration](#configuration)
4. [Proxy Setup](#proxy-setup)
5. [Codex CLI Integration](#codex-cli-integration)
6. [Testing](#testing)
7. [Troubleshooting](#troubleshooting)

## Prerequisites

### System Requirements

- **Operating System**: Linux, macOS, or Windows
- **Go Version**: 1.21 or later
- **Memory**: At least 512MB RAM
- **Disk Space**: 100MB for binaries and logs

### Required Software

1. **Go Programming Language**
   ```bash
   # Ubuntu/Debian
   sudo apt update
   sudo apt install golang-go
   
   # macOS (with Homebrew)
   brew install go
   
   # Windows
   # Download from https://golang.org/dl/
   ```

2. **Codex CLI**
   ```bash
   # Install via npm
   npm install -g @openai/codex
   
   # Or download from GitHub releases
   ```

3. **Optional Tools**
   ```bash
   # jq for JSON processing
   sudo apt install jq
   
   # curl for testing
   sudo apt install curl
   ```

## Installation

### Step 1: Download and Setup

```bash
# Navigate to the proxy parser directory
cd /workspace/proxy-parser

# Make scripts executable
chmod +x scripts/*.sh

# Run the setup script
./scripts/setup-proxy.sh
```

The setup script will:
- Check Go installation and version
- Build the proxy binary
- Create necessary directories
- Set up environment variables
- Create systemd service (if running as root)

### Step 2: Verify Installation

```bash
# Check if binary was created
ls -la bin/codex-proxy

# Test the binary
./bin/codex-proxy --help
```

## Configuration

### Basic Configuration

The proxy uses YAML configuration files. The default configuration is in `config.yaml`:

```yaml
server:
  port: 8080
  host: "0.0.0.0"
  read_timeout: 30
  write_timeout: 30

proxy:
  target_url: "https://api.openai.com"
  timeout: 30
  skip_tls_verify: false
  headers:
    User-Agent: "Codex-Proxy-Parser/1.0"

parser:
  enabled: true
  parse_genai: true
  parse_vibe_coding: true
  max_body_size: 10485760  # 10MB
  keywords:
    - "openai"
    - "anthropic"
    - "claude"
    - "gpt"
    - "codex"
    - "cursor"

logging:
  level: "info"
  format: "json"
  file: ""

output:
  format: "json"
  file: "traffic.log"
  console: true
  max_entries: 1000
```

### Advanced Configuration

For production use, consider these settings:

```yaml
server:
  port: 8080
  host: "127.0.0.1"  # Only localhost for security
  read_timeout: 60
  write_timeout: 60

proxy:
  target_url: "https://api.openai.com"
  timeout: 60
  skip_tls_verify: false
  headers:
    User-Agent: "Codex-Proxy-Parser/1.0"
    Authorization: "Bearer YOUR_API_KEY"  # If needed

parser:
  enabled: true
  parse_genai: true
  parse_vibe_coding: true
  max_body_size: 52428800  # 50MB
  keywords:
    - "openai"
    - "anthropic"
    - "claude"
    - "gpt"
    - "codex"
    - "cursor"
    - "github"
    - "copilot"

logging:
  level: "debug"
  format: "json"
  file: "/var/log/codex-proxy.log"

output:
  format: "json"
  file: "/var/log/codex-traffic.json"
  console: true
  max_entries: 10000
```

## Proxy Setup

### Method 1: Environment Variables (Recommended)

```bash
# Set proxy environment variables
export HTTP_PROXY=http://127.0.0.1:8080
export HTTPS_PROXY=http://127.0.0.1:8080
export NO_PROXY=localhost,127.0.0.1,::1

# Or use the provided .env file
source .env
```

### Method 2: Codex CLI Configuration

Create a Codex configuration file:

```bash
# Create config directory
mkdir -p ~/.config/codex

# Create configuration file
cat > ~/.config/codex/config.yaml << EOF
proxy:
  http: "http://127.0.0.1:8080"
  https: "http://127.0.0.1:8080"
  no_proxy: "localhost,127.0.0.1,::1"
EOF
```

### Method 3: System-wide Proxy

For Linux systems, you can set system-wide proxy:

```bash
# Add to /etc/environment
echo 'HTTP_PROXY="http://127.0.0.1:8080"' >> /etc/environment
echo 'HTTPS_PROXY="http://127.0.0.1:8080"' >> /etc/environment
echo 'NO_PROXY="localhost,127.0.0.1,::1"' >> /etc/environment

# Reload environment
source /etc/environment
```

## Codex CLI Integration

### Step 1: Start the Proxy

```bash
# Start proxy in foreground
./scripts/start-proxy.sh

# Or start as daemon
./scripts/start-proxy.sh --daemon
```

### Step 2: Configure Codex CLI

```bash
# Set environment variables
source .env

# Verify proxy is working
echo $HTTP_PROXY
echo $HTTPS_PROXY
```

### Step 3: Test Codex CLI

```bash
# Test basic functionality
codex --help

# Test with a simple command
codex exec "echo 'Hello from Codex CLI'"

# Test with AI features (if available)
codex chat "What is the capital of France?"
```

## Testing

### Test 1: Proxy Connectivity

```bash
# Test if proxy is listening
curl -I http://127.0.0.1:8080

# Test proxy with a simple request
curl -x http://127.0.0.1:8080 http://httpbin.org/ip
```

### Test 2: Traffic Parsing

```bash
# Start proxy in verbose mode
./scripts/start-proxy.sh --verbose

# In another terminal, make a test request
curl -x http://127.0.0.1:8080 https://api.openai.com/v1/models

# Check logs for parsed data
tail -f logs/traffic.json | jq .
```

### Test 3: Codex CLI Integration

```bash
# Set proxy environment
source .env

# Run a Codex command that makes network requests
codex --help

# Check proxy logs
tail -f logs/proxy.log
```

## Troubleshooting

### Common Issues

#### 1. Port Already in Use

**Error**: `bind: address already in use`

**Solution**:
```bash
# Find process using port 8080
lsof -i :8080

# Kill the process
sudo kill -9 <PID>

# Or change port in config.yaml
```

#### 2. Codex CLI Not Using Proxy

**Error**: Codex CLI makes direct requests without going through proxy

**Solution**:
```bash
# Verify environment variables
env | grep -i proxy

# Check if Codex CLI respects proxy settings
codex --help 2>&1 | grep -i proxy

# Try explicit proxy setting
codex --proxy http://127.0.0.1:8080 --help
```

#### 3. TLS Certificate Errors

**Error**: `x509: certificate signed by unknown authority`

**Solution**:
```bash
# For testing, disable TLS verification
# Edit config.yaml:
proxy:
  skip_tls_verify: true

# Or install proper certificates
sudo apt install ca-certificates
```

#### 4. Permission Denied

**Error**: `permission denied` when starting proxy

**Solution**:
```bash
# Make sure binary is executable
chmod +x bin/codex-proxy

# Check file permissions
ls -la bin/codex-proxy

# Run with proper permissions
sudo ./scripts/start-proxy.sh
```

### Debug Mode

Enable verbose logging for detailed troubleshooting:

```bash
# Start with debug logging
./scripts/start-proxy.sh --verbose

# Check all log files
ls -la logs/

# Monitor logs in real-time
tail -f logs/proxy.log
tail -f logs/traffic.json
```

### Log Analysis

```bash
# Parse traffic logs
jq '.is_genai' logs/traffic.json | sort | uniq -c

# Count request types
jq '.type' logs/traffic.json | sort | uniq -c

# Filter GenAI requests
jq 'select(.is_genai == true)' logs/traffic.json

# Filter vibe coding requests
jq 'select(.is_vibe_coding == true)' logs/traffic.json
```

## Security Best Practices

1. **Run on localhost only** for development
2. **Use proper authentication** when forwarding requests
3. **Limit log retention** to prevent disk space issues
4. **Encrypt sensitive data** in logs
5. **Regular security updates** for dependencies

## Performance Tuning

1. **Adjust timeout values** based on your network
2. **Limit body size** to prevent memory issues
3. **Use appropriate log levels** (info vs debug)
4. **Monitor resource usage** during operation
5. **Consider load balancing** for high traffic

## Next Steps

After successful setup:

1. **Monitor traffic patterns** to understand usage
2. **Customize parsing rules** for your specific needs
3. **Set up log rotation** for long-term operation
4. **Integrate with monitoring systems** (Prometheus, Grafana)
5. **Develop custom analysis tools** based on parsed data
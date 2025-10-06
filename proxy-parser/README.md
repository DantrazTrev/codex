# Codex CLI Proxy Parser

This tool provides a proxy server and traffic parser for monitoring and analyzing network traffic from the Codex CLI. It's designed to help understand and debug GenAI/Vibe coding tools traffic without modifying the CLI itself.

## Features

- **HTTP/HTTPS Proxy Server**: Intercepts and logs all traffic from Codex CLI
- **Traffic Parser**: Analyzes requests/responses to identify patterns
- **GenAI Traffic Analyzer**: Specifically tracks AI model API calls
- **Request/Response Logging**: Detailed logging with filtering capabilities
- **Statistics & Metrics**: Traffic analysis and reporting

## Quick Start

### 1. Build the Proxy Server

```bash
cd proxy-parser
go mod download
go build -o codex-proxy ./cmd/proxy
```

### 2. Run the Proxy Server

```bash
# Start proxy on default port (8080)
./codex-proxy start

# Start proxy on custom port
./codex-proxy start --port 9090

# Start with verbose logging
./codex-proxy start --verbose

# Save traffic to file
./codex-proxy start --output traffic.json
```

### 3. Configure Codex CLI to Use the Proxy

#### Method 1: Environment Variables (Recommended)

```bash
# For HTTP traffic
export HTTP_PROXY=http://localhost:8080
export http_proxy=http://localhost:8080

# For HTTPS traffic
export HTTPS_PROXY=http://localhost:8080
export https_proxy=http://localhost:8080

# Optional: Bypass proxy for certain hosts
export NO_PROXY=localhost,127.0.0.1,internal.corp

# Now run codex commands
codex chat "Hello, world"
```

#### Method 2: Node.js Specific Configuration

Since Codex CLI uses Node.js, you can also use Node-specific proxy settings:

```bash
# Using NODE_OPTIONS
export NODE_OPTIONS="--proxy-server=http://localhost:8080"

# Or using npm config
npm config set proxy http://localhost:8080
npm config set https-proxy http://localhost:8080
```

#### Method 3: System-wide Proxy (macOS/Linux)

```bash
# macOS
networksetup -setwebproxy "Wi-Fi" localhost 8080
networksetup -setsecurewebproxy "Wi-Fi" localhost 8080

# Linux (using environment)
echo 'export HTTP_PROXY=http://localhost:8080' >> ~/.bashrc
echo 'export HTTPS_PROXY=http://localhost:8080' >> ~/.bashrc
source ~/.bashrc
```

## Traffic Analysis

### Parse Captured Traffic

```bash
# Analyze traffic from JSON file
./codex-proxy analyze --input traffic.json

# Filter by endpoint
./codex-proxy analyze --input traffic.json --filter-endpoint "/api/codex"

# Show only GenAI traffic
./codex-proxy analyze --input traffic.json --genai-only

# Generate statistics report
./codex-proxy stats --input traffic.json
```

### Real-time Monitoring

```bash
# Start proxy with real-time analysis
./codex-proxy start --analyze --genai-highlight

# Monitor specific patterns
./codex-proxy start --monitor "gpt-4,claude,completion"
```

## Understanding the Traffic

The proxy captures and analyzes:

1. **API Endpoints**: All HTTP/HTTPS requests to backend services
2. **Authentication**: Bearer tokens and session management
3. **Model Requests**: Specific calls to AI models (GPT-4, Claude, etc.)
4. **Tool Calls**: Function/tool invocations in conversations
5. **Response Streaming**: Server-sent events and streaming responses

### Common Endpoints Monitored

- `/api/codex/*` - Codex-specific API calls
- `/backend-api/*` - ChatGPT backend calls
- `/wham/*` - WHAM API endpoints
- Model-specific endpoints for completions and chat

## Advanced Configuration

### Custom CA Certificate (for HTTPS interception)

```bash
# Generate CA certificate
./codex-proxy generate-cert

# Install CA certificate (macOS)
./codex-proxy install-cert

# Manual installation
# The CA certificate is saved at ~/.codex-proxy/ca-cert.pem
```

### Configuration File

Create `config.yaml`:

```yaml
proxy:
  port: 8080
  host: "0.0.0.0"
  
logging:
  level: "info"
  output: "traffic.log"
  format: "json"
  
filtering:
  include_endpoints:
    - "/api/codex"
    - "/backend-api"
  exclude_headers:
    - "Cookie"
    - "Set-Cookie"
    
analysis:
  genai_patterns:
    - "gpt-4"
    - "gpt-3.5"
    - "claude"
    - "completion"
    - "chat/conversation"
  
  highlight_keywords:
    - "model"
    - "temperature"
    - "max_tokens"
    - "stream"
```

Run with config:

```bash
./codex-proxy start --config config.yaml
```

## Troubleshooting

### Proxy Connection Issues

1. **Check if proxy is running**: 
   ```bash
   curl -x http://localhost:8080 http://httpbin.org/ip
   ```

2. **Verify environment variables**:
   ```bash
   env | grep -i proxy
   ```

3. **Test with Node.js directly**:
   ```bash
   node -e "console.log(process.env.HTTP_PROXY)"
   ```

### SSL Certificate Issues

If you see SSL errors when intercepting HTTPS:

1. Install the CA certificate (see above)
2. Or disable SSL verification (not recommended for production):
   ```bash
   export NODE_TLS_REJECT_UNAUTHORIZED=0
   ```

### No Traffic Captured

1. Ensure Codex CLI is using the proxy:
   ```bash
   # Check if requests are reaching proxy
   ./codex-proxy start --verbose
   ```

2. Try explicit proxy configuration in Node.js:
   ```bash
   # Use global-agent for Node.js
   npm install -g global-agent
   export GLOBAL_AGENT_HTTP_PROXY=http://localhost:8080
   ```

## Architecture

```
┌─────────────┐       ┌──────────────┐       ┌──────────────┐
│  Codex CLI  │──────▶│ Proxy Server │──────▶│ Backend APIs │
└─────────────┘       └──────────────┘       └──────────────┘
                             │
                             ▼
                      ┌──────────────┐
                      │   Parser &   │
                      │   Analyzer   │
                      └──────────────┘
                             │
                             ▼
                      ┌──────────────┐
                      │   Output:    │
                      │  JSON/Stats  │
                      └──────────────┘
```

## Development

### Project Structure

```
proxy-parser/
├── cmd/
│   └── proxy/
│       └── main.go          # CLI entry point
├── pkg/
│   ├── proxy/
│   │   ├── server.go        # Proxy server implementation
│   │   └── handler.go       # Request/response handlers
│   ├── parser/
│   │   ├── parser.go        # Traffic parser
│   │   └── analyzer.go      # GenAI traffic analyzer
│   ├── logger/
│   │   └── logger.go        # Structured logging
│   └── storage/
│       └── storage.go       # Traffic storage
├── config/
│   └── config.go            # Configuration management
├── go.mod
├── go.sum
└── README.md
```

### Testing

```bash
# Run tests
go test ./...

# Run with coverage
go test -cover ./...

# Integration test with real traffic
./scripts/test-integration.sh
```

## License

This tool is part of the Codex project and follows the same Apache 2.0 license.
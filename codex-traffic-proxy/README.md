# Codex Traffic Proxy

A transparent HTTP proxy server designed to monitor and analyze traffic from the Codex CLI to various AI service endpoints. This tool helps developers understand API usage patterns, extract sensitive information like API keys, and monitor request/response behavior.

## Features

- **Traffic Interception**: Captures all HTTP/HTTPS requests and responses from the Codex CLI
- **API Key Detection**: Automatically identifies and extracts API keys from requests
- **Endpoint Analysis**: Categorizes requests by service (OpenAI, ChatGPT, Codex, etc.)
- **Token Extraction**: Identifies authentication tokens and session cookies
- **Usage Monitoring**: Tracks token usage and API consumption patterns
- **Pattern Recognition**: Identifies common request patterns (streaming, WebSocket, etc.)
- **Comprehensive Logging**: Structured JSON logging with customizable output formats
- **Configuration Management**: YAML-based configuration with environment variable overrides

## Installation

### Prerequisites

- Go 1.21 or later
- Make (optional, for using Makefile)

### Build from Source

```bash
git clone <repository-url>
cd codex-traffic-proxy
go mod download
go build -o codex-traffic-proxy ./cmd
```

### Using Make (if available)

```bash
make build
```

## Quick Start

1. **Initialize Configuration** (optional):
   ```bash
   ./codex-traffic-proxy config init
   ```

2. **Start the Proxy**:
   ```bash
   ./codex-traffic-proxy start
   ```

3. **Configure Codex CLI** to use the proxy:
   ```bash
   export HTTP_PROXY=http://127.0.0.1:8080
   export HTTPS_PROXY=http://127.0.0.1:8080
   ```

4. **Run Codex CLI** - all traffic will now be monitored:
   ```bash
   codex --help  # Test with a simple command first
   ```

## Configuration

### Configuration File

The proxy uses a YAML configuration file located at `~/.codex-traffic-proxy/config.yaml`. You can generate a default configuration using:

```bash
./codex-traffic-proxy config init
```

### Environment Variables

You can override configuration values using environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `PROXY_LISTEN_ADDR` | Proxy server listen address | `127.0.0.1` |
| `PROXY_PORT` | Proxy server port | `8080` |
| `PROXY_VERBOSE` | Enable verbose logging | `false` |
| `LOG_LEVEL` | Logging level (debug, info, warn, error) | `info` |
| `LOG_FORMAT` | Log format (json, text) | `json` |
| `LOG_OUTPUT` | Log output destination (stdout, stderr) | `stdout` |
| `STORAGE_DIR` | Directory for storing logs | `./logs` |
| `STORAGE_RETENTION` | Log retention in days | `30` |

### Example Configuration

```yaml
proxy:
  listen_addr: "127.0.0.1"
  port: 8080
  verbose: false

logger:
  level: "info"
  format: "json"
  output: "stdout"

parser:
  enabled: true
  extract_api_keys: true
  extract_endpoints: true
  extract_tokens: true
  sensitive_patterns:
    - "sk-[a-zA-Z0-9]{32,}"
    - "Bearer [a-zA-Z0-9\\-._~+/]+=*"
    - "authorization: Bearer [a-zA-Z0-9\\-._~+/]+=*"

storage:
  directory: "./logs"
  retention_days: 30
```

## Usage

### Basic Commands

```bash
# Start the proxy server
./codex-traffic-proxy start

# Initialize default configuration
./codex-traffic-proxy config init

# View help
./codex-traffic-proxy --help
```

### Monitoring Codex CLI Traffic

1. Start the proxy:
   ```bash
   ./codex-traffic-proxy start
   ```

2. Configure your environment to use the proxy:
   ```bash
   export HTTP_PROXY=http://127.0.0.1:8080
   export HTTPS_PROXY=http://127.0.0.1:8080
   ```

3. Run Codex CLI commands. The proxy will intercept and analyze all traffic.

### Understanding the Logs

The proxy generates structured logs that include:

- **Request Details**: Method, URL, headers, timing information
- **Response Details**: Status codes, response headers, response body analysis
- **API Key Detection**: Identifies and logs API keys found in requests
- **Endpoint Classification**: Categorizes requests by AI service (OpenAI, ChatGPT, Codex)
- **Token Usage**: Tracks API token consumption
- **Pattern Recognition**: Identifies streaming requests, WebSocket connections, etc.

### Log Output Example

```json
{
  "level": "info",
  "timestamp": "2024-01-15T10:30:45Z",
  "message": "Request intercepted",
  "fields": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "method": "POST",
    "host": "api.openai.com",
    "path": "/v1/chat/completions",
    "api_keys": [
      {
        "key": "sk-...abcd",
        "location": "Authorization header",
        "timestamp": "2024-01-15T10:30:45Z"
      }
    ],
    "endpoints": [
      {
        "url": "https://api.openai.com/v1/chat/completions",
        "host": "api.openai.com",
        "path": "/v1/chat/completions",
        "method": "POST",
        "service": "openai",
        "timestamp": "2024-01-15T10:30:45Z"
      }
    ],
    "patterns": ["chat_completion"]
  }
}
```

## Advanced Features

### Custom Sensitive Patterns

You can customize the patterns used to detect sensitive information:

```yaml
parser:
  sensitive_patterns:
    - "sk-[a-zA-Z0-9]{32,}"  # OpenAI API keys
    - "x-api-key:[a-zA-Z0-9]{32,}"  # Custom API key headers
    - "token:[a-zA-Z0-9\\-._]+"  # Token patterns
```

### Log Filtering

Configure logging levels and output formats:

```yaml
logger:
  level: "debug"  # Show all log levels
  format: "text"  # Human-readable text format
  output: "file"  # Log to file instead of stdout
```

### Storage Configuration

Configure where logs are stored and how long to retain them:

```yaml
storage:
  directory: "/var/log/codex-proxy"
  retention_days: 90
```

## Integration with Codex CLI

The proxy is designed to work transparently with the existing Codex CLI without requiring any code modifications. Simply set the proxy environment variables and all HTTP/HTTPS traffic will be routed through the proxy for monitoring.

### Shell Integration

Add to your `.bashrc` or `.zshrc`:

```bash
# Codex Traffic Proxy
export HTTP_PROXY=http://127.0.0.1:8080
export HTTPS_PROXY=http://127.0.0.1:8080

# Optional: Start proxy automatically
alias start-codex-proxy="codex-traffic-proxy start"
```

### Development Workflow

1. Start the proxy in one terminal
2. Set proxy environment variables in your shell
3. Use Codex CLI normally - all traffic is monitored
4. Review logs to understand API usage patterns

## Security Considerations

- The proxy logs may contain sensitive information (API keys, tokens)
- Store logs securely and consider log rotation policies
- Use the proxy only in development/staging environments
- Consider encrypting stored logs if they contain sensitive data

## Troubleshooting

### Common Issues

1. **Proxy not intercepting traffic**:
   - Ensure `HTTP_PROXY` and `HTTPS_PROXY` are set correctly
   - Verify the proxy is running on the correct port
   - Check if firewall is blocking the proxy port

2. **SSL/TLS certificate errors**:
   - The proxy uses a self-signed certificate for HTTPS interception
   - Configure your system to trust the proxy's certificate

3. **Performance impact**:
   - The proxy adds minimal overhead to requests
   - Monitor resource usage if processing high volumes of traffic

### Debug Mode

Run the proxy in verbose mode for detailed logging:

```bash
PROXY_VERBOSE=true ./codex-traffic-proxy start
```

Or set in configuration:

```yaml
proxy:
  verbose: true
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests for new functionality
5. Submit a pull request

## License

[Add appropriate license information here]
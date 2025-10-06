# Codex CLI Traffic Proxy and Parser

A Go-based HTTP proxy server designed to intercept, monitor, and parse traffic from the Codex CLI, specifically targeting GenAI and vibe coding tool requests.

## Features

- **HTTP Proxy Server**: Intercepts all HTTP/HTTPS traffic from Codex CLI
- **Traffic Parsing**: Automatically detects and parses GenAI and vibe coding requests
- **Configurable**: Flexible configuration for different use cases
- **Logging**: Comprehensive logging with multiple output formats
- **Easy Setup**: Simple scripts for installation and management
- **Docker Support**: Containerized deployment option

## Quick Start

### Prerequisites

- Go 1.21 or later
- Codex CLI installed
- Basic understanding of HTTP proxies

### Installation

1. **Clone and setup**:
   ```bash
   cd /workspace/proxy-parser
   ./scripts/setup-proxy.sh
   ```

2. **Start the proxy**:
   ```bash
   ./scripts/start-proxy.sh
   ```

3. **Configure Codex CLI**:
   ```bash
   source .env
   # Now run codex commands normally
   codex --help
   ```

## Configuration

### Basic Configuration

The proxy uses YAML configuration files. See `config.yaml` for the default configuration:

```yaml
server:
  port: 8080
  host: "0.0.0.0"

proxy:
  target_url: "https://api.openai.com"
  timeout: 30

parser:
  enabled: true
  parse_genai: true
  parse_vibe_coding: true
  keywords:
    - "openai"
    - "codex"
    - "cursor"
```

### Environment Variables

The proxy respects standard HTTP proxy environment variables:

- `HTTP_PROXY`: HTTP proxy URL
- `HTTPS_PROXY`: HTTPS proxy URL
- `NO_PROXY`: Comma-separated list of hosts to bypass

## Usage

### Starting the Proxy

```bash
# Start with default config
./scripts/start-proxy.sh

# Start with custom config
./scripts/start-proxy.sh --config ./examples/codex-proxy-config.yaml

# Start in verbose mode
./scripts/start-proxy.sh --verbose

# Start as daemon
./scripts/start-proxy.sh --daemon
```

### Stopping the Proxy

```bash
./scripts/stop-proxy.sh
```

### Using with Codex CLI

1. **Set environment variables**:
   ```bash
   export HTTP_PROXY=http://127.0.0.1:8080
   export HTTPS_PROXY=http://127.0.0.1:8080
   ```

2. **Run Codex CLI commands**:
   ```bash
   codex --help
   codex exec "echo hello"
   ```

## Traffic Parsing

The proxy automatically parses traffic to identify:

### GenAI Requests
- OpenAI API calls
- Anthropic Claude requests
- GPT model interactions
- Image generation requests
- Audio transcription/translation

### Vibe Coding Requests
- Codex-specific requests
- Cursor IDE interactions
- GitHub Copilot requests
- Tabnine suggestions
- IntelliCode requests

### Parsed Data Structure

```json
{
  "timestamp": "2024-01-15T10:30:00Z",
  "type": "chat_completion",
  "endpoint": "/v1/chat/completions",
  "method": "POST",
  "model": "gpt-4",
  "tokens": 150,
  "is_genai": true,
  "is_vibe_coding": false,
  "request_id": "req_123456",
  "user_agent": "Codex-CLI/1.0",
  "content_length": 1024
}
```

## Output Formats

The proxy supports multiple output formats:

- **JSON**: Structured data for programmatic processing
- **YAML**: Human-readable configuration format
- **CSV**: Tabular data for spreadsheet analysis

## Docker Support

### Build Docker Image

```bash
docker build -t codex-proxy .
```

### Run with Docker

```bash
docker run -d \
  --name codex-proxy \
  -p 8080:8080 \
  -v $(pwd)/config.yaml:/app/config.yaml \
  -v $(pwd)/logs:/app/logs \
  codex-proxy
```

## Monitoring and Logging

### Log Files

- `logs/proxy.log`: General proxy logs
- `logs/traffic.json`: Parsed traffic data
- `logs/proxy.pid`: Process ID file

### Real-time Monitoring

```bash
# Watch logs in real-time
tail -f logs/proxy.log

# Monitor parsed traffic
tail -f logs/traffic.json | jq .
```

## Security Considerations

- The proxy runs on localhost by default for security
- TLS verification can be disabled for testing (not recommended for production)
- Sensitive data in request bodies can be logged (configure `max_body_size` appropriately)
- Use proper authentication headers when forwarding requests

## Troubleshooting

### Common Issues

1. **Port already in use**:
   ```bash
   # Check what's using the port
   lsof -i :8080
   # Kill the process or change port in config
   ```

2. **Codex CLI not using proxy**:
   ```bash
   # Verify environment variables
   echo $HTTP_PROXY
   echo $HTTPS_PROXY
   ```

3. **TLS errors**:
   ```bash
   # Check if target URL is accessible
   curl -I https://api.openai.com
   ```

### Debug Mode

Enable verbose logging for troubleshooting:

```bash
./scripts/start-proxy.sh --verbose
```

## Development

### Building from Source

```bash
go mod tidy
go build -o bin/codex-proxy ./cmd/proxy
```

### Running Tests

```bash
go test ./...
```

### Adding New Parsers

1. Extend the `TrafficParser` struct in `internal/parser/parser.go`
2. Add new detection patterns
3. Update configuration schema
4. Add tests

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## License

This project is licensed under the Apache 2.0 License - see the LICENSE file for details.

## Support

For issues and questions:
- Create an issue in the repository
- Check the troubleshooting section
- Review the configuration examples
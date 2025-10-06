# Codex CLI Integration Guide

This guide explains how to integrate the Codex Traffic Proxy with the existing Codex CLI without modifying the CLI source code.

## How Proxy Integration Works

The Codex CLI uses the `reqwest` HTTP client library, which automatically respects standard proxy environment variables:

- `HTTP_PROXY` - Proxy URL for HTTP requests
- `HTTPS_PROXY` - Proxy URL for HTTPS requests
- `NO_PROXY` - Comma-separated list of hosts to bypass proxy

When these environment variables are set, all HTTP/HTTPS requests from the CLI will be routed through the specified proxy server.

## Integration Steps

### 1. Start the Proxy Server

First, start the traffic proxy server:

```bash
cd /workspace/codex-traffic-proxy
go build -o codex-traffic-proxy ./cmd
./codex-traffic-proxy start
```

The proxy will start listening on `127.0.0.1:8080` by default.

### 2. Configure Environment Variables

Set the proxy environment variables in your shell:

```bash
# Set proxy for both HTTP and HTTPS
export HTTP_PROXY=http://127.0.0.1:8080
export HTTPS_PROXY=http://127.0.0.1:8080

# Optional: Configure hosts to bypass proxy (if needed)
export NO_PROXY="localhost,127.0.0.1"
```

### 3. Test the Integration

Test that the proxy is intercepting traffic:

```bash
# This should now route through the proxy
codex --help

# Or run a simple command that makes API calls
codex exec "echo 'test'" --model gpt-4o-mini
```

### 4. Monitor Traffic

While the CLI is running, you should see logs in the proxy console showing intercepted requests:

```json
{
  "level": "info",
  "timestamp": "2024-01-15T10:30:45Z",
  "message": "Request intercepted",
  "fields": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "method": "POST",
    "host": "api.openai.com",
    "path": "/v1/chat/completions"
  }
}
```

## Environment Variable Persistence

### Temporary (Current Session)

Set the variables in your current shell session:

```bash
export HTTP_PROXY=http://127.0.0.1:8080
export HTTPS_PROXY=http://127.0.0.1:8080
```

### Permanent (Shell Profile)

Add to your `.bashrc`, `.zshrc`, or equivalent:

```bash
# Add these lines to ~/.bashrc or ~/.zshrc
export HTTP_PROXY=http://127.0.0.1:8080
export HTTPS_PROXY=http://127.0.0.1:8080
```

### Reload Shell Configuration

```bash
source ~/.bashrc  # or your shell's configuration file
```

## Advanced Configuration

### Custom Proxy Settings

You can customize the proxy settings using environment variables or the configuration file:

```bash
# Custom proxy port
export PROXY_PORT=9090
export HTTP_PROXY=http://127.0.0.1:9090
export HTTPS_PROXY=http://127.0.0.1:9090

# Custom listen address
export PROXY_LISTEN_ADDR=0.0.0.0
```

### Proxy Authentication

If your proxy requires authentication, include credentials in the URL:

```bash
export HTTP_PROXY=http://username:password@127.0.0.1:8080
export HTTPS_PROXY=http://username:password@127.0.0.1:8080
```

### Bypass Specific Hosts

Configure hosts that should not use the proxy:

```bash
export NO_PROXY="localhost,127.0.0.1,internal.company.com"
```

## Troubleshooting

### Verify Proxy is Running

Check if the proxy server is listening:

```bash
# Check if port 8080 is listening
netstat -tlnp | grep 8080
# or
ss -tlnp | grep 8080

# Test proxy connectivity
curl -x http://127.0.0.1:8080 http://httpbin.org/get
```

### Check Environment Variables

Verify that environment variables are set correctly:

```bash
echo "HTTP_PROXY: $HTTP_PROXY"
echo "HTTPS_PROXY: $HTTPS_PROXY"
echo "NO_PROXY: $NO_PROXY"
```

### Test CLI Without Proxy

To temporarily disable proxy for testing:

```bash
unset HTTP_PROXY HTTPS_PROXY
codex --help
```

### View Proxy Logs

The proxy provides detailed logging. Look for:

- Request interception logs
- API key detection
- Endpoint analysis
- Error messages

### Common Issues

1. **"Connection refused" errors**:
   - Ensure proxy server is running
   - Check proxy port configuration
   - Verify firewall settings

2. **SSL/TLS certificate errors**:
   - The proxy generates self-signed certificates for HTTPS
   - Some systems may require certificate trust configuration

3. **"Proxy authentication required"**:
   - Check if proxy authentication is configured correctly
   - Verify username/password in proxy URL

4. **No traffic being intercepted**:
   - Confirm environment variables are set
   - Restart shell/terminal after setting variables
   - Check if CLI is actually making network requests

## Development Workflow

### Typical Development Session

```bash
# Terminal 1: Start proxy server
cd /workspace/codex-traffic-proxy
./codex-traffic-proxy start

# Terminal 2: Set proxy and use CLI
export HTTP_PROXY=http://127.0.0.1:8080
export HTTPS_PROXY=http://127.0.0.1:8080

# Run CLI commands - all traffic monitored
codex exec "analyze this code"
codex --help

# Monitor proxy logs in Terminal 1 for insights
```

### Log Analysis

The proxy generates structured logs that can be:

- **Piped to log aggregation systems**
- **Filtered for specific patterns**
- **Analyzed for API usage insights**
- **Used for debugging network issues**

Example log filtering:

```bash
# Filter for API key detections
./codex-traffic-proxy start | jq 'select(.fields.api_keys != null)'

# Filter for OpenAI API calls
./codex-traffic-proxy start | jq 'select(.fields.host == "api.openai.com")'
```

## Security Considerations

- **API Keys in Logs**: The proxy logs may contain sensitive information
- **Log Storage**: Consider log rotation and secure storage
- **Network Exposure**: Only run proxy on trusted networks
- **Certificate Handling**: Be aware of self-signed certificate implications

## Integration with IDEs/Editors

You can configure proxy settings in your development environment:

### VS Code

Add to workspace settings or user settings:

```json
{
  "http.proxy": "http://127.0.0.1:8080",
  "http.proxyStrictSSL": false
}
```

### Cursor

Configure proxy in Cursor settings:

```
HTTP Proxy: http://127.0.0.1:8080
```

## Performance Impact

The proxy adds minimal overhead to HTTP requests:

- **Latency**: ~1-5ms additional per request
- **Throughput**: Negligible impact for typical CLI usage
- **Memory**: Stores request/response data for analysis

Monitor resource usage if processing high volumes of traffic.

## Best Practices

1. **Start proxy before CLI usage**
2. **Verify proxy is intercepting traffic before running sensitive operations**
3. **Use configuration files for consistent proxy settings**
4. **Monitor proxy logs for API usage insights**
5. **Secure log storage if containing sensitive data**
6. **Test integration in non-production environments first**

## Support

For issues with proxy integration:

1. Check proxy server logs for error messages
2. Verify environment variable configuration
3. Test proxy connectivity with `curl` or similar tools
4. Ensure Codex CLI version supports proxy environment variables
5. Check network/firewall configuration

The integration leverages standard HTTP proxy mechanisms, so it should work with any properly configured HTTP client, including the Codex CLI.
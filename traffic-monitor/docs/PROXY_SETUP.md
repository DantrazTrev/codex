# Proxy Setup Guide for Codex CLI

This guide explains how to configure the Codex CLI to route traffic through your monitoring proxy.

## Method 1: Environment Variables (Recommended)

The Codex CLI uses the `reqwest` HTTP client library, which automatically respects standard proxy environment variables.

### For HTTP and HTTPS traffic:

```bash
# Set proxy for all HTTP/HTTPS traffic
export HTTP_PROXY=http://localhost:8080
export HTTPS_PROXY=http://localhost:8080

# Alternative lowercase versions (some systems prefer these)
export http_proxy=http://localhost:8080
export https_proxy=http://localhost:8080

# Optional: Bypass proxy for specific domains
export NO_PROXY=localhost,127.0.0.1,.local
export no_proxy=localhost,127.0.0.1,.local
```

### For authentication (if your proxy requires it):

```bash
export HTTP_PROXY=http://username:password@localhost:8080
export HTTPS_PROXY=http://username:password@localhost:8080
```

## Method 2: System Proxy Settings

### macOS:
1. Open System Preferences → Network
2. Select your network connection
3. Click "Advanced" → "Proxies"
4. Check "Web Proxy (HTTP)" and "Secure Web Proxy (HTTPS)"
5. Set Server: `localhost`, Port: `8080`

### Linux:
```bash
# Using gsettings (GNOME)
gsettings set org.gnome.system.proxy mode 'manual'
gsettings set org.gnome.system.proxy.http host 'localhost'
gsettings set org.gnome.system.proxy.http port 8080
gsettings set org.gnome.system.proxy.https host 'localhost'
gsettings set org.gnome.system.proxy.https port 8080

# Or edit /etc/environment
echo "HTTP_PROXY=http://localhost:8080" | sudo tee -a /etc/environment
echo "HTTPS_PROXY=http://localhost:8080" | sudo tee -a /etc/environment
```

### Windows:
1. Open Settings → Network & Internet → Proxy
2. Turn on "Use a proxy server"
3. Set Address: `localhost`, Port: `8080`

## Method 3: Application-Specific Configuration

If you need more control, you can modify the Codex CLI's HTTP client configuration:

### Option A: Modify the reqwest client (requires code changes)

In `codex-rs/core/src/default_client.rs`, modify the `create_client()` function:

```rust
pub fn create_client() -> reqwest::Client {
    use reqwest::header::HeaderMap;

    let mut headers = HeaderMap::new();
    headers.insert("originator", originator().header_value.clone());
    let ua = get_codex_user_agent();

    let mut builder = reqwest::Client::builder()
        .user_agent(ua)
        .default_headers(headers);
    
    // Add proxy configuration
    if let Ok(proxy_url) = std::env::var("CODEX_PROXY_URL") {
        if let Ok(proxy) = reqwest::Proxy::all(&proxy_url) {
            builder = builder.proxy(proxy);
        }
    }
    
    if is_sandboxed() {
        builder = builder.no_proxy();
    }

    builder.build().unwrap_or_else(|_| reqwest::Client::new())
}
```

Then set the environment variable:
```bash
export CODEX_PROXY_URL=http://localhost:8080
```

### Option B: Use a wrapper script

Create a wrapper script that sets proxy environment variables:

```bash
#!/bin/bash
# codex-with-proxy.sh

export HTTP_PROXY=http://localhost:8080
export HTTPS_PROXY=http://localhost:8080

# Start the proxy server in background if not running
if ! pgrep -f "proxy.*8080" > /dev/null; then
    echo "Starting proxy server..."
    ./traffic-monitor/bin/proxy -port 8080 -log-file traffic.log &
    sleep 2
fi

# Run codex with proxy settings
exec codex "$@"
```

Make it executable and use it:
```bash
chmod +x codex-with-proxy.sh
./codex-with-proxy.sh chat "Hello world"
```

## Verification

To verify that the proxy is working:

1. Start the proxy server:
   ```bash
   cd traffic-monitor
   ./bin/proxy -port 8080 -log-file traffic.log -verbose
   ```

2. Set proxy environment variables:
   ```bash
   export HTTP_PROXY=http://localhost:8080
   export HTTPS_PROXY=http://localhost:8080
   ```

3. Run a Codex CLI command:
   ```bash
   codex chat "What is the weather like?"
   ```

4. Check the proxy logs:
   ```bash
   tail -f traffic.log
   ```

You should see requests being logged, especially to AI provider domains like `api.openai.com`.

## Troubleshooting

### Common Issues:

1. **CLI not using proxy**: 
   - Ensure environment variables are set in the same shell session
   - Check for `NO_PROXY` settings that might exclude AI provider domains
   - Verify the proxy server is running and accessible

2. **HTTPS certificate errors**:
   - The proxy uses `InsecureSkipVerify: true` by default
   - For production, configure proper TLS certificates

3. **Proxy authentication**:
   - Include credentials in the proxy URL: `http://user:pass@localhost:8080`
   - Or implement authentication in the proxy server

4. **Performance issues**:
   - Adjust proxy timeout settings
   - Consider running proxy on a different port
   - Check network latency between CLI and proxy

### Debug Commands:

```bash
# Test proxy connectivity
curl -x http://localhost:8080 https://api.openai.com/v1/models

# Check environment variables
env | grep -i proxy

# Test DNS resolution
nslookup api.openai.com

# Check proxy server logs
tail -f traffic.log | grep -i error
```

## Security Considerations

1. **Sensitive Data**: The proxy logs may contain API keys and request/response data
2. **Network Security**: Run proxy on localhost or secure networks only  
3. **Log Rotation**: Configure log rotation to prevent disk space issues
4. **Access Control**: Restrict access to proxy server and log files

## Advanced Configuration

### Custom Certificate Authority

For HTTPS interception with custom certificates:

```bash
# Generate CA certificate
openssl genrsa -out ca-key.pem 4096
openssl req -new -x509 -days 365 -key ca-key.pem -out ca-cert.pem

# Configure proxy to use custom CA
./bin/proxy -port 8080 -tls-cert ca-cert.pem -tls-key ca-key.pem
```

### Load Balancing

For high-volume scenarios, run multiple proxy instances:

```bash
# Start multiple proxy instances
./bin/proxy -port 8080 -log-file traffic-1.log &
./bin/proxy -port 8081 -log-file traffic-2.log &

# Use HAProxy or nginx for load balancing
```

### Integration with Monitoring Systems

Export logs to external systems:

```bash
# Export to Elasticsearch
./bin/parser -input traffic.log -format json | curl -X POST "localhost:9200/codex-traffic/_doc" -H "Content-Type: application/json" -d @-

# Export to Prometheus metrics endpoint
./bin/proxy -port 8080 -metrics-port 9090
```
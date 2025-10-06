# Quick Start Guide - Codex CLI Proxy Parser

## 🚀 Setup in 2 Minutes

### 1. Build the Proxy

```bash
cd proxy-parser
make install
# or
go build -o bin/codex-proxy ./cmd/proxy
```

### 2. Start the Proxy

```bash
# Simple start
make run

# Or manually
./bin/codex-proxy start --port 8080 --output traffic.json --verbose
```

### 3. Configure Codex CLI

In a new terminal:

```bash
# Set proxy environment variables
export HTTP_PROXY=http://localhost:8080
export HTTPS_PROXY=http://localhost:8080

# Run codex commands
codex chat "What is a proxy server?"
```

### 4. Analyze Traffic

```bash
# View GenAI traffic only
./bin/codex-proxy analyze --input traffic.json --genai-only

# Generate statistics
./bin/codex-proxy stats --input traffic.json
```

## 📊 What Gets Captured?

The proxy captures and analyzes:

- **All HTTP/HTTPS requests** from Codex CLI
- **GenAI API calls** to OpenAI, Anthropic, etc.
- **Request/Response bodies** including prompts and completions
- **Authentication tokens** (redacted in logs)
- **Streaming responses** (SSE events)
- **Model usage** (GPT-4, Claude, etc.)
- **Token usage and costs**

## 🔍 Key Features

### Real-time Monitoring
```bash
./bin/codex-proxy start --analyze --genai-highlight
```

### Filter Specific Traffic
```bash
# Only capture certain endpoints
./bin/codex-proxy start --config config.yaml
```

### HTTPS Interception
```bash
# Generate CA certificate
./bin/codex-proxy generate-cert

# Trust the certificate (macOS)
security add-trusted-cert -d -r trustRoot -k ~/Library/Keychains/login.keychain ~/.codex-proxy/ca-cert.pem
```

## 🐳 Docker Usage

```bash
# Build image
docker build -t codex-proxy .

# Run container
docker run -p 8080:8080 -v $(pwd)/traffic:/data codex-proxy

# Configure Codex CLI
export HTTP_PROXY=http://localhost:8080
```

## 📈 Sample Output

### GenAI Request Detection
```
🤖 GenAI Request [Session: 123]
   Model: gpt-4
   POST https://api.openai.com/v1/chat/completions
   Time: 2024-01-20T10:30:00Z
   ✓ Response: 200
```

### Traffic Statistics
```json
{
  "total_requests": 42,
  "genai_requests": 15,
  "models": {
    "gpt-4": 10,
    "gpt-3.5-turbo": 5
  },
  "total_tokens": 5234
}
```

## 🛠️ Troubleshooting

### Proxy Not Capturing Traffic?

1. Check environment variables:
```bash
env | grep -i proxy
```

2. Test proxy directly:
```bash
curl -x http://localhost:8080 http://httpbin.org/ip
```

3. For Node.js apps, try:
```bash
export NODE_OPTIONS="--proxy-server=http://localhost:8080"
```

### SSL/TLS Errors?

1. Generate and install CA certificate:
```bash
./bin/codex-proxy generate-cert
# Follow the instructions printed
```

2. Or disable verification (dev only):
```bash
export NODE_TLS_REJECT_UNAUTHORIZED=0
```

## 📝 Configuration Options

Create `config.yaml`:

```yaml
proxy:
  port: 8080
  
analysis:
  genai_patterns:
    - "gpt-4"
    - "claude"
    
filtering:
  include_endpoints:
    - "/api/codex"
    - "/v1/chat"
```

Run with: `./bin/codex-proxy start --config config.yaml`

## 🎯 Common Use Cases

### Monitor AI Model Usage
Track which models are being used and how often:
```bash
./bin/codex-proxy stats --input traffic.json | jq '.genai.models'
```

### Debug API Errors
Capture full request/response for debugging:
```bash
./bin/codex-proxy start --verbose --output debug.json
```

### Track Token Usage
Monitor token consumption and costs:
```bash
./bin/codex-proxy analyze --input traffic.json | grep -i token
```

### Filter Sensitive Data
Exclude headers and sensitive information:
```yaml
filtering:
  exclude_headers:
    - "Authorization"
    - "Cookie"
```

## 🔗 Integration Examples

### With Shell Scripts
```bash
#!/bin/bash
export HTTP_PROXY=http://localhost:8080
codex chat "Explain this code: $(cat script.sh)"
```

### With Node.js
```javascript
// Use with Node.js HTTP clients
process.env.HTTP_PROXY = 'http://localhost:8080';
const codex = require('@openai/codex');
```

### With Python
```python
import os
os.environ['HTTP_PROXY'] = 'http://localhost:8080'
# Your codex code here
```

## 📚 More Information

- Full documentation: [README.md](README.md)
- Configuration details: [config.yaml](config.yaml)
- Example scripts: [examples/](examples/)

## Need Help?

Run `./bin/codex-proxy --help` for all available commands and options.
# Codex CLI Proxy Parser - Quick Start Guide

## 🚀 Get Started in 5 Minutes

This guide will help you quickly set up the Codex CLI traffic proxy and parser.

### Prerequisites

- Go 1.21+ installed
- Codex CLI installed (`npm install -g @openai/codex`)
- Basic terminal knowledge

### 1. Setup (1 minute)

```bash
cd /workspace/proxy-parser
./scripts/setup-proxy.sh
```

### 2. Start Proxy (30 seconds)

```bash
./scripts/start-proxy.sh
```

### 3. Configure Codex CLI (30 seconds)

```bash
# In a new terminal
source .env
codex --help
```

### 4. Test Integration (2 minutes)

```bash
./examples/test-codex-integration.sh
```

### 5. Analyze Traffic (1 minute)

```bash
./scripts/analyze-traffic.sh --summary
```

## 🎯 What You Get

- **Traffic Monitoring**: All Codex CLI network requests are intercepted
- **GenAI Detection**: Automatically identifies AI/ML requests
- **Vibe Coding Detection**: Detects coding assistant interactions
- **Detailed Logging**: JSON logs with request/response data
- **Easy Analysis**: Built-in traffic analysis tools

## 📊 Sample Output

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
  "request_id": "req_123456"
}
```

## 🔧 Quick Commands

```bash
# Start proxy
make run

# Stop proxy
make stop

# View logs
make logs

# Analyze traffic
make analyze

# Docker setup
make docker-run
```

## 📁 Key Files

- `config.yaml` - Main configuration
- `logs/traffic.json` - Parsed traffic data
- `logs/proxy.log` - Proxy server logs
- `examples/` - Configuration examples
- `scripts/` - Management scripts

## 🆘 Need Help?

- **Full Documentation**: See `README.md`
- **Setup Guide**: See `docs/SETUP_GUIDE.md`
- **Troubleshooting**: Check the troubleshooting section in README
- **Examples**: See `examples/` directory

## 🎉 You're Done!

Your Codex CLI traffic is now being monitored and parsed. Check the logs directory for detailed traffic analysis.
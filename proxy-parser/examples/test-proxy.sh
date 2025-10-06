#!/bin/bash

# Test script for Codex Proxy Parser
# This script demonstrates how to use the proxy with the Codex CLI

set -e

PROXY_PORT=${PROXY_PORT:-8080}
PROXY_HOST=${PROXY_HOST:-localhost}

echo "🚀 Codex Proxy Test Script"
echo "=========================="
echo ""

# Check if proxy binary exists
if [ ! -f "./codex-proxy" ]; then
    echo "Building proxy server..."
    go build -o codex-proxy ./cmd/proxy
fi

# Function to cleanup on exit
cleanup() {
    echo ""
    echo "Stopping proxy server..."
    if [ ! -z "$PROXY_PID" ]; then
        kill $PROXY_PID 2>/dev/null || true
    fi
}
trap cleanup EXIT

# Start proxy server in background
echo "Starting proxy server on port $PROXY_PORT..."
./codex-proxy start --port $PROXY_PORT --output test-traffic.json --verbose &
PROXY_PID=$!

# Wait for proxy to start
sleep 2

# Check if proxy is running
if ! kill -0 $PROXY_PID 2>/dev/null; then
    echo "❌ Failed to start proxy server"
    exit 1
fi

echo "✅ Proxy server started (PID: $PROXY_PID)"
echo ""

# Set proxy environment variables
export HTTP_PROXY=http://$PROXY_HOST:$PROXY_PORT
export HTTPS_PROXY=http://$PROXY_HOST:$PROXY_PORT
export http_proxy=$HTTP_PROXY
export https_proxy=$HTTPS_PROXY

echo "📝 Environment configured:"
echo "   HTTP_PROXY=$HTTP_PROXY"
echo "   HTTPS_PROXY=$HTTPS_PROXY"
echo ""

# Test with curl first
echo "Testing proxy with curl..."
curl -s -x $HTTP_PROXY http://httpbin.org/ip | jq . || echo "curl test failed (install jq for pretty output)"
echo ""

# Check if codex CLI is available
if command -v codex &> /dev/null; then
    echo "Testing with Codex CLI..."
    echo "Running: codex --version"
    codex --version || true
    
    echo ""
    echo "You can now run Codex commands, for example:"
    echo "  codex chat \"Hello, how are you?\""
    echo "  codex explain \"What is a proxy server?\""
else
    echo "⚠️  Codex CLI not found in PATH"
    echo "   Install it first or run from the codex-cli directory"
fi

echo ""
echo "📊 Proxy is running and capturing traffic"
echo "   Output file: test-traffic.json"
echo "   Press Ctrl+C to stop and analyze the traffic"
echo ""

# Wait for user to stop
wait $PROXY_PID
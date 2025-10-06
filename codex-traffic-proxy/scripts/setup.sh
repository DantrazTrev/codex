#!/bin/bash

# Codex Traffic Proxy Setup Script
# This script helps set up and configure the proxy for monitoring Codex CLI traffic

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

echo -e "${BLUE}🚀 Codex Traffic Proxy Setup${NC}"
echo "=================================="

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo -e "${RED}❌ Go is not installed. Please install Go 1.21 or later.${NC}"
    exit 1
fi

# Check Go version
GO_VERSION=$(go version | grep -oE 'go[0-9]+\.[0-9]+' | sed 's/go//')
GO_MAJOR=$(echo $GO_VERSION | cut -d'.' -f1)
GO_MINOR=$(echo $GO_VERSION | cut -d'.' -f2)

if [ "$GO_MAJOR" -lt 1 ] || ([ "$GO_MAJOR" -eq 1 ] && [ "$GO_MINOR" -lt 21 ]); then
    echo -e "${RED}❌ Go version $GO_VERSION is too old. Please install Go 1.21 or later.${NC}"
    exit 1
fi

echo -e "${GREEN}✅ Go $GO_VERSION is installed${NC}"

# Navigate to project root
cd "$PROJECT_ROOT"

# Install dependencies
echo -e "${YELLOW}📦 Installing dependencies...${NC}"
go mod download
go mod tidy

# Build the proxy
echo -e "${YELLOW}🔨 Building proxy...${NC}"
go build -o codex-traffic-proxy ./cmd

if [ ! -f "codex-traffic-proxy" ]; then
    echo -e "${RED}❌ Build failed${NC}"
    exit 1
fi

echo -e "${GREEN}✅ Proxy built successfully${NC}"

# Initialize configuration if it doesn't exist
CONFIG_DIR="$HOME/.codex-traffic-proxy"
CONFIG_FILE="$CONFIG_DIR/config.yaml"

if [ ! -f "$CONFIG_FILE" ]; then
    echo -e "${YELLOW}⚙️  Initializing configuration...${NC}"
    ./codex-traffic-proxy config init
    echo -e "${GREEN}✅ Configuration initialized at $CONFIG_FILE${NC}"
else
    echo -e "${GREEN}✅ Configuration already exists at $CONFIG_FILE${NC}"
fi

# Check if proxy is already running
if pgrep -f "codex-traffic-proxy start" > /dev/null; then
    echo -e "${YELLOW}⚠️  Proxy appears to be already running${NC}"
    echo "Would you like to stop the existing proxy first? (y/n)"
    read -r response
    if [[ "$response" == "y" || "$response" == "Y" ]]; then
        pkill -f "codex-traffic-proxy start"
        sleep 2
    fi
fi

# Set up environment variables
echo -e "${BLUE}🌐 Setting up environment variables...${NC}"
echo ""
echo "To use the proxy with Codex CLI, set these environment variables:"
echo ""
echo "export HTTP_PROXY=http://127.0.0.1:8080"
echo "export HTTPS_PROXY=http://127.0.0.1:8080"
echo ""
echo "You can add these to your ~/.bashrc or ~/.zshrc for persistence."
echo ""

# Ask if user wants to set variables now
echo "Would you like to set the proxy environment variables now? (y/n)"
read -r response

if [[ "$response" == "y" || "$response" == "Y" ]]; then
    export HTTP_PROXY=http://127.0.0.1:8080
    export HTTPS_PROXY=http://127.0.0.1:8080
    echo -e "${GREEN}✅ Environment variables set for current session${NC}"
    echo -e "${YELLOW}⚠️  Note: These will only persist for the current shell session${NC}"
fi

# Start the proxy
echo -e "${GREEN}🚀 Starting proxy server...${NC}"
echo ""
echo "The proxy will start in the background. Press Ctrl+C to stop it."
echo ""
echo "Once running, you can test the integration with:"
echo "  codex --help"
echo ""

# Start proxy in background
./codex-traffic-proxy start &
PROXY_PID=$!

echo -e "${GREEN}✅ Proxy started with PID: $PROXY_PID${NC}"
echo ""
echo -e "${BLUE}📋 Next Steps:${NC}"
echo "1. Set HTTP_PROXY and HTTPS_PROXY environment variables (if not done above)"
echo "2. Run Codex CLI commands - all traffic will be monitored"
echo "3. Monitor the proxy logs above for intercepted requests"
echo ""
echo -e "${YELLOW}💡 Tip: Use './codex-traffic-proxy config init' to reconfigure settings${NC}"
echo ""

# Wait for proxy process
wait $PROXY_PID
#!/bin/bash

# Test Codex CLI Integration Script
# This script demonstrates how to use the proxy with Codex CLI

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
PROXY_PORT=${PROXY_PORT:-8080}
PROXY_HOST=${PROXY_HOST:-127.0.0.1}
CONFIG_FILE=${CONFIG_FILE:-./examples/development-config.yaml}

echo -e "${GREEN}Testing Codex CLI Integration with Proxy${NC}"
echo ""

# Function to show help
show_help() {
    echo "Usage: $0 [OPTIONS]"
    echo "Options:"
    echo "  -p, --port PORT      Proxy port (default: 8080)"
    echo "  -h, --host HOST      Proxy host (default: 127.0.0.1)"
    echo "  -c, --config FILE    Config file (default: ./examples/development-config.yaml)"
    echo "  --no-proxy           Test without proxy"
    echo "  --help               Show this help message"
}

# Parse command line arguments
NO_PROXY=false

while [[ $# -gt 0 ]]; do
    case $1 in
        -p|--port)
            PROXY_PORT="$2"
            shift 2
            ;;
        -h|--host)
            PROXY_HOST="$2"
            shift 2
            ;;
        -c|--config)
            CONFIG_FILE="$2"
            shift 2
            ;;
        --no-proxy)
            NO_PROXY=true
            shift
            ;;
        --help)
            show_help
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            show_help
            exit 1
            ;;
    esac
done

# Check if Codex CLI is installed
if ! command -v codex &> /dev/null; then
    echo -e "${RED}Error: Codex CLI is not installed.${NC}"
    echo "Please install it first:"
    echo "  npm install -g @openai/codex"
    exit 1
fi

echo -e "${GREEN}Codex CLI found: $(codex --version 2>/dev/null || echo 'version unknown')${NC}"

# Check if proxy binary exists
if [ ! -f "./bin/codex-proxy" ]; then
    echo -e "${YELLOW}Proxy binary not found. Building...${NC}"
    make build
fi

# Start proxy if not running
if [ "$NO_PROXY" = false ]; then
    echo -e "${YELLOW}Starting proxy server...${NC}"
    
    # Check if proxy is already running
    if curl -s http://$PROXY_HOST:$PROXY_PORT >/dev/null 2>&1; then
        echo -e "${GREEN}Proxy is already running on $PROXY_HOST:$PROXY_PORT${NC}"
    else
        # Start proxy in background
        ./scripts/start-proxy.sh --config "$CONFIG_FILE" --daemon
        
        # Wait for proxy to start
        echo -e "${YELLOW}Waiting for proxy to start...${NC}"
        for i in {1..10}; do
            if curl -s http://$PROXY_HOST:$PROXY_PORT >/dev/null 2>&1; then
                echo -e "${GREEN}Proxy started successfully!${NC}"
                break
            fi
            sleep 1
        done
        
        if ! curl -s http://$PROXY_HOST:$PROXY_PORT >/dev/null 2>&1; then
            echo -e "${RED}Failed to start proxy${NC}"
            exit 1
        fi
    fi
    
    # Set proxy environment variables
    export HTTP_PROXY=http://$PROXY_HOST:$PROXY_PORT
    export HTTPS_PROXY=http://$PROXY_HOST:$PROXY_PORT
    export NO_PROXY=localhost,127.0.0.1,::1
    
    echo -e "${GREEN}Proxy environment configured${NC}"
    echo "  HTTP_PROXY: $HTTP_PROXY"
    echo "  HTTPS_PROXY: $HTTPS_PROXY"
    echo "  NO_PROXY: $NO_PROXY"
else
    echo -e "${YELLOW}Running without proxy${NC}"
    unset HTTP_PROXY
    unset HTTPS_PROXY
    unset NO_PROXY
fi

echo ""

# Test 1: Basic Codex CLI functionality
echo -e "${BLUE}=== Test 1: Basic Codex CLI Functionality ===${NC}"
echo ""

echo -e "${YELLOW}Running: codex --help${NC}"
if codex --help >/dev/null 2>&1; then
    echo -e "${GREEN}✓ Codex CLI help command successful${NC}"
else
    echo -e "${RED}✗ Codex CLI help command failed${NC}"
fi

echo ""

# Test 2: Codex exec command
echo -e "${BLUE}=== Test 2: Codex Exec Command ===${NC}"
echo ""

echo -e "${YELLOW}Running: codex exec 'echo Hello from Codex'${NC}"
if codex exec 'echo Hello from Codex' >/dev/null 2>&1; then
    echo -e "${GREEN}✓ Codex exec command successful${NC}"
else
    echo -e "${RED}✗ Codex exec command failed${NC}"
fi

echo ""

# Test 3: Check proxy logs
if [ "$NO_PROXY" = false ]; then
    echo -e "${BLUE}=== Test 3: Proxy Logs Analysis ===${NC}"
    echo ""
    
    echo -e "${YELLOW}Checking proxy logs...${NC}"
    
    if [ -f "logs/traffic.json" ]; then
        local request_count=$(jq -r 'length' logs/traffic.json 2>/dev/null || echo "0")
        echo -e "${GREEN}✓ Traffic log found with $request_count requests${NC}"
        
        if [ "$request_count" -gt 0 ]; then
            echo -e "${YELLOW}Sample parsed data:${NC}"
            jq -r '.[0]' logs/traffic.json 2>/dev/null | head -10
        fi
    else
        echo -e "${YELLOW}No traffic log found yet${NC}"
    fi
    
    echo ""
fi

# Test 4: Network connectivity test
echo -e "${BLUE}=== Test 4: Network Connectivity ===${NC}"
echo ""

echo -e "${YELLOW}Testing network connectivity...${NC}"

# Test direct connection
if curl -s --max-time 10 https://api.openai.com/v1/models >/dev/null 2>&1; then
    echo -e "${GREEN}✓ Direct connection to OpenAI API successful${NC}"
else
    echo -e "${YELLOW}⚠ Direct connection to OpenAI API failed (may be expected)${NC}"
fi

# Test proxy connection
if [ "$NO_PROXY" = false ]; then
    if curl -s --max-time 10 -x http://$PROXY_HOST:$PROXY_PORT https://httpbin.org/ip >/dev/null 2>&1; then
        echo -e "${GREEN}✓ Proxy connection successful${NC}"
    else
        echo -e "${RED}✗ Proxy connection failed${NC}"
    fi
fi

echo ""

# Test 5: Traffic analysis
if [ "$NO_PROXY" = false ] && [ -f "logs/traffic.json" ]; then
    echo -e "${BLUE}=== Test 5: Traffic Analysis ===${NC}"
    echo ""
    
    echo -e "${YELLOW}Running traffic analysis...${NC}"
    ./scripts/analyze-traffic.sh --summary
fi

echo ""

# Cleanup
if [ "$NO_PROXY" = false ]; then
    echo -e "${YELLOW}Cleaning up...${NC}"
    ./scripts/stop-proxy.sh
    echo -e "${GREEN}Proxy stopped${NC}"
fi

echo ""
echo -e "${GREEN}Integration test complete!${NC}"

# Show next steps
echo ""
echo -e "${BLUE}Next Steps:${NC}"
echo "1. Review the traffic logs in logs/traffic.json"
echo "2. Run traffic analysis: ./scripts/analyze-traffic.sh"
echo "3. Customize the configuration in $CONFIG_FILE"
echo "4. Set up monitoring and alerting for production use"
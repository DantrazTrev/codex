#!/bin/bash

# Start Codex Proxy Script
# This script starts the proxy server with proper configuration

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Default values
CONFIG_FILE=${CONFIG_FILE:-./config.yaml}
VERBOSE=${VERBOSE:-false}
DAEMON=${DAEMON:-false}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -c|--config)
            CONFIG_FILE="$2"
            shift 2
            ;;
        -v|--verbose)
            VERBOSE=true
            shift
            ;;
        -d|--daemon)
            DAEMON=true
            shift
            ;;
        -h|--help)
            echo "Usage: $0 [OPTIONS]"
            echo "Options:"
            echo "  -c, --config FILE    Configuration file (default: ./config.yaml)"
            echo "  -v, --verbose        Enable verbose output"
            echo "  -d, --daemon         Run as daemon"
            echo "  -h, --help           Show this help message"
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            exit 1
            ;;
    esac
done

# Check if proxy binary exists
if [ ! -f "./bin/codex-proxy" ]; then
    echo -e "${RED}Error: Proxy binary not found. Run setup-proxy.sh first.${NC}"
    exit 1
fi

# Check if config file exists
if [ ! -f "$CONFIG_FILE" ]; then
    echo -e "${RED}Error: Configuration file not found: $CONFIG_FILE${NC}"
    exit 1
fi

# Create logs directory
mkdir -p logs

echo -e "${GREEN}Starting Codex Proxy...${NC}"
echo -e "${YELLOW}Config: $CONFIG_FILE${NC}"
echo -e "${YELLOW}Verbose: $VERBOSE${NC}"

# Build command
CMD="./bin/codex-proxy --config $CONFIG_FILE"
if [ "$VERBOSE" = true ]; then
    CMD="$CMD --verbose"
fi

# Start proxy
if [ "$DAEMON" = true ]; then
    echo -e "${YELLOW}Starting as daemon...${NC}"
    nohup $CMD > logs/proxy.log 2>&1 &
    PID=$!
    echo $PID > logs/proxy.pid
    echo -e "${GREEN}Proxy started with PID: $PID${NC}"
    echo -e "${YELLOW}Logs: logs/proxy.log${NC}"
    echo -e "${YELLOW}PID file: logs/proxy.pid${NC}"
else
    echo -e "${GREEN}Starting proxy (Ctrl+C to stop)...${NC}"
    exec $CMD
fi
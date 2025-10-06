#!/bin/bash

# Stop Codex Proxy Script
# This script stops the running proxy server

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

PID_FILE="logs/proxy.pid"

# Check if PID file exists
if [ ! -f "$PID_FILE" ]; then
    echo -e "${YELLOW}No PID file found. Proxy may not be running.${NC}"
    exit 0
fi

# Read PID
PID=$(cat "$PID_FILE")

# Check if process is running
if ! kill -0 "$PID" 2>/dev/null; then
    echo -e "${YELLOW}Process $PID is not running.${NC}"
    rm -f "$PID_FILE"
    exit 0
fi

echo -e "${YELLOW}Stopping proxy (PID: $PID)...${NC}"

# Send TERM signal
kill -TERM "$PID"

# Wait for process to stop
for i in {1..10}; do
    if ! kill -0 "$PID" 2>/dev/null; then
        echo -e "${GREEN}Proxy stopped successfully.${NC}"
        rm -f "$PID_FILE"
        exit 0
    fi
    sleep 1
done

# Force kill if still running
echo -e "${YELLOW}Force stopping proxy...${NC}"
kill -KILL "$PID" 2>/dev/null || true

# Clean up
rm -f "$PID_FILE"
echo -e "${GREEN}Proxy stopped.${NC}"
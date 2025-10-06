#!/bin/bash

# Codex CLI Proxy Setup Script
# This script sets up the proxy environment for monitoring Codex CLI traffic

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
PROXY_PORT=${PROXY_PORT:-8080}
PROXY_HOST=${PROXY_HOST:-127.0.0.1}
CONFIG_FILE=${CONFIG_FILE:-./config.yaml}

echo -e "${GREEN}Setting up Codex CLI Proxy...${NC}"

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo -e "${RED}Error: Go is not installed. Please install Go 1.21 or later.${NC}"
    exit 1
fi

# Check Go version
GO_VERSION=$(go version | cut -d' ' -f3 | sed 's/go//')
REQUIRED_VERSION="1.21"
if [ "$(printf '%s\n' "$REQUIRED_VERSION" "$GO_VERSION" | sort -V | head -n1)" != "$REQUIRED_VERSION" ]; then
    echo -e "${RED}Error: Go version $GO_VERSION is too old. Please install Go 1.21 or later.${NC}"
    exit 1
fi

echo -e "${GREEN}Go version $GO_VERSION detected.${NC}"

# Build the proxy
echo -e "${YELLOW}Building proxy server...${NC}"
go build -o bin/codex-proxy ./cmd/proxy

if [ $? -eq 0 ]; then
    echo -e "${GREEN}Proxy server built successfully!${NC}"
else
    echo -e "${RED}Failed to build proxy server.${NC}"
    exit 1
fi

# Create log directory
mkdir -p logs

# Set up environment variables
echo -e "${YELLOW}Setting up environment variables...${NC}"

# Create environment file
cat > .env << EOF
# Codex Proxy Environment Variables
export HTTP_PROXY=http://${PROXY_HOST}:${PROXY_PORT}
export HTTPS_PROXY=http://${PROXY_HOST}:${PROXY_PORT}
export NO_PROXY=localhost,127.0.0.1,::1

# Codex CLI specific
export CODEX_PROXY_URL=http://${PROXY_HOST}:${PROXY_PORT}
EOF

echo -e "${GREEN}Environment variables configured in .env file${NC}"

# Create systemd service file (if running as root)
if [ "$EUID" -eq 0 ]; then
    echo -e "${YELLOW}Creating systemd service...${NC}"
    
    cat > /etc/systemd/system/codex-proxy.service << EOF
[Unit]
Description=Codex CLI Traffic Proxy
After=network.target

[Service]
Type=simple
User=codex-proxy
Group=codex-proxy
WorkingDirectory=$(pwd)
ExecStart=$(pwd)/bin/codex-proxy --config $(pwd)/${CONFIG_FILE}
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

    # Create user for the service
    if ! id "codex-proxy" &>/dev/null; then
        useradd -r -s /bin/false codex-proxy
    fi
    
    # Set permissions
    chown -R codex-proxy:codex-proxy .
    chmod +x bin/codex-proxy
    
    echo -e "${GREEN}Systemd service created. Use 'sudo systemctl start codex-proxy' to start.${NC}"
fi

echo -e "${GREEN}Setup complete!${NC}"
echo ""
echo -e "${YELLOW}To use the proxy:${NC}"
echo "1. Start the proxy: ./bin/codex-proxy --config ${CONFIG_FILE}"
echo "2. Set environment variables: source .env"
echo "3. Run codex CLI commands normally"
echo ""
echo -e "${YELLOW}To monitor traffic:${NC}"
echo "- Check logs in the logs/ directory"
echo "- View real-time output in the console"
echo "- Configure output format in ${CONFIG_FILE}"
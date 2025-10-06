#!/bin/bash

# Docker Setup Script for Codex Proxy
# This script sets up and runs the proxy using Docker

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
IMAGE_NAME="codex-proxy"
CONTAINER_NAME="codex-proxy"
PORT=${PORT:-8080}
CONFIG_FILE=${CONFIG_FILE:-./config.yaml}

echo -e "${GREEN}Setting up Codex Proxy with Docker...${NC}"

# Check if Docker is installed
if ! command -v docker &> /dev/null; then
    echo -e "${RED}Error: Docker is not installed. Please install Docker first.${NC}"
    exit 1
fi

# Check if Docker Compose is installed
if ! command -v docker-compose &> /dev/null; then
    echo -e "${RED}Error: Docker Compose is not installed. Please install Docker Compose first.${NC}"
    exit 1
fi

echo -e "${GREEN}Docker and Docker Compose detected.${NC}"

# Create logs directory
mkdir -p logs

# Check if config file exists
if [ ! -f "$CONFIG_FILE" ]; then
    echo -e "${YELLOW}Config file not found. Using default configuration.${NC}"
    cp config.yaml "$CONFIG_FILE"
fi

# Build Docker image
echo -e "${YELLOW}Building Docker image...${NC}"
docker build -t $IMAGE_NAME .

if [ $? -eq 0 ]; then
    echo -e "${GREEN}Docker image built successfully!${NC}"
else
    echo -e "${RED}Failed to build Docker image.${NC}"
    exit 1
fi

# Create docker-compose override file
cat > docker-compose.override.yml << EOF
version: '3.8'

services:
  codex-proxy:
    ports:
      - "$PORT:8080"
    volumes:
      - ./$CONFIG_FILE:/app/config.yaml:ro
      - ./logs:/app/logs
EOF

echo -e "${GREEN}Docker setup complete!${NC}"
echo ""
echo -e "${YELLOW}To start the proxy:${NC}"
echo "  docker-compose up -d"
echo ""
echo -e "${YELLOW}To view logs:${NC}"
echo "  docker-compose logs -f"
echo ""
echo -e "${YELLOW}To stop the proxy:${NC}"
echo "  docker-compose down"
echo ""
echo -e "${YELLOW}To rebuild:${NC}"
echo "  docker-compose build --no-cache"
.PHONY: build run clean test analyze help install cert

# Variables
BINARY_NAME=codex-proxy
MAIN_PATH=./cmd/proxy
OUTPUT_DIR=./bin

# Default target
help:
	@echo "Codex Proxy Parser - Makefile Help"
	@echo "=================================="
	@echo ""
	@echo "Available targets:"
	@echo "  make build      - Build the proxy binary"
	@echo "  make install    - Install dependencies and build"
	@echo "  make run        - Run the proxy server (port 8080)"
	@echo "  make cert       - Generate CA certificate for HTTPS"
	@echo "  make test       - Run tests"
	@echo "  make analyze    - Analyze captured traffic"
	@echo "  make clean      - Clean build artifacts"
	@echo "  make help       - Show this help message"
	@echo ""
	@echo "Environment variables:"
	@echo "  PROXY_PORT      - Proxy port (default: 8080)"
	@echo "  OUTPUT_FILE     - Output file for traffic (default: traffic.json)"
	@echo "  CONFIG_FILE     - Configuration file path"

# Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(OUTPUT_DIR)
	@go build -o $(OUTPUT_DIR)/$(BINARY_NAME) $(MAIN_PATH)
	@echo "✅ Build complete: $(OUTPUT_DIR)/$(BINARY_NAME)"

# Install dependencies and build
install:
	@echo "Installing dependencies..."
	@go mod download
	@go mod tidy
	@$(MAKE) build
	@echo "✅ Installation complete"

# Run the proxy server
run: build
	@echo "Starting proxy server..."
	$(OUTPUT_DIR)/$(BINARY_NAME) start \
		--port $${PROXY_PORT:-8080} \
		--output $${OUTPUT_FILE:-traffic.json} \
		--verbose \
		--analyze

# Run with configuration file
run-config: build
	@if [ -z "$(CONFIG_FILE)" ]; then \
		echo "❌ CONFIG_FILE not specified"; \
		echo "Usage: make run-config CONFIG_FILE=config.yaml"; \
		exit 1; \
	fi
	$(OUTPUT_DIR)/$(BINARY_NAME) start --config $(CONFIG_FILE)

# Generate CA certificate
cert: build
	@echo "Generating CA certificate..."
	@$(OUTPUT_DIR)/$(BINARY_NAME) generate-cert

# Run tests
test:
	@echo "Running tests..."
	@go test -v ./...

# Analyze captured traffic
analyze: build
	@if [ -z "$(INPUT_FILE)" ]; then \
		echo "Analyzing default traffic.json..."; \
		$(OUTPUT_DIR)/$(BINARY_NAME) analyze --input traffic.json --verbose; \
	else \
		echo "Analyzing $(INPUT_FILE)..."; \
		$(OUTPUT_DIR)/$(BINARY_NAME) analyze --input $(INPUT_FILE) --verbose; \
	fi

# Generate statistics
stats: build
	@if [ -z "$(INPUT_FILE)" ]; then \
		$(OUTPUT_DIR)/$(BINARY_NAME) stats --input traffic.json; \
	else \
		$(OUTPUT_DIR)/$(BINARY_NAME) stats --input $(INPUT_FILE); \
	fi

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(OUTPUT_DIR)
	@rm -f $(BINARY_NAME)
	@rm -f traffic.json traffic.log test-traffic.json
	@echo "✅ Clean complete"

# Development targets
dev: build
	@echo "Starting in development mode..."
	$(OUTPUT_DIR)/$(BINARY_NAME) start \
		--port 8080 \
		--output dev-traffic.json \
		--verbose \
		--analyze \
		--genai-highlight

# Quick test with curl
test-proxy:
	@echo "Testing proxy with curl..."
	@curl -x http://localhost:8080 http://httpbin.org/ip || echo "Proxy may not be running"

# Docker build (if needed)
docker-build:
	@echo "Building Docker image..."
	@docker build -t codex-proxy:latest -f Dockerfile .

docker-run:
	@docker run -p 8080:8080 -v $${PWD}/traffic:/data codex-proxy:latest
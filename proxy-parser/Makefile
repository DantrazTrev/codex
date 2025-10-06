# Codex Proxy Parser Makefile

.PHONY: help build run stop clean test docker-build docker-run docker-stop setup install

# Default target
help:
	@echo "Codex Proxy Parser - Available targets:"
	@echo ""
	@echo "  build          - Build the proxy binary"
	@echo "  run            - Start the proxy server"
	@echo "  stop           - Stop the proxy server"
	@echo "  clean          - Clean build artifacts"
	@echo "  test           - Run tests"
	@echo "  setup          - Run initial setup"
	@echo "  install        - Install dependencies"
	@echo ""
	@echo "Docker targets:"
	@echo "  docker-build   - Build Docker image"
	@echo "  docker-run     - Run with Docker Compose"
	@echo "  docker-stop    - Stop Docker containers"
	@echo ""
	@echo "Analysis targets:"
	@echo "  analyze        - Analyze traffic data"
	@echo "  logs           - Show logs"
	@echo "  status         - Show proxy status"

# Build the proxy binary
build:
	@echo "Building proxy binary..."
	go build -o bin/codex-proxy ./cmd/proxy
	@echo "Build complete!"

# Run the proxy server
run:
	@echo "Starting proxy server..."
	./scripts/start-proxy.sh

# Stop the proxy server
stop:
	@echo "Stopping proxy server..."
	./scripts/stop-proxy.sh

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf bin/
	rm -rf logs/
	rm -f .env
	@echo "Clean complete!"

# Run tests
test:
	@echo "Running tests..."
	go test ./...

# Install dependencies
install:
	@echo "Installing dependencies..."
	go mod tidy
	go mod download

# Run initial setup
setup:
	@echo "Running initial setup..."
	./scripts/setup-proxy.sh

# Build Docker image
docker-build:
	@echo "Building Docker image..."
	docker build -t codex-proxy .

# Run with Docker Compose
docker-run:
	@echo "Starting with Docker Compose..."
	docker-compose up -d

# Stop Docker containers
docker-stop:
	@echo "Stopping Docker containers..."
	docker-compose down

# Analyze traffic data
analyze:
	@echo "Analyzing traffic data..."
	./scripts/analyze-traffic.sh

# Show logs
logs:
	@echo "Showing logs..."
	tail -f logs/proxy.log

# Show proxy status
status:
	@echo "Proxy status:"
	@if [ -f logs/proxy.pid ]; then \
		PID=$$(cat logs/proxy.pid); \
		if kill -0 $$PID 2>/dev/null; then \
			echo "  Status: Running (PID: $$PID)"; \
		else \
			echo "  Status: Not running"; \
		fi; \
	else \
		echo "  Status: Not running"; \
	fi

# Development targets
dev: build run

# Production targets
prod: clean build
	@echo "Production build complete!"

# Full setup including Docker
full-setup: setup docker-build
	@echo "Full setup complete!"
# Codex Traffic Proxy Makefile

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
BINARY_NAME=codex-traffic-proxy
BINARY_UNIX=$(BINARY_NAME)_unix

# Build targets
.PHONY: all build clean test coverage deps help

all: deps test build

build:
	$(GOBUILD) -o $(BINARY_NAME) -v ./cmd

build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BINARY_UNIX) -v ./cmd

clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_UNIX)

test:
	$(GOTEST) -v ./...

test-coverage:
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html

deps:
	$(GOMOD) download
	$(GOMOD) tidy

run:
	$(GOBUILD) -o $(BINARY_NAME) ./cmd
	./$(BINARY_NAME) start

run-config:
	$(GOBUILD) -o $(BINARY_NAME) ./cmd
	./$(BINARY_NAME) config init

install-deps:
	$(GOGET) -u github.com/elazarl/goproxy
	$(GOGET) -u github.com/gorilla/mux
	$(GOGET) -u github.com/sirupsen/logrus
	$(GOGET) -u github.com/spf13/cobra
	$(GOGET) -u github.com/spf13/viper

fmt:
	$(GOCMD) fmt ./...

vet:
	$(GOCMD) vet ./...

lint:
	golangci-lint run

# Docker targets (if needed)
docker-build:
	docker build -t $(BINARY_NAME) .

docker-run:
	docker run -p 8080:8080 $(BINARY_NAME)

# Development helpers
dev-setup: install-deps deps
	@echo "Development environment setup complete"

dev-run: build run

# Help target
help:
	@echo "Available targets:"
	@echo "  build        - Build the binary"
	@echo "  build-linux  - Build for Linux"
	@echo "  clean        - Clean build files"
	@echo "  test         - Run tests"
	@echo "  test-coverage- Run tests with coverage"
	@echo "  deps         - Download dependencies"
	@echo "  run          - Build and run the proxy"
	@echo "  run-config   - Initialize default config"
	@echo "  fmt          - Format Go code"
	@echo "  vet          - Run go vet"
	@echo "  lint         - Run golangci-lint"
	@echo "  dev-setup    - Setup development environment"
	@echo "  dev-run      - Build and run for development"
	@echo "  help         - Show this help"
#!/bin/bash

# Codex CLI Traffic Monitor Setup Script

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

echo "🚀 Setting up Codex CLI Traffic Monitor..."

# Check Go installation
if ! command -v go &> /dev/null; then
    echo "❌ Go is not installed. Please install Go 1.21 or later."
    echo "   Visit: https://golang.org/doc/install"
    exit 1
fi

GO_VERSION=$(go version | cut -d' ' -f3 | sed 's/go//')
REQUIRED_VERSION="1.21"

if [ "$(printf '%s\n' "$REQUIRED_VERSION" "$GO_VERSION" | sort -V | head -n1)" != "$REQUIRED_VERSION" ]; then
    echo "❌ Go version $GO_VERSION is too old. Please install Go $REQUIRED_VERSION or later."
    exit 1
fi

echo "✅ Go version $GO_VERSION detected"

# Navigate to project directory
cd "$PROJECT_DIR"

# Initialize Go module if not exists
if [ ! -f "go.mod" ]; then
    echo "📦 Initializing Go module..."
    go mod init codex-traffic-monitor
fi

# Download dependencies
echo "📦 Downloading dependencies..."
go mod tidy

# Create necessary directories
echo "📁 Creating directories..."
mkdir -p bin
mkdir -p logs
mkdir -p data

# Build binaries
echo "🔨 Building proxy server..."
go build -o bin/proxy ./cmd/proxy

echo "🔨 Building traffic parser..."
go build -o bin/parser ./cmd/parser

# Make scripts executable
chmod +x bin/proxy
chmod +x bin/parser
chmod +x scripts/*.sh

# Create default config files if they don't exist
if [ ! -f "config/proxy.yaml" ]; then
    echo "⚙️ Creating default proxy configuration..."
    # Config file already exists from our creation above
fi

if [ ! -f "config/parser.yaml" ]; then
    echo "⚙️ Creating default parser configuration..."
    # Config file already exists from our creation above
fi

# Create systemd service file (optional)
if command -v systemctl &> /dev/null; then
    echo "🔧 Creating systemd service file..."
    cat > codex-proxy.service << EOF
[Unit]
Description=Codex CLI Traffic Proxy
After=network.target

[Service]
Type=simple
User=$USER
WorkingDirectory=$PROJECT_DIR
ExecStart=$PROJECT_DIR/bin/proxy -port 8080 -log-file $PROJECT_DIR/logs/traffic.log
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF
    
    echo "   Service file created: codex-proxy.service"
    echo "   To install: sudo mv codex-proxy.service /etc/systemd/system/"
    echo "   To enable: sudo systemctl enable codex-proxy"
    echo "   To start: sudo systemctl start codex-proxy"
fi

# Create convenience scripts
echo "📝 Creating convenience scripts..."

# Start proxy script
cat > scripts/start-proxy.sh << 'EOF'
#!/bin/bash
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

echo "🚀 Starting Codex CLI Traffic Proxy..."
cd "$PROJECT_DIR"

# Create logs directory if it doesn't exist
mkdir -p logs

# Start proxy with default settings
./bin/proxy -port 8080 -log-file logs/traffic.log -verbose
EOF

# Stop proxy script
cat > scripts/stop-proxy.sh << 'EOF'
#!/bin/bash
echo "🛑 Stopping Codex CLI Traffic Proxy..."
pkill -f "proxy.*8080" || echo "No proxy process found"
EOF

# Analyze traffic script
cat > scripts/analyze-traffic.sh << 'EOF'
#!/bin/bash
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

cd "$PROJECT_DIR"

LOG_FILE="${1:-logs/traffic.log}"
OUTPUT_FILE="${2:-data/analysis.json}"

if [ ! -f "$LOG_FILE" ]; then
    echo "❌ Log file not found: $LOG_FILE"
    echo "Usage: $0 [log-file] [output-file]"
    exit 1
fi

echo "📊 Analyzing traffic from: $LOG_FILE"
./bin/parser -input "$LOG_FILE" -output "$OUTPUT_FILE" -format json

echo "✅ Analysis complete: $OUTPUT_FILE"

# Also generate a summary
echo "📋 Traffic Summary:"
./bin/parser -input "$LOG_FILE" -format summary
EOF

# Setup environment script
cat > scripts/setup-env.sh << 'EOF'
#!/bin/bash
echo "🔧 Setting up proxy environment variables..."

# Check if proxy is running
if ! curl -s --connect-timeout 2 http://localhost:8080 > /dev/null 2>&1; then
    echo "⚠️  Proxy server is not running on port 8080"
    echo "   Start it with: ./scripts/start-proxy.sh"
    echo ""
fi

echo "Add these lines to your shell profile (~/.bashrc, ~/.zshrc, etc.):"
echo ""
echo "# Codex CLI Traffic Monitoring"
echo "export HTTP_PROXY=http://localhost:8080"
echo "export HTTPS_PROXY=http://localhost:8080"
echo "export http_proxy=http://localhost:8080"
echo "export https_proxy=http://localhost:8080"
echo ""
echo "Or run this command to set them for the current session:"
echo "export HTTP_PROXY=http://localhost:8080 HTTPS_PROXY=http://localhost:8080"
EOF

chmod +x scripts/*.sh

# Create sample usage script
cat > scripts/demo.sh << 'EOF'
#!/bin/bash
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

echo "🎯 Codex CLI Traffic Monitor Demo"
echo "=================================="

cd "$PROJECT_DIR"

# Start proxy in background
echo "1. Starting proxy server..."
./bin/proxy -port 8080 -log-file logs/demo-traffic.log &
PROXY_PID=$!

# Wait for proxy to start
sleep 2

# Set proxy environment variables
export HTTP_PROXY=http://localhost:8080
export HTTPS_PROXY=http://localhost:8080

echo "2. Proxy environment variables set"

# Make some test requests
echo "3. Making test requests..."
curl -s "https://api.github.com/repos/microsoft/vscode" > /dev/null || true
curl -s "https://httpbin.org/get" > /dev/null || true

# Simulate AI API request (this will fail but will be logged)
curl -s -X POST "https://api.openai.com/v1/chat/completions" \
  -H "Content-Type: application/json" \
  -d '{"model":"gpt-3.5-turbo","messages":[{"role":"user","content":"Hello"}]}' > /dev/null 2>&1 || true

echo "4. Test requests completed"

# Wait a moment for logs to be written
sleep 1

# Stop proxy
echo "5. Stopping proxy server..."
kill $PROXY_PID 2>/dev/null || true

# Analyze the traffic
echo "6. Analyzing traffic..."
if [ -f "logs/demo-traffic.log" ]; then
    ./bin/parser -input logs/demo-traffic.log -format summary
    echo ""
    echo "📁 Full analysis available in: logs/demo-traffic.log"
else
    echo "❌ No traffic log found"
fi

echo ""
echo "✅ Demo complete!"
echo "   To monitor real Codex CLI traffic:"
echo "   1. Start proxy: ./scripts/start-proxy.sh"
echo "   2. Set environment: source <(./scripts/setup-env.sh | tail -n 1)"
echo "   3. Use Codex CLI normally"
echo "   4. Analyze logs: ./scripts/analyze-traffic.sh"
EOF

chmod +x scripts/demo.sh

echo ""
echo "✅ Setup complete!"
echo ""
echo "📋 Next steps:"
echo "   1. Start the proxy server:"
echo "      ./scripts/start-proxy.sh"
echo ""
echo "   2. Set up environment variables:"
echo "      ./scripts/setup-env.sh"
echo ""
echo "   3. Use Codex CLI normally - traffic will be logged"
echo ""
echo "   4. Analyze traffic:"
echo "      ./scripts/analyze-traffic.sh"
echo ""
echo "   5. Run demo (optional):"
echo "      ./scripts/demo.sh"
echo ""
echo "📚 Documentation:"
echo "   - Proxy setup: docs/PROXY_SETUP.md"
echo "   - Configuration: config/proxy.yaml, config/parser.yaml"
echo ""
echo "🔧 Built binaries:"
echo "   - Proxy server: bin/proxy"
echo "   - Traffic parser: bin/parser"
echo ""
echo "Happy monitoring! 🎉"
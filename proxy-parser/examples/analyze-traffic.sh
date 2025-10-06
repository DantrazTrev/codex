#!/bin/bash

# Script to analyze captured traffic
# Usage: ./analyze-traffic.sh [traffic-file.json]

set -e

TRAFFIC_FILE=${1:-"traffic.log"}

if [ ! -f "$TRAFFIC_FILE" ]; then
    echo "❌ Traffic file not found: $TRAFFIC_FILE"
    echo "Usage: $0 [traffic-file.json]"
    exit 1
fi

echo "📊 Analyzing traffic from: $TRAFFIC_FILE"
echo "========================================="
echo ""

# Build analyzer if needed
if [ ! -f "./codex-proxy" ]; then
    echo "Building analyzer..."
    go build -o codex-proxy ./cmd/proxy
fi

# Basic analysis
echo "1️⃣ Full Traffic Analysis"
echo "-------------------------"
./codex-proxy analyze --input "$TRAFFIC_FILE" --verbose
echo ""

# GenAI-only analysis
echo "2️⃣ GenAI Traffic Only"
echo "---------------------"
./codex-proxy analyze --input "$TRAFFIC_FILE" --genai-only
echo ""

# Generate statistics
echo "3️⃣ Traffic Statistics (JSON)"
echo "----------------------------"
./codex-proxy stats --input "$TRAFFIC_FILE"
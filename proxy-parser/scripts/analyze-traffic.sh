#!/bin/bash

# Traffic Analysis Script
# This script analyzes parsed traffic data from the proxy

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
TRAFFIC_FILE=${TRAFFIC_FILE:-logs/traffic.json}
OUTPUT_FORMAT=${OUTPUT_FORMAT:-table}

# Check if jq is installed
if ! command -v jq &> /dev/null; then
    echo -e "${RED}Error: jq is not installed. Please install jq first.${NC}"
    echo "Ubuntu/Debian: sudo apt install jq"
    echo "macOS: brew install jq"
    exit 1
fi

# Check if traffic file exists
if [ ! -f "$TRAFFIC_FILE" ]; then
    echo -e "${RED}Error: Traffic file not found: $TRAFFIC_FILE${NC}"
    exit 1
fi

echo -e "${GREEN}Analyzing traffic data from: $TRAFFIC_FILE${NC}"
echo ""

# Function to show help
show_help() {
    echo "Usage: $0 [OPTIONS]"
    echo "Options:"
    echo "  -f, --file FILE      Traffic file to analyze (default: logs/traffic.json)"
    echo "  -o, --output FORMAT  Output format: table, json, csv (default: table)"
    echo "  --genai              Show only GenAI requests"
    echo "  --vibe-coding        Show only vibe coding requests"
    echo "  --summary            Show summary statistics"
    echo "  --top-models         Show top models used"
    echo "  --top-endpoints      Show top endpoints"
    echo "  -h, --help           Show this help message"
}

# Parse command line arguments
GENAI_ONLY=false
VIBE_CODING_ONLY=false
SHOW_SUMMARY=false
SHOW_TOP_MODELS=false
SHOW_TOP_ENDPOINTS=false

while [[ $# -gt 0 ]]; do
    case $1 in
        -f|--file)
            TRAFFIC_FILE="$2"
            shift 2
            ;;
        -o|--output)
            OUTPUT_FORMAT="$2"
            shift 2
            ;;
        --genai)
            GENAI_ONLY=true
            shift
            ;;
        --vibe-coding)
            VIBE_CODING_ONLY=true
            shift
            ;;
        --summary)
            SHOW_SUMMARY=true
            shift
            ;;
        --top-models)
            SHOW_TOP_MODELS=true
            shift
            ;;
        --top-endpoints)
            SHOW_TOP_ENDPOINTS=true
            shift
            ;;
        -h|--help)
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

# Function to filter data
filter_data() {
    local filter=""
    if [ "$GENAI_ONLY" = true ]; then
        filter="select(.is_genai == true)"
    elif [ "$VIBE_CODING_ONLY" = true ]; then
        filter="select(.is_vibe_coding == true)"
    else
        filter="."
    fi
    echo "$filter"
}

# Function to show summary
show_summary() {
    echo -e "${BLUE}=== TRAFFIC SUMMARY ===${NC}"
    echo ""
    
    local total_requests=$(jq -r 'length' "$TRAFFIC_FILE")
    echo -e "${YELLOW}Total Requests:${NC} $total_requests"
    
    local genai_requests=$(jq -r '[.[] | select(.is_genai == true)] | length' "$TRAFFIC_FILE")
    echo -e "${YELLOW}GenAI Requests:${NC} $genai_requests"
    
    local vibe_coding_requests=$(jq -r '[.[] | select(.is_vibe_coding == true)] | length' "$TRAFFIC_FILE")
    echo -e "${YELLOW}Vibe Coding Requests:${NC} $vibe_coding_requests"
    
    local total_tokens=$(jq -r '[.[] | .tokens // 0] | add' "$TRAFFIC_FILE")
    echo -e "${YELLOW}Total Tokens:${NC} $total_tokens"
    
    echo ""
}

# Function to show top models
show_top_models() {
    echo -e "${BLUE}=== TOP MODELS ===${NC}"
    echo ""
    
    jq -r '.[] | select(.model != null) | .model' "$TRAFFIC_FILE" | \
    sort | uniq -c | sort -nr | head -10 | \
    while read count model; do
        echo -e "${YELLOW}$count${NC} - $model"
    done
    echo ""
}

# Function to show top endpoints
show_top_endpoints() {
    echo -e "${BLUE}=== TOP ENDPOINTS ===${NC}"
    echo ""
    
    jq -r '.[] | .endpoint' "$TRAFFIC_FILE" | \
    sort | uniq -c | sort -nr | head -10 | \
    while read count endpoint; do
        echo -e "${YELLOW}$count${NC} - $endpoint"
    done
    echo ""
}

# Function to show detailed data
show_detailed_data() {
    echo -e "${BLUE}=== DETAILED TRAFFIC DATA ===${NC}"
    echo ""
    
    local filter=$(filter_data)
    
    case $OUTPUT_FORMAT in
        table)
            echo -e "${YELLOW}Timestamp${NC} | ${YELLOW}Type${NC} | ${YELLOW}Endpoint${NC} | ${YELLOW}Model${NC} | ${YELLOW}Tokens${NC} | ${YELLOW}GenAI${NC} | ${YELLOW}Vibe Coding${NC}"
            echo "----------|------|----------|-------|-------|------|-------------"
            jq -r --arg filter "$filter" '.[] | select('"$filter"') | 
                "\(.timestamp) | \(.type) | \(.endpoint) | \(.model // "N/A") | \(.tokens // 0) | \(.is_genai) | \(.is_vibe_coding)"' "$TRAFFIC_FILE"
            ;;
        json)
            jq -r --arg filter "$filter" '.[] | select('"$filter"')' "$TRAFFIC_FILE"
            ;;
        csv)
            echo "timestamp,type,endpoint,method,model,tokens,is_genai,is_vibe_coding,request_id,user_agent,content_length"
            jq -r --arg filter "$filter" '.[] | select('"$filter"') | 
                "\(.timestamp),\(.type),\(.endpoint),\(.method),\(.model // ""),\(.tokens // 0),\(.is_genai),\(.is_vibe_coding),\(.request_id // ""),\(.user_agent // ""),\(.content_length)"' "$TRAFFIC_FILE"
            ;;
        *)
            echo -e "${RED}Unknown output format: $OUTPUT_FORMAT${NC}"
            exit 1
            ;;
    esac
}

# Main execution
if [ "$SHOW_SUMMARY" = true ]; then
    show_summary
fi

if [ "$SHOW_TOP_MODELS" = true ]; then
    show_top_models
fi

if [ "$SHOW_TOP_ENDPOINTS" = true ]; then
    show_top_endpoints
fi

# If no specific analysis is requested, show detailed data
if [ "$SHOW_SUMMARY" = false ] && [ "$SHOW_TOP_MODELS" = false ] && [ "$SHOW_TOP_ENDPOINTS" = false ]; then
    show_detailed_data
fi

echo -e "${GREEN}Analysis complete!${NC}"
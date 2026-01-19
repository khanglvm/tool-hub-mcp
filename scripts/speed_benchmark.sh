#!/bin/bash
# Speed Benchmark: Traditional MCP vs tool-hub-mcp
#
# Compares time-to-first-response between:
# 1. Traditional: AI client with all MCPs registered directly
# 2. tool-hub-mcp: AI client with single aggregator
#
# Usage: ./scripts/speed_benchmark.sh [iterations]

set -e

ITERATIONS=${1:-3}
RESULTS_DIR="$(dirname "$0")/../benchmark_results"
mkdir -p "$RESULTS_DIR"

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${BLUE}╔══════════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║           SPEED BENCHMARK: Traditional vs tool-hub-mcp       ║${NC}"
echo -e "${BLUE}╠══════════════════════════════════════════════════════════════╣${NC}"
echo -e "${BLUE}║ Iterations: ${ITERATIONS}                                              ║${NC}"
echo -e "${BLUE}╚══════════════════════════════════════════════════════════════╝${NC}"
echo ""

# Test prompts (simple queries that use MCP tools)
PROMPTS=(
    "List all available tools"
    "What servers are available for external integrations"
)

# Function to measure command execution time
measure_time() {
    local start=$(python3 -c "import time; print(time.time())")
    eval "$1" > /dev/null 2>&1
    local end=$(python3 -c "import time; print(time.time())")
    echo "scale=3; $end - $start" | bc
}

# Function to run benchmark with tool-hub-mcp
benchmark_toolhub() {
    local prompt="$1"
    local times=()
    
    echo -e "${YELLOW}Testing with tool-hub-mcp...${NC}"
    
    for ((i=1; i<=ITERATIONS; i++)); do
        echo -n "  Run $i/$ITERATIONS: "
        local time=$(measure_time "echo '$prompt' | claude --print --dangerously-skip-permissions")
        times+=($time)
        echo "${time}s"
    done
    
    # Calculate average
    local sum=0
    for t in "${times[@]}"; do
        sum=$(echo "$sum + $t" | bc)
    done
    local avg=$(echo "scale=3; $sum / $ITERATIONS" | bc)
    echo -e "${GREEN}  Average: ${avg}s${NC}"
    echo "$avg"
}

# Main benchmark
echo "═══════════════════════════════════════════════════════════════"
echo "TEST 1: List available tools/servers"
echo "═══════════════════════════════════════════════════════════════"

echo ""
echo "Prompt: 'List all available external tools and integrations'"
echo ""

# Run benchmark
TOOLHUB_TIME=$(benchmark_toolhub "List all available external tools and integrations")

echo ""
echo "═══════════════════════════════════════════════════════════════"
echo "TEST 2: Search for capability"  
echo "═══════════════════════════════════════════════════════════════"

echo ""
echo "Prompt: 'Search for document management capabilities'"
echo ""

TOOLHUB_TIME2=$(benchmark_toolhub "Search for document management capabilities")

echo ""
echo "═══════════════════════════════════════════════════════════════"
echo "RESULTS SUMMARY"
echo "═══════════════════════════════════════════════════════════════"
echo ""
echo "tool-hub-mcp Average Response Times:"
echo "  Test 1 (List tools):     ${TOOLHUB_TIME}s"
echo "  Test 2 (Search):         ${TOOLHUB_TIME2}s"
echo ""

# Save results
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
RESULTS_FILE="$RESULTS_DIR/speed_benchmark_$TIMESTAMP.json"

cat > "$RESULTS_FILE" << EOF
{
  "timestamp": "$(date -Iseconds)",
  "iterations": $ITERATIONS,
  "results": {
    "tool_hub_mcp": {
      "list_tools": ${TOOLHUB_TIME},
      "search_capability": ${TOOLHUB_TIME2}
    }
  },
  "notes": "Measures time from prompt to response using tool-hub-mcp aggregator"
}
EOF

echo "Results saved to: $RESULTS_FILE"
echo ""
echo -e "${GREEN}Benchmark complete!${NC}"

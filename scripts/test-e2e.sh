#!/bin/bash
#
# new-api-mcp-server E2E test runner
#
# Tests the MCP server by sending JSON-RPC requests over HTTP.
# Requires the Docker stack to be running (docker compose up -d).
#
# Usage:
#   make test-e2e
#   # or directly:
#   bash scripts/test-e2e.sh
#
# Environment variables:
#   MCP_URL     — MCP server URL (default: http://localhost:4051/mcp)
#   MCP_TOKEN   — Auth token for MCP server (default: empty)
#   HEALTH_URL  — Health check URL (default: http://localhost:4051/healthz)
#

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

echo "============================================================"
echo "  new-api-mcp-server E2E Test Runner"
echo "============================================================"
echo ""

# Check if Docker is available
if command -v docker &> /dev/null; then
    # Check if the MCP container is running
    if docker ps --format '{{.Names}}' 2>/dev/null | grep -q 'new-api-mcp'; then
        echo "✓ MCP server container is running"
    else
        echo "○ MCP server container not detected"
        echo "  Start it with: docker compose up -d"
        echo "  Tests may fail if server is not reachable."
    fi
else
    echo "○ Docker not found in PATH"
    echo "  Ensure the MCP server is running at: ${MCP_URL:-http://localhost:4051/mcp}"
fi
echo ""

# Run the Python test script
cd "$PROJECT_DIR"

if command -v python3 &> /dev/null; then
    python3 scripts/test-e2e.py
elif command -v python &> /dev/null; then
    python scripts/test-e2e.py
else
    echo "ERROR: Python 3 is required to run E2E tests."
    echo "Install Python 3 and try again."
    exit 1
fi
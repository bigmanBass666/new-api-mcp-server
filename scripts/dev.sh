#!/usr/bin/env bash
# 本地开发快速启动脚本
# 用法: ./scripts/dev.sh [http|stdio]
#
# http 模式 — 启动 HTTP transport，Inspector / conformance / 浏览器可连接
# stdio 模式 — 启动 stdio transport，通过 Claude Code 对话使用

set -euo pipefail

TRANSPORT="${1:-http}"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# ---- 构建 ----
echo "[build] compiling..."
cd "$PROJECT_ROOT"
go build -o bin/new-api-mcp-server ./cmd/server

# ---- 环境变量 ----
export NEW_API_BASE_URL="${NEW_API_BASE_URL:-http://localhost:4050}"
export NEW_API_SYSTEM_KEY="${NEW_API_SYSTEM_KEY:-}"
export NEW_API_KEY="${NEW_API_KEY:-}"
export MCP_API_TOOLS_ENABLED="${MCP_API_TOOLS_ENABLED:-true}"
export MCP_LOG_LEVEL="${MCP_LOG_LEVEL:-info}"

if [ "$TRANSPORT" = "http" ]; then
    export MCP_TRANSPORT="http"
    export MCP_HTTP_ADDR=":4051"
    export MCP_HTTP_AUTH_TOKEN="${MCP_HTTP_AUTH_TOKEN:-}"

    echo "[dev] starting HTTP transport on :4051"
    echo "       Inspector: npx @modelcontextprotocol/inspector --url http://localhost:4051/mcp"
    echo "       Conformance: npx @modelcontextprotocol/conformance server --url http://localhost:4051/mcp --suite active"
    echo ""
    ./bin/new-api-mcp-server
else
    echo "[dev] starting stdio transport"
    ./bin/new-api-mcp-server
fi

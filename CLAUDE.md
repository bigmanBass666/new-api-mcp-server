# CLAUDE.md

> **重要：代理开发指南已迁移至 `AGENTS.md`** — 包含 MCP 开发工作流、注册流程、测试方式等。本文件仅保留项目概览。
>
> 本地部署的New API部署目录在这儿。D:\Test\installations\new-api

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What This Project Is

A Go MCP (Model Context Protocol) server that wraps New API's OpenAPI endpoints (~160) as MCP tools. It supports stdio and HTTP (Streamable HTTP) transport modes with full observability (Prometheus, OpenTelemetry, slog).

## Build & Run

```bash
make build          # Build binary to bin/new-api-mcp-server
make test           # Run all tests with race detector
make lint           # Run golangci-lint
make run            # go run ./cmd/server

# Docker
make docker-build   # Build Docker image
make docker-up      # Start with Docker Compose

# E2E tests
make test-e2e       # Run E2E test suite (requires Docker stack)
```

Single test: `go test ./internal/config/ -v -run TestLoad_Defaults`

Integration test: `go test -tags=integration -v -run TestIntegration`

## Architecture

- `openapi/` — Embedded OpenAPI specs (api.json, relay.json) via `go:embed`
- `internal/openapi/` — Parses OpenAPI JSON into `[]ToolDef` using kin-openapi
- `internal/registry/` — Filters tools by config and registers them on `mcp.Server` via `server.AddTool()`
- `internal/handler/` — Creates `mcp.ToolHandler` functions that map MCP tool calls to upstream HTTP requests
- `internal/client/` — HTTP client for upstream New API calls with API key injection
- `internal/observability/` — Logging (slog), Metrics (Prometheus), Tracing (OTel)
- `internal/hightools/` — High-level MCP tools (channel/user/token management with built-in logic)
- `cmd/server/` — Entry point wiring everything together

## Key Design Decisions

- Tools are registered dynamically at startup using `server.AddTool(tool, handler)` (low-level API, not generic `AddTool[In,Out]`)
- Two API key types: `NEW_API_KEY` (relay/model tools) and `NEW_API_SYSTEM_KEY` (admin tools)
- API tools use `api_` name prefix; relay tools have no prefix
- API tools default OFF (whole group toggle); relay tools default ON (disable by tag)
- High-level tools (e.g. `add_channel`, `toggle_channel`) simplify common admin operations
- All config via environment variables
- Tool names are sanitized to `[a-zA-Z0-9_\-.]` per MCP SDK requirement
- Non-JSON upstream responses are base64 encoded
- Claude Code automatically enables Tool Search for efficient tool discovery (>200 tools)

## Deployment Options

| Mode | Target | How |
|------|--------|-----|
| stdio | Local/WSL | `make build && ./bin/new-api-mcp-server` |
| HTTP | Local | `MCP_TRANSPORT=http ./bin/new-api-mcp-server` |
| Docker | Container | `docker compose up -d` (includes New API) |

## Git Workflow

本项目为单人开发 + CI，适用"宽松 GitHub Flow"：

- **小改动、当天能完成** → 直接在 `main` 提交
- **改动涉及多个模块、可能破坏现有功能、开发周期超一天** → 创建 `feature/xxx` 分支
- **任何提交前确保 `make test` 通过**
- **原子提交**：一个 commit 只做一件事
- **提交信息格式**：`<type>: <简短描述>`（feat/fix/refactor/chore/docs）
- **禁止**：攒多个改动一次性提交、提交信息写"更新代码"之类废话
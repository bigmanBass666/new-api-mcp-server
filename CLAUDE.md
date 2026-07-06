# CLAUDE.md

> 本地部署的 New API 部署目录在这儿：`D:\Test\installations\new-api`

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

## MCP 开发工作流

本项目的 MCP Server 开发测试标准流程：

### 标准循环

```
编辑代码 → make build → 注册到 .mcp.json → 重启 Claude Code → 对话中测试
```

**不要**手动编译后启动 HTTP 模式再用 curl 测试——这绕过了 Claude Code 的 MCP 集成能力。

### 注册 MCP Server 到项目

使用 `setup-project-mcp` skill（全局路径：`C:\Users\86150\.claude\skills\setup-project-mcp\SKILL.md`）：

1. **编译二进制**：`make build` → 生成 `bin/new-api-mcp-server.exe`
2. **配置 `.mcp.json`**：在项目根目录创建/更新，写入 stdio 类型配置，包含必要的环境变量
3. **重启 Claude Code**：退出当前会话 → 重新进入项目目录启动
4. **批准连接**：输入 `/mcp` 批准新的 MCP 服务器
5. **检查状态**：输入 `/doctor` 确认连接正常
6. **对话测试**：直接向 Claude 提需求，Claude 会自动发现并调用 MCP 工具

### 环境变量配置

通过 `env` 字段注入，典型变量：

| 变量 | 说明 | 必填 |
|------|------|:----:|
| `NEW_API_BASE_URL` | New API 实例地址 | ✅ |
| `NEW_API_SYSTEM_KEY` | 管理员 access_token | ✅ |
| `NEW_API_KEY` | 模型调用的 API key | 按需 |
| `MCP_API_TOOLS_ENABLED` | 是否启用 API 管理工具 | 按需 |
| `MCP_RELAY_ENABLED_GROUPS` | 启用的 relay 分组（`all` 开启全部） | 按需 |
| `MCP_LOG_LEVEL` | 日志级别（debug/info/warn/error） | 按需 |

### 注意事项

- **变更后需重建**：每次修改 Go 代码后 `make build` 重新编译
- **重启才能生效**：MCP Server 进程在 Claude Code 启动时加载，修改后需重启
- **可同时开两个会话**：一个编辑代码，另一个测试（在同一项目目录用 `/mcp` 重新连接）
- **E2E 测试**：`make test-e2e-go` 不需启动 Claude Code，直接验证完整功能
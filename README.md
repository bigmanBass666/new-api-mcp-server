# New API MCP Server

[![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go)](https://go.dev)
[![License](https://img.shields.io/badge/license-MIT-green)](LICENSE)
[![CI](https://github.com/bigmanBass666/new-api-mcp-server/actions/workflows/test.yml/badge.svg)](https://github.com/bigmanBass666/new-api-mcp-server/actions/workflows/test.yml)
[![Docker](https://img.shields.io/badge/docker-ready-2496ED?logo=docker)](docker-compose.yml)

**Turn your [New API](https://github.com/QuantumNous/new-api) into a fully manageable AI tool ecosystem.**

This Go MCP server automatically generates **225+ tools** from New API's OpenAPI specs, giving AI assistants (Claude, etc.) direct access to manage channels, users, tokens, and make model inference calls — all through the standardized [Model Context Protocol](https://modelcontextprotocol.io/).

## ✨ Features

- **225+ MCP tools** — auto-generated from OpenAPI specs (58 relay + 157 admin + 10 high-level)
- **Dual transport** — stdio (CLI) and [Streamable HTTP](https://spec.modelcontextprotocol.io/2025-03-26/basic/transports/http/) (Docker/remote)
- **Dual API key** — separate keys for model relay (`NEW_API_KEY`) and admin management (`NEW_API_SYSTEM_KEY`)
- **10 high-level tools** — `add_channel`, `toggle_channel`, `set_channel_priority`, `toggle_user_status`, `switch_group` and more, with built-in logic
- **Tag-based group control** — enable/disable tool groups by category
- **Full observability** — Prometheus metrics, OpenTelemetry tracing, structured logging (slog)
- **Claude Code Tool Search** — optimized with camelCase names and server instructions; all 225+ tools auto-discovered on demand
- **Single binary** — OpenAPI specs embedded via `go:embed`, zero runtime files
- **Docker ready** — one-command deployment alongside New API

---

## 🚀 Quick Start

### Option 1: Docker (recommended)

The easiest way to get started. Deploys New API + MCP server together.

```bash
# 1. Clone and configure
git clone https://github.com/bigmanBass666/new-api-mcp-server.git
cd new-api-mcp-server
cp .env.example .env
# Edit .env: set SESSION_SECRET, CRYPTO_SECRET, NEW_API_SYSTEM_KEY

# 2. Start everything
docker compose up -d

# 3. Verify it's running
curl http://localhost:4051/healthz
# → {"status":"ok"}

# 4. Connect Claude Code (adds to your project's .mcp.json)
claude mcp add new-api --transport http http://localhost:4051/mcp \
  --header "Authorization: Bearer $(grep MCP_HTTP_AUTH_TOKEN .env | cut -d= -f2)"
```

> 💡 First time? After New API starts, register an admin account at `http://localhost:4050`, then copy the `access_token` from the admin panel into `.env` as `NEW_API_SYSTEM_KEY`. Restart the stack with `docker compose restart new-api-mcp`.

### Option 2: Binary (local/WSL)

```bash
# Build
make build

# Run in stdio mode (for Claude Code CLI)
export NEW_API_BASE_URL=http://localhost:3000
export NEW_API_KEY=sk-your-relay-key
export NEW_API_SYSTEM_KEY=sk-your-admin-key
export MCP_API_TOOLS_ENABLED=true

./bin/new-api-mcp-server
```

### Option 3: Connect Claude Code (stdio)

```bash
claude mcp add new-api -- \
  ./bin/new-api-mcp-server \
  -e NEW_API_BASE_URL=http://localhost:3000 \
  -e NEW_API_KEY=sk-your-relay-key \
  -e NEW_API_SYSTEM_KEY=sk-your-admin-key \
  -e MCP_API_TOOLS_ENABLED=true
```

### Option 4: Add to `.mcp.json`

```json
{
  "mcpServers": {
    "new-api": {
      "type": "http",
      "url": "http://localhost:4051/mcp",
      "headers": {
        "Authorization": "Bearer ${MCP_HTTP_AUTH_TOKEN}"
      }
    }
  }
}
```

---

## 🛠 Tool Categories

### High-Level Tools (10 tools, admin)

Built-in tools that wrap common workflows with smart defaults:

| Tool | Description | Required |
|------|-------------|----------|
| `list_providers` | List all channels with balance, status, priority | `NEW_API_SYSTEM_KEY` |
| `add_channel` | Create a new channel (name, type, key) | `NEW_API_SYSTEM_KEY` |
| `toggle_channel` | Enable/disable a channel by ID | `NEW_API_SYSTEM_KEY` |
| `set_channel_priority` | Set a channel's priority (load balancing weight) | `NEW_API_SYSTEM_KEY` |
| `test_channel` | Test a single channel (sync) | `NEW_API_SYSTEM_KEY` |
| `test_and_report_channels` | Test all channels with progress reporting | `NEW_API_SYSTEM_KEY` |
| `show_balance` | Show channel balances with quota info | `NEW_API_SYSTEM_KEY` |
| `list_users` | List all users with role and status | `NEW_API_SYSTEM_KEY` |
| `toggle_user_status` | Enable/disable a user | `NEW_API_SYSTEM_KEY` |
| `switch_group` | Move a token to a different group | `NEW_API_SYSTEM_KEY` |

### Relay Tools (58 tools, model inference)

Powered by `NEW_API_KEY`. Covers model API endpoints for all major providers:

| Category | Examples |
|----------|----------|
| OpenAI Chat/Responses | `POST /v1/chat/completions`, `POST /v1/responses` |
| Anthropic Messages | `POST /v1/messages` |
| Gemini | `POST /v1beta/models/{model}:generateContent`, `streamGenerateContent` |
| Midjourney | `/mj/submit/imagine`, `/mj/submit/describe`, `/mj/task/{id}/fetch` |
| Image Generation | `POST /v1/images/generations` |
| Video Generation | `POST /v1/video/generations`, `/v1/videos/{video_id}/remix` |
| Audio (TTS/STT) | `POST /v1/audio/speech`, `POST /v1/audio/transcriptions` |
| Embeddings / Rerank | `POST /v1/embeddings`, `POST /v1/rerank` |
| DALL·E / Kling / Jimeng | Provider-specific image/video endpoints |

### API Admin Tools (157 tools, backend management)

Powered by `NEW_API_SYSTEM_KEY`. All admin endpoints with `api_` prefix:

| Category | Examples |
|----------|----------|
| Channel Management | `api_get_channel`, `api_create_channel`, `api_delete_channel` |
| User Management | `api_get_user`, `api_create_user`, `api_manage_user` |
| Token Management | `api_get_token`, `api_create_token`, `api_update_token` |
| Logs & Stats | `api_search_logs`, `api_get_log_stats`, `api_get_log_graphs` |
| Model Management | `api_get_all_models`, `api_create_model` |
| System Settings | `api_get_system_options`, `api_update_system_options` |
| Redemption Codes | `api_create_redemption`, `api_search_redemptions` |

---

## ⚙️ Configuration

All configuration via environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `NEW_API_BASE_URL` | **(required)** | New API base URL |
| `NEW_API_KEY` | | API key for relay (model) tools |
| `NEW_API_SYSTEM_KEY` | | System API key for admin + high-level tools |
| `NEW_API_TIMEOUT` | `30s` | Upstream request timeout |
| `MCP_TRANSPORT` | `stdio` | Transport mode: `stdio` or `http` |
| `MCP_HTTP_ADDR` | `:8080` | HTTP listen address (http mode only) |
| `MCP_HTTP_AUTH_TOKEN` | | Bearer token for HTTP transport auth |
| `MCP_HTTP_CORS_ORIGINS` | `*` | Allowed CORS origins |
| `MCP_HTTP_MAX_BODY_SIZE` | `10485760` | Max request body size (bytes) |
| `MCP_API_TOOLS_ENABLED` | `false` | Enable admin tools (requires system key) |
| `MCP_RATE_LIMIT_RPS` | `0` | Rate limit: requests per second (0 = unlimited) |
| `MCP_RATE_LIMIT_BURST` | `0` | Rate limit: burst size |
| `MCP_RELAY_ENABLED_GROUPS` | | Comma-separated tag groups to enable |
| `MCP_LOG_LEVEL` | `info` | Log level: `debug`, `info`, `warn`, `error` |
| `MCP_LOG_FORMAT` | `json` | Log format: `json` or `text` |
| `MCP_LOG_CONSOLE_ENABLED` | `true` | Console log output |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | | OTLP endpoint for tracing |
| `OTEL_SERVICE_NAME` | `new-api-mcp-server` | Service name for tracing |
| `MCP_METRICS_ADDR` | `:9090` | Prometheus metrics listen address |
| `MCP_METRICS_PATH` | `/metrics` | Prometheus metrics path |

### Behavior Rules

- `NEW_API_KEY` not set → relay tools not registered
- `NEW_API_SYSTEM_KEY` not set or `MCP_API_TOOLS_ENABLED=false` → admin tools + high-level tools not registered
- `MCP_RATE_LIMIT_RPS=0` → rate limiting disabled
- `MCP_RELAY_ENABLED_GROUPS=未实现,视频生成` → enables only matching groups (whitelist)
- `MCP_TRANSPORT=http` → enables health check endpoints (`/healthz`, `/readyz`)

---

## 🐳 Docker Deployment

### Production stack (New API + MCP server)

```bash
# Configure
cp .env.example .env
# Edit .env with your secrets

# Start
docker compose up -d

# Verify
curl http://localhost:4051/healthz
curl http://localhost:4050/api/status

# View logs
docker compose logs -f new-api-mcp

# Stop
docker compose down
```

### Build image only

```bash
make docker-build
```

### Environment variables in Docker

The `docker-compose.yml` uses `${VAR}` syntax from `.env`. Required variables:

| Variable | Where to get it |
|----------|----------------|
| `SESSION_SECRET` | `openssl rand -base64 32` |
| `CRYPTO_SECRET` | `openssl rand -base64 32` |
| `NEW_API_SYSTEM_KEY` | New API admin panel → user's `access_token` |
| `MCP_HTTP_AUTH_TOKEN` | `openssl rand -base64 32` |

---

## 🧪 Testing

```bash
# Run all unit tests (with race detector)
make test

# Run E2E tests (requires Docker stack)
make test-e2e

# Single package
go test ./internal/openapi/ -v -count=1

# Integration with live New API
go test -tags=integration -v -run TestIntegration
```

The E2E test suite (`scripts/test-e2e.py`) runs 4 steps:
1. **Health check** — verifies the MCP server is responding
2. **Tool listing** — confirms 225+ tools are registered
3. **Naming quality** — validates camelCase names and descriptions
4. **Server info** — checks metadata and instructions

---

## 🔭 Observability

### Prometheus Metrics

Available at `http://localhost:9090/metrics` (configurable via `MCP_METRICS_ADDR`).

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `mcp_tool_requests_total` | Counter | `tool`, `status` | Tool call count |
| `mcp_tool_request_duration_seconds` | Histogram | `tool` | Tool call duration |
| `mcp_upstream_requests_total` | Counter | `method`, `path`, `status_code` | Upstream API call count |
| `mcp_upstream_request_duration_seconds` | Histogram | `method`, `path` | Upstream API duration |

Plus Go runtime metrics (goroutines, GC, memory).

### OpenTelemetry Tracing

Each tool call creates a span with:
- Tool name, HTTP method, path template
- Child span for upstream HTTP request with status code
- `trace_id` and `span_id` injected into structured logs

Set `OTEL_EXPORTER_OTLP_ENDPOINT` to enable tracing (noop when unset).

---

## 🏗 Architecture

```
┌───────────────────────────────────────────────────┐
│                   MCP Server                       │
│                                                    │
│  Client ─► ┌──────────┐  ┌──────────┐             │
│  (stdio)   │  stdio   │  │Streamable│  ← HTTP     │
│  or HTTP   │ transport │  │  HTTP    │             │
│            └────┬─────┘  └────┬─────┘             │
│                 └──────┬──────┘                    │
│                        ▼                            │
│            ┌───────────────────┐                   │
│            │   Tool Registry   │  ← Embedded       │
│            │   (225+ tools)    │     OpenAPI specs  │
│            │                   │                   │
│            │  ┌─────────────┐  │                   │
│            │  │ High-Level  │  │  ← System Key     │
│            │  │ (10 tools)  │  │                   │
│            │  ├─────────────┤  │                   │
│            │  │ API Tools   │  │  ← System Key     │
│            │  │ (157 tools) │  │                   │
│            │  ├─────────────┤  │                   │
│            │  │ Relay Tools │  │  ← API Key        │
│            │  │ (58 tools)  │  │                   │
│            │  └─────────────┘  │                   │
│            └────────┬──────────┘                   │
│                     ▼                              │
│            ┌───────────────────┐                   │
│            │   HTTP Client     │  → New API REST   │
│            └───────────────────┘                   │
│                                                    │
│  ┌─────────────────────────────────────────────┐   │
│  │ Prometheus │ OpenTelemetry │ slog (JSON)    │   │
│  └─────────────────────────────────────────────┘   │
└───────────────────────────────────────────────────┘
```

### Key Packages

| Package | Purpose |
|---------|---------|
| `openapi/` | Embedded OpenAPI specs (`go:embed`) |
| `internal/openapi/` | OpenAPI JSON → `[]ToolDef` parser |
| `internal/registry/` | Tool filtering + registration |
| `internal/handler/` | MCP handler → upstream HTTP mapper |
| `internal/client/` | HTTP client with dual auth (relay + system) |
| `internal/hightools/` | High-level tools (channel/user management) |
| `internal/config/` | Env-var configuration with validation |
| `internal/middleware/` | Rate limiting middleware |
| `internal/observability/` | Metrics, tracing, logging |
| `cmd/server/` | Entry point wiring |

---

## 👥 Contributing

Contributions are welcome! Here's how to get started:

1. **Fork** the repository
2. **Create a feature branch**: `git checkout -b feature/your-feature`
3. **Make your changes** with atomic commits
4. **Run tests**: `make test`
5. **Open a PR** against the `main` branch

### What would help

- Additional high-level tools (batch operations, log queries, stats)
- Test coverage improvements
- Documentation improvements
- Performance optimizations for large tool sets

Please ensure `make test` passes before submitting. If you're adding a new feature, include tests.

---

## 📦 Tech Stack

- [modelcontextprotocol/go-sdk](https://github.com/modelcontextprotocol/go-sdk) v1.4 — Official MCP Go SDK
- [getkin/kin-openapi](https://github.com/getkin/kin-openapi) — OpenAPI 3.0 parser
- [prometheus/client_golang](https://github.com/prometheus/client_golang) — Metrics
- [go.opentelemetry.io/otel](https://opentelemetry.io/docs/languages/go/) — Distributed tracing

---

## 📄 License

MIT — see [LICENSE](LICENSE) for details.

---

*Built with [New API](https://github.com/QuantumNous/new-api) — the open-source AI API management platform.*
# New API MCP Server — AGENTS.md

基于 New API OpenAPI 规范（~225 个工具）封装的 Go MCP server。支持 stdio 与 Streamable HTTP 传输，内置 Prometheus + OpenTelemetry 可观测性。

## 开发环境

- **操作系统**：Windows 11，使用 Git Bash
- **Go 版本**：1.25
- **所有命令在项目根目录执行**
- 二进制输出：`bin/new-api-mcp-server.exe`（Windows）/ `bin/new-api-mcp-server`（Unix）
- New API 实例通常部署在 `D:\Test\installations\new-api`

## 构建与测试

| 命令 | 用途 |
|---------|------|
| `make build` | 编译到 `bin/` |
| `make test` | 全量测试（含 race detector） |
| `make lint` | `golangci-lint` |
| `make run` | `go run ./cmd/server` |
| `make test-e2e-go` | 独立 E2E 测试（无需 Docker） |
| `make test-int` | 集成测试（需运行中的 New API） |

| 单测试执行 | `go test ./internal/config/ -v -run TestLoad_Defaults` |
| 单模块 lint | `golangci-lint run ./internal/hightools/` |

## 目录结构

```
cmd/server/          入口、服务编排、E2E 测试
internal/
  openapi/           解析 OpenAPI JSON → ToolDef（kin-openapi）
  registry/          按配置过滤工具，注册到 mcp.Server
  handler/           将 MCP 工具调用映射为上游 HTTP 请求
  client/            New API 的 HTTP 客户端（API key 注入）
  hightools/         高层工具（add_channel、toggle_user 等，含内置逻辑）
  config/            基于环境变量的配置
  observability/     slog 日志、Prometheus 指标、OTel 链路追踪
  extractor/         从 API 响应中提取 schema
  middleware/        限流
openapi/             通过 go:embed 嵌入的 OpenAPI 规范（api.json、relay.json）
```

## 代码约定

- **缩进**：Tab（非空格）——所有 Go 源文件使用 tab 缩进
- **工具命名**：仅 `[a-zA-Z0-9_\-.]`（MCP SDK 要求）；管理端工具加 `api_` 前缀，relay 端无前缀
- **API key**：`NEW_API_SYSTEM_KEY`（管理端）、`NEW_API_KEY`（relay 端）——切勿混用
- **OpenAPI 规范**：通过 `go:embed` 嵌入，运行时不会从磁盘读取
- **非 JSON 响应**：返回 MCP 客户端前会做 base64 编码

## 边界规则

- **始终可做**：编辑 Go 源文件、更新 `openapi/` 下的规范、编写测试
- **先问再做**：改动工具注册逻辑、修改配置/环境变量 schema、Docker/CI 变更
- **绝不触碰**：提交 `.env`、`bin/`、`.omc/`；直接 push 到 `main`；手动修改 OpenAPI 规范（它们由 New API 导出）

## 常见陷阱

- Go 源文件使用 **Tab** 而非空格——Edit 工具在 tab 被转为空格时会失败
- 每次代码变更后必须重新编译（`make build`），然后才能通过 MCP 测试
- `NEW_API_KEY` 和 `NEW_API_SYSTEM_KEY` 对应完全不同的工具组——管理端工具无法用 relay key 调用
- Docker 健康检查使用 **4051 端口**（CI 中硬编码），不是默认的 3000
- CI 中的 E2E 测试需要 `docker compose` 启动完整的 New API 栈——本地用 `make test-e2e-go` 更快

## MCP 开发工作流

```
编辑代码 → make build → 注册到 .mcp.json → 重启 Claude Code → 对话中测试
```

不要手动编译后启动 HTTP 模式再用 curl 测试——绕过了 Claude Code 的 MCP 集成能力。

### 注册 MCP Server

使用 `setup-project-mcp` skill：

1. **编译**：`make build` → `bin/new-api-mcp-server.exe`
2. **配置 `.mcp.json`**：项目根目录写入 stdio 配置，注入环境变量
3. **重启 Claude Code**，输入 `/mcp` 批准连接
4. **验证**：`/doctor` 确认连接正常
5. **对话测试**：直接向 Claude 提需求

### 注意事项

- 每次修改 Go 代码后必须 `make build` 重新编译
- MCP Server 进程在 Claude Code 启动时加载，修改后需重启才生效
- 可同时开两个会话：一个编辑代码，另一个测试
- 本地快速验证用 `make test-e2e-go`，不需要启动 Claude Code

## Agent Skills

- **Issue tracker** — Issues 存放在 GitHub Issues，使用 `gh` CLI 读写。详见 `docs/agents/issue-tracker.md`
- **Triage labels** — 五个 canonical triage roles 使用默认 label names。详见 `docs/agents/triage-labels.md`
- **Domain docs** — 上下文文档在 `CONTEXT.md` + `docs/adr/`。详见 `docs/agents/domain.md`

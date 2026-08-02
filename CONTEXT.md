# CONTEXT.md

> 本项目的领域术语表。Engineering skills 探索 codebase 时应使用这里的词汇。

## 项目定位

new-api-mcp-server 是一个 Go MCP（Model Context Protocol）服务器，将 [New API](https://github.com/Calcium-Ion/new-api) 的 OpenAPI 端点（~160+）包装为 MCP tools。支持 stdio 和 HTTP（Streamable HTTP）两种 transport。

## 核心术语

| 术语 | 含义 |
|------|------|
| **New API** | 上游的 LLM API 管理平台，提供渠道、令牌、用户管理等 REST API |
| **Relay 端点** | 位于 `relay/` 目录的工具，负责将上游 API 转发到实际 LLM 提供方 |
| **API 管理端点** | 位于 `api/` 目录的工具，管理 New API 自身的配置（渠道、用户、令牌等） |
| **高层工具** | `hightools/` 目录中的工具，封装多个 API 调用的复合操作（如 add_channel、toggle_channel） |
| **OpenAPI Spec** | `openapi/` 目录下的 `api.json` 和 `relay.json`，从 New API 提取并通过 `go:embed` 嵌入 |
| **ToolDef** | `internal/openapi/` 解析出的工具定义结构，包含 name、description、inputSchema、handler |
| **Channel** | 上游 LLM 提供方的渠道配置（API key、base URL、模型、速率限制等） |
| **Token** | 用户/渠道的访问令牌，用于认证 |
| **Task** | Tasks 扩展（phase 2 集成），支持异步任务的状态查询、更新和取消 |
| **系统 Key** | `NEW_API_SYSTEM_KEY`，管理员级别的 access_token，用于 API 管理操作 |
| **用户 Key** | `NEW_API_KEY`，模型调用的 API key，用于 relay 操作 |

## 架构约定

- `go:embed` 将 OpenAPI spec 嵌入二进制，运行时不需要外部文件
- 工具注册使用低阶 `server.AddTool(tool, handler)` 接口（非泛型 `AddTool[In,Out]`）
- API 工具默认关闭（整体 toggle），relay 工具默认开启（按 tag 禁用）
- 工具名称按 MCP SDK 要求 sanitize 为 `[a-zA-Z0-9_\-.]`

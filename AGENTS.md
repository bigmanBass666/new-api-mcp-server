# AGENTS.md — 代理开发指南

> 此文件取代 CLAUDE.md 作为主要代理指令文件。CLAUDE.md 保留项目概览，AGENTS.md 聚焦于代理如何工作。

## MCP Server 开发工作流

本项目是一个 MCP Server，开发过程中需要不断测试验证。正确的工作流如下：

### 标准开发循环

```
编辑代码 → make build → 注册到 .mcp.json → 重启 Claude Code → 对话中测试
```

**不要**手动编译后启动 HTTP 模式再用 curl 测试——这绕过了 Claude Code 的 MCP 集成能力。

### 注册 MCP Server 到项目

使用 `setup-project-mcp` skill（全局技能路径：`C:\Users\86150\.claude\skills\setup-project-mcp\SKILL.md`）：

1. **编译二进制**：`make build` → 生成 `bin/new-api-mcp-server.exe`
2. **配置 `.mcp.json`**：在项目根目录创建/更新，写入 stdio 类型配置，包含必要的环境变量（`NEW_API_BASE_URL`、`NEW_API_SYSTEM_KEY`、`NEW_API_KEY` 等）
3. **重启 Claude Code**：退出当前会话 → 重新进入项目目录启动
4. **批准连接**：输入 `/mcp` 批准新的 MCP 服务器
5. **检查状态**：输入 `/doctor` 确认连接正常
6. **对话测试**：直接向 Claude 提需求，Claude 会自动发现并调用 MCP 工具

### 环境变量配置

MCP Server 通过 `env` 字段注入环境变量，典型配置：

| 变量 | 说明 | 必填 |
|------|------|:----:|
| `NEW_API_BASE_URL` | New API 实例地址 | ✅ |
| `NEW_API_SYSTEM_KEY` | 管理员 access_token | ✅ |
| `NEW_API_KEY` | 模型调用的 API key | 按需 |
| `MCP_API_TOOLS_ENABLED` | 是否启用 API 管理工具 | 按需 |
| `MCP_RELAY_ENABLED_GROUPS` | 启用的 relay 分组（设为 `all` 开启全部） | 按需 |
| `MCP_LOG_LEVEL` | 日志级别（debug/info/warn/error） | 按需 |

### 迭代开发时的注意事项

- **变更后需重建**：每次修改 Go 代码后，`make build` 重新编译
- **重启才能生效**：MCP Server 进程在 Claude Code 启动时加载，修改后需重启 Claude Code
- **可同时开两个会话**：一个会话编辑代码，另一个会话（在同一项目目录）用 `/mcp` 重新连接来测试，避免来回切换
- **E2E 测试**：`make test-e2e-go` 不需启动 Claude Code，直接验证完整功能

### 常见开发错误（避免）

| 错误做法 | 正确做法 |
|---------|---------|
| 手动 `go run` + curl 测试 | 注册到 `.mcp.json`，对话中测试 |
| 攒多个改动一次性重新编译 | 每次小改动后立即编译验证 |
| 在 HTTP 模式下手工管理进程生命周期 | 让 Claude Code 自动管理 stdio 子进程 |
| 配置到全局 `.claude.json` | 配置到项目级 `.mcp.json`（隔离干净） |
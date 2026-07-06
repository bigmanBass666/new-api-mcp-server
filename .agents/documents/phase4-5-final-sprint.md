# Phase 4/5 最终冲刺：Tool Search 优化 + Docker 化 + 测试流水线

## Summary

完成项目的最后三个收尾工作：优化 Tool Search 工具名称/描述（Phase 5）、将 MCP 服务器 Docker 化以消除 WSL 跨平台问题、建立非交互测试流水线。

## Current State Analysis

### 工具命名现状
- **Relay 工具（58 个）**：已有良好的 `operationId`，如 `createJimengVideo`、`mjBlend`、`listFiles`——Tool Search 可正常匹配
- **API 工具（157 个）**：**所有 operationId 缺失**（api.json 中全部为 `no-id`），回退到 `generateName()` 生成类似 `get_api_channel` 的 snake_case 名称
- **High-level 工具（10 个）**：名称良好，如 `add_channel`、`toggle_channel` 等
- **描述**：api.json 的 `summary` 字段多为 emoji+中文权限标记（如 `🔓 无需鉴权`），对英文搜索无帮助

### WSL 跨平台现状
- MCP 服务器通过 WSL wrapper 脚本启动：`wsl.exe bash -lc /root/bin/new-api-mcp-wrapper.sh`
- 依赖 WSL 端口转发，不稳定
- `.mcp.json` 的 `env: {}` 为空，环境变量在 wrapper 脚本中硬编码

### 测试现状
- 已手动完成 10 个工具的 E2E 验证，但无自动化测试脚本
- 无 CI/CD 集成

## Proposed Changes

### Direction A: Tool Search 优化（Phase 5）

**涉及文件：**
- `internal/openapi/parser.go`：改进 `generateName()` 和 fallback description
- `cmd/server/main.go`：添加 MCP server instructions（工具类别提示）

**改动详情：**

1. **改进 `generateName()`**（parser.go:232-239）
   - 当前：`get_api_channel`（snake_case）
   - 改为：`getChannel` / `listChannel`（camelCase，动词+路径关键词）
   - 映射表：`get`→`get`, `post`→`create`, `put`→`update`, `delete`→`delete`
   - 对路径中数字 ID 参数用 `ById` 结尾

2. **改进 fallback description**（parser.go:64-71）
   - 当前：如果 `summary` 没有有意义的英文描述，description 为空
   - 改为：生成描述 `"{method} {path} - {tag} endpoint"`，如 `"GET /api/channel/ - Channel management endpoint"`
   - 如果 `summary` 为空或只包含 emoji/中文，使用生成的描述

3. **添加 MCP server instructions**（main.go）
   - 在 `mcp.NewServer()` 时添加 `instructions` 字段
   - 内容：`"Available tool categories: Channel management (api_channel_*), User management (api_user_*), Token management (api_token_*), Log/System (api_log_*, api_about_*), and relay tools for AI model inference."`
   - 这帮助 Tool Search 的初始搜索引导

**验证：**
- `go build ./cmd/server` 通过
- `go test ./internal/openapi/ -v -count=1` 通过（解析器测试）
- `go test ./... -race -count=1` 全部通过

### Direction B: Docker 化 MCP 服务器

**涉及文件：**
- `Dockerfile`（新建）
- `docker-compose.yml`（新建，在 new-api-mcp-server 项目根目录）
- `.mcp.json`（更新为 HTTP transport）
- `scripts/` 目录中保留 wrapper 脚本的备用入口

**改动详情：**

1. **创建 Dockerfile**
   - 多阶段构建：`golang:1.25-alpine` 构建 → `alpine:latest` 运行
   - 暴露端口 8080（HTTP transport）
   - 默认启动 HTTP 模式
   - 健康检查：`/healthz` 端点

2. **创建 docker-compose.yml**（项目根目录）
   ```yaml
   services:
     new-api-mcp:
       build: .
       container_name: new-api-mcp
       ports:
         - "4051:8080"
       environment:
         - NEW_API_BASE_URL=http://new-api:3000
         - NEW_API_KEY=${NEW_API_KEY}
         - NEW_API_SYSTEM_KEY=${NEW_API_SYSTEM_KEY}
         - MCP_API_TOOLS_ENABLED=true
         - MCP_TRANSPORT=http
         - MCP_HTTP_ADDR=:8080
       depends_on:
         new-api:
           condition: service_healthy
       restart: unless-stopped

   networks:
     default:
       name: new-api-network
   ```

3. **更新 `.mcp.json`**
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
   - 使用 `${VAR}` 语法从环境变量展开 token

**验证：**
- `docker build -t new-api-mcp-server .` 编译通过
- `docker compose up -d` 容器启动成功
- `curl http://localhost:4051/healthz` 返回 `{"status":"ok"}`
- `claude mcp list` 显示 Connected

### Direction C: 非交互测试流水线

**涉及文件：**
- `scripts/test-e2e.sh`（新建，Bash 测试脚本）
- `Makefile`（添加 `test-e2e` 和 `test-ci` 目标）
- `.github/workflows/test.yml`（新建，GitHub Actions）

**改动详情：**

1. **创建 `scripts/test-e2e.sh`**
   - 多步骤 E2E 测试脚本
   - Step 1: 检查 New API 服务状态
   - Step 2: 构建 MCP 服务器
   - Step 3: 启动 MCP 服务器（stdio 模式，通过 Go 子进程）
   - Step 4: 用 Python 脚本发送 JSON-RPC `tools/list` 和 `tools/call` 请求
   - Step 5: 验证关键工具正常响应
   - Step 6: 清理

2. **创建 Go E2E 测试文件**（可选）
   - `internal/handler/integration_test.go`（已存在 `TestIntegration`）
   - 扩展为更全面的 E2E 测试

3. **更新 Makefile**
   - `test-e2e`: 运行 scripts/test-e2e.sh
   - `test-ci`: 全量测试 + lint + build

4. **创建 GitHub Actions 工作流**
   - `.github/workflows/test.yml`（CI 配置）
   - 触发条件：push、pull_request
   - 步骤：checkout → setup Go → test → build → lint

**验证：**
- `make test-e2e` 全部通过
- 测试覆盖至少 5 个关键工具

## Assumptions & Decisions

1. **Direction A 的 scope 限定**：不改动 api.json/relay.json 文件内容，只通过改解析器逻辑来提升工具名称和描述质量。更彻底的方式（给 157 个 endpoint 加 operationId）工作量太大，不适合当前场景。
2. **Docker compose 放在项目根目录**：这样用户可以在项目目录一键 `docker compose up -d`，与 installations/new-api 下的部署配置分离。
3. **测试使用 Python JSON-RPC**：而非 `claude -p`，因为 `claude -p` 依赖用户登录状态和环境变量传输，在 CI 中不可靠。
4. **保留 WSL wrapper 作为备选**：Docker 化后 WSL 方案保留，用户可选择任一种方式启动。

## Dependencies

```
Task A (Tool Search) ── 独立，可先做
     │
Task B (Docker) ── 独立，可与 A 并行
     │
     └── Task C (Testing) ── 依赖 B（Docker 部署后可测试 HTTP 模式）
                          ── 但不阻塞，也可先对 stdio 模式做测试
```

## Verification

- [x] go test ./... -race -count=1 全部通过
- [x] go build ./cmd/server 通过
- [x] Docker build 成功
- [x] docker compose up 后 MCP 可访问
- [x] make test-e2e 通过
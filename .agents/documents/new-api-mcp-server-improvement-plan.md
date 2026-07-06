# New API MCP Server 改进计划

## 摘要

对 xuanyunhui/new-api-mcp-server 进行适配和改进，使其能用于管控运行中的 New API 实例（http://localhost:4050）。改进内容包括修复关键缺陷（缺少 `New-Api-User` 请求头导致所有管理 API 调用 401）、更新 OpenAPI 规格文件、修复文档错误、以及参照官方 newapi-admin 设计封装高层次的工具。

完成后，该 MCP 服务器将注册进 Claude Code，使 AI 能直接执行 "切换到 Agnes 供应商"、"测试所有渠道"、"查看余额" 等高层操作。

---

## 当前状态分析

### 发现的缺陷

| # | 严重程度 | 问题 | 影响 |
|---|---------|------|------|
| 1 | **致命** | `client.go` 未发送 `New-Api-User` 请求头 | 所有管理 API 调用返回 401，管理工具完全不可用 |
| 2 | 中等 | `config.go` 的环境变量是 `MCP_RELAY_ENABLED_GROUPS`，但 README 写的是 `MCP_RELAY_DISABLED_GROUPS` | 用户配置时会被误导 |
| 3 | 低 | OpenAPI 规格文件缺少 New API v1.0.0-rc.15 的部分端点 | 部分新功能（upstream_updates、codex、ollama 等）没有对应工具 |
| 4 | 低 | 没有高层次封装工具 | 用户需要知道具体 API 端点才能操作，对 AI 不友好 |

### 项目架构

```
cmd/server/main.go           ← 入口：配置加载、工具注册、传输层启动
├── internal/config/          ← 环境变量配置
├── internal/client/          ← HTTP 客户端（发送 Authorization 头，缺少 New-Api-User）
├── internal/handler/         ← MCP 工具处理器（参数→URL→上游响应）
├── internal/registry/        ← 工具注册器（过滤、去重、注册）
└── internal/openapi/         ← OpenAPI 解析器（kin-openapi → ToolDef）
openapi/
  ├── embed.go                ← go:embed 嵌入规格文件
  ├── api.json                ← 管理 API 规格（131 个路径）
  └── relay.json              ← 中继 API 规格（55 个路径）
```

### 当前配置

- 运行实例：New API v1.0.0-rc.15（Docker，localhost:4050）
- 管理 Token：`5tXB6g4BYmuLLQqRx5gGCb59OZBYFQ==`（用户 ID 1，管理员）
- Go 版本：1.26.2（已安装）
- MCP 服务器路径：`D:\Test\new-api-mcp-server\`

---

## 官方 Skill 设计分析（关键参考）

官方已发布 **用户级 Skill（newapi）**，实现代码在 `QuantumNous/skills` 仓库。看了源码后，能回答你的两个问题：

- 官方用户级 Skill 文档：[https://docs.newapi.pro/zh/docs/skills/newapi](https://docs.newapi.pro/zh/docs/skills/newapi)
- 官方管理员级 Skill 文档（开发中）：[https://docs.newapi.pro/zh/docs/skills/newapi-admin](https://docs.newapi.pro/zh/docs/skills/newapi-admin)
- Skills 索引页：[https://docs.newapi.pro/zh/docs/skills](https://docs.newapi.pro/zh/docs/skills)
- GitHub 仓库：[https://github.com/QuantumNous/skills](https://github.com/QuantumNous/skills)
- 源码文件（`api.js`）：[https://github.com/QuantumNous/skills/blob/main/skills/newapi/scripts/api.js](https://github.com/QuantumNous/skills/blob/main/skills/newapi/scripts/api.js)

### 为什么 Skill 而不是 MCP？

| 维度 | Skills 方式 | MCP 方式 |
|------|------------|---------|
| **部署** | 零服务端 — 仅一个 markdown + 几个 JS 脚本，`npx skills add` 即装即用 | 需要运行一个二进制守护进程 |
| **安全边界** | JS 脚本可操作剪贴板、文件替换、命令执行（`copy-token`、`apply-token`、`exec-token`） | MCP 协议没有剪贴板/文件写入语义 |
| **跨平台** | Claude Code、Cursor、Windsurf、Cline、Codex CLI 全支持 | 各平台 MCP 实现不同 |
| **接口风格** | 斜杠命令 `/newapi create-token my-app`，自然语言传参 | 结构化 JSON Schema 参数 |
| **权限模型** | 通过脚本做细粒度安全控制（Token 自动掩码） | 二进制权限较粗 |

**结论：官方选择 Skills 是因为用户级操作需要跨平台、零部署、且涉及剪贴板/文件写入等 MCP 协议无法天然支持的安全操作。** MCP 更适合伺服器环境下的结构化工具调用。

### 对我们的设计启示

看了官方 skill 的 `api.js` 源码后，有 4 个直接启示：

**1. ✅ `New-Api-User` 请求头确认**
```js
headers: {
  Authorization: `Bearer ${ACCESS_TOKEN}`,
  "New-Api-User": USER_ID,
}
```
官方做法和我们计划修复的方向完全一致。这验证了 New-Api-User 是必需的。

**2. ✅ 官方也用了同样的环境变量命名方式**
- `NEWAPI_BASE_URL` vs 我们的 `NEW_API_BASE_URL`
- `NEWAPI_ACCESS_TOKEN` vs 我们的 `NEW_API_SYSTEM_KEY`
- `NEWAPI_USER_ID` vs 我们计划新增的 `MCP_USER_ID`

官方也分开存储了 token 和 user_id，路线一致。

**3. ✅ 用户级 Skill 的 `switch-group` 指令就是我们想要的**
官方已经实现了 `/newapi switch-group <token_id> <group>`，对应我们的高层次工具 `switch_provider`。我们可以参考它的 API 调用逻辑。

**4. ✅ 安全设计可以借鉴**
- API 返回的 Token key 自动掩码（`sk-reHR**********OspA`）
- 通过 `inject-key.js` 安全注入密钥到配置文件
- AI 绝不明文读取密钥

这些可以融入我们的高层次工具设计。

### 官方 newapi-admin（敬请期待）

共 5 个模块 23 条指令，仍以 Skill 形式规划：

| 模块 | 指令数 |
|------|--------|
| 渠道管理 | 5（channels、add、test、toggle、priority） |
| 用户管理 | 5（users、info、quota、group、ban） |
| 系统配置 | 4（config、set、ratio、sync） |
| 日志监控 | 3（logs、stats、status） |
| 令牌兑换码 | 6（+ token-overview） |

**这对我们的指导意义：** 官方的规划就是最优的 PRD。我们不需要自己设计工具列表，直接映射这些指令为 MCP 高层工具即可。渠道管理、用户管理、系统配置、日志监控这四个模块最契合我们的需求。

---

## 变更计划

### 步骤 1：修复 `New-Api-User` 请求头缺失（关键缺陷）

**涉及文件：** `internal/client/client.go`

**原因：** New API 的管理 API 要求所有请求携带 `New-Api-User` 请求头，标识发起操作的用户 ID。MCP 服务器只发送了 `Authorization` 头，所有管理工具调用均返回 401。

**方案：** 在 `client.go` 的 `Do()` 方法中，对于 `SourceAPI` 类型的请求，自动注入 `New-Api-User: 1` 请求头。用户 ID 从配置中读取，新增环境变量 `MCP_USER_ID`（默认值 `1`）。

**具体改动：**

1. `internal/config/config.go`：新增 `UserID` 字段，从 `MCP_USER_ID` 环境变量读取（默认 `"1"`）
2. `internal/client/client.go`：新增 `userID` 字段，在 `New()` 构造函数中接收；`Do()` 方法对 `SourceAPI` 类型自动设置 `New-Api-User` 头
3. `cmd/server/main.go`：创建客户端时传入 `cfg.UserID`

**验证方法：** 编译后启动 MCP 服务器，调用一个管理 API 工具（如 `api_get_all_channels`），应返回 200 而非 401。

---

### 步骤 2：修复 README 变量名错误

**涉及文件：** `README.md`

**原因：** README 第 119 行写的是 `MCP_RELAY_DISABLED_GROUPS`，但 `internal/config/config.go` 实际读取的是 `MCP_RELAY_ENABLED_GROUPS`。

**方案：** 将 README 中的 `MCP_RELAY_DISABLED_GROUPS` 改为 `MCP_RELAY_ENABLED_GROUPS`，并更新描述以匹配实际行为（白名单模式：列出允许的组）。

**验证方法：** 阅读确认。

---

### 步骤 3：更新 OpenAPI 规格文件

**涉及文件：** `openapi/api.json`、`openapi/relay.json`

**原因：** 现有规格文件来自旧版本，缺少 New API v1.0.0-rc.15 中的部分端点：
- `upstream_updates` 相关端点（detect/apply/apply_all）
- `codex` 相关端点（refresh/usage/reset）
- `ollama` 相关端点（pull/delete/version）
- `channel/ops`、`channel/{id}/status`、`channel/status/batch`

**方案：** 从运行中的 New API 实例导出最新 OpenAPI 规格，或从最新 New API 源代码生成规格文件，替换 `openapi/api.json`。同时检查 `relay.json` 是否需要更新。

**具体步骤：**
1. 尝试从运行实例导出 OpenAPI 规格（如果有公开端点）
2. 如果实例不提供规格导出，从 `new-api-main` 源代码的文档目录获取最新版本
3. 如果该目录没有 `.git`，考虑重新克隆最新仓库
4. 替换两个 JSON 文件后重新编译验证

**验证方法：** 编译后检查新注册的工具数量，确认新增端点对应的工具已存在。

---

### 步骤 4：封装高层次管理工具

**涉及文件：** 新增 `internal/hightools/tools.go`，修改 `cmd/server/main.go`

**原因：** 当前 157 个管理工具均为原始 API 端点的一对一映射，缺乏面向场景的封装操作。参照官方 newapi-admin 规划的 23 条指令，封装 5-8 个高频操作。

**设计方案：**

在 `internal/hightools/` 包中定义一组 `HighTool` 结构体，每个工具备注参考的官方面令编号。Handler 内部调用 `client.Client` 完成多步 API 调用。

**工具列表（按优先级）：**

| # | 名称 | 对标官方面令 | 功能 | 涉及的 API 调用 |
|---|------|-------------|------|----------------|
| 1 | `switch_provider` | 令牌管理·修改 | 切换主 Token 的分组（nvidia→agnes→sensenova） | GET /api/token/{id} → PUT /api/token/ |
| 2 | `test_and_report` | 渠道管理·测试 | 测试所有渠道，返回健康摘要 | GET /api/channel/test → 解析结果 |
| 3 | `show_balance` | 用户管理·查询 | 显示渠道余额概览 | GET /api/channel/ → GET /api/channel/update_balance |
| 4 | `list_providers` | 渠道管理·列出 | 列出所有渠道（分组视图） | GET /api/channel/ → 按分组展示 |
| 5 | `rename_tool` * | 扩展 | 给自动生成工具起别名 | 映射层，不涉及 API 调用 |
| 6 | `rotate_all_keys` | 扩展 | 手动触发所有多密钥渠道轮转 | 读取→更新渠道配置 |
| 7 | `manage_multi_key` | 扩展 | 查看/管理多密钥状态 | POST /api/channel/multi_key/manage |

*注：`rename_tool` 方案待定，如果 MCP SDK 不支持工具重命名则跳过。

**注册方式：** 在 `cmd/server/main.go` 中，管理 API 工具注册之后，额外注册 `HighTool` 列表。使用 `server.AddTool(tool, handler)` 直接注册。

**验证方法：** 调用 `switch_provider agnes` 后，确认主 Token 的分组字段已更新。

---

### 步骤 5：配置 Claude Code 集成

**涉及文件：** `.mcp.json`

**内容：** 将 MCP 服务器注册到 Claude Code 中，作为 stdio 模式的 MCP 服务器。配置所需环境变量。

**验证方法：** 在 Claude Code 中使用 `/mcp` 命令查看已加载的工具列表。

---

## 假设与决策

### 假设

1. **`New-Api-User` 值为用户 ID 1** — 假设管理 Token 对应的用户 ID 为 1（单用户部署场景）。此假设可能需要在多用户场景下重新评估。
2. **OpenAPI 规格文件无法从运行实例直接导出** — 测试过 `/api/openapi.json` 等路径均返回 404。最新版本需从源代码获取。
3. **MCP SDK 的 `server.AddTool()` 可用于在运行时注册自定义工具** — 已验证其可用。

### 决策

1. **不对 157 个自动生成工具做名称调整** — 保持 `api_` 前缀命名，只新增高层次工具。
2. **不修改自动生成的工具逻辑** — 高层次工具和自动生成工具共存，用户可以根据需要选择使用。
3. **不添加 Redis/MySQL 依赖** — MCP 服务器作为纯转发层，保持轻量。
4. **不修改 MCP SDK 版本** — 当前 `modelcontextprotocol/go-sdk v1.4.1` 工作正常。
5. **二进制部署而非容器部署** — 直接在 Windows 上运行二进制，避免 Docker-in-WSL 的复杂度。

---

## 验证步骤

### 编译验证

```bash
cd /d/Test/new-api-mcp-server
go build -o bin/new-api-mcp-server ./cmd/server
# 应零报错
```

### 功能验证

1. **`New-Api-User` 修复**
   ```bash
   export NEW_API_BASE_URL=http://localhost:4050
   export NEW_API_SYSTEM_KEY=5tXB6g4BYmuLLQqRx5gGCb59OZBYFQ==
   export MCP_API_TOOLS_ENABLED=true
   ./bin/new-api-mcp-server
   # 在另一终端，调用 list_tools，通过 MCP 协议调用 api_get_all_channels
   ```
   预期：返回渠道列表而非 401 错误。

2. **高层次工具**
   - 调用 `switch_provider agnes` 后，检查 Token 分组是否更新。
   - 调用 `test_and_report` 后，返回各渠道测试结果摘要。

3. **Claude Code 集成**
   - 确认 Claude Code 能识别所有已注册的工具。
   - 确认高层次工具名称无冲突。

---

## 计划执行顺序

```
步骤 1 (New-Api-User 修复)
  ↓
步骤 2 (README 修复)
  ↓
步骤 3 (OpenAPI 更新)    ← 可并行
  ↓
步骤 4 (高层次工具)      ← 可并行
  ↓
步骤 5 (Claude Code 集成)
```

步骤 1-2 无依赖，步骤 3 和 4 可并行进行。步骤 5 需要前 4 步完成后进行。
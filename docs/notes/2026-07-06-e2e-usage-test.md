# 实际使用测试报告

> **测试时间：** 2026-07-06
> **测试方式：** 编译 MCP 服务器二进制 → 启动 New API（从源码编译） → 启动 MCP 服务器（HTTP 模式）→ 实际发送 JSON-RPC 请求验证各功能

---

## 测试环境

| 组件 | 方式 |
|------|------|
| New API | 从源码编译运行（`D:\Test\installations\new-api`），SQLite 数据库，端口 3000 |
| MCP Server | HTTP 传输模式，端口 18080 |
| 认证 | `NEW_API_SYSTEM_KEY` = admin access_token, `NEW_API_KEY` = token 表 key |
| 测试方法 | curl 发送原始 JSON-RPC 请求，解析 SSE 响应 |

---

## 测试通过的项

| 功能 | 结果 | 备注 |
|------|:----:|------|
| MCP initialize | ✅ | 正确返回 serverInfo、capabilities、extensions |
| notifications/initialized | ✅ | 会话正常建立 |
| tools/list 分页 | ✅ | 两页共 174（或 232）个工具 |
| tools/list 含 tasks 工具 | ✅ | `tasks_get/update/cancel/test_and_report` 均在第二页 |
| test_and_report 异步返回 | ✅ | 返回 task_id，不阻塞 |
| tasks_get 查询任务 | ✅ | 返回完整状态（state, progress, metadata, error） |
| tasks_update 恢复任务 | ✅ | `retry` 操作成功，返回 `state:"running"` |
| tasks_cancel 取消任务 | ✅ | 返回 `state:"cancelled"` |
| list_providers 列渠道 | ✅ | 正确返回 3 个渠道的完整信息 |

---

## 发现的缺陷与问题

### 🔴 1. [严重] 中继/模型工具默认不注册

**现象：** 即使设置了 `NEW_API_KEY`，`createChatCompletion`、`createImageGeneration`、`listModels`、`mjImagine` 等 58 个模型访问工具均未注册。

**原因：** `main.go` 注册中继工具时使用 `registry.Options{EnabledGroups: cfg.RelayEnabledGroups, AllGroups: cfg.RelayAllGroups}`，但这些选项默认都是空/零值。`RegisterTools` 中只有 `AllGroups=true` 或 `EnabledGroups` 非空时才注册工具。没有任何默认"全部注册"的逻辑。

**影响范围：**
- `createChatCompletion` — Claude Code 无法通过 MCP 调用模型
- `createImageGeneration` — 无法生成图片
- `listModels` — 无法列出可用模型
- 所有 58 个中继工具全被静默过滤

**修复方案：** 当 `NEW_API_KEY` 已设置且 `MCP_RELAY_ENABLED_GROUPS` 未显式设置时，默认应注册所有中继工具（`AllGroups=true`）。

**相关代码：**
- `cmd/server/main.go:98` — `RegisterTools(server, relayDefs, registry.Options{...})`
- `internal/registry/registry.go:27` — `!opts.AllGroups && !isEnabled(def, enabled)` 过滤逻辑

---

### 🟠 2. [中等] test_and_report 任务租约过期

**现象：** `test_and_report` 启动后台 goroutine 轮询 New API 的 `/api/system-task/{id}` 端点。但 New API 没有对应的 system task，导致 goroutine 轮询后进入 `input_required` 状态，错误信息为：

```
"error": "task lease expired",
"metadata": {
  "error_type": "poll_failure",
  "message": "渠道测试执行异常: task lease expired。请选择操作：resume(继续) 或 retry(重试)"
}
```

**影响：** test_and_report 始终无法完成，用户必须不断 resume/retry，但每次都会再次失败。形成了**死循环**。

**根本原因分析：** `test_and_report` 的异步实现假设 New API 会创建一个 system task（有 `/api/system-task/{id}` 端点可轮询）。但在实际部署中，New API 的渠道测试是同步接口（`POST /api/channel/test`），不会创建 system task。后台 goroutine 轮询不到结果，租约过期，进入 `input_required`。

**修复方向思考：**
1. 让 `test_and_report` 直接调用同步的 `/api/channel/test` 接口——但这需要重写，因为高并发测试多个渠道仍然需要异步
2. 修改后台 goroutine 的逻辑：如果上游返回 404（无 system task），应直接调 `/api/channel/test` 同步测试
3. 或直接改为同步调用，不要用 TaskManager 包装
- **详见 #tasks-ext-async-flow**

---

### 🟡 3. [低] 工具分页导致关键工具隐藏

**现象：** 174 个工具（API 157 + 高等级 17）中，`tasks_get/update/cancel/test_and_report` 等关键工具全在**第二页**。

**影响：** 如果 MCP 客户端不遍历分页（或只做一次 `tools/list`），会遗漏这些工具。Claude Code 会分页，但增加了启动时的延迟和 token 消耗。

**建议：** 考虑增大默认 PageSize（目前 100），或按使用频率重新排序。

---

### 🟡 4. [低] Metrics 端口 9090 冲突

**现象：** 第二次启动时：
```
{"level":"ERROR","msg":"metrics server error","error":"listen tcp :9090: bind: Only one usage of each socket address"}
```

**影响：** 不影响主服务，但指标采集失败。自用场景不需要 metrics，但错误日志会让用户困惑。

**建议：** metrics server 启动失败应为 Warning 而非 Error，或默认不启用 metrics。

---

### 🟡 5. [低] `.env.example` 中 `MCP_RELAY_ENABLED_GROUPS` 未说明

文件 `.env.example` 第 24 行：
```
MCP_RELAY_ENABLED_GROUPS=
```

为空值且无注释说明。用户不知道需要设置 `all` 才能让中继工具生效（见问题 #1）。

**建议：** 添加注释，如 `# 设置为 "all" 启用全部中继工具，或按逗号分隔指定组名`

---

## 总结

| 严重度 | 数量 | 影响 |
|--------|:----:|------|
| 🔴 严重 | 1 | 中继工具不注册 = 核心功能不可用 |
| 🟠 中等 | 1 | test_and_report 死循环 |
| 🟡 低 | 3 | 分页、metrics 端口、文档 |

**最关键的问题**是 #1（中继工具不注册）。在自用场景中，调用大模型是核心需求，但这个功能需要用户额外配置 `MCP_RELAY_ENABLED_GROUPS=all` 才能开启。这应该默认开启。

问题 #2（test_and_report 死循环）影响渠道测试功能，在自用场景中影响较小，但如果用户想使用这个功能就会遇到无法完成的问题。
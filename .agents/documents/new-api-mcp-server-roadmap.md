# New API MCP Server — 全面开发路线图

## 摘要

本路线图覆盖 new-api-mcp-server 从当前状态到生产级 MCP Server 的完整演进路径。整合了现有缺陷修复（.agents/documents/new-api-mcp-server-improvement-plan.md）和上一轮分析的架构级改进，按优先级分 5 个阶段推进。

---

## 当前状态分析

### 代码库健康度

| 维度 | 状态 | 详情 |
|------|------|------|
| 测试覆盖 | ✅ 良好 | 所有核心模块均有单元测试，handler 测试覆盖完整（GET/POST/Path/Query/非JSON/错误） |
| 可观测性 | ✅ 优秀 | Prometheus 指标 + OTel 链路追踪 + slog 结构化日志 + 敏感字段脱敏 |
| 并发安全 | ⚠️ 可改进 | 无显式竞态控制，但当前架构无共享可变状态（无状态 handler） |
| 错误处理 | ⚠️ 可改进 | 错误信息以纯文本返回，缺少结构化错误码 |
| 安全性 | ⚠️ 可改进 | 路径参数直接替换有注入风险、缺少速率限制、缺少 CORS 配置 |
| 部署 | ✅ 良好 | 双传输（stdio + HTTP）、容器化（Containerfile）、优雅关闭 |

### 测试现状评估

参照你的 Testing Trophy 模型评估当前测试库：

| 层 | 现有文件 | 符合你的 Trophy？ | 评价 |
|---|---------|-----------------|------|
| **静态分析** | Makefile: `make lint` | ✅ 零成本，最高 ROI | 已集成 golangci-lint |
| **全链路集成** (trophy 顶层) | `integration_test.go` | ✅ 最高价值 | 用 MCP 内存传输模拟真实客户端 → 服务器 → mock 上游，完整管道测试 |
| **Handler 集成** (trophy 主力) | `handler_test.go` | ✅ 你的实践认可的模式 | mock 上游 HTTP + 真实工具调用，是"集成验收测试"的正确形态 |
| **核心逻辑单元测试** | `registry_test.go` `parser_test.go` `client_test.go` | ✅ 必要 | 测试标签过滤、OpenAPI 解析、auth 注入——全是复杂逻辑 |
| **简单逻辑单元测试** | `config_test.go` `logging_test.go` | ⚠️ 可精简 | 简单 getter/setter 和日志格式测试 ROI 较低 |

**缺少的关键测试（按验收三问映射）：**

| 缺失测试 | 验收三问关联 | 发现优先级 |
|---------|------------|:--------:|
| 上游超时/断连 — 服务器不 panic，返回结构化错误 | **破坏测试** | 🔴 |
| API Key 无效 — 认证失败时 `IsError=true` + 可解析错误 | **破坏测试** | 🔴 |
| 路径注入攻击 — `../../etc` 被拒绝而非透传 | **破坏测试** | 🟠 |
| 错误码契约 — 错误格式变更时测试失败 | **生效证明** | 🟠 |
| stdio 子进程端到端 | **关键路径覆盖** | 🟡 |

---

## 测试策略：MCP Server 的 Testing Trophy

### 模型

```
优先级排序（从高到低）：
1. 静态分析（go vet, go build, golangci-lint）— 零成本，最高 ROI
2. 全链路集成测试（MCP 内存传输 + mock 上游）— 最高 ROI
3. Handler 集成测试（mock 上游 HTTP + 真实工具调用）— 主力
4. 单元测试（复杂内部逻辑，如参数校验、错误码构造）
5. 传输层端到端测试（stdio/HTTP 子进程）— 关键路径覆盖
```

### 验收三问 × MCP 场景

| 问题 | MCP 场景对应 | 测试写法 |
|------|-------------|---------|
| **用户可见行为** | AI 客户端通过 MCP 协议看到什么？ | 断言 `CallToolResult.Content` 结构、`IsError` 状态、错误 JSON 格式 |
| **生效证明** | 删除这行代码后测试会不会失败？ | 断言具体的错误消息、状态码值，不测模糊的"不为空"、"有内容" |
| **破坏测试** | 上游挂了/Key 无效/参数畸形，服务器怎么反应？ | 验证返回 `IsError=true` + 可解析的错误结构，不是 panic 或空响应 |

### MCP 特定测试模式

```
测试模式 A — 全链路集成（已实现）
  MCP Client (in-memory) → MCP Server → Handler → mock Upstream (httptest)
  验证：工具列表 → 工具调用 → 响应内容 → 错误状态

测试模式 B — 破坏性测试（需新增）
  mock Upstream 返回 500/超时/畸形响应
  验证：MCP 工具调用返回 IsError=true，不 panic

测试模式 C — 关键路径覆盖（需新增）
  stdio: 启动子进程 → stdin 写入 JSON-RPC → 读取 stdout 响应
  HTTP: httptest 启动 Server → POST /mcp → 验证响应
```

### 已识别的缺陷（来自现有计划）

| # | 严重程度 | 问题 | 涉及文件 |
|---|---------|------|---------|
| F1 | **致命** | `New-Api-User` 请求头缺失，管理 API 全部 401 | `internal/client/client.go` |
| F2 | 中等 | README 变量名 `MCP_RELAY_DISABLED_GROUPS` 与实际代码 `MCP_RELAY_ENABLED_GROUPS` 不符 | `README.md` |
| F3 | 低 | OpenAPI 规格文件过时，缺少部分端点 | `openapi/api.json`、`openapi/relay.json` |
| F4 | 低 | 缺少高层次封装工具，AI 需要了解原始 API 端点 | 新增 `internal/hightools/` |

### 架构级改进机会（来自上一轮分析）

| # | 优先级 | 改进项 | 收益 |
|---|--------|--------|------|
| A1 | 高 | 结构化错误码体系 | 客户端可编程解析错误，而非解析纯文本 |
| A2 | 高 | 路径参数注入防护 | 防止路径遍历攻击（如果上游有文件系统参数） |
| A3 | 中 | 健康检查端点 | K8s 存活/就绪探针、运维监控 |
| A4 | 中 | 速率限制 | 防止滥用，保护上游 New API |
| A5 | 中 | 工具搜索（Tool Search） | 195 个工具定义节省上下文窗口 |
| A6 | 中 | CORS 配置 | HTTP 模式下浏览器端 MCP 客户端访问 |
| A7 | 低 | 请求/响应日志采样 | 高流量下减少日志 IO |
| A8 | 低 | 配置热重载 | 无重启更新配置 |

---

## 阶段计划

### 阶段 0：紧急修复（立即执行）

**目标：** 修复阻止项目正常使用的缺陷。

**测试信条：** 修复缺陷前先写一个能证明缺陷存在的测试（验收三问：生效证明）。

| 步骤 | 描述 | 涉及文件 | 执行模式 | 测试要求 |
|------|------|---------|---------|---------|
| 0.1 | 添加 `New-Api-User` 请求头（F1） | `internal/config/config.go`、`internal/client/client.go`、`cmd/server/main.go` | direct+test | **先写**：`client_test.go` 中新增测试，验证 `SourceAPI` 请求携带 `New-Api-User` 头。**后改**：实现逻辑。验证方式：无此头的测试失败 → 加头后测试通过 |
| 0.2 | 修复 README 变量名（F2） | `README.md` | direct | 无需测试 |
| 0.3 | 更新 OpenAPI 规格文件（F3） | `openapi/api.json`、`openapi/relay.json` | direct | **修改后运行**：`TestParse_RealAPISpec` 和 `TestParse_RealRelaySpec` 确认工具数量增加 |
| 0.4 | 修复 `.mcp.json` 配置 | `.mcp.json` | direct | 手动验证 |

**验证：** `go test ./... -v -race -count=1` 全部通过。新增的 `New-Api-User` 测试能证明缺陷已修复（删除代码头后测试失败）。

---

### 阶段 1：高层工具（按官方规划实现，2-3 天）

**目标：** 参照官方 newapi-admin 规划，实现核心 MCP 高层工具，覆盖 80% 管理场景。

**执行策略：** 先做第一批（渠道管理 5 + 用户管理 5），第二批等官方规划稳定后再做。

**设计依据：** 官方 newapi-admin 规划（.agents/documents/new-api-mcp-server-improvement-plan.md 第 98-111 行）：5 模块 23 条指令。

| 步骤 | 描述 | 涉及文件 | 执行模式 | 测试要求 |
|------|------|---------|---------|---------|
| **1.1 渠道管理** | 对标官方渠道模块 5 指令 | | spec-first | |
| | 1.1.1 `list_providers` | `internal/hightools/tools.go` | spec-first | **先写集成测试**：mock 渠道列表 API → 按分组展示 |
| | 1.1.2 `test_and_report` | `internal/hightools/tools.go` | spec-first | **破坏测试**：某渠道不可用时，报告中有错误标志而非 panic |
| | 1.1.3 `toggle_channel` | `internal/hightools/tools.go` | spec-first | **生效证明**：测试验证渠道状态实际变更 |
| | 1.1.4 `set_channel_priority` | `internal/hightools/tools.go` | spec-first | **验证**：优先级数值正确传递 |
| | 1.1.5 `add_channel` | `internal/hightools/tools.go` | spec-first | 集成测试：创建渠道后列表中出现 |
| **1.2 用户管理** | 对标官方用户模块 5 指令 | | spec-first | |
| | 1.2.1 `list_users` | `internal/hightools/tools.go` | spec-first | 集成测试：返回用户列表 |
| | 1.2.2 `show_balance` | `internal/hightools/tools.go` | spec-first | mock 渠道列表 + 余额 → 输出摘要 |
| | 1.2.3 `set_user_quota` | `internal/hightools/tools.go` | spec-first | **破坏测试**：无效配额值被拒绝而非错误传递 |
| | 1.2.4 `switch_group` | `internal/hightools/tools.go` | spec-first | 对标官方 `switch-group`，参考 `api.js` 实现逻辑 |
| | 1.2.5 `toggle_user_status` | `internal/hightools/tools.go` | spec-first | **生效证明**：禁用用户后该用户 API 调用失败 |
| **1.3 工具注册** | 注册到主入口 | `cmd/server/main.go` | direct | 全链路测试：in-memory transport 列出工具，确认新工具出现在列表中 |

**验证：**
- 10 个高层工具全部可通过 MCP 协议调用
- `go test ./... -v -race -count=1` 全部通过
- 每个工具的验收三问已自检

---

### 阶段 2：安全加固（1-2 天）

**目标：** 修复安全漏洞，建立安全基线。

**测试信条：** 破坏测试优先。每项安全加固都要写"如果有人试图绕过它，测试会抓住它"。

| 步骤 | 描述 | 涉及文件 | 执行模式 | 测试要求 |
|------|------|---------|---------|---------|
| **2.1 路径参数注入防护** | | `internal/handler/handler.go` | | |
| | 2.1.1 添加路径参数校验 | `internal/handler/handler.go` | direct+test | **handler_test.go 新增**：传入 `{"id": "../etc/passwd"}`，验证上游收到的 path 不含 `../` |
| | 2.1.2 添加路径参数类型检查 | `internal/handler/handler.go` | direct+test | **破坏测试**：传入字符串当整数参数，验证返回 `IsError=true` |
| **2.2 结构化错误码体系** | | `internal/handler/errors.go` | | |
| | 2.2.1 定义错误码类型 | `internal/handler/errors.go`（新建） | direct | 错误码常量 + 构造函数 |
| | 2.2.2 返回结构化错误 | `internal/handler/handler.go` | direct+test | **生效证明**：新增测试校验错误 JSON 包含 `code` 字段。未来改变错误格式时此测试失败 |
| **2.3 请求头安全** | | | | |
| | 2.3.1 审查 header 参数注入 | `internal/client/client.go` | direct+test | **client_test.go 补充**：尝试通过 header 参数覆盖 `Authorization`，验证未生效 |
| **2.4 HTTP 传输安全** | | `cmd/server/main.go` | | |
| | 2.4.1 可配置 CORS | `cmd/server/main.go` | direct+test | 集成测试：发送 OPTIONS 预检请求验证 CORS 头 |
| | 2.4.2 请求大小限制 | `cmd/server/main.go` | direct+test | 破坏测试：发送超大请求体验证 413 |

**验证：** 
- `go test ./... -v -race -count=1` 全部通过
- **关键破坏测试会失败如果安全防护被移除**：删除参数校验逻辑后，注入测试应该失败

---

### 阶段 3：生产就绪（2-3 天）

**目标：** 使项目达到生产级运维标准。

**测试信条：** 关键路径覆盖纪律。任何可以从 CLI/HTTP 入口到达的代码必须通过入口测试验证。

| 步骤 | 描述 | 涉及文件 | 执行模式 | 测试要求 |
|------|------|---------|---------|---------|
| **3.1 健康检查端点** | | `cmd/server/main.go` | | |
| | 3.1.1 添加 `/healthz` 存活端点 | `cmd/server/main.go` | direct+test | HTTP 集成测试：GET `/healthz` → 200 |
| | 3.1.2 添加 `/readyz` 就绪端点 | `cmd/server/main.go` | direct+test | **破坏测试**：上游不可达时 `/readyz` → 503 |
| **3.2 速率限制** | | 新增 `internal/middleware/` | | |
| | 3.2.1 实现令牌桶中间件 | `internal/middleware/ratelimit.go` | direct+test | 单测验证：消耗令牌 → 速率超过 → 拒绝 |
| | 3.2.2 配置化速率限制 | `internal/config/config.go` + `internal/middleware/` | direct+test | 集成测试：RPS 设为 1，连续发 3 请求，验证第 3 个返回 429 |
| | 3.2.3 集成到 HTTP 传输 | `cmd/server/main.go` | direct | **生效证明**：去掉中间件后 429 测试失败 |
| **3.3 优雅关闭增强** | | `cmd/server/main.go` | | |
| | 3.3.1 添加 HTTP 传输的 drain 逻辑 | `cmd/server/main.go` | direct+test | 子进程测试：发送 SIGTERM，验证正在处理的请求完成 |
| | 3.3.2 添加 stdio 传输的关闭信号 | `cmd/server/main.go` | direct+test | stdio 子进程：关闭 stdin，验证进程退出码 0 |
| **3.4 配置验证** | | `internal/config/config.go` | | |
| | 3.4.1 添加配置校验逻辑 | `internal/config/config.go` | direct+test | **破坏测试**：无效 URL 格式 → 启动失败 + stderr 有错误消息，不是 panic |

**验证：** 
- `curl http://localhost:8080/healthz` 返回 200
- 连续高频请求触发 429 响应
- `go test ./... -v -race -count=1` 全部通过
- **验收三问自检**：每个改动回答，确认测试覆盖

---

### 阶段 4：扩展高层工具（等官方规划稳定，待定）

**目标：** 待官方 newapi-admin 规划稳定后，补充剩余模块。

**前置条件：** 官方 newapi-admin 正式发布或规划稳定。

| 步骤 | 描述 | 涉及文件 | 执行模式 |
|------|------|---------|---------|
| **4.1 系统配置**（对标官方 4 条指令） | 配置查看/修改、倍率设置、同步 | `internal/hightools/` | spec-first |
| **4.2 日志监控**（对标官方 3 条指令） | 日志查询、统计、状态 | `internal/hightools/` | spec-first |
| **4.3 令牌兑换码**（对标官方 6 条指令） | Token 管理、兑换码操作 | `internal/hightools/` | spec-first |

---

### 阶段 5：性能优化（3-5 天）

**目标：** 优化大规模工具集下的性能，支持扩展。

**测试信条：** 性能测试不是功能测试。用基准测试（benchmark）而非单元测试验证性能改进。

| 步骤 | 描述 | 涉及文件 | 执行模式 | 测试要求 |
|------|------|---------|---------|---------|
| **5.1 工具搜索（Tool Search）** | | 多个文件 | | |
| | 5.1.1 研究 MCP Tool Search 协议 | 文档研究 | 研究 | 了解 MCP 协议中 tool search 的实现方式 |
| | 5.1.2 实现工具分组/索引 | 新增 `internal/toolsearch/` | spec-first | 单测验证索引正确性 |
| | 5.1.3 实现搜索接口 | 修改 `internal/registry/` | spec-first | **基准测试**：195 工具下搜索 vs 全量加载的 token 消耗对比 |
| **5.2 响应缓存** | | 新增 `internal/cache/` | | |
| | 5.2.1 实现简单的内存缓存 | `internal/cache/cache.go` | direct+test | 单测：缓存命中返回、TTL 过期失效、并发安全 |
| | 5.2.2 可配置的缓存策略 | `internal/config/config.go` | direct+test | 集成测试：缓存启用时重复请求只转发一次到上游 |
| | 5.2.3 缓存指标 | `internal/cache/metrics.go` | direct+test | 断言指标值正确 |
| **5.3 请求/响应日志采样** | | `internal/observability/logging.go` | | |
| | 5.3.1 实现日志采样器 | `internal/observability/sampling.go` | direct+test | **before/after 对比**：采样率 0.5 时运行 1000 次，日志量约 500 ± 容差 |
| | 5.3.2 配置采样率 | `internal/config/config.go` | direct | config_test 扩展 |

**验证：**
- 工具搜索加载后，195 个工具定义不全部进入上下文
- `go test -bench=. -benchmem` 确认缓存有性能提升
- 高流量下日志量减少 90%

---

### 阶段 6：高级特性（待定）

**目标：** 面向未来扩展，提升 MCP Server 的能力边界。

| 步骤 | 描述 | 详情 |
|------|------|------|
| **6.1 配置热重载** | 监听 SIGHUP 或文件变化，无重启更新配置 | 需要重构配置生命周期 |
| **6.2 动态工具发现** | 从运行中的 New API 实例动态获取端点，无需重启 | 利用 New API 的 OpenAPI 导出端点 |
| **6.3 多租户支持** | 不同用户使用不同的 API Key 和权限 | 需要认证层 + 会话管理 |
| **6.4 WebSocket 传输** | 双向通信，支持流式工具结果推送 | MCP 协议扩展 |
| **6.5 插件系统** | 第三方开发者可编写插件扩展工具 | 基于 Go plugin 或 WASM |

---

## 决策记录

| 决策 | 选项 | 选择 | 理由 |
|------|------|------|------|
| 错误格式 | 纯文本 / JSON 结构体 | **JSON 结构体** | 客户端可编程解析，兼容纯文本降级 |
| 速率限制算法 | 令牌桶 / 漏桶 / 滑动窗口 | **令牌桶** | Go 标准库 `x/time/rate` 支持，社区成熟 |
| 缓存存储 | 内存 / Redis | **内存** | 当前无 Redis 依赖，保持轻量级 |
| 高层次工具位置 | 独立包 / 合并到 handler | **独立包 `internal/hightools/`** | 关注点分离，独立测试 |
| URL 路径参数编码 | 透传 / URL 编码 | **URL 编码** | 防止路径遍历和特殊字符注入 |

---

## 验证计划

### 每个阶段的金标准

| 阶段 | 验证方式 | 标准 |
|------|---------|------|
| 阶段 0 | `go build` + 手动调用工具 | 编译零错误，管理工具返回 200 |
| 阶段 1 | 安全测试 + 错误格式检查 | 路径注入被拒绝，错误返回 JSON 结构 |
| 阶段 2 | `make test` + 集成测试 | 全部测试通过，健康检查端点正常工作 |
| 阶段 3 | 基准测试 + 指标观察 | 上下文消耗减少，缓存命中率 > 50% |

### 测试纪律验证（每阶段结束时执行）

1. **`make test` 全部通过** — `go test ./... -v -race -count=1`
2. **`make lint` 零报错** — `golangci-lint run ./...`
3. **新增测试的验收三问回检** — 逐条确认：
   - 测试失败时是否对应一个可观察的 MCP 行为？
   - 删除被测试的代码后测试是否确实失败？
   - 是否覆盖了破坏场景？
4. **违反案例库检查** — 本次改动没有踩进已知的坑
5. **（PR 合并前）集成测试通过** — `go test -tags=integration -v -run TestIntegration`

---

## 测试纪律（执行过程中必须遵守）

### 每次改动前的自问清单

每次改动（新功能、Bug 修复、重构）提交前，按你的 **验收三问** 自检：

1. **用户可见行为** — AI 客户端通过 MCP 协议看到这次改动的什么变化？
   - 不要写"代码跑通了"，要写"错误时 `IsError=true` 且内容包含 `error.code`"
2. **生效证明** — 有没有一个测试能证明这个行为真的生效了？这个测试会不会在改动被删除时失败？
   - 不要写"模块测通了"，要写"删除这行校验逻辑后注入测试失败"
3. **破坏测试** — 如果我故意捣乱（坏配置、空数据、上游挂了、网络断了），MCP 服务器怎么死？这个死法你接受吗？
   - 不要只测正常路径，测上游超时、APK Key 无效、参数类型错误

### MCP 项目特有规则

| 规则 | 说明 |
|------|------|
| **全链路优先** | 能用 `integration_test.go` 模式（MCP 内存传输）测通完整链路的，不要只测 handler 或 registry |
| **http.Handler 测试是中间层** | 已有 `handler_test.go` 保留当契约文档，新功能不优先写——除非是 HTTP 传输层特有的逻辑 |
| **破坏测试必须写** | MCP 客户端的"用户"是 AI，AI 对上游异常的容忍度比人类更低——必须确保上游挂了返回可用错误而非让 AI 困惑 |
| **end_to_end 测试标记为 integration** | 启动子进程的测试用 `//go:build integration` 隔离，不做日常 CI 但 PR 合并前必须跑 |
| **before/after 对比，不测快照** | 测试缓存的代码，不测 `缓存命中=5`，测 `命中后重复请求的 upstream 调用数 < 未命中时的调用数` |

### 违反案例库（MCP 项目专属）

| 违反 | 后果 | 预防 |
|------|------|------|
| 只测 handler 不测全链路 | MCP 协议消息格式不对导致客户端解析失败 | 至少一个全链路集成测试覆盖新工具 |
| 不测上游异常 | 上游超时导致 MCP 服务器 30s 无响应，AI 客户端超时报错 | 每个工具 handler 至少一个上游异常测试 |
| 只测 IsError 不测错误内容 | `IsError=true` 了但 AI 看不懂错误消息 | 断言错误 JSON 包含有意义的 `code` 和 `message` |
| 不改测试 | 改了一行 handler 没改测试 → 不知道改坏了什么 | 改代码前先确认有没有对应测试，没有就先写验收测试

```
阶段 0：紧急修复（立即）
├── 0.1 New-Api-User 头  ← 最优先，阻塞项
├── 0.2 README 修复      ← 可并行
├── 0.3 OpenAPI 规格更新  ← 可并行
└── 0.4 .mcp.json 配置    ← 可并行

     ↓

阶段 1：高层工具（2-3 天）← 直接价值最高
├── 1.1 渠道管理（5 个工具）
│   ├── list_providers
│   ├── test_and_report
│   ├── toggle_channel
│   ├── set_channel_priority
│   └── add_channel
├── 1.2 用户管理（5 个工具）
│   ├── list_users
│   ├── show_balance
│   ├── set_user_quota
│   ├── switch_group
│   └── toggle_user_status
└── 1.3 注册 + 集成测试

     ↓

阶段 2：安全加固（1-2 天）
├── 2.1 路径参数注入防护
├── 2.2 结构化错误码
├── 2.3 请求头安全
└── 2.4 HTTP 传输安全（CORS + 大小限制）

     ↓

阶段 3：生产就绪（2-3 天）
├── 3.1 健康检查
├── 3.2 速率限制
├── 3.3 优雅关闭增强
└── 3.4 配置验证

     ↓

阶段 4：扩展高层工具（等官方规划稳定，待定）
├── 4.1 系统配置（4 个）
├── 4.2 日志监控（3 个）
└── 4.3 令牌兑换码（6 个）

     ↓

阶段 5：性能优化（3-5 天）
├── 5.1 工具搜索
├── 5.2 响应缓存
└── 5.3 日志采样

     ↓

阶段 6：高级特性（待定）
├── 6.1 配置热重载
├── 6.2 动态工具发现
├── 6.3 多租户
├── 6.4 WebSocket 传输
└── 6.5 插件系统
```

---

## 资源估算

| 阶段 | 预估工时 | 风险等级 | 关键依赖 |
|------|---------|---------|---------|
| 阶段 0 | 2h | 低 | 无 |
| 阶段 1 | 2-3 天 | 中 | 阶段 0 完成 |
| 阶段 2 | 1-2 天 | 低 | 无 |
| 阶段 3 | 2-3 天 | 中 | 无 |
| 阶段 4 | 待定 | 中 | 官方 newapi-admin 规划稳定 |
| 阶段 5 | 3-5 天 | 高 | 需要 MCP 协议理解 |
| 阶段 6 | 待定 | 高 | 需要业务需求确认 |

---

## Workflow 自动化编排

本路线图的执行由 Workflow 脚本 `.agents/workflows/execute-roadmap.mjs` 编排。

**执行方式：** 在当前会话中运行以下命令（由我触发）：

```
Workflow({scriptPath: '.agents/workflows/execute-roadmap.mjs'})
```

**编排规则：**

| 阶段内 | 阶段间 |
|--------|--------|
| 无依赖的步骤自动并行（如 0.2/0.3/0.4） | 阶段 1 依赖阶段 0 完成 |
| 有依赖的步骤串行（如 1.0→1.1→1.3） | 阶段 4 依赖官方规划稳定 |
| 每个步骤执行后自动跑 `go test` 验证 | 其余阶段相对独立 |

**当前编排覆盖：** 阶段 0（全部步骤）+ 阶段 1（全部 10 个工具 + 注册 + 测试）。阶段 2-6 的编排脚本将在后续扩展。

---

## 文档索引

| 文档 | 位置 | 用途 |
|------|------|------|
| 开发路线图 | `.agents/documents/new-api-mcp-server-roadmap.md` | 战略方向、阶段结构、测试纪律 |
| 改进计划（含官方 Skill 分析） | `.agents/documents/new-api-mcp-server-improvement-plan.md` | 缺陷追踪、官方设计参考 |
| Workflow 编排脚本 | `.agents/workflows/execute-roadmap.mjs` | 自动化执行 |
| 阶段 0/1 Spec 文件 | `.agents/specs/` | 执行时由子代理自动生成 |
# MCP 官方文档调研驱动的后续发展路线图

> **创建日期：** 2026-07-06
> **触发事件：** 完整阅读 [MCP 官方文档索引](https://modelcontextprotocol.io/llms.txt)，对照分析 [new-api-mcp-server](https://github.com/QuantumNous/new-api-mcp-server) 当前代码状态后产出的路线图
> **前置文档：** [实现计划](docs/superpowers/plans/2026-03-19-new-api-mcp-server.md) · [设计文档](docs/superpowers/specs/2026-03-19-new-api-mcp-server-design.md)

---

## 1. 背景与起因

### 1.1 为什么会有这份文档

此前，new-api-mcp-server 的开发路线图基于 2025 年初对 MCP 协议的理解制定。期间 MCP 协议经历了从 `2025-06-18` 到 `2025-11-25` 的版本升级，新增了大量重要特性（Tasks 扩展、Extensions 体系、JSON Schema 2020-12、MCP Apps 等），并成立了多个 Working Group 推进协议演进。

2026-07-06，我们完整阅读了 [MCP 官方文档索引](https://modelcontextprotocol.io/llms.txt) 中列出的全部章节，涵盖：

- 最新规范 [Specification 2025-11-25](https://modelcontextprotocol.io/specification/2025-11-25/index.md) 及 [变更日志](https://modelcontextprotocol.io/specification/2025-11-25/changelog.md)
- [Extensions 扩展体系](https://modelcontextprotocol.io/extensions/overview.md)（Tasks、Apps、Authorization）
- [官方路线图](https://modelcontextprotocol.io/development/roadmap.md)（2026-03-05 更新）
- [SDK 分级体系](https://modelcontextprotocol.io/community/sdk-tiers.md) 和 [MCP Registry](https://modelcontextprotocol.io/registry/about.md)
- [SEP（规范增强提案）](https://modelcontextprotocol.io/seps/index.md) 完整列表
- [Security Best Practices](https://modelcontextprotocol.io/docs/tutorials/security/security_best_practices.md)

基于这次全面调研，我们发现当前项目与 MCP 最新规范之间存在若干差距，也发现了多个可以显著提升项目价值的扩展方向。本路线图记录这些发现，并为后续开发提供结构化指引。

### 1.2 如何阅读这份文档

每个发展方向均包含以下结构：

| 章节 | 内容 |
|------|------|
| **相关文档链接** | MCP 官方文档中与此方向直接相关的页面 URL |
| **当前状态** | 项目在此方向上的现状 |
| **做什么** | 具体需要实现的功能或变更 |
| **优先级判断** | 基于投入/产出比的推荐优先级 |
| **为什么值得做** | 用户/项目从这个方向获得什么价值 |

---

## 2. MCP 官方文档概览 — 我们读到了什么

### 2.1 最新规范（2025-11-25）关键变化

[变更日志](https://modelcontextprotocol.io/specification/2025-11-25/changelog.md) 列出了从 2025-06-18 到 2025-11-25 的主要变化，对项目有直接影响的有：

| 变化 | 影响 |
|------|------|
| [JSON Schema 2020-12 成为默认方言](https://modelcontextprotocol.io/seps/1613-establish-json-schema-2020-12-as-default-dialect-f.md)（SEP-1613） | 项目生成的 `InputSchema` 可能需要更新 |
| [工具可以暴露 Icons 元数据](https://modelcontextprotocol.io/seps/973-expose-additional-metadata-for-implementations-res.md)（SEP-973） | 可为注册的工具添加显示图标 |
| [工具命名规范指引](https://modelcontextprotocol.io/seps/986-specify-format-for-tool-names.md)（SEP-986） | 检查项目工具命名是否符合新规范 |
| [输入验证错误应返回 Tool Execution Error](https://modelcontextprotocol.io/seps/1303-input-validation-errors-as-tool-execution-errors.md)（SEP-1303） | 修改错误处理以允许模型自纠正 |
| [工具调用支持加入 Sampling](https://modelcontextprotocol.io/seps/1577--sampling-with-tools.md)（SEP-1577） | 若实现 sampling 功能时参考 |
| [Tasks 的实验性支持](https://modelcontextprotocol.io/extensions/tasks/overview.md)（SEP-1686） | 异步任务扩展 |
| [Streamable HTTP SSE 轮询支持](https://modelcontextprotocol.io/seps/1699-support-sse-polling-via-server-side-disconnect.md)（SEP-1699） | 改进 HTTP 传输模式 |
| [$ref 解耦与参数模式独立](https://modelcontextprotocol.io/seps/1319-decouple-request-payload-from-rpc-methods-definiti.md)（SEP-1319） | 影响 OpenAPI 解析逻辑 |

### 2.2 Extensions 扩展体系

[Extensions 概述](https://modelcontextprotocol.io/extensions/overview.md) 引入了官方扩展机制，每个扩展有唯一标识符（如 `io.modelcontextprotocol/tasks`），通过 capability 协商启用：

| 扩展 | 说明 | 适用场景 |
|------|------|----------|
| [MCP Tasks](https://modelcontextprotocol.io/extensions/tasks/overview.md) | 异步任务执行，支持轮询、中间输入、持久化句柄 | 长时间操作（批量测试、日志导出、批量用户操作） |
| [MCP Apps](https://modelcontextprotocol.io/extensions/apps/overview.md) | 交互式 HTML UI 渲染在聊天气泡中 | 仪表盘、复杂表单、数据可视化（HTTP 部署模式） |
| [OAuth Client Credentials](https://modelcontextprotocol.io/extensions/auth/oauth-client-credentials.md) | 机器对机器 OAuth 认证 | 远程部署时替代 API Key |
| [Enterprise-Managed Authorization](https://modelcontextprotocol.io/extensions/auth/enterprise-managed-authorization.md) | 企业级集中访问控制 | 多租户/企业部署场景 |

### 2.3 官方路线图重点

[Roadmap](https://modelcontextprotocol.io/development/roadmap.md)（2026-03-05 更新）列出了四个优先领域：

1. **Transport Evolution and Scalability** — 无状态 Streamable HTTP、水平扩展、Server Cards
2. **Agent Communication** — Tasks 原语的语义细化（重试、过期策略）
3. **Governance Maturation** — 贡献者阶梯、委托模型
4. **Enterprise Readiness** — 审计追踪、企业级 Auth、网关模式、配置可移植性

此外 [On the Horizon](https://modelcontextprotocol.io/development/roadmap.md#on-the-horizon) 领域包括：Triggers & Events、Result Type Improvements、Security & Authorization、Extensions Ecosystem。

### 2.4 SDK 与生态

- [Go SDK](https://go.sdk.modelcontextprotocol.io) 被列为 [Tier 1 SDK](https://modelcontextprotocol.io/community/sdk-tiers.md)（最高等级），功能完整度有保障
- [MCP Registry](https://modelcontextprotocol.io/registry/about.md) 已上线，支持服务器发布和发现
- [Agent Skills](https://modelcontextprotocol.io/docs/develop/build-with-agent-skills.md) 为 MCP 服务器开发提供可复用的技能集

---

## 3. 项目现状快照

取自代码库核心结构（截至 2026-07-06）：

| 领域 | 状态 | 说明 |
|------|------|------|
| OpenAPI 解析 → 工具注册 | ✅ 完成 | ~160 端点自动映射为 MCP Tools |
| 高级工具（hightools） | ✅ 完成 | 14 个内置逻辑工具（渠道/用户/令牌管理等） |
| 双传输模式（stdio + Streamable HTTP） | ✅ 完成 | 通过 `MCP_TRANSPORT` 切换 |
| 可观测性（slog + Prometheus + OTel） | ✅ 完成 | 完整的三支柱可观测性 |
| 中间件（Rate Limit） | ✅ 完成 | 内置速率限制 |
| Docker / CI/CD (Tekton) | ✅ 完成 | 容器化部署与持续集成 |
| Schema 提取器（extractor） | ✅ 完成 | 自动从运行实例推断 OpenAPI 结构 |
| **Resources（资源）** | ❌ 未实现 | MCP 一等公民，项目未使用 |
| **Prompts（提示模板）** | ❌ 未实现 | MCP 一等公民，项目未使用 |
| **Tasks（异步任务）** | ❌ 未实现 | 官方扩展，可优化长时间操作 |
| **JSON Schema 2020-12** | ❌ 未跟进 | 项目使用旧版 Schema 格式 |
| **Extensions 协商** | ❌ 未实现 | 存在且应该声明的能力未声明 |
| **MCP Registry** | ❌ 未发布 | 服务器未在官方 Registry 注册 |
| **MCP Apps** | ❌ 未实现 | 基于 UI 的交互能力缺失 |

---

## 4. 可发展方向（详细）

---

### 方向 A：实现 Resources（资源）支持

**相关文档链接：**
- [Spec: Resources](https://modelcontextprotocol.io/specification/2025-11-25/server/resources.md) — 资源完整定义
- [Spec: Resource Templates](https://modelcontextprotocol.io/specification/2025-11-25/server/resources.md#resource-templates) — URI 模板
- [Spec: Pagination](https://modelcontextprotocol.io/specification/2025-11-25/server/utilities/pagination.md) — 列表分页
- [Spec: Subscriptions](https://modelcontextprotocol.io/specification/2025-11-25/server/resources.md#subscriptions) — 资源变更订阅
- [Spec: Completion](https://modelcontextprotocol.io/specification/2025-11-25/server/utilities/completion.md) — 参数自动补全
- [SEP-973: Additional Metadata](https://modelcontextprotocol.io/seps/973-expose-additional-metadata-for-implementations-res.md) — 资源图标与注解

**当前状态：** 项目只实现了 Tools 机制。MCP 协议定义了 Resources 作为一等公民（与 Tools、Prompts 并列），通过 `resources/list`、`resources/read`、`resources/templates/list` 等方法访问。当前项目完全没有使用这一机制。

**做什么：**

1. 在 `internal/resources/` 包中定义数据源接口
2. 实现 `resources/list` handler：返回渠道列表、令牌列表、供应商列表等
3. 实现 `resources/templates/list`：定义 URI 模板（如 `newapi://channel/{id}`、`newapi://token/{id}`）
4. 实现 `resources/read`：根据 URI 返回具体资源内容
5. 可选：实现 `resources/subscribe`，当渠道状态变化时推送 `notifications/resources/updated` 通知

**为什么值得做：**
- Resources 是 MCP 协议的一等公民，缺失它使得这个 MCP 服务器在协议层面不完整
- 客户端（特别是带 UI 的客户端）可以像浏览文件系统一样浏览 New API 的数据，提升用户体验
- 资源模板自动补全（completion）让 LLM 可以更准确地构造参数

**推荐优先级：** ⭐⭐⭐ 高

---

### 方向 B：实现 Prompts（提示模板）支持

**相关文档链接：**
- [Spec: Prompts](https://modelcontextprotocol.io/specification/2025-11-25/server/prompts.md) — 提示模板完整定义
- [Spec: Completion](https://modelcontextprotocol.io/specification/2025-11-25/server/utilities/completion.md) — 参数自动补全
- [Spec: Messages](https://modelcontextprotocol.io/specification/2025-11-25/server/prompts.md#messages) — 提示消息格式与角色
- [Spec: Arguments](https://modelcontextprotocol.io/specification/2025-11-25/server/prompts.md#arguments) — 模板参数定义

**当前状态：** 项目完全没有使用 Prompts 机制。MCP 协议中 Prompts 是与 Tools、Resources 并列的一等公民，用于提供可复用的提示模板。CLAUDE.md 和设计文档中均未提及此方向。

**做什么：**

1. 在 `internal/prompts/` 包中定义提示模板集合
2. 实现 `prompts/list`：注册预定义的提示模板
3. 实现 `prompts/get`：返回带参数填充的提示内容
4. 模板示例：
   - `channel-setup-guide`：引导创建新渠道的步骤式模板（参数：`provider`、`model_type`）
   - `troubleshoot-channel`：渠道故障排查模板（参数：`channel_id`）
   - `usage-report`：使用量统计报告模板（参数：`time_range`、`group_by`）
   - `batch-operation`：批量操作模板（参数：`operation`、`target_type`）
5. 可选：为 arguments 实现 `completion/complete` 端点（如自动补全渠道 ID）

**为什么值得做：**
- Prompts 让 LLM 在正确引导下调用工具，减少执行出错率
- 对于复杂操作（如创建渠道需填写十余个参数），预定义模板可以大幅降低误操作风险
- 提升 MCP 服务器的"开箱即用"体验，降低学习成本

**推荐优先级：** ⭐⭐⭐ 高

---

### 方向 C：集成 MCP Tasks 扩展（异步任务）

**相关文档链接：**
- [Extensions: Tasks Overview](https://modelcontextprotocol.io/extensions/tasks/overview.md) — 完整的功能说明和生命周期
- [Spec: Tasks (experimental)](https://modelcontextprotocol.io/specification/2025-11-25/basic/utilities/tasks.md) — 规范定义
- [SEP-1686: Tasks](https://modelcontextprotocol.io/seps/1686-tasks.md) — 增强提案
- [SEP-2663: Tasks Extension](https://modelcontextprotocol.io/seps/2663-tasks-extension.md) — 扩展版的 Tasks
- [Extensions: Overview → Negotiation](https://modelcontextprotocol.io/extensions/overview.md#negotiation) — 扩展协商机制
- [Extensions: Client Matrix](https://modelcontextprotocol.io/extensions/client-matrix.md) — 客户端支持矩阵

**当前状态：** `test_and_report`（测试所有渠道）是同步阻塞的。当渠道数量较多（>50）时，这个调用会长时间占用连接，在 Streamable HTTP 模式下可能超时。其他批量操作（如批量创建令牌、批量更新配额）同样面临这个问题。

**做什么：**

1. 添加 `io.modelcontextprotocol/tasks` 扩展的 capability 声明
2. 实现 `tasks/get`、`tasks/update`、`tasks/cancel` handler
3. 改造 `test_and_report` 返回 `CreateTaskResult`：
   - 后台异步执行渠道批量测试
   - 客户端通过 `tasks/get` 轮询进度
   - 遇到渠道异常时进入 `input_required` 状态等待用户确认
4. 在 `internal/hightools/` 中提供 Tasks 工具基类，方便后续工具复用

**为什么值得做：**
- 解决长时间操作阻塞传输连接的实际问题
- Tasks 的 `input_required` 能力让批量操作可以暂停等待用户决策，而不是全部失败或全部跳过
- 与 MCP 官方路线图方向一致（Agent Communication 是 Roadmap 优先领域）
- 当渠道数量较多时显著提升使用体验

**推荐优先级：** ⭐⭐⭐ 高

---

### 方向 D：JSON Schema 升级至 2020-12

**相关文档链接：**
- [SEP-1613: Establish JSON Schema 2020-12 as Default Dialect](https://modelcontextprotocol.io/seps/1613-establish-json-schema-2020-12-as-default-dialect-f.md) — 确立默认方言
- [SEP-2106: Tools inputSchema & outputSchema Conform to JSON Schema 2020-12](https://modelcontextprotocol.io/seps/2106-json-schema-2020-12.md) — 工具 schema 合规
- [Spec: Tools → inputSchema](https://modelcontextprotocol.io/specification/2025-11-25/server/tools.md) — 工具的 inputSchema 定义
- [JSON Schema 2020-12 Specification](https://json-schema.org/specification) — 官方规范

**当前状态：** 项目的 `schemaToMap()` 函数（`internal/openapi/parser.go`）生成的 JSON Schema 遵循 OpenAPI 3.0 时代的标准格式，未显式声明 `$schema` 字段，也未采用 JSON Schema 2020-12 的语法。

**做什么：**

1. 在生成 `InputSchema` 时添加 `"$schema": "https://json-schema.org/draft/2020-12/schema"` 字段
2. 检查现有 schema 格式是否符合 2020-12 规范（如 `const`、`examples`、`deprecated` 等新关键字支持）
3. 更新 `schemaToMap()` 以支持 2020-12 的新特性（如有需要）
4. 更新 `internal/hightools/tooldef.go` 的结构体注解以明确标注 2020-12 兼容

**为什么值得做：**
- MCP SDK 在 validateToolName 等校验中已隐含了对新规范的期待
- 明确的 `$schema` 声明让 IDE、验证器等工具可以正确校验输入
- 这是一个变更量小但符合规范要求的改进

**推荐优先级：** ⭐⭐ 中

---

### 方向 E：为工具添加图标与元数据（SEP-973）

**相关文档链接：**
- [SEP-973: Additional Metadata for Implementations, Resources, Tools, Prompts](https://modelcontextprotocol.io/seps/973-expose-additional-metadata-for-implementations-res.md) — 增强元数据提案
- [Spec: Tools → Icons](https://modelcontextprotocol.io/specification/2025-11-25/server/tools.md) — 工具图标字段
- [Spec: Resources → Annotations](https://modelcontextprotocol.io/specification/2025-11-25/server/resources.md#annotations) — 资源注解字段（audience、priority、lastModified）

**当前状态：** 工具的 `mcp.Tool` 结构体只包含 `Name`、`Description`、`InputSchema` 三个字段，未使用 `icons` 和 `annotations`。

**做什么：**

1. 为每个高级工具（hightools）添加语义化的 `icons` 定义
2. 为 OpenAPI 自动生成的工具分组添加分类图标
3. 在 `internal/openapi/parser.go` 中解析 OpenAPI 的 `x-logo`、`x-icon` 等扩展字段（如果 spec 中有定义）
4. 利用 `Tool.InputSchema` 或扩展字段添加 `annotations`（如渠道管理工具标记 `audience: ["user", "assistant"]`）

**为什么值得做：**
- 对支持图标显示的 MCP host（如 VS Code GitHub Copilot、Claude Desktop），图标让工具列表更直观
- `annotations` 中的 `priority` 字段可以帮助 LLM 判断工具使用优先级
- 低投入、高可见度的改进

**推荐优先级：** ⭐⭐ 中

---

### 方向 F：实现 MCP Apps 交互式 UI（扩展）

**相关文档链接：**
- [Extensions: MCP Apps Overview](https://modelcontextprotocol.io/extensions/apps/overview.md) — 交互式 UI 完整指南
- [Extensions: Build an MCP App](https://modelcontextprotocol.io/extensions/apps/build.md) — 快速开始
- [MCP Apps 官方 API 文档](https://apps.extensions.modelcontextprotocol.io) — 完整 API 参考
- [SEP-1865: MCP Apps](https://modelcontextprotocol.io/seps/1865-mcp-apps-interactive-user-interfaces-for-mcp.md) — 增强提案
- [Extensions: Client Matrix](https://modelcontextprotocol.io/extensions/client-matrix.md) — 客户端支持矩阵

**当前状态：** 项目所有工具都以 text/JSON 方式返回结果。对于数据可视化场景（如系统概览、渠道测试报告、使用量分析），纯文本输出展示效果有限。当前没有使用任何 MCP Apps 扩展点。

**做什么：**

1. 在 `cmd/server/` 的 capability 声明中添加 `io.modelcontextprotocol/ui` 扩展
2. 设计并实现交互式仪表盘 App：
   - 系统概览（渠道数、令牌数、今日请求量、余额概览）
   - 渠道测试结果可视化（成功/失败分布、响应时间图表）
   - 使用量分析（按供应商、按模型的请求量趋势）
3. 在工具定义中添加 `_meta.ui.resourceUri` 字段，将工具结果与 UI 资源关联
4. UI 资源通过 `ui://` scheme 的 Resource 提供服务

**为什么值得做：**
- MCP Apps 在 Claude Desktop、VS Code GitHub Copilot、Microsoft 365 Copilot 等主流 Host 中得到支持
- 对于 HTTP 远程部署模式，MCP Apps 提供比纯文本更直观的操作界面
- 数据可视化场景（渠道测试报告、使用量分析）受益最大
- MCP Apps 的 sandboxed iframe 安全模型保障了安全性

**推荐优先级：** ⭐⭐ 中（适用于 HTTP 远程部署场景）

---

### 方向 G：发布到 MCP Registry

**相关文档链接：**
- [Registry: About](https://modelcontextprotocol.io/registry/about.md) — Registry 介绍
- [Registry: Quickstart](https://modelcontextprotocol.io/registry/quickstart.md) — 发布快速开始
- [Registry: Authentication](https://modelcontextprotocol.io/registry/authentication.md) — 发布认证
- [Registry: GitHub Actions](https://modelcontextprotocol.io/registry/github-actions.md) — 自动发布 CI
- [Registry: Package Types](https://modelcontextprotocol.io/registry/package-types.md) — 支持的包类型
- [Registry: Versioning](https://modelcontextprotocol.io/registry/versioning.md) — 版本管理
- [Registry: Remote Servers](https://modelcontextprotocol.io/registry/remote-servers.md) — 远程服务器发布
- [Registry: FAQ](https://modelcontextprotocol.io/registry/faq.md) — 常见问题

**当前状态：** 项目只存在于 GitHub 仓库，未在 MCP Registry 注册。用户只能通过手动配置或克隆仓库来使用。

**做什么：**

1. 创建 `server.json` 元数据文件（含服务器名称、描述、命令、环境变量、认证方式等）
2. 配置 GitHub Actions 自动发布 workflow
3. 设计 package 格式：支持 stdio 二进制安装包
4. 添加 README 中的 Registry 徽章和安装指引
5. 可选：发布为 MCP Bundle (`.mcpb`) 格式以降低安装门槛

**为什么值得做：**
- MCP Registry 是官方服务器分发渠道，被主流客户端直接搜索
- 注册后用户可以一键安装，无需手动配置
- 提升项目可见度和用户基数
- Registry 支持版本管理，用户可自动接收更新

**推荐优先级：** ⭐⭐ 中

---

### 方向 H：实现扩展协商（Extension Negotiation）

**相关文档链接：**
- [Extensions: Overview → Negotiation](https://modelcontextprotocol.io/extensions/overview.md#negotiation) — 扩展协商机制完整说明
- [Extensions: Overview → Graceful Degradation](https://modelcontextprotocol.io/extensions/overview.md#graceful-degradation) — 优雅降级
- [Spec: Lifecycle → Initialize](https://modelcontextprotocol.io/specification/2025-11-25/basic/lifecycle.md) — initialize 流程

**当前状态：** 项目在 `mcp.NewServer` 中使用默认 capabilities，未自定义 `extensions` 字段。虽然实际使用的 Streamable HTTP 传输模式需要特定的 capability 协商，但代码中没有显式声明。

**做什么：**

1. 在 `cmd/server/main.go` 中修改 server 初始化，添加 extensions capabilities 声明
2. 声明 `io.modelcontextprotocol/streamable-http` 扩展支持
3. 添加扩展协商的版本匹配逻辑（参考 spec 中的 protocol version 协商）
4. 如果实现了 Tasks 扩展，在 capability 中添加对应的 extension identifier
5. 实现优雅降级：根据客户端不支持某些扩展时的 fallback 行为

**为什么值得做：**
- 扩展协商是 MCP Extensions 体系的核心机制
- 显式声明 capabilities 让客户端可以准确了解服务器能力
- 为后续实现更多扩展（Tasks、Apps）打下基础
- 当前项目通过了扩展协商但未显式声明，属于"隐性正确"

**推荐优先级：** ⭐⭐ 中

---

### 方向 I：跟进 Server Card 标准（Roadmap 方向）

**相关文档链接：**
- [Roadmap: Transport Evolution → Server Cards](https://modelcontextprotocol.io/development/roadmap.md#1-transport-evolution-and-scalability) — 路线图描述
- [Spec: Architecture](https://modelcontextprotocol.io/specification/2025-11-25/architecture/index.md) — 架构概览
- [Server Card Working Group Charter](https://modelcontextprotocol.io/community/working-groups/server-card.md) — WG 章程

**当前状态：** Server Card 标准仍在 Working Group 阶段，尚未发布正式规范。项目暂不需要实现，但应持续关注进展。

**做什么：**

1. 关注 Server Card WG 的进展和 SEP 发布
2. 标准发布后，在 `/.well-known/mcp-server.json` 路径提供服务
3. 元数据应包括：服务器名称、版本、功能列表、认证方式、连接信息
4. 确保 OpenAPI 解析后的工具列表可通过 Server Card 自动发现

**为什么值得做：**
- Server Card 是 Roadmap 优先领域的一部分
- 实现后浏览器、爬虫和 Registry 无需连接即可发现服务器能力
- 与 MCP Registry 发布配合，形成完整的可发现性基础设施

**推荐优先级：** ⭐（观望，等待标准成熟）

---

### 方向 J：认证与授权增强

**相关文档链接：**
- [Spec: Authorization](https://modelcontextprotocol.io/specification/2025-11-25/basic/authorization.md) — 认证规范
- [Extensions: OAuth Client Credentials](https://modelcontextprotocol.io/extensions/auth/oauth-client-credentials.md) — OAuth 机器对机器流
- [Extensions: Enterprise-Managed Authorization](https://modelcontextprotocol.io/extensions/auth/enterprise-managed-authorization.md) — 企业级授权
- [SEP-1046: OAuth Client Credentials in Authorization](https://modelcontextprotocol.io/seps/1046-support-oauth-client-credentials-flow-in-authoriza.md) — OAuth 增强提案
- [Security Best Practices](https://modelcontextprotocol.io/docs/tutorials/security/security_best_practices.md) — 安全最佳实践
- [Roadmap: Enterprise Readiness](https://modelcontextprotocol.io/development/roadmap.md#4-enterprise-readiness) — 企业需求路线图

**当前状态：** HTTP 传输模式下没有任何内置认证（设计文档明确说明"由前置网关/反向代理负责"）。对于没有前置网关的部署场景，所有 API tools（启用的 admin 工具）暴露在未认证环境中。

**做什么：**

1. 实现 OAuth Client Credentials 扩展：
   - 服务器注册 OAuth client
   - 客户端通过 `token` 端点获取 access token
   - 请求携带 `Authorization: Bearer` header
2. 添加 per-tool 的细粒度权限控制：
   - relay tools 使用 `NEW_API_KEY`
   - admin tools 使用 `NEW_API_SYSTEM_KEY`
   - 允许更细粒度的 scope 控制
3. 添加网关代理支持（与 Roadmap 的 Gateway/Proxy 方向一致）

**为什么值得做：**
- HTTP 模式下无认证是已知的安全缺口
- OAuth Client Credentials 是最小侵入的解决方案
- 企业部署场景下认证是硬性要求

**推荐优先级：** ⭐⭐ 中（HTTP 远程部署时才需要）

---

### 方向 K：工具命名规范化（SEP-986）

**相关文档链接：**
- [SEP-986: Specify Format for Tool Names](https://modelcontextprotocol.io/seps/986-specify-format-for-tool-names.md) — 工具命名规范
- [Spec: Tools](https://modelcontextprotocol.io/specification/2025-11-25/server/tools.md) — 工具定义

**当前状态：** 项目的 `sanitizeToolName()` 函数（`internal/openapi/parser.go:855`）处理了名称中的非法字符。但命名风格未完全对齐 SEP-986 的建议。目前自动生成的名称如 `post_api_items`、`get_item` 等使用蛇形命名法，而规范推荐使用 `kebab-case`。

**做什么：**

1. 检查 `generateName()` 函数（`internal/openapi/parser.go:844`）生成的名称格式
2. 将蛇形命名法转换为 kebab-case（如 `post_api_items` → `post-api-items`）
3. 统一高亮工具（hightools）的命名风格
4. 添加名称唯一性校验的测试用例覆盖 SEP-986 的场景
5. **注意：** 这是一个破坏性变更！所有已配置的 MCP 客户端需要更新工具名称。建议在版本大更新时一起做。

**为什么值得做：**
- 与官方规范对齐，减少兼容性问题
- kebab-case 在 URL 和文件名中更常用

**推荐优先级：** ⭐（低优先级，等待重大版本更新时一并处理）

---

### 方向 L：输入验证错误处理规范化（SEP-1303）

**相关文档链接：**
- [SEP-1303: Input Validation Errors as Tool Execution Errors](https://modelcontextprotocol.io/seps/1303-input-validation-errors-as-tool-execution-errors.md) — 输入验证错误处理
- [Spec: Tools → Tool Execution Errors](https://modelcontextprotocol.io/specification/2025-11-25/server/tools.md) — 工具执行错误定义

**当前状态：** 当前 handler 的验证逻辑在参数解析失败时返回带有 `IsError=true` 的 `CallToolResult`，但错误消息格式未按 Tool Execution Error 规范处理。

**做什么：**

1. 检查 `internal/handler/handler.go` 中的错误返回逻辑
2. 确保验证错误使用标准的 JSON-RPC error 格式（`code: -32000`）
3. 错误消息应包含可读性描述，以允许 LLM 模型自纠正（模型可以读取错误消息并修正参数后重试）
4. 添加错误格式的测试用例

**为什么值得做：**
- 规范明确要求：输入验证错误应作为 Tool Execution Error 而非 Protocol Error 返回
- 正确的错误格式可以让 LLM 自动纠正参数错误，减少用户干预
- 当前行为虽然"能用"，但不完全符合规范

**推荐优先级：** ⭐⭐ 中

---

### 方向 M：Agent Skills 集成

**相关文档链接：**
- [Build with Agent Skills](https://modelcontextprotocol.io/docs/develop/build-with-agent-skills.md) — Agent Skills 开发指南
- [mcp-server-dev plugin (GitHub)](https://github.com/anthropics/claude-plugins-official/tree/main/plugins/mcp-server-dev) — MCP 服务器开发技能集
- [Claude Code Skills](https://code.claude.com/docs/skills.html) — Claude Code 技能系统

**当前状态：** 项目自身使用了很多 Claude Code skills（trae-spec、trae-plan 等）来辅助开发，但没有结构化为可复用的 MCP 开发技能。

**做什么：**

1. （可选）创建一个 `build-new-api-mcp-server` skill，包含：
   - New API 的 OpenAPI 结构知识
   - 项目架构和关键设计决策
   - 常用开发工作流（更新 spec、注册新工具、测试等）
2. 参考 `mcp-server-dev` 插件中的 `build-mcp-server` skill 模式

**为什么值得做：**
- MCP Server 开发技能集是 Anthropic 官方推荐的方法论
- 结构化的 skill 可以让新手开发者快速上手
- 但项目已大部分完成，收益有限

**推荐优先级：** ⭐（低优先级，项目已基本完成）

---

### 方向 N：可观测性增强 — 审计追踪（Audit Trail）

**相关文档链接：**
- [Roadmap: Enterprise Readiness → Audit Trails](https://modelcontextprotocol.io/development/roadmap.md#4-enterprise-readiness) — Roadmap 描述
- [Spec: Logging](https://modelcontextprotocol.io/specification/2025-11-25/server/utilities/logging.md) — 日志规范
- [Security Best Practices](https://modelcontextprotocol.io/docs/tutorials/security/security_best_practices.md) — 安全最佳实践

**当前状态：** 项目有完整的可观测性（slog + Prometheus + OTel Tracing），但日志设计偏向运维监控而非审计合规。没有区分操作审计日志和系统运行日志。谁在什么时间执行了什么操作没有被结构化记录。

**做什么：**

1. 在 `internal/observability/` 中添加专门的 Audit Logger：
   - 记录每次 tool 调用的操作者身份（如果有 source 信息）
   - 记录操作类型、目标资源、操作结果
   - 按结构化格式输出，便于导入外部审计系统
2. 添加审计事件的 Prometheus counter（`mcp_audit_events_total`）
3. 确保审计日志不记录敏感数据（API keys、tokens 等）

**为什么值得做：**
- Enterprise Readiness 是 Roadmap 四大优先领域之一
- 审计追踪是企业部署的基本要求
- 当前日志虽然记录了操作，但没有明确的审计语义区分

**推荐优先级：** ⭐⭐ 中（企业部署场景才需要）

---

## 5. 阶段规划

### 5.1 优先级矩阵

| 方向 | 投入 | 产出 | 推荐优先级 |
|------|------|------|-----------|
| **A: Resources** | 中（~3-5 天） | 高（协议完整性） | ⭐⭐⭐ |
| **B: Prompts** | 中（~2-3 天） | 高（LLM 交互质量） | ⭐⭐⭐ |
| **C: Tasks** | 高（~5-8 天） | 高（突破性体验改进） | ⭐⭐⭐ |
| **D: JSON Schema 2020-12** | 低（~0.5 天） | 中（规范合规） | ⭐⭐ |
| **E: Tool Icons** | 低（~0.5 天） | 中（显示增强） | ⭐⭐ |
| **F: MCP Apps** | 高（~5-10 天） | 中（仅 HTTP 模式受益） | ⭐⭐ |
| **G: Registry** | 低（~1 天） | 中（分发渠道） | ⭐⭐ |
| **H: Extension Negotiation** | 低（~0.5 天） | 中（基础能力） | ⭐⭐ |
| **I: Server Card** | — | — | ⭐（观望） |
| **J: OAuth/Auth** | 高（~3-5 天） | 中（远程部署才需要） | ⭐⭐ |
| **K: 命名规范** | 中（~1-2 天） | 低（破坏性变更） | ⭐ |
| **L: 错误处理规范** | 低（~0.5 天） | 中（规范合规） | ⭐⭐ |
| **M: Agent Skills** | 低（~0.5 天） | 低（项目已基本完成） | ⭐ |
| **N: Audit Trail** | 中（~2-3 天） | 中（企业场景） | ⭐⭐ |

### 5.2 阶段划分

#### 阶段 1：近期（下一个 Sprint，建议优先实施）

| 方向 | 估算工作量 | 预期产出 |
|------|-----------|---------|
| **A: Resources** | 3-5 天 | `internal/resources/` 包，实现 `resources/list`、`resources/read`、`resources/templates/list` |
| **B: Prompts** | 2-3 天 | `internal/prompts/` 包，3-5 个模板，实现 `prompts/list`、`prompts/get` |
| **D: JSON Schema 2020-12** | 0.5 天 | 更新 `schemaToMap()` 添加 `$schema` 字段 |
| **H: Extension Negotiation** | 0.5 天 | 在 `cmd/server/main.go` 中添加 capabilities 声明 |
| **L: 错误处理规范** | 0.5 天 | 更新 `handler.go` 错误格式 |

**阶段 1 总计：** ~7-9 天

#### 阶段 2：短期（下一迭代）

| 方向 | 估算工作量 | 预期产出 |
|------|-----------|---------|
| **C: Tasks** | 5-8 天 | 异步任务底座 + 改造 `test_and_report` |
| **E: Tool Icons** | 0.5 天 | 为每个 hightool 添加 icons |
| **G: Registry** | 1 天 | `server.json` + GitHub Actions workflow |

**阶段 2 总计：** ~6-10 天

#### 阶段 3：中期（按需推进）

| 方向 | 触发条件 |
|------|---------|
| **F: MCP Apps** | 项目主要部署在 HTTP 模式且需要可视化 |
| **J: OAuth/Auth** | 部署到没有前置网关的远程环境 |
| **N: Audit Trail** | 企业部署要求审计合规 |

#### 阶段 4：长期关注

| 方向 | 状态 |
|------|------|
| **I: Server Card** | 等待标准成熟 |
| **K: 命名规范** | 等待重大版本更新时一并处理 |
| **M: Agent Skills** | 作为社区贡献，无硬性需求 |

---

## 6. 附录

### 6.1 MCP 文档索引速查表

以下是本路线图中引用到的所有 MCP 官方文档链接分类汇总：

**规范核心：**
- [Spec 2025-11-25 首页](https://modelcontextprotocol.io/specification/2025-11-25/index.md)
- [变更日志](https://modelcontextprotocol.io/specification/2025-11-25/changelog.md)
- [架构](https://modelcontextprotocol.io/specification/2025-11-25/architecture/index.md)
- [生命周期](https://modelcontextprotocol.io/specification/2025-11-25/basic/lifecycle.md)
- [传输层](https://modelcontextprotocol.io/specification/2025-11-25/basic/transports.md)

**一等公民：**
- [Tools](https://modelcontextprotocol.io/specification/2025-11-25/server/tools.md)
- [Resources](https://modelcontextprotocol.io/specification/2025-11-25/server/resources.md)
- [Prompts](https://modelcontextprotocol.io/specification/2025-11-25/server/prompts.md)

**扩展：**
- [Extensions 总览](https://modelcontextprotocol.io/extensions/overview.md)
- [Tasks](https://modelcontextprotocol.io/extensions/tasks/overview.md)
- [MCP Apps](https://modelcontextprotocol.io/extensions/apps/overview.md)
- [OAuth Client Credentials](https://modelcontextprotocol.io/extensions/auth/oauth-client-credentials.md)
- [Enterprise-Managed Authorization](https://modelcontextprotocol.io/extensions/auth/enterprise-managed-authorization.md)

**Registry：**
- [Registry 总览](https://modelcontextprotocol.io/registry/about.md)
- [快速开始](https://modelcontextprotocol.io/registry/quickstart.md)
- [GitHub Actions 自动发布](https://modelcontextprotocol.io/registry/github-actions.md)
- [认证方式](https://modelcontextprotocol.io/registry/authentication.md)

**SEP（规范增强提案）：**
- [SEP 索引](https://modelcontextprotocol.io/seps/index.md)
- [SEP-973: 工具/资源元数据](https://modelcontextprotocol.io/seps/973-expose-additional-metadata-for-implementations-res.md)
- [SEP-986: 工具命名规范](https://modelcontextprotocol.io/seps/986-specify-format-for-tool-names.md)
- [SEP-1303: 输入验证错误处理](https://modelcontextprotocol.io/seps/1303-input-validation-errors-as-tool-execution-errors.md)
- [SEP-1613: JSON Schema 2020-12](https://modelcontextprotocol.io/seps/1613-establish-json-schema-2020-12-as-default-dialect-f.md)
- [SEP-1686: Tasks](https://modelcontextprotocol.io/seps/1686-tasks.md)
- [SEP-1865: MCP Apps](https://modelcontextprotocol.io/seps/1865-mcp-apps-interactive-user-interfaces-for-mcp.md)
- [SEP-2106: Tools Schema 2020-12](https://modelcontextprotocol.io/seps/2106-json-schema-2020-12.md)
- [SEP-2663: Tasks Extension](https://modelcontextprotocol.io/seps/2663-tasks-extension.md)

**路线图与社区：**
- [官方路线图](https://modelcontextprotocol.io/development/roadmap.md)
- [SDK 分级](https://modelcontextprotocol.io/community/sdk-tiers.md)
- [安全最佳实践](https://modelcontextprotocol.io/docs/tutorials/security/security_best_practices.md)
- [Agent Skills](https://modelcontextprotocol.io/docs/develop/build-with-agent-skills.md)
- [Server Card WG](https://modelcontextprotocol.io/community/working-groups/server-card.md)

### 6.2 与既有文档的关系

| 文档 | 关系 |
|------|------|
| [2026-03-19-new-api-mcp-server.md](2026-03-19-new-api-mcp-server.md) | 原始实现计划，已完成，产出当前代码库 |
| [2026-03-19-new-api-mcp-server-design.md](../specs/2026-03-19-new-api-mcp-server-design.md) | 原始设计文档，定义了项目架构决策 |
| 本路线图 | 基于 MCP 最新规范调研的后续发展指引 |

### 6.3 更新记录

| 日期 | 版本 | 变更 |
|------|------|------|
| 2026-07-06 | v1.0 | 初版，基于 MCP 官方文档完整调研产出 14 个可发展方向 |
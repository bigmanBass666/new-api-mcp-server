# 自用优先执行路线图

> **创建日期：** 2026-07-06
> **策略：** 先满足自用需求，有余力再考虑公开
> **前驱文档：** [MCP 官方文档调研驱动的后续发展路线图](../notes/2026-07-06-mcp-docs-driven-roadmap.md)（完整调研，14 个方向 A-N）

---

## 1. 策略说明

基于当前"自用优先"的方针，所有方向按以下两级分类：

- **立即执行** — 当前使用中有真实痛点的改进
- **按需触发** — 不主动做，等到遇到具体场景再补

原路线图中的 14 个方向（A-N）在本文档中重新标注优先级，具体分析见原文档。

---

## 2. 立即执行

### 2.1 Tasks 扩展集成（原方向 C）✅ 已完成

**验收结果：**
- TaskManager 核心实现（6 状态状态机 + TTL 清理 + 线程安全）
- 声明 `io.modelcontextprotocol/tasks` 扩展 capability
- `tasks_get` 工具：按 task_id 查询异步任务状态
- `tasks_update` 工具：input_required 状态下提交 resume/retry 决策
- `tasks_cancel` 工具：取消运行中任务
- `test_and_report` 改为异步：触发即返回 task_id + 后台 goroutine 轮询
- 33 个单元测试覆盖所有模块和边界场景
- Race detector 全通过（0 竞态）
- `make test` 全通过

**关键发现：**
1. MCP Go SDK v1.4.1 无原生 Tasks 扩展支持，TaskManager 需从零实现
2. 线程安全的 TaskStore 使用 `sync.RWMutex` + map，`GetTask` 需返回副本而非内部指针以防止读写竞态
3. 后台 goroutine 必须使用 `context.Background()` 而非 handler 传入的 context（handler 返回后 context 会被取消）
4. 取消信号通过 channel 传递（`GetCancelCh` 返回只读 channel），resume 通过带缓冲 channel 传递
5. `test_and_report` 的测试从同步改为异步后，需添加 `waitForTask` 辅助函数轮询 TaskManager 等待后台完成
6. 使用 worktree 隔离子代理时，要注意主工作目录和工作树的区分（build 命令的路径解析）

### 2.2 顺手修复（原方向 L + H）✅ 已完成

**验收结果：**
- L — 输入验证错误规范化：handler.go 中 JSON 参数解析和路径参数类型验证失败返回 `&jsonrpc.Error{Code: -32000}` 协议级错误
- H — 扩展协商声明：ServerOptions.Capabilities 显式声明 logging + `io.modelcontextprotocol/streamable-http` 扩展
- 单元测试全部通过（race detector 无竞态）
- 验收测试确认 initialize 响应中包含 `extensions.io.modelcontextprotocol/streamable-http: {}`
- 提交记录：070f4b1（H）、b9c1fc2（L）

**关键发现（供后续参考）：**
1. MCP SDK 的 `mcp.Server` 不暴露 Capabilities 的读取方法，只能在构造时通过 ServerOptions.Capabilities 设置
2. 设置 Capabilities 为非 nil 会覆盖 SDK 默认的 logging 能力，需显式保留 `Logging: &mcp.LoggingCapabilities{}`
3. 扩展声明（streamable-http/tasks）仅是一个声明，不依赖实际传输模式——无论 stdio/HTTP 都可以声明
4. ToolHandler 返回 `(nil, &jsonrpc.Error{Code: -32000, ...})` 会在 JSON-RPC 层面产生协议错误，适合验证类错误
5. 其他运行时错误（上游错误、超时等）保持 IsError: true 内容级错误，SDK 文档明确建议这样

---

## 3. 按需触发

以下方向**不做预实施**，仅在满足触发条件时再做：

| 方向 | 触发条件 |
|------|---------|
| **A: Resources** | 开始使用 GUI 客户端（Claude Desktop 等），发现缺少资源浏览影响体验 |
| **E: Tool Icons** | 同上，GUI 客户端中工具列表太难看 |
| **D: JSON Schema 2020-12** | 遇到兼容性报错或工具校验失败 |
| **G: Registry** | 决定公开此项目 |

**其余方向（B, F, I, J, K, M, N）在本策略下不执行，** 除非需求发生根本变化（如：决定企业部署、决定开源推广）。

---

## 4. 与原路线图的关系

| 原路线图 | 本计划 |
|---------|--------|
| 14 个方向 A-N 的完整调研 | 精选出对自用真正有用的 2 项 |
| 通用投入/产出比评估 | 基于自用场景的实战优先级 |
| 4 阶段规划 | 两级分类：立即执行 / 按需触发 |
| 完整保留不动 | 轻量执行指南 |

---

## 5. 更新记录

| 日期 | 版本 | 变更 |
|------|------|------|
| 2026-07-06 | v3.0 | 阶段二（Tasks 扩展集成 C）完成并验收通过。所有代码已实现、测试通过、race detector 无竞态 |
| 2026-07-06 | v2.0 | 阶段一（L+H 顺手修复）完成并验收通过。下一阶段：Tasks 扩展集成（方向 C）|
| 2026-07-06 | v1.0 | 初版，基于自用优先策略 |
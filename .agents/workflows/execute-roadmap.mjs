export const meta = {
  name: 'execute-roadmap',
  description: 'Execute new-api-mcp-server roadmap phase by phase: fixes (Phase 0) then high-level tools (Phase 1)',
  phases: [
    { title: 'Phase 0: 紧急修复', detail: 'New-Api-User, README, OpenAPI, .mcp.json' },
    { title: 'Phase 1: 高层工具', detail: '渠道管理 5 + 用户管理 5 (spec-first, 强制调 Skill)' },
    { title: 'Verify', detail: '编译 + 全部测试 + lint 检查' },
  ],
}

// ─── Phase 0: 紧急修复 ───────────────────────────────────────
phase('Phase 0: 紧急修复')
log('=== 阶段 0：紧急修复 开始 ===')

// 0.1 最优先
const fixNewApiUser = await agent(
  `你正在执行 new-api-mcp-server 路线图的阶段 0.1：添加 New-Api-User 请求头。

## 背景
New API 的管理 API 要求所有请求携带 "New-Api-User" 请求头。当前 MCP 服务器只发送了 Authorization 头，导致所有管理 API 调用返回 401。

## 需要修改的文件
1. internal/config/config.go — 新增 UserID 字段，从 MCP_USER_ID 环境变量读取（默认 "1"）
2. internal/client/client.go — 新增 userID 字段，在 New() 中接收；Do() 对 SourceAPI 类型自动设置 New-Api-User 头
3. cmd/server/main.go — 创建客户端时传入 cfg.UserID
4. internal/client/client_test.go — 新增 TestClient_Do_NewApiUserHeader，验证 SourceAPI 请求携带 New-Api-User 头

## 测试要求（验收三问）
- 用户可见行为：SourceAPI 请求中应包含 "New-Api-User: 1" 头
- 生效证明：先写测试（它应该失败），再实现逻辑（测试通过），删除注入代码后测试失败
- 破坏测试：MCP_USER_ID 未设置时使用默认值 "1"

## 执行步骤
1. 先读 client_test.go 了解现有测试模式
2. 在 client_test.go 追加 TestClient_Do_NewApiUserHeader
3. 依次修改 config.go → client.go → main.go
4. 运行 go test ./internal/client/ -v -run TestClient_Do_NewApiUserHeader 确认通过
5. 运行 go build ./... 确认编译通过`,
  { agentType: 'general-purpose', label: '0.1 New-Api-User 头', phase: 'Phase 0: 紧急修复' }
)

// 0.2, 0.3, 0.4 可并行
const [fixReadme, fixOpenapi, fixMcpJson] = await parallel([
  () => agent(
    `修复 README.md 中的环境变量名错误。

    当前 README 写的是 MCP_RELAY_DISABLED_GROUPS，但 config.go 实际读取的是 MCP_RELAY_ENABLED_GROUPS。
    修改 README.md 中的变量名，并更新描述以匹配白名单模式（列出允许的组）。
    先读 README.md 找到对应行，确认后再修改。`,
    { agentType: 'general-purpose', label: '0.2 README 修复', phase: 'Phase 0: 紧急修复' }
  ),
  () => agent(
    `更新 openapi/api.json 和 openapi/relay.json 为最新版本。

    当前规格文件来自旧版本，缺少 New API v1.0.0-rc.15 的部分端点。
    从 https://github.com/QuantumNous/new-api 获取最新 OpenAPI 规格文件，替换两个文件。
    运行 go test ./internal/openapi/ -v -run TestParse_Real 确认工具数量增加。
    不要修改 openapi/embed.go。`,
    { agentType: 'general-purpose', label: '0.3 OpenAPI 规格更新', phase: 'Phase 0: 紧急修复' }
  ),
  () => agent(
    `检查并修复 .mcp.json 配置。先读 .mcp.json 确认当前配置是否正确，确保与 README.md 示例一致。`,
    { agentType: 'general-purpose', label: '0.4 .mcp.json 配置', phase: 'Phase 0: 紧急修复' }
  ),
])

// Phase 0 验证
const verifyPhase0 = await agent(
  `验证阶段 0 的改动全部正确。

  1. go build ./... 编译通过
  2. go test ./... -v -race -count=1 测试全部通过
  3. go vet ./... 无静态分析问题
  4. 检查 TestClient_Do_NewApiUserHeader 测试通过
  5. 检查 TestParse_RealAPISpec 和 TestParse_RealRelaySpec 通过

  报告结果：编译状态、测试通过数量、新增工具数量。`,
  { agentType: 'general-purpose', label: 'Phase 0 验证', phase: 'Phase 0: 紧急修复' }
)

log('=== 阶段 0 完成 ===')

// ─── Phase 1: 高层工具 ───────────────────────────────────────
phase('Phase 1: 高层工具')
log('=== 阶段 1：高层工具 开始 ===')

// 1.0 先创建包结构
const createHightoolsPkg = await agent(
  `创建 internal/hightools/ 包，作为高层工具的容器。

  定义：
  1. ToolDef 结构体 — 描述一个高层工具（名称、描述、输入Schema、Handler 函数）
  2. RegisterAll() 函数 — 返回所有高层工具的列表，供 main.go 注册
  3. 每个工具对应一个独立的构造函数

  先读 internal/handler/handler.go 和 internal/registry/registry.go 了解现有模式。`,
  { agentType: 'general-purpose', label: '1.0 创建 hightools 包', phase: 'Phase 1: 高层工具' }
)

// 渠道管理 5 个 spec-first
const channelTools = await parallel([
  () => agent(
    `你现在要实现的工具是 list_providers（列出所有渠道，按分组展示）。

    ╔══════════════════════════════════════════════════════════╗
    ║  第 1 步：调用 Skill tool（强制，不可跳过）              ║
    ║  参数：{skill: "trae-spec", args: "为 list_providers    ║
    ║         创建 spec，change-id: list-providers，           ║
    ║         路径: .agents/specs/list-providers/"}           ║
    ║  第 2 步：Skill 成功后按指导写三个文件，失败则手动写      ║
    ║  第 3 步：trae-spec 要求委派子代理实现，不要自己写代码   ║
    ╚══════════════════════════════════════════════════════════╝

    工具参考：
    - API: GET /api/channel/
    - 输出: 按分组展示渠道（ID、名称、状态、模型、优先级）
    - 测试: httptest.Server mock 上游，验证分组信息`,
    { agentType: 'general-purpose', label: '1.1.1 list_providers', phase: 'Phase 1: 高层工具' }
  ),
  () => agent(
    `你现在要实现的工具是 test_and_report（测试所有渠道，返回健康摘要）。

    ╔══════════════════════════════════════════════════════════╗
    ║  第 1 步：调用 Skill tool（强制，不可跳过）              ║
    ║  参数：{skill: "trae-spec", args: "为 test_and_report  ║
    ║         创建 spec，change-id: test-and-report，          ║
    ║         路径: .agents/specs/test-and-report/"}          ║
    ║  第 2 步：Skill 成功后按指导写三个文件，失败则手动写      ║
    ║  第 3 步：trae-spec 要求委派子代理实现，不要自己写代码   ║
    ╚══════════════════════════════════════════════════════════╝

    工具参考：
    - API: GET /api/channel/test
    - 输出: 每个渠道的测试结果（通过/失败、延迟、错误信息）
    - 破坏测试: 某渠道不可用时报告有错误标志而非 panic`,
    { agentType: 'general-purpose', label: '1.1.2 test_and_report', phase: 'Phase 1: 高层工具' }
  ),
  () => agent(
    `你现在要实现的工具是 toggle_channel（启用或禁用指定渠道）。

    ╔══════════════════════════════════════════════════════════╗
    ║  第 1 步：调用 Skill tool（强制，不可跳过）              ║
    ║  参数：{skill: "trae-spec", args: "为 toggle_channel   ║
    ║         创建 spec，change-id: toggle-channel，           ║
    ║         路径: .agents/specs/toggle-channel/"}           ║
    ║  第 2 步：Skill 成功后按指导写三个文件，失败则手动写      ║
    ║  第 3 步：trae-spec 要求委派子代理实现，不要自己写代码   ║
    ╚══════════════════════════════════════════════════════════╝

    工具参考：
    - API: PUT /api/channel/{id}
    - 参数: id（必填）、enabled（必填）
    - 生效证明: 测试验证渠道状态实际变更`,
    { agentType: 'general-purpose', label: '1.1.3 toggle_channel', phase: 'Phase 1: 高层工具' }
  ),
  () => agent(
    `你现在要实现的工具是 set_channel_priority（设置渠道优先级）。

    ╔══════════════════════════════════════════════════════════╗
    ║  第 1 步：调用 Skill tool（强制，不可跳过）              ║
    ║  参数：{skill: "trae-spec", args: "为                  ║
    ║         set_channel_priority 创建 spec，                 ║
    ║         change-id: set-channel-priority，                ║
    ║         路径: .agents/specs/set-channel-priority/"}     ║
    ║  第 2 步：Skill 成功后按指导写三个文件，失败则手动写      ║
    ║  第 3 步：trae-spec 要求委派子代理实现，不要自己写代码   ║
    ╚══════════════════════════════════════════════════════════╝

    工具参考：
    - API: PUT /api/channel/{id}
    - 参数: id（必填）、priority（必填，整数）
    - 破坏测试: 非整数优先级值被拒绝`,
    { agentType: 'general-purpose', label: '1.1.4 set_channel_priority', phase: 'Phase 1: 高层工具' }
  ),
  () => agent(
    `你现在要实现的工具是 add_channel（添加渠道）。

    ╔══════════════════════════════════════════════════════════╗
    ║  第 1 步：调用 Skill tool（强制，不可跳过）              ║
    ║  参数：{skill: "trae-spec", args: "为 add_channel      ║
    ║         创建 spec，change-id: add-channel，              ║
    ║         路径: .agents/specs/add-channel/"}              ║
    ║  第 2 步：Skill 成功后按指导写三个文件，失败则手动写      ║
    ║  第 3 步：trae-spec 要求委派子代理实现，不要自己写代码   ║
    ╚══════════════════════════════════════════════════════════╝

    工具参考：
    - API: POST /api/channel/
    - 参数: name（必填）、type（必填）、models、group、priority
    - 集成测试: 创建渠道后列表中出现新渠道`,
    { agentType: 'general-purpose', label: '1.1.5 add_channel', phase: 'Phase 1: 高层工具' }
  ),
])

// 用户管理 5 个 spec-first
const userTools = await parallel([
  () => agent(
    `你现在要实现的工具是 list_users（列出所有用户）。

    ╔══════════════════════════════════════════════════════════╗
    ║  第 1 步：调用 Skill tool（强制，不可跳过）              ║
    ║  参数：{skill: "trae-spec", args: "为 list_users       ║
    ║         创建 spec，change-id: list-users，               ║
    ║         路径: .agents/specs/list-users/"}               ║
    ║  第 2 步：Skill 成功后按指导写三个文件，失败则手动写      ║
    ║  第 3 步：trae-spec 要求委派子代理实现，不要自己写代码   ║
    ╚══════════════════════════════════════════════════════════╝

    工具参考：
    - API: GET /api/user/
    - 输出: 每个用户显示 ID、用户名、角色、状态、配额
    - 集成测试: 返回用户列表`,
    { agentType: 'general-purpose', label: '1.2.1 list_users', phase: 'Phase 1: 高层工具' }
  ),
  () => agent(
    `你现在要实现的工具是 show_balance（显示余额概览）。

    ╔══════════════════════════════════════════════════════════╗
    ║  第 1 步：调用 Skill tool（强制，不可跳过）              ║
    ║  参数：{skill: "trae-spec", args: "为 show_balance     ║
    ║         创建 spec，change-id: show-balance，             ║
    ║         路径: .agents/specs/show-balance/"}             ║
    ║  第 2 步：Skill 成功后按指导写三个文件，失败则手动写      ║
    ║  第 3 步：trae-spec 要求委派子代理实现，不要自己写代码   ║
    ╚══════════════════════════════════════════════════════════╝

    工具参考：
    - API: 1. GET /api/channel/ 2. GET /api/channel/update_balance
    - 输出: 按渠道展示剩余额度、已用额度、过期时间
    - mock 两个 API 端点验证组合调用正确`,
    { agentType: 'general-purpose', label: '1.2.2 show_balance', phase: 'Phase 1: 高层工具' }
  ),
  () => agent(
    `你现在要实现的工具是 set_user_quota（设置用户配额）。

    ╔══════════════════════════════════════════════════════════╗
    ║  第 1 步：调用 Skill tool（强制，不可跳过）              ║
    ║  参数：{skill: "trae-spec", args: "为 set_user_quota   ║
    ║         创建 spec，change-id: set-user-quota，           ║
    ║         路径: .agents/specs/set-user-quota/"}           ║
    ║  第 2 步：Skill 成功后按指导写三个文件，失败则手动写      ║
    ║  第 3 步：trae-spec 要求委派子代理实现，不要自己写代码   ║
    ╚══════════════════════════════════════════════════════════╝

    工具参考：
    - API: PUT /api/user/{id}
    - 参数: id（必填）、quota（必填，数字）
    - 破坏测试: 无效配额值被拒绝`,
    { agentType: 'general-purpose', label: '1.2.3 set_user_quota', phase: 'Phase 1: 高层工具' }
  ),
  () => agent(
    `你现在要实现的工具是 switch_group（切换 Token 分组）。

    ╔══════════════════════════════════════════════════════════╗
    ║  第 1 步：调用 Skill tool（强制，不可跳过）              ║
    ║  参数：{skill: "trae-spec", args: "为 switch_group     ║
    ║         创建 spec，change-id: switch-group，             ║
    ║         路径: .agents/specs/switch-group/"}             ║
    ║  第 2 步：Skill 成功后按指导写三个文件，失败则手动写      ║
    ║  第 3 步：trae-spec 要求委派子代理实现，不要自己写代码   ║
    ╚══════════════════════════════════════════════════════════╝

    工具参考：
    - API: 1. GET /api/token/{id} 2. PUT /api/token/
    - 参数: token_id（必填）、group（必填）
    - 参考官方 api.js 的 switch-group 实现逻辑`,
    { agentType: 'general-purpose', label: '1.2.4 switch_group', phase: 'Phase 1: 高层工具' }
  ),
  () => agent(
    `你现在要实现的工具是 toggle_user_status（启用/禁用用户）。

    ╔══════════════════════════════════════════════════════════╗
    ║  第 1 步：调用 Skill tool（强制，不可跳过）              ║
    ║  参数：{skill: "trae-spec", args: "为                  ║
    ║         toggle_user_status 创建 spec，                   ║
    ║         change-id: toggle-user-status，                  ║
    ║         路径: .agents/specs/toggle-user-status/"}       ║
    ║  第 2 步：Skill 成功后按指导写三个文件，失败则手动写      ║
    ║  第 3 步：trae-spec 要求委派子代理实现，不要自己写代码   ║
    ╚══════════════════════════════════════════════════════════╝

    工具参考：
    - API: PUT /api/user/{id}
    - 参数: id（必填）、enabled（必填）
    - 生效证明: 禁用用户后该用户状态已更改`,
    { agentType: 'general-purpose', label: '1.2.5 toggle_user_status', phase: 'Phase 1: 高层工具' }
  ),
])

// 1.3 注册到主入口 + 集成测试
const registerAndTest = await agent(
  `将 internal/hightools 包中的高层工具注册到 cmd/server/main.go，并添加集成测试。

  1. 读 cmd/server/main.go 了解现有工具注册方式
  2. 在 main.go 中管理 API 工具注册之后调用 hightools.RegisterAll() 注册高层工具
  3. 在 integration_test.go 中添加高层工具的集成测试：
     - 创建 mock upstream
     - 解析 OpenAPI spec 注册基础工具
     - 注册高层工具
     - 使用 in-memory transport 连接 MCP 客户端
     - 调用 ListTools 确认高层工具出现在列表中
     - 至少调用一个高层工具验证响应

  验证：go build ./... && go test ./... -v -race -count=1 && go test -tags=integration -v -run TestIntegration`,
  { agentType: 'general-purpose', label: '1.3 注册 + 集成测试', phase: 'Phase 1: 高层工具' }
)

// Phase 1 验证
const verifyPhase1 = await agent(
  `验证阶段 1 的改动全部正确。

  1. go build ./... 编译通过
  2. go test ./... -v -race -count=1 测试全部通过
  3. go vet ./... 无静态分析问题
  4. go test -tags=integration -v -run TestIntegration 集成测试通过
  5. 检查所有 10 个高层工具是否都已注册

  报告结果：编译状态、测试通过数量、高层工具列表。`,
  { agentType: 'general-purpose', label: 'Phase 1 验证', phase: 'Phase 1: 高层工具' }
)

log('=== 阶段 1 完成 ===')
log('=== 阶段 2-6 的 Workflow 扩展将在后续更新 ===')

return {
  phase0: {
    newApiUser: fixNewApiUser,
    readme: fixReadme,
    openapi: fixOpenapi,
    mcpJson: fixMcpJson,
    verification: verifyPhase0,
  },
  phase1: {
    hightoolsPkg: createHightoolsPkg,
    channelTools: channelTools,
    userTools: userTools,
    registration: registerAndTest,
    verification: verifyPhase1,
  },
}
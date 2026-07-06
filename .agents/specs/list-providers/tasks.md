# Tasks

- [x] Task 1: 实现 list_providers 工具的 handler 逻辑
  - 在 `internal/hightools/list_providers.go` 中实现 `NewListProvidersTool(client)` 构造函数
  - 调用 `GET /api/channel/` 获取所有渠道
  - 解析 JSON 响应为 Channel 结构体数组
  - 按 `groups` 字段分组
  - 按 `priority` 降序排列
  - 格式化为 markdown 表格输出
  - 支持可选的 `group` 和 `status` 筛选参数
- [x] Task 2: 注册 list_providers 工具到服务器
  - 修改 `internal/hightools/register.go`，在 `RegisterAll()` 中加入 `NewListProvidersTool(client)`
  - 修改 `cmd/server/main.go`，在 API 工具注册后调用 `hightools.RegisterAll()` 并注册返回的工具列表
  - 仅在 `cfg.SystemKey != "" && cfg.APIToolsEnabled` 时注册
- [x] Task 3: 编写 list_providers 的测试
  - 在 `internal/hightools/list_providers_test.go` 中编写测试
  - 使用 `httptest.Server` mock 上游 GET /api/channel/ 响应
  - 验证正常场景：多渠道、多分组
  - 验证空列表场景
  - 验证上游错误场景
  - 验证筛选参数（group、status）过滤
  - 验证分组排序（按 priority 降序）

# Task Dependencies

- Task 2 依赖 Task 1（需要先有工具实现才能注册）
- Task 3 可与 Task 1 并行（测试可以先写预期行为）
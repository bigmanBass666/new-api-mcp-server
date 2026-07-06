# Tasks

- [x] Task 1: 实现 `set_user_quota` 工具
  - 创建 `internal/hightools/set_user_quota.go`，遵循 `set_channel_priority.go` 的代码模式
  - 构造函数 `NewSetUserQuotaTool(c *client.Client, metrics *observability.Metrics) ToolDef`
  - 参数：`id`（必填，整数）、`quota`（必填，整数）
  - 验证：`id` 必须为整数，`quota` 必须为非负整数（拒绝负数、小数、字符串）
  - 调用 `PUT /api/user/{id}`，body 为 `{"quota": quota}`
  - 使用 `c.Do(ctx, client.SourceAPI, "PUT", path, nil, nil, bodyBytes)` 发送请求
  - 注册 metrics 和 logging 与 `set_channel_priority` 一致
  - 在 `register.go` 的 `RegisterAll` 中添加 `NewSetUserQuotaTool(c, metrics)`

- [x] Task 2: 编写 `set_user_quota` 测试
  - 创建 `internal/hightools/set_user_quota_test.go`
  - 测试用例包括：
    - 成功设置配额
    - 缺少 id 参数
    - 缺少 quota 参数
    - 非整数 id（字符串）
    - 负数的 quota
    - 非整数的 quota（小数）
    - 非数字的 quota（字符串）
    - 上游错误（500）

## Task Dependencies

- [Task 2] depends on [Task 1]
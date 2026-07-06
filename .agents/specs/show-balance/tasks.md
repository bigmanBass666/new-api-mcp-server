# Tasks

- [x] Task 1: 实现 show_balance 工具
  - [x] SubTask 1.1: 在 `internal/hightools/show_balance.go` 中定义 ChannelWithBalance 结构体（包含 balance、used_quota、balance_updated_time 字段）
  - [x] SubTask 1.2: 定义 API 响应 wrapper 结构体（listChannelsResponse / listData）
  - [x] SubTask 1.3: 实现 `NewShowBalanceTool()` 构造函数，返回 ToolDef
  - [x] SubTask 1.4: 实现 handler 逻辑：依次调用两个上游 API，组合输出结构化的 Markdown 表格
  - [x] SubTask 1.5: 处理边缘情况：刷新失败可容忍、渠道列表为空、响应解析错误

- [x] Task 2: 注册 show_balance 工具
  - [x] SubTask 2.1: 在 `internal/hightools/register.go` 的 `RegisterAll()` 中添加 `NewShowBalanceTool(c, metrics)`

- [x] Task 3: 编写单元测试
  - [x] SubTask 3.1: 在 `internal/hightools/show_balance_test.go` 中实现 Mock 双端点的测试用例
  - [x] SubTask 3.2: 测试正常流程（两个 API 都成功返回、输出包含余额列）
  - [x] SubTask 3.3: 测试刷新余额失败但渠道列表成功（不标记 IsError、包含警告信息）
  - [x] SubTask 3.4: 测试渠道列表为空
  - [x] SubTask 3.5: 测试渠道列表返回错误（IsError=true）
  - [x] SubTask 3.6: 测试指定 channel_id 参数（调用 /api/channel/update_balance/{id}）
  - [x] SubTask 3.7: 测试 balance_updated_time=0 时显示 "(从未更新)"

# Task Dependencies

- Task 3 依赖 Task 1
- Task 2 依赖 Task 1
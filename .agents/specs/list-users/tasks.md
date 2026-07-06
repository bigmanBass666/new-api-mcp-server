# Tasks

- [x] Task 1: Implement list_users tool (`internal/hightools/list_users.go`)
  - 根据上游 `GET /api/user/` 响应格式定义 User 结构体
  - 实现 `NewListUsersTool(c, metrics) ToolDef` 构造函数
  - 实现 handler 逻辑：调用上游 API、解析分页响应、格式化为 Markdown 表格
  - 处理空列表、错误响应、解析失败等异常情况
  - 记录 Prometheus 指标和 slog 日志

- [x] Task 2: Register list_users in `internal/hightools/register.go`
  - 在 RegisterAll() 返回的切片中添加 `NewListUsersTool(c, metrics)`

- [x] Task 3: Write unit tests (`internal/hightools/list_users_test.go`)
  - 使用 `httptest.Server` mock 上游 GET /api/user/ 接口
  - 测试正常返回：多个用户、不同角色/状态
  - 测试空列表
  - 测试上游返回 success=false
  - 测试非 200 响应

## Task Dependencies

- [Task 2] depends on [Task 1]
- [Task 3] depends on [Task 1]
# Tasks

- [x] Task 1: 创建 `add_channel_test.go`
  - 遵循 `set_channel_priority_test.go` 的模式
  - 覆盖：成功创建（含可选参数）、缺失 name、缺失 type、缺失 key、type 类型错误、priority 浮点数、上游 500 错误
  - 验证请求方法为 `POST`，路径为 `/api/channel/`
  - 验证请求体包含 `mode` 和 `channel` 包装

- [x] Task 2: 创建 `toggle_channel_test.go`
  - 覆盖：启用（status=1）、禁用（status=2）、缺失 id、缺失 enabled、id 类型错误、上游 500 错误
  - 验证请求方法为 `POST`，路径包含 `/api/channel/{id}/status`

- [x] Task 3: 创建 `toggle_user_status_test.go`
  - 覆盖：启用（action=enable）、禁用（action=disable）、缺失 id、缺失 enabled、id 类型错误、上游 500 错误
  - 验证请求方法为 `POST`，路径为 `/api/user/manage`
  - 验证请求体包含 `id` 和 `action` 字段

- [x] Task 4: 创建 `switch_group_test.go`
  - 覆盖：成功切换、缺失 token_id、缺失 group、token_id 类型错误、group 为空字符串、上游 500 错误
  - 验证请求方法为 `PUT`，路径为 `/api/token/`
  - 验证请求体包含 `id` 和 `group` 字段

- [x] Task 5: 验证全部测试通过
  - `go test ./internal/hightools/ -v -count=1` 全部通过
  - `go build ./cmd/server` 编译无错误

# Task Dependencies

- 所有 Task 1-4 可并行执行（无依赖关系）
- Task 5 依赖 Task 1-4 全部完成
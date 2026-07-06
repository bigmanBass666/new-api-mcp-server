# Checklist

- [x] `internal/hightools/set_user_quota.go` 文件存在且包含正确的工具定义
- [x] `set_user_quota` 在 `register.go` 中注册
- [x] 参数 `id` 和 `quota` 均为必填
- [x] 非整数 `id` 被拒绝并返回 IsError
- [x] 负数 `quota` 被拒绝并返回 IsError
- [x] 非整数 `quota`（小数）被拒绝并返回 IsError
- [x] 字符串 `quota` 被拒绝并返回 IsError
- [x] 有效调用发送 `PUT /api/user/{id}` 到上游
- [x] 正确记录 metrics（ToolRequestsTotal、ToolRequestDuration、UpstreamRequestsTotal、UpstreamRequestDuration）
- [x] 所有测试通过（`go test ./internal/hightools/ -v -run TestSetUserQuota`）
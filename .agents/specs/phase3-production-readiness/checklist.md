# 检查清单

- [x] `internal/middleware/ratelimit.go` 存在，包含令牌桶速率限制实现
- [x] `internal/middleware/ratelimit_test.go` 存在，覆盖正常请求、超限拒绝、不限速跳过
- [x] `internal/config/config.go` 包含 BaseURL 格式校验、Transport 值校验、Timeout 最小值校验
- [x] `internal/config/config_test.go` 扩展，覆盖无效 URL、无效传输模式、超时过低
- [x] `cmd/server/main.go` 包含 `/healthz` 和 `/readyz` 端点，HTTP 200 和 503 正确返回
- [x] `cmd/server/main.go` 包含速率限制中间件集成（RPS=0 时跳过）
- [x] `cmd/server/main.go` 包含可配置的优雅关闭超时
- [x] `cmd/server/main_test.go` 存在，覆盖健康检查 HTTP 响应和优雅关闭信号处理
- [x] `go test ./... -v -race -count=1` 全部通过
- [x] `go build ./cmd/server` 编译无错误
- [x] `go vet ./...` 无警告
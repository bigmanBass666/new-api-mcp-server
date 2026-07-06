# Tasks

- [x] Task 1: 新增 `golang.org/x/time/rate` 依赖
  - 运行 `go get golang.org/x/time/rate` 添加依赖
  - 验证：`go mod tidy` 无错误

- [x] Task 2: 速率限制中间件
  - 创建 `internal/middleware/ratelimit.go`
  - 实现 `NewRateLimiter(rps, burst int) func(http.Handler) http.Handler`
  - 支持 `rps=0` 时跳过限速（不创建令牌桶）
  - 创建 `internal/middleware/ratelimit_test.go`
  - 测试：正常请求通过、超限被拒绝 429、不限速跳过
  - 验证：`go test ./internal/middleware/ -v -count=1`

- [x] Task 3: 配置校验增强
  - 修改 `internal/config/config.go`：在 `Load()` 中添加校验逻辑
  - 校验项：BaseURL 格式（URL Parse）、Transport 值（stdio|http）、Timeout 最小值（>1s）
  - 扩展 `internal/config/config_test.go`：添加无效 URL、无效传输模式、超时过低测试
  - 验证：`go test ./internal/config/ -v -count=1`

- [x] Task 4: 健康检查端点
  - 修改 `cmd/server/main.go`：在 HTTP 传输中添加 `/healthz` 和 `/readyz` 端点
  - `/healthz`：返回 200 + `{"status":"ok"}`
  - `/readyz`：HEAD 请求上游 BaseURL，可达返回 200，不可达返回 503
  - 创建 `cmd/server/main_test.go`：HTTP 集成测试验证健康检查端点
  - 验证：`go test ./cmd/server/ -v -count=1`

- [x] Task 5: 优雅关闭增强
  - 修改 `cmd/server/main.go`：添加可配置的关闭超时（`MCP_SHUTDOWN_TIMEOUT`，默认 15s）
  - 增强 HTTP 传输的关闭逻辑：启动时创建 `http.Server` 的 `shutdownCtx`，SIGTERM 时调用 `shutdown()`
  - 修改 `internal/config/config.go`：添加 `ShutdownTimeout` 字段
  - 验证：`go test ./cmd/server/ -v -count=1`

- [x] Task 6: 速率限制集成到 HTTP 传输
  - 修改 `cmd/server/main.go`：在 HTTP handler 链中插入速率限制中间件
  - 在 `config.Load()` 中读取 `MCP_RATE_LIMIT_RPS`（默认 0=不限速）和 `MCP_RATE_LIMIT_BURST`（默认 0）
  - 验证：`go build ./cmd/server` 编译通过

- [x] Task 7: 最终验证
  - 全量测试：`go test ./... -v -race -count=1`
  - 编译检查：`go build ./cmd/server`
  - 静态分析：`go vet ./...`

# Task Dependencies

- Task 1 → Task 2（Task 2 需要 x/time 依赖）
- Task 2 → Task 6（Task 6 集成速率限制到 HTTP 传输）
- Task 3 → Task 4（Task 4 需要配置校验先完成以确认配置正确）
- Task 4 → Task 5（Task 5 在健康检查基础上增强关闭逻辑）
- Task 5 → Task 6（Task 6 在 HTTP 传输上叠加速率限制）
- Task 6 → Task 7（最终验证依赖所有任务完成）
- Task 1, 3 可并行
- Task 2, 4 可并行（依赖 Task 1 和 Task 3）
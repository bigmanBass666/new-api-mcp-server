# Phase 2 安全加固 — 实施计划

## 摘要

对 new-api-mcp-server 进行安全加固，包括结构化错误码体系、路径参数注入防护、CORS 配置和请求大小限制。基于 Phase 0+1 完成后的代码状态（编译零错误，60+ 测试全部通过）。

---

## 当前状态分析

### 代码库状态（Phase 1 完成后）

- 编译：`go build ./...` 零错误
- 测试：`go test ./... -v -race -count=1` 全部通过（8 个包，60+ 测试用例）
- 新增：10 个高层 MCP 工具（`internal/hightools/`），4133 行代码
- 共 195 个工具定义（157 api + 38 relay）

### 已识别的安全缺口

| # | 问题 | 位置 | 风险等级 |
|---|------|------|---------|
| 1 | 路径参数使用 `fmt.Sprintf("%v", v)` 直接替换，无校验和编码 | `handler.go:56-58` | **高** — 路径遍历注入 |
| 2 | 错误信息纯文本返回，无结构化错误码 | `handler.go:160-167` | **中** — 客户端无法编程式解析 |
| 3 | HTTP 模式无 CORS 头 | `main.go`（HTTP 传输） | **中** — 浏览器端 MCP 客户端无法访问 |
| 4 | HTTP 模式无请求体大小限制 | `main.go`（HTTP 传输） | **中** — 潜在滥用 |

---

## 变更计划

### 2.1 路径参数注入防护（`internal/handler/handler.go`）

**What：** 对路径参数做 URL 编码和类型校验。

**Why：** 当前直接 `fmt.Sprintf("%v", v)` 替换，`../` 或 `%2e%2e%2f` 等路径遍历字符可被透传给上游。

**How：**
1. 新增 `url` 和 `strconv` import
2. 在 `MakeHandler` 的路径替换循环中，对每个路径参数值：
   - 如果 `p.Schema` 有 `type: "integer"`，用 `strconv.ParseInt` 校验
   - 调用 `url.PathEscape(value)` 编码后再替换
3. 创建 `sanitizePathParam(name, value string, schema map[string]any) (string, error)` 辅助函数

**涉及文件：** `internal/handler/handler.go`（修改，约 10 行新增）

### 2.2 结构化错误码（`internal/handler/errors.go` + `internal/handler/handler.go`）

**What：** 定义错误码枚举，返回 JSON 格式的结构化错误。

**Why：** 当前 `errorResult("msg")` 返回纯文本，MCP 客户端无法编程式解析错误类型。

**How：**
1. 新建 `internal/handler/errors.go`，定义：

```go
type ErrorCode string

const (
    ErrInvalidParams    ErrorCode = "INVALID_PARAMS"
    ErrUpstreamError    ErrorCode = "UPSTREAM_ERROR"
    ErrUpstreamTimeout  ErrorCode = "UPSTREAM_TIMEOUT"
    ErrUpstreamAuth     ErrorCode = "UPSTREAM_AUTH"
    ErrInternal         ErrorCode = "INTERNAL_ERROR"
    ErrUpstreamNotFound ErrorCode = "UPSTREAM_NOT_FOUND"
)

type ToolError struct {
    Code       ErrorCode `json:"code"`
    Message    string    `json:"message"`
    StatusCode int       `json:"status_code,omitempty"`
}
```

2. 将 `errorResult(msg string)` 改为 `errorResult(code ErrorCode, msg string) \*mcp.CallToolResult`，返回 JSON 序列化的 `ToolError`
3. 更新 `handler.go` 中所有 `errorResult()` 调用点（约 6 处），根据错误类型选择合适错误码
4. 错误码映射规则：
   - 上游 400 → `ErrInvalidParams`
   - 上游 401/403 → `ErrUpstreamAuth`
   - 上游 404 → `ErrUpstreamNotFound`
   - 上游 4xx/5xx → `ErrUpstreamError`
   - 上游超时 → `ErrUpstreamTimeout`
   - 参数校验失败 → `ErrInvalidParams`
   - 内部错误 → `ErrInternal`

**涉及文件：**
- 新增 `internal/handler/errors.go`（约 30 行）
- 修改 `internal/handler/handler.go`（约 10 行）
- 修改 `internal/handler/handler_test.go`（约 10 行，验证错误 JSON 包含 `code` 字段）

### 2.3 CORS 配置（`cmd/server/main.go` + `internal/config/config.go`）

**What：** HTTP 模式下添加可配置 CORS 支持。

**Why：** 浏览器端 MCP 客户端需要 CORS 头才能访问。

**How：**
1. `internal/config/config.go`：新增 `HTTPCORSOrigins string` 字段，从 `MCP_HTTP_CORS_ORIGINS` 读取（默认 `"*"`）
2. `cmd/server/main.go`：在 HTTP handler 链中包装 CORS 中间件，处理 OPTIONS 预检请求

**涉及文件：**
- `internal/config/config.go`（修改，约 2 行）
- `cmd/server/main.go`（修改，约 15 行）

### 2.4 请求大小限制（`cmd/server/main.go` + `internal/config/config.go`）

**What：** HTTP 模式下限制请求体大小。

**Why：** 防止滥用或恶意大请求消耗服务器资源。

**How：**
1. `internal/config/config.go`：新增 `HTTPMaxBodySize int64` 字段，从 `MCP_HTTP_MAX_BODY_SIZE` 读取（默认 `10485760` = 10MB）
2. `cmd/server/main.go`：在 HTTP handler 中包装 `http.MaxBytesReader`

**涉及文件：**
- `internal/config/config.go`（修改，约 2 行）
- `cmd/server/main.go`（修改，约 5 行）

---

## 假设与决策

| 决策 | 选择 | 理由 |
|------|------|------|
| 错误格式 | JSON 结构体 | 客户端可编程解析，兼容纯文本降级 |
| 路径参数编码 | `url.PathEscape` | 标准库函数，兼顾安全性和可读性 |
| CORS 默认值 | `*` | 最小阻力开发体验，生产环境可配置 |
| 请求体大小默认值 | 10MB | 足够容纳所有正常请求 |

---

## 验证步骤

```bash
# 1. 编译验证
go build ./...

# 2. 全部测试
go test ./... -v -race -count=1

# 3. 静态分析
go vet ./...

# 4. 手动验证错误格式（可选）
# 启动服务器，调用一个错误工具，确认返回 JSON 包含 "code" 字段
```

### 关键验收标准

1. 路径参数注入测试：`{"id": "../etc/passwd"}` 被拒绝而非透传
2. 错误码测试：`handler_test.go` 中新增测试验证 `result.IsError` 时，`result.Content[0].Text` 包含 `"code"` 字段
3. CORS 测试：`OPTIONS` 预检请求返回 `Access-Control-Allow-Origin` 头
4. 大小限制测试：超 10MB 请求体被拒绝

---

## 执行顺序

```
2.2 结构化错误码  ← 优先，因为修改 handler.go 的 errorResult 签名
  ↓
2.1 路径注入防护  ← 修改同一文件，先完成 errorResult 重构再改路径逻辑
  ↓
2.3 CORS        ← 独立，可并行
2.4 请求大小限制  ← 独立，可并行
  ↓
最终验证
```
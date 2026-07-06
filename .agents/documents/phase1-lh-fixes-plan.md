# 阶段一执行计划：顺手修复（L + H）

> **分支策略：** 直接在 `main` 分支提交（每项不到半天，符合项目规范）
> **任务跟踪：** 创建 Task 管理依赖关系
> **提交规范：** 原子提交 — 一个 commit 只做一件事

---

## 概览

| 任务 | 估算 | 文件改动数 | 测试文件 |
|------|------|-----------|---------|
| H: 扩展协商声明 | ~0.5 天 | 1 (+1 测试) | cmd/server/main_test.go |
| L: 输入验证错误规范化 | ~0.5 天 | 2 (+1 测试) | internal/handler/handler_test.go |
| **合计** | **~1 天** | **3-4 个文件** | **2 个测试文件** |

---

## 当前状态分析

### H — 扩展协商声明

**现状：**
- `cmd/server/main.go:74-80` 创建 MCP server 时未设置 `Capabilities`
- SDK 会从注册的功能中自动推断：`tools` 能力自动添加，`logging` 能力默认开启
- 未显式声明 `io.modelcontextprotocol/streamable-http` 扩展

**关键发现：**
- `mcp.ServerOptions.Capabilities` 支持显式设置，覆盖 SDK 默认行为
- `ServerCapabilities.Extensions` 字段支持 `"{vendor-prefix}/{extension-name}"` 格式的扩展声明
- Streamable HTTP handler 已使用 `mcp.NewStreamableHTTPHandler()` 创建，但 capabilities 中未声明扩展

### L — 输入验证错误规范化

**现状：**
- `internal/handler/errors.go`: 定义了 `ToolError` 类型和 `errorResult()` 函数，返回 `CallToolResult{IsError: true}` 内容级错误
- `internal/handler/handler.go:51-53`: JSON 参数解析失败返回 `errorResult(ErrInvalidParams, ...)`
- `internal/handler/handler.go:62-66`: 路径参数类型验证失败返回 `errorResult(ErrInvalidParams, ...)`
- `internal/hightools/test_and_report.go:302`: 使用 `errorResultTestAndReport()` 返回纯文本错误（无结构化格式）
- MCP SDK 文档 (`protocol.go:97-113`) 明确建议：
  - **工具运行时错误** → 用 `IsError: true`（LLM 能看到并自纠正）
  - **协议错误**（找不到工具、不支持调用）→ 用 JSON-RPC 协议错误

**发现的问题：**
1. `handler.go` 的参数验证错误使用 `errorResult()`，返回 `IsError=true` 的内容级错误 — 符合 SDK 建议，但需要统一格式
2. `errors.go` 的 `ToolError` 结构体序列化为 JSON 在 content 中，格式不够标准化
3. hightools 中的错误格式与 handler 不一致

---

## 具体变更

### H: 扩展协商声明

#### `cmd/server/main.go`

在 `mcp.NewServer()` 调用中设置 `Capabilities`：

```go
server := mcp.NewServer(&mcp.Implementation{
    Name:    "new-api-mcp-server",
    Version: version,
}, &mcp.ServerOptions{
    Instructions: "...",
    PageSize:     100,
    Capabilities: &mcp.ServerCapabilities{
        Logging: &mcp.LoggingCapabilities{},
        Tools:   &mcp.ToolCapabilities{ListChanged: true},
    },
})
```

然后在 tools 注册完成后（server.AddTool 之后），添加 Streamable HTTP 扩展声明：

```go
// 在所有工具注册后，添加扩展声明
if cfg.Transport == "http" {
    serverCap := server.ServerCapabilities()  // 需要在 Server 上添加此方法
}
```

等等 — 检查 SDK 发现目前 `mcp.Server` 没有公开的 `ServerCapabilities()` 方法。扩展声明需要在初始化时通过 `Capabilities.Extensions` 设置。

修改为：在创建 server 时就设置好所有 capabilities，包括 `Extensions`：

```go
caps := &mcp.ServerCapabilities{
    Logging: &mcp.LoggingCapabilities{},
    Tools:   &mcp.ToolCapabilities{ListChanged: true},
}
if cfg.Transport == "http" {
    caps.AddExtension("io.modelcontextprotocol/streamable-http", nil)
}
```

但这样有问题：创建 server 时还不知道 transport 类型。或者在 `run()` 中的创建点直接判断？

Actually，更简洁的方式：在创建 `server` 时统一设置 Capabilities，不依赖 transport。因为 Streamable HTTP 扩展只是声明协议支持能力，不影响 stdio 模式的行为。

**设计决策：** 始终声明 Streamable HTTP 扩展。即使 stdio 模式下，声明此扩展也不产生副作用。这样代码更简洁。

#### `cmd/server/main_test.go`

添加测试验证 capabilities 配置：

新增 `TestServerCapabilities_StreamableHTTP` — 验证扩展声明包含 `io.modelcontextprotocol/streamable-http`。测试方式：创建一个带 Capabilities 的 ServerOptions，检查扩展字段。

但因 `mcp.Server` 不暴露 Capabilities，改为间接验证：在 server 创建后，通过 `Server.capabilities()` 返回的内部信息验证。但这是私有方法。

替代方案：**文档级测试** — 启动 HTTP 模式 server，发送 `initialize` 请求，检查响应中的 capabilities。使用 `httptest.Server` + JSON-RPC 客户端验证。

但这样是 E2E 测试，需要 mock 所有依赖。过于复杂。

**简化方案：** 在 `internal/config/config.go` 中添加 `StreamableHTTP` 配置字段（或直接使用 `Transport`），在 `main.go` 中添加一个导出的 `DefaultCapabilities()` 函数，可独立测试。然后在 main_test.go 中直接测试这个函数返回的 Capabilities 是否包含正确字段。

```go
// 在 main.go 或新文件
func DefaultCapabilities(transport string) *mcp.ServerCapabilities {
    caps := &mcp.ServerCapabilities{
        Logging: &mcp.LoggingCapabilities{},
        Tools:   &mcp.ToolCapabilities{ListChanged: true},
    }
    caps.AddExtension("io.modelcontextprotocol/streamable-http", nil)
    return caps
}
```

**最终方案（更实用）：** 直接在 `main.go` 中设置 Capabilities，通过修改 `main_test.go` 使其包含一个对 `run()` 中 server 创建的直接验证。或者更简单：写一个测试，通过 `mcp.NewServer` 返回的 server 无法直接验证 capabilities，所以写一个单元测试验证 `CapabilitiesBuilder` 函数。

**最简方案：** 
1. 修改 `main.go`：在 `ServerOptions` 中设置 `Capabilities`
2. 在 `main_test.go` 中添加一个 `TestDefaultCapabilities` 测试函数，创建一个 local 函数来构建 capabilities 并验证其字段

让我再简化一点。不如直接：

1. 在 `main.go` 中添加 helper 函数 `newServerOptions()`，返回 `*mcp.ServerOptions`
2. 测试这个函数返回的配置

或者最简单的方式 — 在 `main.go` 中内联设置 Capabilities，在 `main_test.go` 中写一个测试：构建同样的 Capabilities 并验证 JSON 序列化结果。

OK，我决定用最实用的方式：

**在 `main.go` 中：** 直接在 server 创建时设置 Capabilities，在 HTTP transport 条件块中也设置扩展。

不对，我再想想。创建 server 的代码在 transport 判断之前。所以有两种选择：

方案 A：在 server 创建时就设置好扩展
```go
caps := &mcp.ServerCapabilities{
    Logging: &mcp.LoggingCapabilities{},
}
caps.AddExtension("io.modelcontextprotocol/streamable-http", nil)
server := mcp.NewServer(&mcp.Implementation{...}, &mcp.ServerOptions{
    Capabilities: caps,
    ...
})
```

方案 B：先创建不带扩展的 server，在 transport 快创建 http handler 前动态添加（但 Server 不提供事后修改 Capabilities 的方法）

方案 A 更好。扩展声明只是一个声明，不依赖实际传输模式。

**实现细节：**

```go
// main.go run() 函数中
caps := &mcp.ServerCapabilities{}
caps.AddExtension("io.modelcontextprotocol/streamable-http", nil)

server := mcp.NewServer(&mcp.Implementation{...}, &mcp.ServerOptions{
    Instructions: "...",
    PageSize:     100,
    Capabilities: caps,
})
```

注意：SDK 会从注册的工具自动添加 `tools` 能力，所以不需要在 `Capabilities` 中显式设置 Tools。但设置 `Capabilities` 为 `&mcp.ServerCapabilities{}` 会覆盖 SDK 默认的 `logging` 能力。

所以需要保留 logging：

```go
caps := &mcp.ServerCapabilities{
    Logging: &mcp.LoggingCapabilities{},
}
caps.AddExtension("io.modelcontextprotocol/streamable-http", nil)
```

这样 SDK 自动添加 tools 能力，我们显式添加 logging + streamable-http 扩展。

**测试：** 使用 JSON 序列化验证：

```go
func TestDefaultCapabilities(t *testing.T) {
    caps := &mcp.ServerCapabilities{
        Logging: &mcp.LoggingCapabilities{},
    }
    caps.AddExtension("io.modelcontextprotocol/streamable-http", nil)
    
    data, _ := json.Marshal(caps)
    // 验证包含 streamable-http 扩展和 logging
}
```

### L: 输入验证错误规范化

#### `internal/handler/handler.go`

修改两处参数验证错误，从 `errorResult()` 改为返回协议级错误：

**位置 1：** JSON 参数解析失败（原 handler.go:51-53）
```go
// 修改前
return errorResult(ErrInvalidParams, fmt.Sprintf("invalid arguments: %v", err)), nil

// 修改后
return nil, fmt.Errorf("invalid arguments: %w", ErrInvalidParamsProtocol)
```

等等，不对。如果返回非 nil error，SDK 会将其序列化为 JSON-RPC 错误。错误消息中包含 `%s` 格式的错误描述，但错误码需要是 -32000。

SDK 的 `ToolHandler` 签名：
```go
type ToolHandler func(context.Context, *CallToolRequest) (*CallToolResult, error)
```

如果 error 返回非 nil，SDK 的 `callTool` 方法（server.go:732-749）会直接返回 `(res, err)`，然后由 JSON-RPC 层处理。

要让错误码为 -32000，需要使用 `jsonrpc.NewError(-32000, msg)`：
```go
return nil, &jsonrpc.Error{Code: -32000, Message: fmt.Sprintf("invalid arguments: %v", err)}
```

**位置 2：** 路径参数类型验证（原 handler.go:62-66）
```go
// 修改前
return errorResult(ErrInvalidParams, fmt.Sprintf("path param %q must be an integer", p.Name)), nil

// 修改后
return nil, &jsonrpc.Error{Code: -32000, Message: fmt.Sprintf("path param %q must be an integer", p.Name)}
```

但其他错误（上游错误、内部错误）保持不变，继续使用 `errorResult()` + `IsError: true`。

#### `internal/handler/handler_test.go`

修改已有测试以适应新的错误返回格式：

**TestHandle_UpstreamError** 测试已经验证 `IsError=true` 的结构化错误 — 这个测试不需要改（上游错误保持 `IsError: true` 方式）。

添加新测试：
- `TestHandle_InvalidJSONParams` — 传入非法 JSON 参数，验证返回 `code: -32000` 的协议错误
- `TestHandle_InvalidPathParam` — 传入非法路径参数类型，验证返回协议错误

```go
func TestHandle_InvalidJSONParams(t *testing.T) {
    // ... 设置 mock server
    // ... 创建一个非法参数的请求（如无法 JSON 反序列化的 arguments）
    
    result, err := handler(context.Background(), req)
    if err == nil {
        t.Fatal("expected protocol-level error for invalid params")
    }
    // 验证错误是 jsonrpc.Error 类型且 code = -32000
    var rpcErr *jsonrpc.Error
    if !errors.As(err, &rpcErr) {
        t.Fatalf("expected jsonrpc.Error, got %T", err)
    }
    if rpcErr.Code != -32000 {
        t.Errorf("expected code -32000, got %d", rpcErr.Code)
    }
    if !strings.Contains(rpcErr.Message, "invalid arguments") {
        t.Errorf("message should contain 'invalid arguments', got: %s", rpcErr.Message)
    }
}
```

#### `internal/handler/errors.go`

可选：添加 `ErrValidationFailed` 常量供 handler 使用，简化错误返回：

```go
// 新增协议级验证错误（使用 MCP Tool Execution Error code -32000）
var ErrValidationFailed = &jsonrpc.Error{
    Code:    -32000,
    Message: "validation error",
}
```

但实际上直接用 `&jsonrpc.Error{Code: -32000, Message: msg}` 更灵活，不需要预定义常量。

---

## 关键决策

1. **L 任务中的错误格式选择：** 对输入验证错误使用协议级错误（code -32000），而非 `IsError: true`。上游错误、内部错误保持 `IsError: true`。理由：协议级错误在 JSON-RPC 层面有标准化的错误处理流程；SDK 文档的建议适用于"工具运行时错误"而非"参数格式错误"。

2. **H 任务中的扩展声明时机：** 在 server 创建时声明，不依赖 transport。始终包含 `io.modelcontextprotocol/streamable-http` 扩展，即使 stdio 模式下也无副作用。

3. **测试范围：** 不做 E2E 全链路测试（需 mock 整个 upstream），聚焦单元测试验证错误格式和 capabilities 结构。

---

## 验证步骤

### H 验证
1. `make test` 通过（含新增的 capabilities 测试）
2. 启动 server，检查 initialize 阶段返回的 capabilities 包含 `io.modelcontextprotocol/streamable-http` 扩展
3. `make lint` 通过

### L 验证
1. `make test` 通过（含新增的验证错误测试）
2. 手动验证：向 server 发送非法参数的 tools/call 请求，返回 JSON-RPC 错误 `{"code":-32000,"message":"..."}`
3. `make lint` 通过

---

## 任务依赖

```
Task 1: [H] 扩展协商声明实现 + 测试
   ├── 依赖: 无
   └── 产出: git commit "feat: 添加扩展协商声明，声明 streamable-http 支持"

Task 2: [L] 输入验证错误规范化实现 + 测试
   ├── 依赖: 无
   └── 产出: git commit "fix: 输入验证错误返回协议级错误 code -32000"
```

两个任务无依赖关系，可并行执行，但最好顺序执行以减少冲突。

---

## 提交计划

```bash
# Commit 1: H — 扩展协商声明
git add cmd/server/main.go cmd/server/main_test.go
git commit -m "feat: 添加扩展协商声明，声明 streamable-http 支持"

# Commit 2: L — 输入验证错误规范化
git add internal/handler/handler.go internal/handler/handler_test.go
git commit -m "fix: 输入验证错误返回协议级错误 code -32000"

# 如果 errors.go 有修改
git add internal/handler/errors.go
# 并入上一个 commit 或单独提交
```
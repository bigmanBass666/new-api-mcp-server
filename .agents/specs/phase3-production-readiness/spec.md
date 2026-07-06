# Phase 3：生产就绪加固 Spec

## Why

当前 MCP 服务器缺少生产运维所需的健康检查、速率限制、配置校验和优雅关闭。这四个特性是容器化部署（K8s/Cloudflare 等）的标配，缺失会导致运维盲区和服务不可靠。

## What Changes

- 新增 `/healthz`（存活）和 `/readyz`（就绪）HTTP 端点
- 新增可配置的令牌桶速率限制中间件
- 新增配置启动校验（URL、超时、传输模式）
- 增强 HTTP 传输的优雅关闭（drain + 超时配置）
- 新增 `internal/middleware/` 包（速率限制）
- 所有新增功能附带单元测试 + 集成测试

## Impact

- Affected code: `cmd/server/main.go`、`internal/config/config.go`、新文件 `internal/middleware/ratelimit.go`
- Affected tests: `cmd/server/main_test.go`（新增）、`internal/config/config_test.go`（扩展）、`internal/middleware/ratelimit_test.go`（新增）
- New dependency: `golang.org/x/time/rate`
- **BREAKING**: 无

---

## ADDED Requirements

### Requirement: 健康检查端点

The system SHALL provide `/healthz` and `/readyz` HTTP endpoints for container orchestration.

#### Scenario: /healthz 存活检测
- **WHEN** HTTP GET `/healthz`
- **THEN** 返回 `{"status":"ok"}`，HTTP 200

#### Scenario: /readyz 就绪检测（上游可达）
- **WHEN** 上游 New API 正常工作，HTTP GET `/readyz`
- **THEN** 返回 `{"status":"ok","upstream":"reachable"}`，HTTP 200

#### Scenario: /readyz 就绪检测（上游不可达）
- **WHEN** 上游 New API 不可达，HTTP GET `/readyz`
- **THEN** 返回包含 `{"status":"unhealthy"}` 的 JSON，HTTP 503

#### Scenario: 健康检查端点不干扰 MCP 协议
- **WHEN** MCP 客户端正常调用工具
- **THEN** 健康检查端点不影响 MCP 请求处理

---

### Requirement: 速率限制中间件

The system SHALL provide a configurable token-bucket rate limiter for the HTTP transport.

#### Scenario: 正常请求通过
- **WHEN** MCP_RATE_LIMIT_RPS=100，每秒发送 50 个请求
- **THEN** 所有请求正常被处理（HTTP 200）

#### Scenario: 超过速率限制被拒绝
- **WHEN** MCP_RATE_LIMIT_RPS=1，连续发送 3 个请求
- **THEN** 第 3 个请求返回 HTTP 429 Too Many Requests，响应 JSON 包含 `"error":"rate limit exceeded"`

#### Scenario: 速率限制可通过环境变量配置
- **WHEN** 设置 `MCP_RATE_LIMIT_RPS` 和 `MCP_RATE_LIMIT_BURST`
- **THEN** 配置正确生效，零值或负数表示不限速

#### Scenario: 速率限制不启用时不影响性能
- **WHEN** `MCP_RATE_LIMIT_RPS` 未设置或为 0
- **THEN** 请求正常通过，无速率限制（middleware 跳过）

---

### Requirement: 配置启动校验

The system SHALL validate configuration at startup and exit with a clear error message.

#### Scenario: 无效 BaseURL 格式
- **WHEN** `NEW_API_BASE_URL=http://`（空 host）
- **THEN** `config.Load()` 返回错误，消息包含 "invalid base URL"

#### Scenario: 无效传输模式
- **WHEN** `MCP_TRANSPORT=grpc`
- **THEN** 启动失败，错误信息包含 "unsupported transport"

#### Scenario: 超时值在合理范围内
- **WHEN** `NEW_API_TIMEOUT=5ms`
- **THEN** 启动失败，错误信息包含 "timeout too low"

#### Scenario: 有效配置通过校验
- **WHEN** 所有环境变量有效
- **THEN** 启动正常，无错误

---

### Requirement: HTTP 优雅关闭增强

The system SHALL gracefully drain pending HTTP requests before shutting down.

#### Scenario: SIGTERM 时等待请求完成
- **WHEN** 正在处理 HTTP 请求时收到 SIGTERM
- **THEN** 服务器停止接受新请求，等待当前请求完成（不超过配置的超时时间），然后关闭

#### Scenario: 可配置关闭超时
- **WHEN** 设置 `MCP_SHUTDOWN_TIMEOUT=30s`
- **THEN** 优雅关闭等待最长 30 秒

#### Scenario: 超过关闭超时强制退出
- **WHEN** 请求处理超过配置的关闭超时时间
- **THEN** 服务器强制关闭，不再等待
# E2E 测试套件 Spec

## Why

Phase 2（Tasks 扩展集成）的新功能（tasks_get/update/cancel、异步 test_and_report）目前只有单元测试覆盖，缺少端到端验证。需要一套真正的 E2E 测试：编译真实二进制、启动 server、发送真实 JSON-RPC 请求、验证响应。

## What Changes

- **新增** `cmd/server/e2e_test.go` — Go 版 E2E 测试（`//go:build e2e`）
- **新增** `cmd/server/e2e_mock_test.go` — mock 上游 HTTP server（模拟 New API）
- **新增** `Makefile` target: `make test-e2e-go`
- **修改** `.gitignore` — 忽略 e2e 编译产生的临时二进制
- 借鉴 Alvus-fork 的 E2E 测试模式：编译二进制 → mock 上游 → 真实 HTTP 协议验证

## Impact

- **Affected specs**: 测试覆盖·E2E·Tasks 扩展验证
- **Affected code**:
  - `cmd/server/e2e_test.go` (新增)
  - `cmd/server/e2e_mock_test.go` (新增)
  - `Makefile` (修改)

## ADDED Requirements

### Requirement: E2E Test Infrastructure

The system SHALL provide a Go-based E2E test suite that compiles the server binary, starts it against a mock upstream, and validates behavior through real HTTP JSON-RPC requests.

#### Scenario: Build and start server

- **WHEN** E2E test starts
- **THEN** it builds the server binary with `go build -o <temp>/new-api-mcp-server.exe ./cmd/server`
- **AND** sets up a mock upstream with `httptest.NewServer` simulating New API endpoints
- **AND** starts the server process with env vars pointing to mock upstream
- **AND** waits for the server to be ready via `/healthz` polling

#### Scenario: Clean shutdown

- **WHEN** E2E test completes (pass or fail)
- **THEN** server process is killed
- **AND** mock upstream server is closed
- **AND** temp binary is cleaned up

### Requirement: Tasks Extension Verification

#### Scenario: initialize includes tasks extension

- **WHEN** E2E test sends `initialize` request
- **THEN** response `capabilities.extensions["io.modelcontextprotocol/tasks"]` is present

### Requirement: Tool Registration Verification

#### Scenario: New tools are registered

- **WHEN** E2E test sends `tools/list` request
- **THEN** response includes `tasks_get`, `tasks_update`, `tasks_cancel` in the tools list
- **AND** also includes `test_and_report`

### Requirement: Async test_and_report E2E Flow

#### Scenario: test_and_report returns immediately with task_id

- **WHEN** E2E test sends `tools/call` with name `test_and_report`
- **AND** mock upstream responds with a valid task_id
- **THEN** response contains a task_id in the text (immediate return, no waiting)

#### Scenario: tasks_get retrieves task status

- **WHEN** E2E test sends `tools/call` with name `tasks_get` and valid `task_id`
- **THEN** response includes the task state and related information

#### Scenario: tasks_cancel cancels a running task

- **WHEN** E2E test sends `tools/call` with name `tasks_cancel` and valid `task_id`
- **THEN** response indicates the task was cancelled

## MODIFIED Requirements

N/A — all new functionality.

## REMOVED Requirements

N/A
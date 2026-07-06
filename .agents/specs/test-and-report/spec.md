# test_and_report Spec

## Why

New API 的渠道测试是异步的（`GET /api/channel/test` 返回 task_id 而非直接结果），AI 代理需要一种高层次的抽象：触发全量测试、等待完成、返回格式化的健康摘要，而不是手动拼装多步 API 调用。

## What Changes

- **新增** `internal/hightools/test_and_report.go` — `NewTestAndReportTool` 构造函数
- **修改** `internal/hightools/register.go` — 在 RegisterAll 列表中追加 test_and_report
- **修改** `cmd/server/main.go` — 传入 client 注册高层次的 hightools（如果原来没有接入）
- 依赖关系：需要 `internal/client.Client`、`internal/observability.Metrics`

## Impact

- **Affected specs**: 渠道管理·测试能力
- **Affected code**:
  - `internal/hightools/test_and_report.go` (新增)
  - `internal/hightools/register.go` (追加一行注册)
  - `cmd/server/main.go` (可选 — 若 hightools 尚未集成则需接线)

## ADDED Requirements

### Requirement: test_and_report Tool

The system SHALL register an MCP tool named `test_and_report` that triggers a full channel test against the upstream New API instance, waits for the async test task to complete, and returns a formatted health summary.

#### Scenario: 触发测试并等待完成

- **WHEN** user calls `test_and_report`
- **THEN** system calls `GET /api/channel/test` to enqueue a channel_test system task
- **AND** system extracts `task_id` from upstream response
- **AND** system polls `GET /api/system-task/{task_id}` every 2s until status is `succeeded` or `failed`
- **AND** system returns a formatted summary (tested/succeeded/failed/disabled/enabled counts)

#### Scenario: 已有测试任务正在运行

- **WHEN** user calls `test_and_report` while a channel test is already active
- **THEN** upstream returns HTTP 409
- **AND** system returns a clear message indicating a task is already in progress (including task_id and status), **not** a panic or error

#### Scenario: 测试任务超时

- **WHEN** the poll loop exceeds a configured deadline (default 120s)
- **THEN** system returns a partial summary with the status `"timeout"` and latest available state

#### Scenario: 某渠道不可用

- **WHEN** the upstream test completes and some channels failed
- **THEN** the returned summary reports per-channel errors naturally as part of the output
- **AND** the tool returns `IsError: false` (the tool itself succeeded; channel failures are data, not an error)

### Requirement: Output Format

The tool SHALL return a clearly formatted text summary following this template:

```
# 渠道测试报告

## 概览
- 测试数: 10
- 通过: 8
- 失败: 2
- 已禁用: 0
- 已启用: 0
- 耗时: 45.2s

## 失败渠道详情
- 渠道 #3 (Slack) - 连接超时
- 渠道 #7 (Agnes) - 401 Unauthorized

## 测试状态: ✅ 已完成
```

When all channels pass, the "失败渠道详情" section SHALL be omitted.

## MODIFIED Requirements

N/A — all new functionality.

## REMOVED Requirements

N/A
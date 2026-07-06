# Tasks 扩展集成 Spec

## Why

`test_and_report` 当前是同步阻塞的——触发渠道测试后最多等 120 秒才能返回结果。渠道多时体验极差。
需要将任务生命周期管理抽象为 MCP Tasks 扩展，支持异步触发、轮询、异常确认和取消。

## What Changes

- **声明** `io.modelcontextprotocol/tasks` 扩展 capability（类似 Phase 1 的 streamable-http）
- **新增** `TaskManager` 核心——线程安全的内存 Task 状态管理（6 状态状态机 + TTL 清理）
- **新增** `tasks_get` 工具——查询异步任务状态
- **新增** `tasks_update` 工具——input_required 状态下提交决策（resume/retry）
- **新增** `tasks_cancel` 工具——取消运行中任务
- **改造** `test_and_report`——从同步轮询改为异步：触发即返回 task_id，后台 goroutine 执行
- **注册** 新工具到 `RegisterAll()` 并接入 `defaultCapabilities()`
- 所有新工具命名遵循 `tasks_` 前缀风格（与 `api_` 前缀一致）

## Impact

- **Affected specs**: 渠道管理·异步测试能力、任务生命周期管理
- **Affected code**:
  - `internal/hightools/task_manager.go` (新增)
  - `internal/hightools/task_manager_test.go` (新增)
  - `internal/hightools/tasks_get.go` (新增)
  - `internal/hightools/tasks_get_test.go` (新增)
  - `internal/hightools/tasks_update.go` (新增)
  - `internal/hightools/tasks_update_test.go` (新增)
  - `internal/hightools/tasks_cancel.go` (新增)
  - `internal/hightools/tasks_cancel_test.go` (新增)
  - `internal/hightools/test_and_report.go` (修改)
  - `internal/hightools/test_and_report_test.go` (修改)
  - `internal/hightools/register.go` (修改)
  - `cmd/server/main.go` (修改)
  - `cmd/server/main_test.go` (修改)

## ADDED Requirements

### Requirement: TaskManager Core

The system SHALL provide a thread-safe in-memory task manager that tracks async task lifecycle.

#### Scenario: Create and retrieve a task

- **WHEN** `CreateTask(type, metadata)` is called
- **THEN** a new task is created with `id` (UUID), `state: "pending"`, and `created_at` timestamp
- **AND** the task can be retrieved by `GetTask(id)`

#### Scenario: Update task state with validation

- **WHEN** `UpdateTask(id, newState)` is called
- **THEN** the state transition is validated against the state machine
- **AND** invalid transitions return an error (e.g., `pending → succeeded`)
- **AND** valid transitions succeed and update `updated_at`
- **AND** terminal states (`succeeded`, `failed`, `cancelled`) reject further updates

#### Scenario: Cancel a running task

- **WHEN** `CancelTask(id)` is called on a task in `running` or `input_required` state
- **THEN** the task transitions to `cancelled`
- **AND** the associated background worker is stopped (via cancel channel)

#### Scenario: Task TTL cleanup

- **WHEN** `Reap()` is called
- **THEN** tasks in terminal state (`succeeded`, `failed`, `cancelled`) older than TTL (default 5 minutes) are removed
- **AND** active tasks (`pending`, `running`, `input_required`) are never removed
- **AND** the total task count does not exceed `maxTasks` (default 100)

### Requirement: tasks_get Tool

#### Scenario: Query existing task

- **WHEN** user calls `tasks_get` with `task_id`
- **THEN** system returns the full task state (id, type, state, progress, result, error, metadata, timestamps)

#### Scenario: Query non-existent task

- **WHEN** user calls `tasks_get` with a non-existent `task_id`
- **THEN** system returns a clear error message

### Requirement: tasks_update Tool

#### Scenario: Resume a paused task (input_required → running)

- **WHEN** user calls `tasks_update` with `task_id` and `action: "resume"`
- **AND** the task is in `input_required` state
- **THEN** system resumes the background worker
- **AND** task state transitions to `running`

#### Scenario: Retry a paused task

- **WHEN** user calls `tasks_update` with `task_id` and `action: "retry"`
- **AND** the task is in `input_required` state
- **THEN** system retries the failed operation
- **AND** task state transitions to `running`

#### Scenario: Update non-paused task

- **WHEN** user calls `tasks_update` on a task NOT in `input_required` state
- **THEN** system returns an error

### Requirement: tasks_cancel Tool

- **WHEN** user calls `tasks_cancel` with `task_id`
- **AND** the task is in `running` or `input_required` state
- **THEN** system transitions task to `cancelled`
- **AND** background goroutine receives cancellation signal

### Requirement: Async test_and_report

#### Scenario: Trigger and return immediately

- **WHEN** user calls `test_and_report`
- **THEN** system creates a new Task with type `channel_test`
- **AND** system triggers `GET /api/channel/test` to enqueue the test
- **AND** system starts a background goroutine to poll for completion
- **AND** system returns `{task_id, state: "running", message: "渠道测试已启动"}` immediately

#### Scenario: Background worker completes normally

- **WHEN** the background goroutine polls `GET /api/system-task/{task_id}`
- **AND** the task status is `succeeded`
- **THEN** the goroutine decodes the result and updates the Task to `state: "succeeded"` with result data

#### Scenario: Background worker encounters channel exception

- **WHEN** any channel test fails during background execution
- **THEN** the goroutine sets `state: "input_required"` with metadata containing exception details and options

#### Scenario: Background worker completes after resume

- **WHEN** user calls `tasks_update` with `action: "resume"`
- **THEN** the background worker continues (skips or retries based on action)
- **AND** eventually updates the Task to `succeeded` or `failed`

#### Scenario: Query task progress

- **WHEN** user calls `tasks_get` after `test_and_report` has been triggered
- **THEN** system returns current task state (progress can be estimated from polled status)

### Requirement: Tasks Extension Declaration

The system SHALL declare `io.modelcontextprotocol/tasks` extension in `ServerCapabilities`.

- **WHEN** server responds to `initialize`
- **THEN** capabilities response includes `extensions["io.modelcontextprotocol/tasks"]: {}`

### Requirement: Tool Naming and Registration

- `tasks_get` SHALL be registered as a high-level tool
- `tasks_update` SHALL be registered as a high-level tool
- `tasks_cancel` SHALL be registered as a high-level tool
- All three SHALL appear in `RegisterAll()`
- Registration SHALL only occur when `cfg.APIToolsEnabled && cfg.SystemKey != ""`

## MODIFIED Requirements

### Requirement: test_and_report Tool (Modified)

**变更前**: 同步阻塞，触发测试后轮询等待最多 120 秒，返回格式化报告。

**变更后**: 异步触发，立即返回 `task_id`，后台 goroutine 执行轮询，用户通过 `tasks_get`/`tasks_update`/`tasks_cancel` 管理任务生命周期。

#### Scenario: 触发测试并立即返回

- **WHEN** user calls `test_and_report`
- **THEN** system creates a task, triggers the test, starts background worker
- **AND** returns immediately with `{task_id, state: "running"}`
- **AND** the background worker polls `GET /api/system-task/{task_id}`

#### Scenario: 测试任务正常完成（通过 tasks_get 查询）

- **WHEN** background worker detects `status: "succeeded"`
- **THEN** it updates task state to `succeeded` with formatted result
- **AND** user can retrieve the result via `tasks_get`

#### Scenario: 测试任务遇到渠道异常（input_required）

- **WHEN** background worker detects channel failures
- **THEN** it sets `state: "input_required"` with exception details
- **AND** pauses until user calls `tasks_update` with `action: "resume"` or `action: "retry"`

#### Scenario: 测试任务超时

- **WHEN** background poll loop exceeds 120s timeout
- **THEN** background worker updates task to `state: "failed"` with timeout message
- **AND** partial result (if any) is available in metadata

## REMOVED Requirements

### Requirement: Synchronous test_and_report (Removed)

**Reason**: 被异步模型替代，不再同步阻塞等待。

**Migration**: 旧的使用方式（等待并直接返回报告）不再支持。需要改为两步走：先调 `test_and_report` 获取 `task_id`，再调 `tasks_get` 轮询结果。
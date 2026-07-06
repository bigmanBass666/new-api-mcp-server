# Tasks 扩展设计文档

> **创建日期：** 2026-07-06
> **状态：** 草稿
> **对应路线图：** [自用优先路线图](../plans/2026-07-06-self-use-roadmap.md) — 方向 C（Tasks 扩展集成）
> **前驱调研：** MCP Go SDK v1.4.1 无原生 Tasks 支持，从零实现

---

## 1. 概述

本设计实现 MCP 扩展 `io.modelcontextprotocol/tasks`，提供异步任务生命周期管理能力。核心场景是将 `test_and_report` 从同步阻塞改造为异步轮询。

### 1.1 设计目标

- 将 `test_and_report` 从同步阻塞（~120s 超时）改为异步（立即返回 + 后台执行）
- 提供标准任务查询/更新/取消接口
- 支持 `input_required` 异常处理流程
- 良好的并发安全
- 资源可控（TTL + 清理）

### 1.2 非目标

- 不做 MCP 协议级别的任务调度器（如跨会话任务持久化）
- 不做分布式任务队列（单进程内存存储）
- 不做任务编排（DAG、依赖）

---

## 2. 任务状态机

### 2.1 状态定义

```go
type TaskState string

const (
    TaskStatePending        TaskState = "pending"
    TaskStateRunning        TaskState = "running"
    TaskStateSucceeded      TaskState = "succeeded"
    TaskStateFailed         TaskState = "failed"
    TaskStateInputRequired  TaskState = "input_required"
    TaskStateCancelled      TaskState = "cancelled"
)
```

### 2.2 状态转换图

```
                  ┌──────────┐
                  │  pending  │
                  └────┬─────┘
                       │ start
                  ┌────▼─────┐
                  │  running  │◄──────────────┐
                  └────┬─────┘               │
                       │                      │
          ┌────────────┼──────────┐           │
          │            │          │           │
     ┌────▼───┐  ┌────▼───┐ ┌────▼────┐     │
     │succeeded│  │ failed │ │input_   │─────┘
     └────────┘  └────────┘ │required │(resume)
                            └────┬────┘
                                 │ cancel
                            ┌────▼────┐
                            │cancelled│
                            └─────────┘
```

### 2.3 合法转换表

| 当前状态 → 目标状态 | 合法？ | 触发条件 |
|-------------------|--------|---------|
| pending → running | ✅ | 后台 worker 开始执行 |
| running → succeeded | ✅ | 任务正常完成 |
| running → failed | ✅ | 任务执行出错 |
| running → input_required | ✅ | 渠道异常，需要用户确认 |
| running → cancelled | ✅ | 用户主动取消 |
| input_required → running | ✅ | 用户提交决策后继续 |
| input_required → cancelled | ✅ | 用户取消 |
| 其他 | ❌ | 终结状态不可变 |

---

## 3. Task 数据结构

### 3.1 Task struct

```go
type Task struct {
    ID        string     `json:"id"`
    Type      string     `json:"type"`       // 任务类型，如 "channel_test"
    State     TaskState  `json:"state"`
    Progress  float64    `json:"progress,omitempty"`  // 0.0 ~ 1.0
    Result    any        `json:"result,omitempty"`
    Error     string     `json:"error,omitempty"`
    Metadata  map[string]any `json:"metadata,omitempty"` // 额外上下文
    CreatedAt time.Time  `json:"created_at"`
    UpdatedAt time.Time  `json:"updated_at"`
}
```

### 3.2 TaskManager 接口

```go
type TaskManager struct {
    mu       sync.RWMutex
    tasks    map[string]*Task
    ttl      time.Duration  // 完成后保留时间，默认 5min
    maxTasks int            // 最大任务数，默认 100
}

func NewTaskManager(ttl time.Duration, maxTasks int) *TaskManager

func (tm *TaskManager) CreateTask(taskType string, metadata map[string]any) (*Task, error)
func (tm *TaskManager) GetTask(id string) (*Task, error)
func (tm *TaskManager) UpdateTask(id string, state TaskState, opts ...TaskOption) error
func (tm *TaskManager) CancelTask(id string) error
func (tm *TaskManager) ListTasks(filter TaskFilter) []*Task
func (tm *TaskManager) Reap() int  // 清理过期任务，返回清理数
```

---

## 4. 工具定义

### 4.1 tasks/get

| 字段 | 值 |
|------|-----|
| name | `tasks_get` |
| description | 查询异步任务状态 |
| input schema | `{"type":"object","properties":{"task_id":{"type":"string"}}, "required":["task_id"]}` |
| output | Task 结构体 JSON |

### 4.2 tasks/update

| 字段 | 值 |
|------|-----|
| name | `tasks_update` |
| description | 更新异步任务（如 input_required 状态下提交决策） |
| input schema | `{"type":"object","properties":{"task_id":{"type":"string"},"action":{"type":"string","enum":["resume","retry"]},"payload":{"type":"object"}}, "required":["task_id","action"]}` |
| output | 更新后的 Task 状态 |

### 4.3 tasks/cancel

| 字段 | 值 |
|------|-----|
| name | `tasks_cancel` |
| description | 取消运行中的异步任务 |
| input schema | `{"type":"object","properties":{"task_id":{"type":"string"}}, "required":["task_id"]}` |
| output | 取消确认信息 |

---

## 5. test_and_report 异步改造

### 5.1 当前流程（同步）

```
test_and_report handler
  ├─ triggerChannelTest()       → GET /api/channel/test
  ├─ pollTaskCompletion()       → 轮询 GET /api/system-task/{id}（最长 120s）
  ├─ formatTestReport()
  └─ return CallToolResult
```

### 5.2 改造后流程（异步）

```
test_and_report handler
  ├─ TaskManager.CreateTask("channel_test")
  ├─ triggerChannelTest()       → GET /api/channel/test
  ├─ go worker()                → 后台 goroutine
  └─ return { task_id, status: "running", message: "测试已启动" }

后台 worker():
  ├─ 轮询 GET /api/system-task/{id}
  │   ├─ 正常完成 → UpdateTask(succeeded, result)
  │   ├─ 异常 → UpdateTask(input_required, {异常信息, options: [skip, retry, abort]})
  │   └─ 失败 → UpdateTask(failed, error)
  └─ 结束
```

### 5.3 input_required 处理流程

```
用户视角:
  1. tasks_get(task_id="xxx")
     → {state: "input_required", metadata: {异常: "渠道 A 超时", options: ["skip","retry","abort"]}}

  2. tasks_update(task_id="xxx", action="skip", payload={})
     → {state: "running", message: "已跳过异常渠道，继续测试"}

  3. tasks_get(task_id="xxx")
     → {state: "succeeded", result: {tested: 10, succeeded: 9, failed: 1}}

TaskManager 视角:
  - input_required 状态下，后台 worker pause
  - tasks_update("resume") 被调用后，worker 从 channel 收到信号
  - worker 根据 action 决定后续逻辑（跳过/重试/终止）
  - worker 继续执行，最终更新状态为 succeeded/failed
```

### 5.4 Worker 实现要点

```go
// 后台 worker 管理 goroutine 生命周期
type taskWorker struct {
    taskID    string
    taskMgr   *TaskManager
    client    *client.Client
    metrics   *observability.Metrics
    resumeCh  chan ResumeSignal  // input_required 恢复信号
    cancelCh  chan struct{}      // 取消信号
}

type ResumeSignal struct {
    Action  string
    Payload map[string]any
}

func (w *taskWorker) Run(ctx context.Context) {
    // 1. 更新任务为 running
    // 2. 执行实际工作
    // 3. 处理 input_required 暂停/恢复
    // 4. 处理取消请求
    // 5. 更新最终状态
}
```

---

## 6. 资源管理

### 6.1 TTL 策略

- 已完成任务（succeeded/failed/cancelled）：TTL = 5 分钟
- 活跃任务（pending/running/input_required）：不被清理
- 后台 goroutine 每分钟执行一次 `Reap()`

### 6.2 容量限制

- 最大任务数：100 个
- 超过时，优先清理最旧的已完成任务
- 活跃任务数量不限（但受 goroutine 数限制）

### 6.3 并发安全

- `TaskManager.tasks` 使用 `sync.RWMutex` 保护
- 读操作（`GetTask`、`ListTasks`）使用 RLock
- 写操作（`CreateTask`、`UpdateTask`、`CancelTask`、`Reap`）使用 Lock
- 后台 worker 与 handler 通过 channel 通信，无共享内存

---

## 7. 扩展声明

在 `cmd/server/main.go` 的 `defaultCapabilities()` 中添加：

```go
func defaultCapabilities() *mcp.ServerCapabilities {
    caps := &mcp.ServerCapabilities{
        Logging: &mcp.LoggingCapabilities{},
    }
    caps.AddExtension("io.modelcontextprotocol/streamable-http", nil)
    caps.AddExtension("io.modelcontextprotocol/tasks", nil)  // 新增
    return caps
}
```

---

## 8. 文件改动清单

| 文件 | 操作 | 说明 |
|------|------|------|
| `internal/hightools/task_manager.go` | 新增 | TaskManager 核心实现 |
| `internal/hightools/task_manager_test.go` | 新增 | TaskManager 单元测试 |
| `internal/hightools/tasks_get.go` | 新增 | tasks_get 工具 |
| `internal/hightools/tasks_get_test.go` | 新增 | tasks_get 测试 |
| `internal/hightools/tasks_update.go` | 新增 | tasks_update 工具 |
| `internal/hightools/tasks_update_test.go` | 新增 | tasks_update 测试 |
| `internal/hightools/tasks_cancel.go` | 新增 | tasks_cancel 工具 |
| `internal/hightools/tasks_cancel_test.go` | 新增 | tasks_cancel 测试 |
| `internal/hightools/test_and_report.go` | 修改 | 改为异步 |
| `internal/hightools/test_and_report_test.go` | 修改 | 适配新行为 |
| `internal/hightools/register.go` | 修改 | 注册新工具 |
| `cmd/server/main.go` | 修改 | 添加 Tasks 扩展声明 |
| `cmd/server/main_test.go` | 修改 | 验证扩展声明 |
| `docs/plans/2026-07-06-self-use-roadmap.md` | 修改 | 标记完成 |

---

## 9. 估算

| 阶段 | 任务 | 估算 |
|------|------|------|
| 核心 | TaskManager 实现 | 0.5 天 |
| 工具 | tasks/get + update + cancel | 1 天 |
| 改造 | test_and_report 异步化 | 1.5 天 |
| 测试 | 全部单元测试 + 集成 | 1.5 天 |
| 验收 | 文档 + 验收测试 | 0.5 天 |
| **合计** | | **~5 天** |
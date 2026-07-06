# Tasks

- [x] Task 1: TaskManager 核心实现
  - [x] 1a. 定义 Task struct、TaskState 枚举、状态转换合法性验证（validTransitions map）
  - [x] 1b. 实现 TaskManager（sync.RWMutex + map[string]*Task）
  - [x] 1c. CreateTask / GetTask / UpdateTask / CancelTask / ListTasks / Reap 完整方法集
  - [x] 1d. input_required 状态下的 resume/resumeCh 机制
  - [x] 1e. TTL 清理（5min） + 容量限制（100 tasks）

- [x] Task 2: tasks_get / tasks_update / tasks_cancel 工具实现
  - [x] 2a. tasks_get: 按 task_id 查询返回完整 Task 状态
  - [x] 2b. tasks_update: input_required 状态下提交 resume/retry 决策
  - [x] 2c. tasks_cancel: 取消 running/input_required 状态的任务

- [x] Task 3: test_and_report 异步改造
  - [x] 3a. handler 改为：创建 Task → 触发测试 → 启动后台 goroutine → 立即返回 task_id
  - [x] 3b. 后台 worker goroutine：轮询 /api/system-task/{id} → 更新 Task 状态
  - [x] 3c. input_required 处理：渠道异常时暂停 worker，等待 resume/retry 信号
  - [x] 3d. 取消处理：通过 cancelCh 响应 tasks_cancel
  - [x] 3e. 保持向后兼容输出格式（tasks_get 返回格式化结果）

- [x] Task 4: 扩展声明 + 注册
  - [x] 4a. defaultCapabilities() 添加 io.modelcontextprotocol/tasks 扩展
  - [x] 4b. register.go 追加 tasks_get / tasks_update / tasks_cancel
  - [x] 4c. 确保 api tools 启用的条件才注册 tasks 工具

- [x] Task 5: 测试
  - [x] 5a. TaskManager 单元测试（创建/查询/更新/取消/状态校验/TTL）
  - [x] 5b. tasks_get 测试（存在/不存在）
  - [x] 5c. tasks_update 测试（正常 resume / 错误状态）
  - [x] 5d. tasks_cancel 测试（运行中/已终结）
  - [x] 5e. test_and_report 异步测试（触发返回 / 后台完成 / input_required）
  - [x] 5f. make test 全通过 + race detector 无竞态

- [x] Task 6: 验收测试
  - [x] 6a. 验证 initialize 响应包含 extensions.io.modelcontextprotocol/tasks
  - [x] 6b. 验证 test_and_report 返回 task_id 而非等待
  - [x] 6c. 验证 tasks_get 能查到任务状态

# Task Dependencies
- Task 2 depends on Task 1 (tools 需要 TaskManager)
- Task 3 depends on Task 1 (test_and_report 需要 TaskManager)
- Task 5 depends on Task 1, Task 2, Task 3 (测试需要实现完成)
- Task 6 depends on Task 4 (扩展声明需要先注册)
- Task 1 → Task 2 → Task 3 → ... 推荐顺序执行，Task 4 可在 Task 2 之后并行
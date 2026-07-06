# Checklist

## TaskManager 核心
- [x] Task struct 定义完整（ID, Type, State, Progress, Result, Error, Metadata, CreatedAt, UpdatedAt）
- [x] TaskState 枚举覆盖 6 个状态
- [x] 状态转换合法性验证生效（非法转换返回 error）
- [x] CreateTask 返回 UUID、state=pending
- [x] UpdateTask 验证状态转换并更新时间戳
- [x] CancelTask 停止 running/input_required 任务
- [x] Reap 清理过期任务、不碰活跃任务、不超过容量上限
- [x] 所有操作线程安全（race detector 无竞态）

## Tasks 工具
- [x] tasks_get 按 task_id 查询返回完整状态
- [x] tasks_get 不存在时返回友好错误
- [x] tasks_update input_required → running (resume) 转换成功
- [x] tasks_update 非 input_required 状态返回错误
- [x] tasks_cancel 取消 running 任务成功
- [x] tasks_cancel 取消已终结任务返回错误

## test_and_report 异步改造
- [x] test_and_report 返回 task_id 而非等待完成
- [x] 后台 worker 正常轮询并完成
- [x] 渠道异常时进入 input_required 状态
- [x] input_required → tasks_update resume 后继续执行
- [x] tasks_cancel 能停止后台 worker
- [x] 超时场景正确处理（task state: input_required + timeout message）

## 扩展声明 & 注册
- [x] initialize 响应包含 extensions.io.modelcontextprotocol/tasks
- [x] tasks_get / tasks_update / tasks_cancel 在 RegisterAll() 中注册
- [x] 仅当 api tools 启用时注册 tasks 工具

## 测试通过
- [x] make test 全通过
- [x] race detector 无竞态
- [x] 验收测试验证扩展声明 + 核心流程

## 文档
- [x] 路线图 docs/plans/2026-07-06-self-use-roadmap.md 更新
- [x] 关键发现记录（类似 Phase 1 的 5 条发现）
# Tasks

- [x] Task 1: 创建 mock 上游服务器（e2e_mock_test.go）
  - [x] 1a. 实现 `/api/channel/test` handler（返回 task_id）
  - [x] 1b. 实现 `/api/system-task/{id}` handler（支持 succeeded/running 状态切换）
  - [x] 1c. 实现 `/healthz` handler

- [x] Task 2: 创建 E2E 测试骨架（e2e_test.go）
  - [x] 2a. `//go:build e2e` 标签
  - [x] 2b. 编译二进制、起 mock 上游、起 server、停 server 的完整流程
  - [x] 2c. `mcpClient` 结构体（发送 JSON-RPC 请求、维持会话、解析结果）

- [x] Task 3: 实现测试场景
  - [x] 3a. 验证 tasks 扩展声明
  - [x] 3b. 验证新工具已注册（含分页遍历）
  - [x] 3c. 验证异步返回
  - [x] 3d. 验证任务状态查询
  - [x] 3e. 验证任务取消

- [x] Task 4: 更新 Makefile
  - [x] 4a. 添加 `test-e2e-go` target
  - [x] 4b. 与已有 Python E2E 不冲突

- [x] Task 5: 运行验证
  - [x] 5a. `go test -tags=e2e ./cmd/server/ -v` 全部通过
  - [x] 5b. 不影响现有 `make test`（e2e 标签隔离）

# Task Dependencies
- Task 2 depends on Task 1 (测试需要 mock 上游就绪)
- Task 3 depends on Task 1 and Task 2 (测试场景需要基础设施)
- Task 4 depends on Task 3 (Makefile 需要在测试写好后再加)
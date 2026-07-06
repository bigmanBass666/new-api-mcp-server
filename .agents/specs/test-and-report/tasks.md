# Tasks

- [x] Task 1: 实现 test_and_report.go — NewTestAndReportTool 构造函数
  - [x] 1a. 定义 InputSchema（无参数即可）
  - [x] 1b. 实现 Handler：调用 GET /api/channel/test
  - [x] 1c. 实现 Handler：解析 task_id 并轮询 GET /api/system-task/{task_id}（每 2s，最长 120s）
  - [x] 1d. 实现 Handler：从 task.Result 解析 channelTestSummary（tested/succeeded/failed/disabled/enabled）
  - [x] 1e. 实现 Handler：格式化输出（概览 + 失败渠道详情）
  - [x] 1f. 处理边缘情况：已有任务冲突（409）、超时、上游不可达
  - [x] 1g. 使用 client.Client 和 metrics 记录调用指标

- [x] Task 2: 在 register.go 的 RegisterAll 中追加 test_and_report
  - [x] 2a. 将 NewTestAndReportTool 加入返回列表
  - [x] 2b. 确保 TestAndReportTool 接收 client 和 metrics 参数

- [x] Task 3: 在 cmd/server/main.go 中接线注册 hightools
  - [x] 3a. 在 API tools 注册之后，调用 hightools.RegisterAll()
  - [x] 3b. 遍历返回的 ToolDef 列表，调用 server.AddTool()
  - [x] 3c. 仅当 cfg.APIToolsEnabled 且 cfg.SystemKey 不为空时注册（与 API tools 相同的条件）

- [x] Task 4: 编写测试
  - [x] 4a. 单元测试：用 httptest 模拟 upstream 返回 task_id，模拟轮询过程
  - [x] 4b. 单元测试：模拟冲突（409）响应
  - [x] 4c. 单元测试：模拟超时场景
  - [x] 4d. 单元测试：验证输出格式符合 spec

# Task Dependencies
- Task 3 depends on Task 1 and Task 2
- Task 4 is independent (can be done after Task 1 or in parallel)
# Checklist

## Mock 服务器
- [x] 支持 `GET /api/channel/test` 返回有效 task_id
- [x] 支持 `GET /api/system-task/{id}` 返回不同状态
- [x] 正确响应 healthz

## E2E 基础设施
- [x] 编译二进制成功
- [x] 启动 server 并等待健康检查通过
- [x] 发送 JSON-RPC 请求并解析响应
- [x] 测试结束后清理进程和临时文件

## 测试场景
- [x] initialize 响应包含 `extensions.io.modelcontextprotocol/tasks`
- [x] tools/list 包含 tasks_get / tasks_update / tasks_cancel / test_and_report
- [x] test_and_report 返回 task_id 而非同步等待
- [x] tasks_get 能查询任务状态
- [x] tasks_cancel 能取消任务

## 集成
- [x] `make test-e2e-go` 可运行
- [x] `make test` 不受影响（标签隔离）
- [x] 脚本不依赖外部 Docker/网络
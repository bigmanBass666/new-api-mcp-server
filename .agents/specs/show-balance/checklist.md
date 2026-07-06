# Checklist

- [x] show_balance.go 实现文件的代码风格与现有 hightools 一致（tab indentation、otel tracer、slog 日志、metrics 记录）
- [x] 工具先调用 GET /api/channel/update_balance 再调用 GET /api/channel/，顺序正确
- [x] 刷新余额失败时不阻断流程，只输出警告
- [x] 输出格式为 Markdown 表格，包含 ID、名称、状态、剩余额度、已用额度、最后更新时间
- [x] register.go 中正确添加了 NewShowBalanceTool 的条目
- [x] 所有测试用例通过（go test）
- [x] balance_updated_time=0 时正确显示 "(从未更新)"
- [x] 指定 channel_id 参数时调用 /api/channel/update_balance/{id} 而非全量刷新
- [x] 不需要修改 list_providers.go 或 cmd/server/main.go
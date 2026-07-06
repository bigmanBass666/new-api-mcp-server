# show_balance Spec

## Why

当前 MCP 服务器已有 `list_providers` 工具列出渠道基本信息，但缺少渠道余额概览能力。用户（AI 客户端）需要查看各渠道的剩余额度、已用额度及额度更新时间来监控渠道健康状态。

`show_balance` 是渠道管理的余额监控工具。它先调用 `GET /api/channel/update_balance` 触发上游刷新所有渠道的余额缓存，再调用 `GET /api/channel/` 获取刷新后的渠道列表，以结构性文本展示每个渠道的余额信息。

## What Changes

- 新增 `internal/hightools/show_balance.go`：实现 `show_balance` 工具的构造函数和 handler 逻辑
- 修改 `internal/hightools/register.go`：将 `show_balance` 加入 `RegisterAll()` 返回列表
- 新增 `internal/hightools/show_balance_test.go`：使用 `httptest.Server` mock 两个上游端点（`/api/channel/update_balance` 和 `/api/channel/`），验证组合调用和输出格式

## Impact

- Affected specs: 渠道管理模块
- Affected code:
  - `internal/hightools/register.go` — 列出所有高层工具（修改）
  - `internal/hightools/show_balance.go` — 新工具实现
  - `internal/hightools/show_balance_test.go` — 新工具测试

## ADDED Requirements

### Requirement: show_balance 工具

系统 SHALL 提供一个名为 `show_balance` 的 MCP 工具，用于展示所有渠道的余额概览。

#### 工具流程

工具执行 SHALL 按以下顺序组合调用两个上游 API：

1. **刷新余额（触发）** — 调用 `GET /api/channel/update_balance`（无参数），上游会触发所有渠道余额的异步/同步刷新
2. **获取渠道列表** — 调用 `GET /api/channel/?page_size=100` 获取所有渠道的完整信息，包含余额字段

#### Scenario: 正常获取余额概览

- **WHEN** 用户调用 `show_balance` 工具
- **THEN** 系统依次调用 `GET /api/channel/update_balance` 和 `GET /api/channel/`
- **AND** 以结构化表格形式返回结果，每个渠道一行，包含：ID、名称、状态（启用/禁用）、剩余额度（$）、已用额度、最后更新时间

#### Scenario: 刷新余额失败（可容忍）

- **WHEN** `GET /api/channel/update_balance` 返回非 200 或 `success=false`
- **THEN** 系统 SHALL 在最终输出顶部包含警告信息 "余额刷新失败，显示的是缓存数据"，AND 仍然继续调用 `GET /api/channel/` 获取当前已缓存的数据
- **AND** 不将整个工具标记为 IsError（上游渠道列表仍可能成功）

#### Scenario: 获取渠道列表失败

- **WHEN** `GET /api/channel/` 返回非 200 或 `success=false` 或解析失败
- **THEN** 系统返回 `IsError=true`，包含错误描述

#### Scenario: 渠道列表为空

- **WHEN** `GET /api/channel/` 返回空数组 `[]`
- **THEN** 系统返回提示信息 "没有找到任何渠道"

### Requirement: 输出格式

系统 SHALL 按以下 Markdown 表格格式展示余额概览：

```
## 余额概览

| ID | 名称 | 状态 | 剩余额度(USD) | 已用额度 | 最后更新时间 |
|----|------|------|--------------|---------|-------------|
| 1 | 渠道A | ✅ 启用 | 100.00 | 500000 | 2026-07-04 10:00:00 |
| 2 | 渠道B | ❌ 禁用 | 0.00 | 1200000 | 2026-07-03 08:30:00 |
```

- 状态：status=1 显示 ✅ 启用，其他显示 ❌ 禁用
- 剩余额度：从 `balance` 字段读取，保留两位小数，格式化为 `$X.XX`
- 已用额度：从 `used_quota` 字段读取，格式为整数
- 最后更新时间：从 `balance_updated_time`（Unix 时间戳，秒）转为 `YYYY-MM-DD HH:mm:ss` 格式
- `balance_updated_time` 为 0 时显示 `(从未更新)`

### Requirement: 工具的 InputSchema

工具无必填参数，可选参数包括：
- `channel_id` (integer, optional): 如果指定，只刷新并显示指定渠道的余额（调用 `GET /api/channel/update_balance/{id}` 而非全量刷新）
- `group` (string, optional): 按分组筛选渠道

### Requirement: 工具注册

`show_balance` SHALL 以 `show_balance` 名称注册到 MCP 服务器。注册方式与其他高层工具一致。

### Requirement: 复用现有响应结构

- 在 `show_balance.go` 中定义包含 `balance`、`used_quota`、`balance_updated_time` 字段的 Channel 响应结构体，与 `list_providers.go` 中的 Channel 结构体分离，避免修改已有代码
- API 响应解析结构与 `list_providers.go` 保持一致（`listChannelsResponse` / `listData` wrapper）
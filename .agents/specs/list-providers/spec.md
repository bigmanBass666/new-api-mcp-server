# list_providers Spec

## Why

当前 MCP 服务器有 160+ 直接映射 API 端点的原子工具，但缺少面向场景的高层封装。用户（AI 客户端）若要查看所有渠道的概况，需要自行调用 `/api/channel/` 并处理 JSON 响应，这对 AI 不友好。

`list_providers` 是渠道管理的核心入口工具，对标官方 newapi-admin 的 "channels" 指令。它聚合了上游 GET /api/channel/ 的响应，按分组展示渠道信息，让 AI 能一句话获得渠道全貌。

## What Changes

- 新增 `internal/hightools/list_providers.go`：实现 `list_providers` 工具的构造函数和 handler 逻辑
- 修改 `internal/hightools/register.go`：将 `list_providers` 加入 `RegisterAll()` 返回列表
- 修改 `cmd/server/main.go`：在 API 工具注册后，注册 `hightools.RegisterAll()` 返回的高层工具
- 新增 `internal/hightools/list_providers_test.go`：使用 `httptest.Server` mock 上游渠道列表 API，验证分组输出

## Impact

- Affected specs: 渠道管理模块
- Affected code:
  - `internal/hightools/tooldef.go` — 定义 ToolDef 结构体（已有）
  - `internal/hightools/register.go` — 列出所有高层工具（修改）
  - `internal/hightools/list_providers.go` — 新工具实现
  - `internal/hightools/list_providers_test.go` — 新工具测试
  - `cmd/server/main.go` — 注册高层工具

## ADDED Requirements

### Requirement: list_providers 工具

系统 SHALL 提供一个名为 `list_providers` 的 MCP 工具，用于列出所有渠道并按分组展示。

#### Scenario: 正常获取并分组展示

- **WHEN** 用户调用 `list_providers` 工具
- **THEN** 系统调用 `GET /api/channel/` 获取所有渠道
- **AND** 按渠道的 `groups` 字段进行分组
- **AND** 每个分组内按 `priority` 降序排列
- **AND** 以结构化文本形式返回分组视图，包含每个渠道的：ID、名称、状态、模型列表、优先级

#### Scenario: 上游返回空列表

- **WHEN** 上游 `GET /api/channel/` 返回空数组 `[]`
- **THEN** 系统返回提示信息 "没有找到任何渠道"

#### Scenario: 上游返回错误

- **WHEN** 上游 `GET /api/channel/` 返回非 200 状态码或非 JSON 响应
- **THEN** 系统返回 `IsError=true`，包含错误描述

### Requirement: 分组展示格式

系统 SHALL 按以下格式展示按分组聚合的渠道视图：

```
## 分组名（default）
| ID | 名称 | 状态 | 模型 | 优先级 |
|----|------|------|------|--------|
| 1 | 渠道A | ✅ 启用 | gpt-4, gpt-3.5 | 1 |

## 分组名（vip）
| ID | 名称 | 状态 | 模型 | 优先级 |
|----|------|------|------|--------|
| 2 | 渠道B | ❌ 禁用 | claude-3 | 2 |
```

- 状态：status=1 显示 ✅ 启用，其他显示 ❌ 禁用
- 模型：以逗号分隔展示
- 分组为空时显示 "(无分组)"
- 若无渠道属于某分组则不展示该分组

### Requirement: 工具的 InputSchema

工具无必填参数，可选参数包括：
- `group` (string, optional): 筛选指定分组，只展示该分组下的渠道
- `status` (integer, optional): 筛选指定状态，1=启用，其他=禁用

### Requirement: 工具注册

`list_providers` SHALL 以 `list_providers` 名称注册到 MCP 服务器。
工具无 `api_` 前缀。注册时使用 `hightools.ToolDef.Handler` 直接作为 handler，而非通过 `handler.MakeHandler` 代理。

工具注册在 API 工具注册之后进行，以确保 API 工具配置已就绪。仅当 `cfg.SystemKey != "" && cfg.APIToolsEnabled` 时注册。
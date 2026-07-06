# Set Channel Priority Spec

## Why

渠道管理场景中，AI 需要能够调整渠道的优先级（priority）以控制请求分发权重。当前没有高层工具封装此操作，AI 需要通过自动生成的 OpenAPI proxy 调用 PUT /api/channel/{id} 并手动构造 body，增加了 AI 出错的概率。

## What Changes

- 新增 `internal/hightools/set_channel_priority.go` — `NewSetChannelPriorityTool(client, metrics)` 构造函数
- 更新 `internal/hightools/register.go` — 在 `RegisterAll()` 返回列表中追加新工具
- 新增 `internal/hightools/set_channel_priority_test.go` — 覆盖成功、验证、破坏三种场景

## Impact

- Affected specs: 新增 hightools 工具定义
- Affected code: `internal/hightools/set_channel_priority.go`（新建）、`internal/hightools/register.go`（追加一行）、`internal/hightools/set_channel_priority_test.go`（新建）
- No changes to existing OpenAPI proxy tools or packages outside hightools

## ADDED Requirements

### Requirement: set_channel_priority Tool

The system SHALL provide a high-level MCP tool named `set_channel_priority` that sets a channel's priority via PUT /api/channel/{id}.

#### Input Schema

```json
{
  "type": "object",
  "properties": {
    "id": {
      "type": "integer",
      "description": "Channel ID"
    },
    "priority": {
      "type": "integer",
      "description": "New priority value"
    }
  },
  "required": ["id", "priority"]
}
```

#### Scenario: Success case
- **WHEN** user calls `set_channel_priority` with valid `id` (integer) and `priority` (integer)
- **THEN** the tool sends `PUT /api/channel/{id}` with JSON body `{"priority": <priority>}` to upstream
- **AND** returns upstream response content as MCP tool result (not IsError)

#### Scenario: Non-integer priority rejected
- **WHEN** user calls `set_channel_priority` with `priority` that is not an integer (e.g. string "high", float 1.5)
- **THEN** the tool SHALL return `IsError=true` with an error message indicating invalid priority value
- **AND** SHALL NOT send any HTTP request to upstream

#### Scenario: Upstream failure
- **WHEN** upstream returns HTTP error (4xx/5xx) for the PUT request
- **THEN** the tool SHALL return `IsError=true` with upstream error details
- **AND** SHALL NOT panic or hang
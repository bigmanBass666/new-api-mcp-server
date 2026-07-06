# Set User Quota Spec

## Why

运维人员需要能够直接设置用户的配额值，而不必使用完整的"更新用户"端点。提供专注、安全且经过验证的配额设置工具，避免误操作。

## What Changes

- 新增 `set_user_quota` 高阶层 MCP 工具（hightool）
- 调用上游 `PUT /api/user/{id}` 接口，仅传递 `quota` 字段
- 参数：`id`（必填，整数用户 ID）、`quota`（必填，整数配额值）
- 验证逻辑：quota 必须是非负整数，拒绝负数、小数、字符串等无效值
- 注册到 `internal/hightools/register.go` 中的 `RegisterAll` 列表

## Impact

**BREAKING**: None. New tool addition only.

- Affected specs: 工具规范（新增 `set_user_quota`）
- Affected code:
  - `internal/hightools/set_user_quota.go`（新增）
  - `internal/hightools/set_user_quota_test.go`（新增）
  - `internal/hightools/register.go`（注册）

## ADDED Requirements

### Requirement: Set User Quota

The system SHALL provide a tool named `set_user_quota` that sets a user's quota by calling `PUT /api/user/{id}` with a JSON body containing only the `quota` field.

#### Scenario: Success case — set quota for existing user

- **GIVEN** a running MCP server with a valid `NEW_API_SYSTEM_KEY`
- **WHEN** user calls `set_user_quota` with `{"id": 1, "quota": 100000}`
- **THEN** the tool calls `PUT /api/user/1` with body `{"quota": 100000}`
- **AND** returns the upstream response

#### Scenario: Missing required parameter

- **GIVEN** the MCP server
- **WHEN** user calls `set_user_quota` without `id` or without `quota`
- **THEN** the tool returns an `IsError` result with a descriptive message

#### Scenario: Invalid quota value rejected

- **GIVEN** the MCP server
- **WHEN** user calls `set_user_quota` with `quota` being a string, negative number, or non-integer float
- **THEN** the tool returns an `IsError` result with a descriptive message
- **AND** does not make any upstream call

#### Scenario: Upstream error

- **GIVEN** the MCP server
- **WHEN** the upstream New API returns a non-2xx status
- **THEN** the tool returns an `IsError` result with the upstream error body
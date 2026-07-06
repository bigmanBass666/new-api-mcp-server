# 为 hightools 添加单元测试 Spec

## Why

`internal/hightools/` 有 10 个高级工具，但目前只有 6 个有测试文件。缺少测试的 4 个工具（`add_channel`、`toggle_channel`、`toggle_user_status`、`switch_group`）是直接操作 New API 后端的关键写操作工具，没有单元测试覆盖无法保证重构安全和 CI 质量。

## What Changes

- 为 4 个缺少测试的 hightools 添加单元测试文件
- 遵循现有 `set_channel_priority_test.go` 的测试模式（mock upstream、参数校验、错误路径）
- 不修改任何现有源代码，只新增测试文件

## Impact

- Affected specs: 无（纯新增测试文件）
- Affected code: 新增 4 个 `*_test.go` 文件在 `internal/hightools/`

## ADDED Requirements

### Requirement: add_channel 单元测试

文件 `internal/hightools/add_channel_test.go`

#### Scenario: 成功创建渠道
- **WHEN** 传入 `name="test"`, `type=1`, `key="sk-test"` 以及可选参数 `models="gpt-4"`, `group="default"`, `priority=5`
- **THEN** 调用 `POST /api/channel/`，请求体包含 `{"mode":"single","channel":{name,type,key,models,group,priority}}`

#### Scenario: 缺失必填参数
- **WHEN** 缺少 `name` / `type` / `key`
- **THEN** 返回 `IsError=true`，错误消息指明缺失参数名

#### Scenario: 参数类型错误
- **WHEN** `type` 传入字符串 / `priority` 传入浮点数
- **THEN** 返回 `IsError=true`，错误消息包含类型信息

#### Scenario: 上游错误
- **WHEN** mock 返回 500
- **THEN** 返回 `IsError=true`，透传上游响应体

### Requirement: toggle_channel 单元测试

文件 `internal/hightools/toggle_channel_test.go`

#### Scenario: 成功启用渠道
- **WHEN** 传入 `id=1`, `enabled=true`
- **THEN** 调用 `POST /api/channel/1/status`，请求体 `{"status":1}`

#### Scenario: 成功禁用渠道
- **WHEN** 传入 `id=1`, `enabled=false`
- **THEN** 调用 `POST /api/channel/1/status`，请求体 `{"status":2}`

#### Scenario: 缺失必填参数
- **WHEN** 缺少 `id` / `enabled`
- **THEN** 返回 `IsError=true`

#### Scenario: 参数类型错误
- **WHEN** `id` 传入字符串
- **THEN** 返回 `IsError=true`

#### Scenario: 上游错误
- **WHEN** mock 返回 500
- **THEN** 返回 `IsError=true`

### Requirement: toggle_user_status 单元测试

文件 `internal/hightools/toggle_user_status_test.go`

#### Scenario: 成功启用用户
- **WHEN** 传入 `id=2`, `enabled=true`
- **THEN** 调用 `POST /api/user/manage`，请求体 `{"id":2,"action":"enable"}`

#### Scenario: 成功禁用用户
- **WHEN** 传入 `id=2`, `enabled=false`
- **THEN** 调用 `POST /api/user/manage`，请求体 `{"id":2,"action":"disable"}`

#### Scenario: 缺失必填参数
- **WHEN** 缺少 `id` / `enabled`
- **THEN** 返回 `IsError=true`

#### Scenario: 参数类型错误
- **WHEN** `id` 传入字符串
- **THEN** 返回 `IsError=true`

#### Scenario: 上游错误
- **WHEN** mock 返回 500
- **THEN** 返回 `IsError=true`

### Requirement: switch_group 单元测试

文件 `internal/hightools/switch_group_test.go`

#### Scenario: 成功切换分组
- **WHEN** 传入 `token_id=5`, `group="vip"`
- **THEN** 调用 `PUT /api/token/`，请求体 `{"id":5,"group":"vip"}`

#### Scenario: 缺失必填参数
- **WHEN** 缺少 `token_id` / `group`
- **THEN** 返回 `IsError=true`

#### Scenario: 参数类型错误
- **WHEN** `token_id` 传入字符串 / `group` 传入空字符串
- **THEN** 返回 `IsError=true`

#### Scenario: 上游错误
- **WHEN** mock 返回 500
- **THEN** 返回 `IsError=true`
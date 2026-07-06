# v2 架构重构 Spec

## Why

现有 OpenAPI 规范是手工编写的，与真实 New API 存在严重偏差：Channel 模型缺失 18 个字段（含阻断性的 key 和 channel_info），Log 模型缺失 16/21 个字段，所有端点响应缺少 schema 引用。基于残缺规范"自动生成"的 225 个工具看似强大，实则关键业务路径不可用。

## What Changes

- **BREAKING**: 用运行时提取器替代手工编写的 OpenAPI 规范（api.json 将从提取器生成）
- **BREAKING**: 解析器增强以支持更丰富的 Schema（$ref、nullable、组合类型）
- **BREAKING**: 高层面工具从"API 端点驱动"改为"业务路径驱动"
- 新增规范提取管道 `internal/extractor/`
- 新增 7+ 个高层面工具覆盖完整业务路径
- 新增集成测试套件

## Impact

- Affected specs: 工具自动生成流程、高层面工具设计、测试体系
- Affected code: `openapi/`, `internal/openapi/`, `internal/hightools/`, `cmd/server/`
- New code: `internal/extractor/`

---

## ADDED Requirements

### Requirement: 规范提取器

The system SHALL provide a spec extraction tool that connects to a running New API instance and auto-generates complete OpenAPI specs.

#### Scenario: 成功提取 Channel Schema

- **WHEN** extractor connects to `localhost:4050` and requests `/api/channel/`
- **THEN** generated Channel schema includes: id, name, type, key, status, models, group, priority, weight, base_url, tag, channel_info (is_multi_key, multi_key_size, multi_key_mode), setting, settings, model_mapping, param_override, header_override, auto_ban, used_quota, balance, created_time, test_time, response_time, other, other_info, remark

#### Scenario: 提取输出是有效的 OpenAPI 3.0.1

- **WHEN** extractor completes
- **THEN** output can be loaded by `kin-openapi` parser without error

#### Scenario: 提取不修改 New API

- **WHEN** extractor runs
- **THEN** New API state is unchanged (read-only queries)

### Requirement: 解析器增强

The system SHALL enhance the OpenAPI parser to handle $ref resolution, nullable types, oneOf/anyOf composition, and response schemas.

#### Scenario: 解析带 $ref 的 Schema

- **WHEN** input spec contains `$ref: "#/components/schemas/Channel"`
- **THEN** parser resolves the reference and produces complete ToolDef with all referenced fields

### Requirement: 增强 add_channel 工具

The system SHALL enhance `add_channel` to support multi-key channel creation with mode and multi_key_mode parameters.

#### Scenario: 创建多 Key 轮转渠道

- **WHEN** user calls `add_channel(name="Agnes Cluster", type=1, key="key1\nkey2\nkey3", mode="multi_to_single", multi_key_mode="polling")`
- **THEN** creates a single channel with 3 keys in polling rotation mode

### Requirement: add_channel_keys 工具

The system SHALL provide a tool to append keys to an existing channel.

#### Scenario: 往已有渠道追加 Key

- **WHEN** user calls `add_channel_keys(channel_id=24, keys="key4\nkey5")`
- **THEN** channel 24 has its key list extended by 2 keys (total multi_key_size increased by 2)

### Requirement: 令牌管理工具集

The system SHALL provide `create_token`, `revoke_token`, and `list_tokens` tools.

#### Scenario: 创建令牌

- **WHEN** user calls `create_token(name="test-user-token", remained_quota=1000000, group="default")`
- **THEN** a new token is created and its key is returned

### Requirement: 日志查询工具

The system SHALL provide `search_logs` and `get_log_stats` tools for querying request logs and statistics.

---

## MODIFIED Requirements

### Requirement: 现有高层面工具保持兼容

All existing 10 high-level tools SHALL continue to work unchanged in their existing API shape. Only `add_channel` gains new optional parameters (`mode`, `multi_key_mode`) — existing single-key usage remains fully backward compatible.

---

## REMOVED Requirements

### Requirement: 手工维护 OpenAPI 规范

**Reason**: 手工编写的 api.json 和 relay.json 不可靠，已审计发现大量字段缺失和错误。改为提取器自动生成 + 引用官方规范。

**Migration**: 现有 api.json 作为提取器的"端点骨架"输入（只读路径和方法，schema 将被替换）。relay.json 将引用 OpenAI 官方规范 + 手工修复。
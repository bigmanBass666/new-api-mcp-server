# Add Channel Tool Spec

## Why

New API users need to add AI provider channels (e.g., OpenAI, Anthropic, Azure) through the MCP server. The raw POST /api/channel/ endpoint requires a complex nested JSON body (`{mode, channel: {name, type, ...}}`), making it error-prone for AI clients. A high-level `add_channel` tool provides a flat, documented parameter interface and handles the body construction internally.

## What Changes

- New file `internal/hightools/add_channel.go` — `NewAddChannelTool(client)` constructor returning `hightools.ToolDef`
- Update `internal/hightools/register.go` — add `NewAddChannelTool(client)` to `RegisterAll()` return list
- Update `cmd/server/main.go` — register high-level tools from `hightools.RegisterAll(client)`
- Integration test in `internal/hightools/add_channel_test.go` — mock upstream, verify tool is callable and channel appears in list

## Impact

- Affected specs: Phase 1 (High-level Tools), item 1.1.5 "add_channel"
- Affected code:
  - `internal/hightools/add_channel.go` (new)
  - `internal/hightools/register.go` (modify)
  - `cmd/server/main.go` (modify)
  - `internal/hightools/add_channel_test.go` (new)
- No breaking changes — existing tools unaffected

## Requirements

### Requirement: Channel Creation via High-Level Tool

The system SHALL provide an `add_channel` MCP tool that creates a new AI provider channel in the upstream New API instance.

#### Scenario: Create a channel with required fields
- **WHEN** user calls `add_channel` with `name` and `type`
- **THEN** tool constructs `{mode: "single", channel: {name, type, ...}}` body, sends POST /api/channel/
- **AND** returns the upstream response (channel creation result)

#### Scenario: Create a channel with all optional fields
- **WHEN** user calls `add_channel` with `name`, `type`, `models`, `group`, `priority`
- **THEN** all provided optional fields are included in the `channel` object of the request body

#### Scenario: Missing required field returns error
- **WHEN** user calls `add_channel` without `name` or `type`
- **THEN** tool returns `IsError=true` with a clear message about the missing required field

#### Scenario: Added channel appears in channel list
- **WHEN** channel is successfully created
- **THEN** subsequent `api_get_all_channels` call lists the new channel

## Tool Interface

### Name
`add_channel`

### Parameters
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| name      | string | yes | Channel name (e.g., "Azure OpenAI East US") |
| type      | integer | yes | Channel type code (e.g., 1=OpenAI, 2=Azure, 3=Custom) |
| models    | string | no | Comma-separated model list (e.g., "gpt-4,gpt-3.5-turbo") |
| group     | string | no | Channel group tag (e.g., "default", "vip") |
| priority  | integer | no | Channel priority (higher = preferred) |

### Implementation Details

The handler function will:
1. Validate required parameters (`name`, `type`)
2. Construct request body:
   ```json
   {
     "mode": "single",
     "channel": {
       "name": "<name>",
       "type": <type>,
       "models": "<models>",
       "group": "<group>",
       "priority": <priority>
     }
   }
   ```
3. Send POST /api/channel/ via `client.Do(ctx, SourceAPI, "POST", "/api/channel/", nil, nil, body)`
4. Return the upstream JSON response as-is (includes channel id on success)

### Error Handling
- Missing `name`: `IsError=true`, message "required parameter 'name' is missing"
- Missing `type`: `IsError=true`, message "required parameter 'type' is missing"
- Upstream failure (non-2xx): `IsError=true`, upstream error body forwarded
- Network/transport error: `IsError=true`, message "upstream error: <detail>"
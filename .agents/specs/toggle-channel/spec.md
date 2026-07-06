# Toggle Channel Spec

## Why

Add a dedicated `toggle_channel` MCP tool that enables or disables a channel by ID, replacing the need to manually construct a raw PUT request to `/api/channel/{id}`. This is a frequent admin operation in New API management.

## What Changes

- Create a new high-level tool `toggle_channel` in `internal/hightools/toggle_channel.go`
- Register it in `internal/hightools/register.go` under the `RegisterAll()` function
- The tool accepts `id` (integer, required) and `enabled` (boolean, required)
- Sends PUT to `/api/channel/{id}` with body `{"enabled": true|false}`
- Uses `SourceAPI` (system key) for authentication
- Returns the upstream response directly (success or error)
- Tool is registered alongside existing high-level tools, no new config toggles needed

## Impact

- Affected specs: Channel management capability
- Affected code:
  - `internal/hightools/toggle_channel.go` (new)
  - `internal/hightools/register.go` (modify — import and register)

## Non-Goals

- No changes to the OpenAPI spec parser (this is a high-level tool, not auto-generated)
- No additional config flags or env vars
- No batch toggle operation — single channel toggle only

## Requirements

### Requirement: Toggle Channel Tool

The system SHALL provide a `toggle_channel` tool that toggles a channel's enabled state.

#### Scenario: Success case
- **WHEN** user calls `toggle_channel` with integer `id` and boolean `enabled`
- **THEN** the system sends PUT `/api/channel/{id}` with body `{"enabled": <enabled>}`
- **AND** returns the upstream API response

#### Scenario: Missing required parameters
- **WHEN** user calls `toggle_channel` without `id` or `enabled`
- **THEN** the system returns an error indicating the missing required parameter

#### Scenario: Invalid parameter types
- **WHEN** user calls `toggle_channel` with non-integer `id` or non-boolean `enabled`
- **THEN** the tool returns a clear error message

#### Scenario: Upstream error
- **WHEN** the upstream API returns a non-2xx response
- **THEN** the tool returns the error with upstream status code
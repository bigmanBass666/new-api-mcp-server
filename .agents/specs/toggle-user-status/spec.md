# Toggle User Status Spec

## Why

Add a dedicated `toggle_user_status` MCP tool that enables or disables a user by ID, replacing the need to manually construct a raw PUT request to `/api/user/{id}`. This is a frequent admin operation in New API user management.

## What Changes

- Create a new high-level tool `toggle_user_status` in `internal/hightools/toggle_user_status.go`
- Register it in `internal/hightools/register.go` under the `RegisterAll()` function
- The tool accepts `id` (integer, required) and `enabled` (boolean, required)
- Sends PUT to `/api/user/{id}` with body `{"enabled": true|false}`
- Uses `SourceAPI` (system key) for authentication
- Returns the upstream response directly (success or error)
- Tool is registered alongside existing high-level tools, no new config toggles needed

## Impact

- Affected specs: User management capability
- Affected code:
  - `internal/hightools/toggle_user_status.go` (new)
  - `internal/hightools/register.go` (modify — import and register)

## Non-Goals

- No changes to the OpenAPI spec parser (this is a high-level tool, not auto-generated)
- No additional config flags or env vars
- No batch toggle operation — single user toggle only
- No user creation or deletion — toggle status only

## Requirements

### Requirement: Toggle User Status Tool

The system SHALL provide a `toggle_user_status` tool that toggles a user's enabled state.

#### Scenario: Success case
- **WHEN** user calls `toggle_user_status` with integer `id` and boolean `enabled`
- **THEN** the system sends PUT `/api/user/{id}` with body `{"enabled": <enabled>}`
- **AND** returns the upstream API response

#### Scenario: Missing required parameters
- **WHEN** user calls `toggle_user_status` without `id` or `enabled`
- **THEN** the system returns an error indicating the missing required parameter

#### Scenario: Invalid parameter types
- **WHEN** user calls `toggle_user_status` with non-integer `id` or non-boolean `enabled`
- **THEN** the tool returns a clear error message

#### Scenario: Upstream error
- **WHEN** the upstream API returns a non-2xx response
- **THEN** the tool returns the error with upstream status code
# Switch Token Group Spec

## Why

New API users need to change a token's group assignment. The raw PUT `/api/token/` endpoint requires constructing the full body with the token ID and group field. A dedicated `switch_group` MCP tool provides a simple, focused interface that accept a token ID and a new group name, performs the underlying GET+PUT workflow, and returns the result.

## What Changes

- Create a new high-level tool `switch_group` in `internal/hightools/switch_group.go`
- Register it in `internal/hightools/register.go` under the `RegisterAll()` function
- The tool accepts `token_id` (integer, required) and `group` (string, required)
- Workflow:
  1. GET `/api/token/{token_id}` to retrieve current token data (validate token exists)
  2. PUT `/api/token/` with body `{"id": <token_id>, "group": "<group>"}` to update the group
- Uses `client.SourceAPI` (system key) for authentication
- Returns the upstream PUT response directly (success or error)
- Tool is registered alongside existing high-level tools, no new config toggles needed

## Impact

- Affected specs: Token management capability
- Affected code:
  - `internal/hightools/switch_group.go` (new)
  - `internal/hightools/register.go` (modify — import and register)

## Non-Goals

- No changes to the OpenAPI spec parser (this is a high-level tool, not auto-generated)
- No additional config flags or env vars
- No batch switch operation — single token group switch only
- No validation of group existence (upstream API handles this)

## Requirements

### Requirement: Switch Group Tool

The system SHALL provide a `switch_group` tool that changes a token's group.

#### Scenario: Success case
- **WHEN** user calls `switch_group` with integer `token_id` and string `group`
- **THEN** the system first GETs `/api/token/{token_id}` to validate the token
- **AND** sends PUT `/api/token/` with body `{"id": <token_id>, "group": "<group>"}`
- **AND** returns the upstream API response

#### Scenario: Missing required parameters
- **WHEN** user calls `switch_group` without `token_id` or `group`
- **THEN** the system returns an error indicating the missing required parameter

#### Scenario: Invalid parameter types
- **WHEN** user calls `switch_group` with non-integer `token_id`
- **THEN** the tool returns a clear error message

#### Scenario: Token not found (GET fails)
- **WHEN** the upstream GET `/api/token/{id}` returns a non-2xx response
- **THEN** the tool returns the error with upstream status code

#### Scenario: Upstream PUT error
- **WHEN** the upstream PUT `/api/token/` returns a non-2xx response
- **THEN** the tool returns the error with upstream status code
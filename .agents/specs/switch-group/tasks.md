# Tasks

- [x] Task 1: Create switch_group.go — high-level tool with constructor function
  - [x] Implement `NewSwitchGroupTool(client, metrics)` returning `ToolDef`
  - [x] Tool name: `switch_group`
  - [x] Input schema: `token_id` (integer, required), `group` (string, required)
  - [x] Handler logic:
    1. Validate params (`token_id`, `group`)
    2. PUT `/api/token/` with body `{"id": <token_id>, "group": "<group>"}`
    3. Return upstream response
  - [x] Error handling for missing/invalid params and upstream PUT failure
  - [x] Metrics recording (ToolRequestsTotal, ToolRequestDuration, UpstreamRequestsTotal, UpstreamRequestDuration)

- [x] Task 2: Register the tool in register.go
  - [x] No new import needed (same client/observability packages already imported)
  - [x] Add `NewSwitchGroupTool(client, metrics),` to the `RegisterAll()` return slice

# Task Dependencies

- Task 2 depends on Task 1
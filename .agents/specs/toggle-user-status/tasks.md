# Tasks

- [x] Task 1: Create toggle_user_status.go — high-level tool with constructor function
  - Implement `NewToggleUserStatusTool(client, metrics)` returning `ToolDef`
  - Tool name: `toggle_user_status`
  - Input schema: `id` (integer, required), `enabled` (boolean, required)
  - Handler logic: validate params, build body, call PUT `/api/user/{id}`, return response
  - Handler uses `client.Do(ctx, client.SourceAPI, "PUT", path, nil, nil, body)`
  - Proper error handling for missing/invalid params
  - Follow the exact same pattern as `toggle_channel.go`

- [x] Task 2: Register the tool in register.go
  - Add import for the new file (same package, no import needed beyond existing `client` and `observability`)
  - Add `NewToggleUserStatusTool(client, metrics),` to the `RegisterAll()` return slice

# Task Dependencies

- Task 2 depends on Task 1
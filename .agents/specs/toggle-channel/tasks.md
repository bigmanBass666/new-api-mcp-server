# Tasks

- [x] Task 1: Create toggle_channel.go — high-level tool with constructor function
  - Implement `NewToggleChannelTool(client, metrics)` returning `ToolDef`
  - Tool name: `toggle_channel`
  - Input schema: `id` (integer, required), `enabled` (boolean, required)
  - Handler logic: validate params, build body, call PUT `/api/channel/{id}`, return response
  - Handler uses `client.Do(ctx, client.SourceAPI, "PUT", path, nil, nil, body)`
  - Proper error handling for missing/invalid params

- [x] Task 2: Register the tool in register.go
  - Add import for the new file
  - Add `NewToggleChannelTool(client, metrics),` to the `RegisterAll()` return slice

# Task Dependencies

- Task 2 depends on Task 1
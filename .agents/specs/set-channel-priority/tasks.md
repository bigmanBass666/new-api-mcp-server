# Tasks

- [x] Task 1: Implement `set_channel_priority` tool — Create `internal/hightools/set_channel_priority.go` with `NewSetChannelPriorityTool(client, metrics)` constructor
  - Function signature: `NewSetChannelPriorityTool(client *client.Client, metrics *observability.Metrics) ToolDef`
  - Tool name: `set_channel_priority`
  - Input schema: `id` (integer, required), `priority` (integer, required)
  - Handler logic:
    1. Parse `id` and `priority` from arguments (validate types)
    2. Marshal JSON body: `{"priority": priority}`
    3. Call `client.Do(ctx, client.SourceAPI, "PUT", "/api/channel/{id}", nil, nil, body)` with id substituted in path
    4. Return upstream response content
  - Error handling: Invalid type for id/priority → return IsError=true without upstream call
  - Error handling: Upstream failure → return IsError=true with upstream error

- [x] Task 2: Register tool in `RegisterAll()` — Add one line to `internal/hightools/register.go`
  - Append `NewSetChannelPriorityTool(client, metrics)` to the returned slice
  - Note: constructor parameters (client, metrics) will be passed by the caller when integration wiring is done

- [x] Task 3: Write tests — Create `internal/hightools/set_channel_priority_test.go` with:
  - **Success test**: mock upstream returns 200, verify tool returns success with expected body
  - **Destructive test**: non-integer priority (string) → IsError=true, no upstream call
  - **Destructive test**: non-integer priority (float) → IsError=true, no upstream call
  - **Upstream failure test**: mock upstream returns 500 → IsError=true with error details

## Task Dependencies

- Task 2 depends on Task 1 (can be done together in same PR)
- Task 3 can be done in parallel with Task 1+2
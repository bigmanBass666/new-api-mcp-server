# Checklist: Add Channel Tool

## Code Implementation
- [ ] `internal/hightools/add_channel.go` exists with `NewAddChannelTool` function
- [ ] `NewAddChannelTool` validates required params (`name`, `type`)
- [ ] `NewAddChannelTool` constructs proper request body with `mode: "single"` and `channel` object
- [ ] Optional fields (`models`, `group`, `priority`) are included when provided
- [ ] Tool name is `add_channel` matching `[a-zA-Z0-9_\-.]` pattern
- [ ] Handler uses `client.SourceAPI` for upstream calls
- [ ] `internal/hightools/register.go` includes `NewAddChannelTool(client)` in return slice
- [ ] `cmd/server/main.go` registers high-level tools from `hightools.RegisterAll()`

## Tests
- [ ] Test: create channel with only required fields (`name`, `type`) succeeds
- [ ] Test: create channel with all fields (`name`, `type`, `models`, `group`, `priority`) succeeds
- [ ] Test: missing `name` returns `IsError=true`
- [ ] Test: missing `type` returns `IsError=true`
- [ ] Test: upstream error response returns `IsError=true` with error content forwarded
- [ ] Tests use httptest to mock upstream New API

## Build and Quality
- [ ] `go build ./cmd/server` compiles without errors
- [ ] `go test ./internal/hightools/ -v -run TestAddChannel` passes
- [ ] `go test ./... -v -race -count=1` passes
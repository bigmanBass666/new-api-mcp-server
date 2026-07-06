# Tasks: Add Channel Tool

## Tasks
- [ ] Task 1: Create add_channel tool implementation
  - [ ] 1.1 Create `internal/hightools/add_channel.go` with `NewAddChannelTool(client)` constructor
    - Input params: name (string, required), type (int, required), models (string, optional), group (string, optional), priority (int, optional)
    - Construct `{mode: "single", channel: {name, type, models, group, priority}}` body
    - Call `client.Do(ctx, SourceAPI, "POST", "/api/channel/", nil, nil, body)`
    - Return upstream response text
  - [ ] 1.2 Update `internal/hightools/register.go` — add `NewAddChannelTool(client)` to RegisterAll() return list
  - [ ] 1.3 Update `cmd/server/main.go` — register high-level tools from hightools.RegisterAll() with client injection

- [ ] Task 2: Write integration test
  - [ ] 2.1 Create `internal/hightools/add_channel_test.go`
  - [ ] 2.2 Mock upstream: POST /api/channel/ returns success JSON with channel id
  - [ ] 2.3 Mock upstream: GET /api/channel/ returns list containing the created channel
  - [ ] 2.4 Test: create channel with required fields only
  - [ ] 2.5 Test: create channel with all fields
  - [ ] 2.6 Test: missing required fields returns error
  - [ ] 2.7 Test: upstream error returns IsError=true

- [ ] Task 3: Verify compilation and tests pass
  - [ ] 3.1 `go build ./cmd/server` — zero errors
  - [ ] 3.2 `go test ./internal/hightools/ -v -run TestAddChannel` — all pass
  - [ ] 3.3 `go test ./... -v -race -count=1` — all passing

## Task Dependencies
- Task 1 and Task 2 are parallel (code and tests)
- Task 3 depends on both Task 1 and Task 2
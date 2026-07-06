# Checklist

## Code Implementation
- [x] `internal/hightools/switch_group.go` exists with `NewSwitchGroupTool` function
- [x] `NewSwitchGroupTool` validates required params (`token_id`, `group`)
- [x] `switch_group` handler validates token_id is integer, group is non-empty string
- [x] PUT response error is forwarded to caller
- [x] Tool name is `switch_group` matching `[a-zA-Z0-9_\-.]` pattern
- [x] Handler uses `client.SourceAPI` for upstream calls
- [x] `internal/hightools/register.go` includes `NewSwitchGroupTool(client, metrics)` in return slice

## Build and Quality
- [x] `go build ./cmd/server` compiles without errors
- [x] `go build ./...` compiles without errors

## Integration (Optional)
- [ ] (Integration) Verify switch_group actually changes a token's group via a running New API instance
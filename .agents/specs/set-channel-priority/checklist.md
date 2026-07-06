# Checklist

- [x] Task 1: `<internal/hightools/set_channel_priority.go>` implements `NewSetChannelPriorityTool` with correct input schema and handler logic
- [x] `id` parameter is required with integer type in input schema
- [x] `priority` parameter is required with integer type in input schema
- [x] Handler validates priority is integer before making upstream call
- [x] PUT /api/channel/{id} is called with JSON body `{"priority": <priority>}`
- [x] `register.go` includes `NewSetChannelPriorityTool` in `RegisterAll()` returned list
- [x] Test: valid id+priority → upstream PUT called with correct path and body → success response returned (IsError=false)
- [x] Test: string priority "high" → IsError=true, no upstream call made
- [x] Test: float priority 1.5 → IsError=true, no upstream call made
- [x] Test: upstream returns 500 → IsError=true with error message, no panic
- [x] All tests pass: `go test ./internal/hightools/... -v -count=1`
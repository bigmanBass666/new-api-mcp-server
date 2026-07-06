# Checklist

- [x] Spec file(s) created under internal/hightools/toggle_channel.go
- [x] Tool registered in internal/hightools/register.go
- [x] Tool compiles: `go build ./...` passes
- [x] Tool tests pass: `go test ./internal/hightools/...` passes (or no tests required if in same package)
- [x] Full project compiles: `make build` passes (verified via `go build ./...`)
- [ ] (Integration) Verify toggle_channel actually changes channel status via a running New API instance
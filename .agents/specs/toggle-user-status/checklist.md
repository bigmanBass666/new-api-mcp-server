# Checklist

- [x] Spec file(s) created under internal/hightools/toggle_user_status.go
- [x] Tool registered in internal/hightools/register.go
- [x] Tool compiles: `go build ./...` passes
- [x] Full project compiles: `make build` passes (verified via `go build ./...`)
- [ ] (Integration) Verify toggle_user_status actually changes user status via a running New API instance
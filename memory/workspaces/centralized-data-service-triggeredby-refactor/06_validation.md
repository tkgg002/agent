# Validation Plan

- `go test ./internal/activity ./internal/handler`
- If impacted dependencies require broader confidence: `go test ./...`
- Build worker binary if tests pass far enough: `go build ./cmd/worker`
- Security/review scan:
  - changed files list via `git diff --name-only`
  - hardcoded secret scan on changed files
  - raw SQL/user input scan where applicable


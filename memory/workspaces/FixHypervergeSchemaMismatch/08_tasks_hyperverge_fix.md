# Task List: Remove Forced "_id" -> "id" Overrides in Handlers

## Tasks
- [ ] **Task 1:** Remove forced override `if pgPKField == "_id" { pgPKField = "id" }` (lines 353-355) and `if !mappedPK && pkField == "_id" { pgPKField = "id" }` (lines 384-386) in `internal/handler/shadow/event_handler.go`.
- [ ] **Task 2:** Remove forced override `if resolved.pgPKField == "" || resolved.pgPKField == "_id" { resolved.pgPKField = "id" }` (lines 281-283) in `internal/handler/source/bridge_handler.go`.
- [ ] **Task 3:** Run `go test ./internal/handler/shadow/...` and `go test ./internal/handler/source/...` to verify test suite passes cleanly.
- [ ] **Task 4:** Pass Governance verification linter (`python3 agent/tooling/verify_governance.py`).

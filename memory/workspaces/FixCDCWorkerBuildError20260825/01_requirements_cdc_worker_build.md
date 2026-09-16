# Requirements - Fix CDC Worker Build Error (declared and not used: mappedPK)

## 1. Scope & Objective
Fix compile-time error in `centralized-data-service`:
`internal/handler/shadow/event_handler.go:371:3: declared and not used: mappedPK`
occurring during `go build` for `cdc-worker`.

## 2. Requirements Traceability
- [x] Fix unused variable `mappedPK` in `internal/handler/shadow/event_handler.go`.
- [x] Ensure `go build ./cmd/worker/` compiles cleanly without warnings/errors.
- [x] Pass all unit tests in `centralized-data-service`.

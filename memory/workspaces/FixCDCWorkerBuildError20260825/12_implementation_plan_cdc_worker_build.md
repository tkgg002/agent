# Implementation Plan - Fix CDC Worker Build Error

## Goal
Resolve `declared and not used: mappedPK` compilation error in `internal/handler/shadow/event_handler.go` of `centralized-data-service`.

## Proposed Changes
### `centralized-data-service`
#### [MODIFY] [event_handler.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/shadow/event_handler.go#L370-L380)
- Remove unused variable declarations `mappedPK := false` and `mappedPK = true`.

## Verification Plan
1. Run `CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o cdc-worker ./cmd/worker/` in `/Users/trainguyen/Documents/work/data-hub/centralized-data-service`.
2. Run `go test ./internal/handler/shadow/...`.

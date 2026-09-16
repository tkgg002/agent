# Task List - Fix CDC Worker Build Error

- [x] Task 1: Remove unused variable `mappedPK` from `internal/handler/shadow/event_handler.go`.
- [x] Task 2: Run `CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o cdc-worker ./cmd/worker/` to verify clean build.
- [x] Task 3: Clean up transient binaries.

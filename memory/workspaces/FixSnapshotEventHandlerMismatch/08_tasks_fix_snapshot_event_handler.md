# Tasks Checklist: Fix Snapshot EventHandler Mismatch

- [x] Task 1: Sửa interface `snapshotEventHandler` trong `internal/handler/orchestration/snapshot_runner_handler.go` về 3 tham số.
- [x] Task 2: Sửa câu lệnh gọi `r.eventHandler.HandleRaw(ctx, subject, nil, envelope)` thành `r.eventHandler.HandleRaw(ctx, subject, envelope)` tại `snapshot_runner_handler.go`.
- [x] Task 3: Chạy `go test ./internal/handler/orchestration/...` và `go check` / `go build ./cmd/worker/main.go` để xác nhận lỗi biên dịch đã được khắc phục hoàn toàn.

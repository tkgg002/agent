# Kế hoạch Khắc phục Lỗi Biên dịch snapshotEventHandler.HandleRaw Mismatch

Khắc phục lỗi biên dịch Go khi chạy `make run` (`go run cmd/worker/main.go`).

## Proposed Changes

### `centralized-data-service/internal/handler/orchestration`

#### [MODIFY] `snapshot_runner_handler.go`
- Sửa interface `snapshotEventHandler` dòng 54-58.
- Sửa điểm gọi `r.eventHandler.HandleRaw` tại dòng 751.

## Verification Plan
1. `go build ./cmd/worker/main.go` -> PASS
2. `go test -v ./internal/handler/orchestration/...` -> PASS

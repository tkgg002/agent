# Kế hoạch Khắc phục Lỗi Biên dịch snapshotEventHandler.HandleRaw Mismatch

Khắc phục lỗi biên dịch Go khi chạy `make run` (`go run cmd/worker/main.go`):
`internal/server/server_setup.go:358:3: cannot use eventHandler (variable of type *"centralized-data-service/internal/handler/shadow".EventHandler) as "centralized-data-service/internal/handler/orchestration".snapshotEventHandler value in argument to handlerorchestration.NewSnapshotRunner`.

## Proposed Changes

### `centralized-data-service/internal/handler/orchestration`

#### [MODIFY] `snapshot_runner_handler.go`
- Sửa interface `snapshotEventHandler` dòng 54-58:
  ```go
  type snapshotEventHandler interface {
      HandleRaw(ctx context.Context, subject string, data []byte) (int, error)
      FlushBatchBuffer(ctx context.Context) (int, error)
      FlushCache()
  }
  ```
- Sửa điểm gọi `r.eventHandler.HandleRaw` tại dòng 751:
  ```go
  written, err := r.eventHandler.HandleRaw(ctx, subject, envelope)
  ```

## Verification Plan
1. `go build ./cmd/worker/main.go` -> PASS
2. `go test -v ./internal/handler/orchestration/...` -> PASS

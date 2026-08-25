# Hồ sơ Giải pháp Kỹ thuật: Fix Snapshot EventHandler Mismatch

## 1. Nguyên nhân Gốc rễ (Root Cause Analysis)
Trong đợt cập nhật gần đây ở file `snapshot_runner_handler.go`:
- Interface `snapshotEventHandler` bị đổi chữ ký (signature) của `HandleRaw`:
  ```go
  type snapshotEventHandler interface {
      HandleRaw(ctx context.Context, subject string, key, data []byte) (int, error)
      FlushBatchBuffer(ctx context.Context) (int, error)
      FlushCache()
  }
  ```
- Tuy nhiên, struct `EventHandler` trong package `internal/handler/shadow/event_handler.go` và toàn bộ codebase (bao gồm `recon_base_handler.go`, `kafka_consumer.go`, `snapshot_runner_test.go`) vẫn giữ nguyên chữ ký chuẩn 3 tham số:
  ```go
  func (h *EventHandler) HandleRaw(ctx context.Context, subject string, data []byte) (rows int, err error)
  ```
- Sự lệch chữ ký phương thức khiến `*"centralized-data-service/internal/handler/shadow".EventHandler` không còn thỏa mãn interface `snapshotEventHandler` của package `orchestration`, dẫn tới lỗi biên dịch Go tại `internal/server/server_setup.go:358`.

## 2. Phương án Khắc phục Tối ưu (Single Best Approach)
Khôi phục lại chữ ký chuẩn 3 tham số cho interface `snapshotEventHandler` và điểm gọi `HandleRaw` trong `snapshot_runner_handler.go`:

### Chi tiết thay đổi:
File: `centralized-data-service/internal/handler/orchestration/snapshot_runner_handler.go`

```diff
type snapshotEventHandler interface {
-	HandleRaw(ctx context.Context, subject string, key, data []byte) (int, error)
+	HandleRaw(ctx context.Context, subject string, data []byte) (int, error)
	FlushBatchBuffer(ctx context.Context) (int, error)
	FlushCache()
}
```

Và tại line 748 (hoặc 751):
```diff
-	written, err := r.eventHandler.HandleRaw(ctx, subject, nil, envelope)
+	written, err := r.eventHandler.HandleRaw(ctx, subject, envelope)
```

## 3. Kế hoạch Kiểm thử & Kiểm định (Verification)
1. Thử nghiệm biên dịch gói `internal/server` và `cmd/worker/main.go`:
   `go build ./cmd/worker/main.go`
2. Chạy test suite `internal/handler/orchestration/...`:
   `go test -v ./internal/handler/orchestration/...`

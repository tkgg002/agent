# Requirements: Fix Snapshot EventHandler Interface Mismatch

## Context & Background
Khi thực hiện command `make run` tại repository `centralized-data-service`, biên dịch Go thất bại tại `internal/server/server_setup.go:358`:
```
cannot use eventHandler (variable of type *"centralized-data-service/internal/handler/shadow".EventHandler) as "centralized-data-service/internal/handler/orchestration".snapshotEventHandler value in argument to handlerorchestration.NewSnapshotRunner: *"centralized-data-service/internal/handler/shadow".EventHandler does not implement "centralized-data-service/internal/handler/orchestration".snapshotEventHandler (wrong type for method HandleRaw)
have HandleRaw(context.Context, string, []byte) (int, error)
want HandleRaw(context.Context, string, []byte, []byte) (int, error)
```

## Functional & Technical Requirements
1. **Khôi phục Interface Consistency:** Đồng bộ lại signature của method `HandleRaw` trong interface `snapshotEventHandler` tại `internal/handler/orchestration/snapshot_runner_handler.go` về đúng signature chuẩn hệ thống: `HandleRaw(ctx context.Context, subject string, data []byte) (int, error)`.
2. **Cập nhật Call Site:** Sửa câu lệnh gọi `HandleRaw` trong `snapshot_runner_handler.go` để truyền đúng 3 tham số `(ctx, subject, envelope)`.
3. **Biên dịch & Verify:** Đảm bảo `go check` / `go build` / `make run` và unit tests pass 100% không còn lỗi biên dịch.

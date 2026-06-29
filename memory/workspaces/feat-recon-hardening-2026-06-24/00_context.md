# Context — feat-recon-hardening-2026-06-24

**Project**: centralized-data-service (data-hub)
**Created**: 2026-06-24T17:14 +07:00
**Scope**: Harden Reconcile ALL pipeline + CDC Observability via NATS

## Mục tiêu

Phân tích phiên trước (doc-architecture-flow-2026-06-22) phát hiện 3 hidden bottleneck trong
luồng `Reconcile ALL` cần được fix để hệ thống có thể scale up an toàn:

1. **Advisory Lock Leak** — pg_try_advisory_lock là Session-level lock; GORM pool có thể
   trả connection về pool trước khi unlock → lock bị kẹt vĩnh viễn.

2. **Adaptive Freeze Blindness** — Khi lag > 60 phút (max clamp), `adaptiveFreeze` trả 60m
   nhưng `pickScanRangeWithLag` vẫn tiếp tục gọi `EstimatedCount` → False Drift alarm.

3. **Thundering Herd** — Khi nhiều bảng cùng drift, tất cả goroutines cùng gọi `BucketCounts`
   (MongoDB aggregate) mà không có rate limiting riêng → spike tải MongoDB.

4. **CDC Observability** — Cần realtime monitoring cho SigNoz qua NATS events mà không tác
   động đến DB production.

## Service liên quan

- `centralized-data-service` (Go)
- `internal/service/recon/recon_tier_a.go` — withTableLock, adaptiveFreeze, RunTier1
- `internal/service/recon/recon_engine_run.go` — CheckAll goroutine orchestration

## Files cần chỉnh sửa

- `recon_tier_a.go` — Fix 1 (lock pinning), Fix 2 (lag circuit breaker)
- `recon_engine_run.go` — Fix 3 (drill-down semaphore)
- `pkgs/metrics/` — Thêm metric mới (optional Fix 4)

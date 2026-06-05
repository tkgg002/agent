# Context — fix-slow-sql-queries-2026-05-29

## Trigger
User dán 3 dòng log SLOW SQL từ prod (≥200ms), kèm directive: **"triệt để mấy cái này"**.

```
SLOW SQL >= 200ms [268.108125ms] [rows:0] UPDATE cdc_system.cdc_jobs ...
   stuck_job_reaper.go:127

SLOW SQL >= 200ms [260.196ms] [rows:0] SELECT ... FROM cdc_system.cdc_activity_log
   system_health_queries.go:95

SLOW SQL >= 200ms [398.064375ms] [rows:1] SELECT count(*) FROM cdc_system.failed_sync_logs
   system_health_queries.go:74
```

## Phạm vi
- Service: `cdc-cms-service`
- Component: `internal/infra/observability/system_health_queries.go`, `internal/infra/messaging/stuck_job_reaper.go`
- DB: `gpay-postgres-cdc` / schema `cdc_system` (PG 16, RANGE partitioned)
- Frequency: 3 query đều chạy mỗi 15–30s → tổng impact ≈ (268+260+398)/45 ≈ 21ms/s wasted DB time

## Đối tượng kỹ thuật
- `cdc_jobs`: bảng monolithic (~vài ngàn rows), index `(type, status, created_at DESC)` — không phù hợp filter `status='running'`.
- `cdc_activity_log`: RANGE partitioned daily, index `idx_act_created (created_at DESC, id)` + `idx_act_started (started_at DESC)`.
- `failed_sync_logs`: RANGE partitioned monthly, index `idx_fsl_created (created_at DESC, id)`.
- Migration runner: `internal/migrate/runner.go` wrap mỗi file trong tx → KHÔNG thể dùng `CONCURRENTLY`.

## Constraint
- §11 GEMINI: APPEND-only memory.
- §12: Brain không sửa code. Phiên hiện tại là Muscle (CC CLI) trực tiếp execute.
- Migration runner: dùng `CREATE INDEX IF NOT EXISTS` regular — partial idx (status='running') subset rất nhỏ nên lock acceptable.

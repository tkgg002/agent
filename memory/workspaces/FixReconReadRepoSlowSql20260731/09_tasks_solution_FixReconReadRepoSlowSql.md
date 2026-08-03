# Hồ Sơ Giải Pháp Kỹ Thuật (Technical Solution Profile)

## 1. Migration File SQL Mới
Tạo `migrations/schema/recon_dlq/101_optimize_recon_read_indexes.sql`:
```sql
-- ------------------------------------------------------------
-- Composite Indexes for cdc_system.recon_runs & cdc_recon_smoke_result
-- ------------------------------------------------------------
CREATE INDEX IF NOT EXISTS idx_recon_runs_tier_started ON cdc_system.recon_runs (tier, started_at DESC);
CREATE INDEX IF NOT EXISTS idx_smoke_result_checked_at ON cdc_system.cdc_recon_smoke_result (checked_at DESC);
```

## 2. Refactor Code Go: `internal/infra/persistence/recon/recon_read_repo_gorm.go`

### A. Tối ưu `listLatestPrimary` (API `/api/reconciliation/report`):
Bổ sung cờ time-window pruning `WHERE checked_at >= NOW() - INTERVAL '7 days'` trong CTE `smoke_latest`:
```sql
smoke_latest AS (
    SELECT DISTINCT ON (COALESCE(shadow_schema, ''), shadow_table, COALESCE(NULLIF(master_schema, ''), ''), COALESCE(NULLIF(master_table, ''), ''), COALESCE(segment, 'source_shadow'))
           id, run_id, trace_id, cycle_id, segment, source_type, source_host, source_db,
           ...
    FROM cdc_system.cdc_recon_smoke_result
    WHERE checked_at >= NOW() - INTERVAL '7 days'
    ORDER BY COALESCE(shadow_schema, ''), shadow_table, COALESCE(NULLIF(master_schema, ''), ''), COALESCE(NULLIF(master_table, ''), ''), COALESCE(segment, 'source_shadow'), checked_at DESC
)
```

### B. Tối ưu `GetBackfillStatus` (API `/api/recon/backfill-source-ts/status`):
Bổ sung cờ time pruning `created_at / started_at` và tận dụng index `(tier, started_at DESC)`:
```go
q := r.db.WithContext(ctx).
    Table("recon_runs").
    Where("tier = ? AND started_at >= NOW() - INTERVAL '7 days'", 4).
    Where("instance_id LIKE ?", "backfill:%").
    Order("started_at DESC").
    Limit(limit)
```

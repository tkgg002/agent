# Plan

## Q1 — cdc_jobs UPDATE 268ms (stuck_job_reaper)
**Root cause**: filter `status='running' AND started_at IS NOT NULL`. Hiện chỉ có `idx_cdc_jobs_type_status (type, status, created_at)` — leading column `type`, không trỏ thẳng vào status → seq scan toàn bảng mỗi 30s.

**Fix**: partial B-tree index trên `started_at`, scope `WHERE status='running'`:
```sql
CREATE INDEX IF NOT EXISTS idx_cdc_jobs_running_started
    ON cdc_system.cdc_jobs (started_at)
    WHERE status = 'running';
```
- Lá rất nhỏ (running set luôn là subset bé).
- Reaper filter `started_at + interval < NOW()` áp dụng sau khi partial idx narrow → ~<10ms thay vì 268ms.
- Migration `053_perf_idx_cdc_jobs_running.sql`, regular CREATE (runner wrap tx, không CONCURRENTLY).

## Q2 — cdc_activity_log SELECT 260ms (queryRecentEvents)
**Root cause**: WHERE filter `created_at > ?`, nhưng ORDER BY dùng `started_at DESC` → mismatch index leading column. Index `idx_act_created (created_at DESC, id)` tồn tại; `idx_act_started` riêng → planner phải sort sau khi scan.

**Fix**: đổi ORDER BY từ `started_at DESC` → `created_at DESC`. Cả 2 cột đều default `NOW()` tại INSERT → semantic identical cho "10 events mới nhất".

## Q3 — failed_sync_logs COUNT 398ms (queryFailedCount)
**Root cause**: 2 query `Count()` riêng biệt → mỗi cái mở 24h partition window → tổng 2x partition scan.

**Fix**: gộp về 1 Raw query với FILTER expression:
```sql
SELECT
  count(*) AS count_24h,
  count(*) FILTER (WHERE created_at > $1) AS count_1h
FROM cdc_system.failed_sync_logs
WHERE created_at > $2 AND created_at <= $3
```
Outer WHERE prune partition; inner FILTER count subset trong cùng 1 scan.

## Sequence
1. Read tất cả file liên quan (stuck_job_reaper.go, system_health_queries.go, migration 052, partitioning 010, migrate runner).
2. Tạo migration 053.
3. Edit `system_health_queries.go` × 2 hàm.
4. `go build ./...` + `go vet ./...`.
5. Bootstrap workspace docs + report.

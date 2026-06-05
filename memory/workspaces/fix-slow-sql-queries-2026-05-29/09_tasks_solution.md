# Solution Summary

## 3 SLOW SQL → 3 fix riêng biệt, đồng nhất pattern "match index với WHERE/ORDER BY"

### Q1 — cdc_jobs (stuck_job_reaper) 268ms → ~<10ms
Index `(type, status, created_at)` không serve được filter `status='running'` standalone (leading column sai). Partial idx `(started_at) WHERE status='running'` narrow direct vào subset cần, ORDER BY `started_at` được idx phục vụ luôn.

### Q2 — cdc_activity_log (queryRecentEvents) 260ms → ~<30ms
WHERE filter `created_at` + ORDER BY `started_at` = mismatch → planner scan + sort. Đổi ORDER BY `created_at DESC` match `idx_act_created (created_at DESC, id)` → index scan reverse stop-at-LIMIT 10.

### Q3 — failed_sync_logs (queryFailedCount) 398ms → ~<60ms
2 separate `count(*)` × 2 partition scan. Merge thành 1 query với FILTER expression — partition pruning chạy 1 lần, FILTER tính subset 1h trong cùng scan. Saves 1 round-trip + 1 partition scan.

## Pattern Global (Lesson)

**Pattern [A query B with WHERE column X + ORDER BY column Y] → Result: index miss vì index trên X không serve được ORDER BY Y.**

Đúng: align ORDER BY cùng column với WHERE filter HOẶC tạo composite/partial index có leading column khớp `WHERE` và secondary column khớp `ORDER BY`.

Áp dụng được cho ≥3 dự án khác:
- Activity log / audit log với 2 timestamp cột (created/started, posted/updated, requested/completed).
- Job queue với 2 cột thời gian (enqueued_at, started_at) — ORDER BY phải match filter.
- Notification/inbox với delivered_at vs read_at.

## Files modified

| # | File | Loại | LOC delta |
|---|---|---|---|
| 1 | `cdc-cms-service/migrations/schema/ops/053_perf_idx_cdc_jobs_running.sql` | NEW | +34 |
| 2 | `cdc-cms-service/internal/infra/observability/system_health_queries.go` | EDIT | +20 / -10 |

NET ≈ +44 LOC, -10 LOC.

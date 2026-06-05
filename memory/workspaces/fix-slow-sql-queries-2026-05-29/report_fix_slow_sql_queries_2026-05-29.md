# Report — Fix 3 SLOW SQL queries

- **Workspace**: `fix-slow-sql-queries-2026-05-29`
- **Date**: 2026-05-29
- **Severity**: P2 — perf hotfix; mỗi 30s reaper sweep + 15s health collector tick.
- **Service**: `cdc-cms-service`
- **Directive**: "triệt để mấy cái này" (user)

## 1. Input logs

```
SLOW SQL >= 200ms [268.108125ms] [rows:0] UPDATE cdc_system.cdc_jobs
   stuck_job_reaper.go:127  (every 30s)

SLOW SQL >= 200ms [260.196ms]   [rows:0] SELECT ... cdc_activity_log
   system_health_queries.go:95  (every 15s)

SLOW SQL >= 200ms [398.064375ms][rows:1] SELECT count(*) failed_sync_logs
   system_health_queries.go:74  (every 15s × 2 calls)
```

DB load wasted ≈ (268 + 260 + 398×2) / 15s ≈ 87ms/s = ~5.8% one core busy on hot path queries.

## 2. Fixes applied

| # | Layer | Change | Before | After (target) |
|---|---|---|---|---|
| Q1 | DB schema | Partial idx `(started_at) WHERE status='running'` | 268ms seq scan | <10ms partial idx scan |
| Q2 | Code | ORDER BY `created_at DESC` (align với WHERE) | 260ms sort | <30ms idx reverse scan |
| Q3 | Code | Merge 2 COUNT → 1 với FILTER | 398ms × 2 scan | <60ms 1 scan |

## 3. Files modified

| # | File | Loại | LOC delta |
|---|---|---|---|
| 1 | `cdc-cms-service/migrations/schema/ops/053_perf_idx_cdc_jobs_running.sql` | NEW | +34 |
| 2 | `cdc-cms-service/internal/infra/observability/system_health_queries.go` | EDIT (2 funcs) | +20 / -10 |

NET: +54 / -10.

## 4. Verify evidence

| Item | Result |
|---|---|
| `go build ./...` | PASS (exit 0, no output) |
| `go vet ./internal/infra/observability/... ./internal/infra/messaging/...` | PASS (no output) |
| Test packages affected | No test files existing — skip |
| Migration re-runnable | `IF NOT EXISTS` guard ✓ |
| Migration runner compat | Regular CREATE (runner wrap tx, không CONCURRENTLY) ✓ |
| Semantic preservation Q2 | `created_at` và `started_at` đều default `NOW()` tại INSERT → rowset identical ✓ |
| Semantic preservation Q3 | Outer WHERE bound 24h + FILTER tính 1h subset = identical với 2-query form ✓ |

## 5. Ops deploy note

- **Auto path**: Restart cdc-cms-service → migrate runner pickup `053_*` via advisory lock + apply.
- **Manual prod path (recommended)**: Trước khi deploy, ops chạy:
  ```sql
  CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_cdc_jobs_running_started
      ON cdc_system.cdc_jobs (started_at)
      WHERE status = 'running';
  ```
  Sau đó deploy bình thường — runner sẽ no-op qua `IF NOT EXISTS`. Đây là path an toàn vì `cdc_jobs` có concurrent writes từ reaper + worker.
- **Index size estimate**: Partial idx chỉ index rows `status='running'` (thường <100 rows tại bất kỳ thời điểm nào) → footprint <1 MB, lock thời gian ngắn nếu apply không-CONCURRENTLY.

## 6. Rollback

- Code: `git revert` 2 file (`system_health_queries.go` + `053_*.sql`).
- Index: `DROP INDEX IF EXISTS cdc_system.idx_cdc_jobs_running_started;` (chỉ cần nếu phát hiện write amplification — không kỳ vọng).

## 7. Out of scope

- Không add unit test infra mới cho `internal/infra/observability` (package chưa có test file existing → scope creep).
- Không refactor `stuck_job_reaper.go` UPDATE CASE statement (semantic + logic giữ nguyên, chỉ index serve filter).
- Không touch `queryReconciliation` (DISTINCT ON) — không có trong slow log.

## 8. Lesson (global pattern)

**Pattern [Query với WHERE column X + ORDER BY column Y where Y ≠ X] → planner scan + sort, 100-300ms range.**

Đúng: align ORDER BY cùng cột với WHERE filter HOẶC tạo composite index `(X, Y)` để index serve cả filter + sort. Khi cột Y semantic-tương-đương với X (both default NOW() tại INSERT), refactor ORDER BY là zero-risk. Khi không tương đương, cần composite idx.

Áp dụng được: activity_log / audit_log / job_queue / notification_inbox / event_log — bất kỳ table nào có ≥2 timestamp column.

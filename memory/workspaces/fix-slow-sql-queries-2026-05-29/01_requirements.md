# Requirements

## Functional
- F1: Mỗi tick reaper sweep `cdc_jobs WHERE status='running'` < 50ms (target).
- F2: `queryRecentEvents` (cdc_activity_log) < 50ms tail.
- F3: `queryFailedCount` (failed_sync_logs) < 80ms tail (gộp 2 COUNT về 1).

## Non-functional
- NF1: Semantic identical với prior behavior (cùng rowset trả về).
- NF2: Migration re-runnable (`IF NOT EXISTS` guard).
- NF3: Không thay đổi public API/interface — chỉ internal observability + index.
- NF4: Không thêm dependency mới.

## Definition of Done
- [x] Migration file mới: `053_perf_idx_cdc_jobs_running.sql`.
- [x] Code edits: `system_health_queries.go` Q2 (ORDER BY) + Q3 (gộp COUNT).
- [x] `go build ./...` PASS.
- [x] `go vet ./...` PASS các package liên quan.
- [x] Comment in-source giải thích WHY (slow SQL hotfix date 2026-05-29).
- [x] Workspace docs đầy đủ (00_context → 09_solution).
- [x] Report file kèm verify evidence.

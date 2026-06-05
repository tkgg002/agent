# Progress (APPEND-only)

## Entry 1 — 2026-05-29
Nhận task từ user: "triệt để mấy cái này" + 3 SLOW SQL logs.

## Entry 2 — 2026-05-29
Read root cause:
- `stuck_job_reaper.go:127` WHERE `status='running' AND started_at IS NOT NULL`. Index `(type, status, created_at)` không match leading column → seq scan.
- `system_health_queries.go:95` ORDER BY `started_at DESC` nhưng WHERE filter `created_at`. Idx `idx_act_created` không được dùng.
- `system_health_queries.go:74` 2 `Count()` riêng × 2 partition scan.

## Entry 3 — 2026-05-29
Confirm migration runner KHÔNG hỗ trợ `CONCURRENTLY` (runner.go wrap tx + strip BEGIN/COMMIT). Chuyển hướng → regular CREATE với IF NOT EXISTS + partial idx subset nhỏ nên lock acceptable.

## Entry 4 — 2026-05-29
Wrote `053_perf_idx_cdc_jobs_running.sql`. Partial idx (started_at) WHERE status='running' + COMMENT + verification SELECT.

## Entry 5 — 2026-05-29
Edited `system_health_queries.go`:
- `queryFailedCount`: merge 2 COUNT → 1 Raw + FILTER expression. Updated doc comment.
- `queryRecentEvents`: ORDER BY `created_at DESC` thay `started_at DESC`. Updated doc comment.

## Entry 6 — 2026-05-29
Verify:
- `go build ./...` → PASS (no output, exit 0).
- `go vet ./internal/infra/observability/... ./internal/infra/messaging/...` → PASS (no output).
- No test files in affected packages → skip test run.

## Entry 7 — 2026-05-29
Bootstrap workspace docs: 00_context, 01_requirements, 02_plan, 03_implementation, 05_progress (this), 07_status, 09_tasks_solution, report.

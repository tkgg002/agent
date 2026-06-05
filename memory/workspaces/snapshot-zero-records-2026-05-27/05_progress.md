# 05_progress — Audit Snapshot Zero Records (APPEND-ONLY)

## Entry 1 — 2026-05-27 17:36 ICT — Workspace init
- Bootstrap workspace theo §7 GEMINI.
- Trigger: user report 17:33:55 ICT — snapshot báo 161/161 nhưng `export_jobs_2` 0 rows.
- Mục tiêu phase 1: trace chain, RCA, đề xuất fix Plan A.

## Entry 2 — 2026-05-27 17:40 ICT — Chain trace + RCA hoàn tất
- Đọc `snapshot_runner_handler.go` (`runSnapshot` line 165-562, focus 380-562).
- Đọc `event_handler.go` (`HandleRaw` + `processEvent` line 60-191).
- Đọc `batch_buffer.go` đầy đủ (`Add` 127-139, `Flush` 158-194, `batchUpsert` 196-306).
- Đọc `schema_adapter.go` (`PrepareForCDCInsertInSchema` 121-230, `BuildBatchUpsertSQLsInSchema` 366-461, `BuildUpsertSQLInSchema` 294-350, `cdcMetadataColumns` 19-32).
- **Root cause xác định**: 4-layer silent swallow chain.
  - Layer 4: `processEvent:173-175` counter từ enqueue (`Add(record); written := 1`).
  - Layer 5: `runSnapshot:516,550` gọi `FlushBatchBuffer()` ignore return; line 521 `rowsTotal += batchWritten`.
  - Layer 6: `FlushBatchBuffer():61-63` void proxy.
  - Layer 7: `Flush():158-194` log error rồi drop, không propagate.
- Cross-check lesson `lessons.md` 2026-05-26 line 3417-3421 "Define DoD at the destination" → bug hôm nay là case study trực tiếp.

## Entry 3 — 2026-05-27 17:45 ICT — Plan A đề xuất + doc set viết
- Files trong workspace: `00_context`, `01_requirements`, `02_plan`, `03_implementation_audit`, `04_decisions`, `05_progress` (file này), `06_validation` (placeholder pre-fix), `07_status`, `08_tasks_audit`, `09_tasks_solution_snapshot`, `10_gap_analysis`, `report_audit_snapshot_zero_records_2026-05-27.md`.
- Code demo cho 5 SOL patch site đã viết đầy đủ trong `09_tasks_solution_snapshot.md`.
- Trạng thái: AUDIT COMPLETE — chờ user verb "làm đi" để sang Muscle apply.

## Entry 4 — 2026-05-28 — Fix phase applied (user verb "làm đi")
- Apply 5 SOL atomically:
  - SOL-1 `batch_buffer.go::batchUpsert` → `(written int, err error)`, TX + fallback đếm `RowsAffected`.
  - SOL-2 `batch_buffer.go::Flush` → `(written int, err error)`, capture first err + sum written.
  - SOL-3 `event_handler.go::FlushBatchBuffer` → `(written int, err error)` pass-through.
  - SOL-4 `snapshot_runner_handler.go` line 516 (per-batch) + line 561 (final) consume return; counter từ `persisted`, không từ enqueue.
  - SOL-5 `batch_buffer.go::timerLoop` 3 caller bọc `_, _ = bb.Flush()`.
- Build verify:
  - `centralized-data-service` `go build ./...` → PASS.
  - `cdc-cms-service` `go build ./...` → PASS (zero-touch sanity).
  - `cdc-cms-web` `npx vite build` → PASS (built in 718ms).
- Vet: `go vet ./internal/handler/` báo 2 lỗi tại `pkgs/idgen/sonyflake.go:77,82` (`ResetForTest` copy `sync.Once`) — pre-existing, không liên quan patch.
- Test: `go test ./internal/handler/... -count=1 -timeout 60s` → PASS (`ok ... 3.769s`).
- LOC delta: batch_buffer.go +37, event_handler.go +1, snapshot_runner_handler.go +16 → **+54 NET**.
- Report: `report_fix_snapshot_zero_records_2026-05-27.md` đã viết.
- Trạng thái: **FIX COMPLETE**. Chờ user verb cho `/security-agent` gate (§8) nếu cần.

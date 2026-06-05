# 02_plan — Audit Snapshot Zero Records

## Phase 1 — Trace chain (DONE)
1. Đọc `snapshot_runner_handler.go` runSnapshot → xác định counter source.
2. Đọc `event_handler.go` HandleRaw + processEvent → xác định counter từ đâu return.
3. Đọc `batch_buffer.go` Add + Flush + batchUpsert → xác định layer persist.
4. Đọc `schema_adapter.go` BuildBatchUpsertSQLInSchema → xác định SQL pattern + ON CONFLICT.

## Phase 2 — Root cause analysis (DONE)
- Map mỗi layer với "counter này đếm gì?".
- Xác định layer đầu tiên mà counter ≠ persistence reality.
- Cross-reference với lesson "Define DoD at the destination".

## Phase 3 — Đề xuất fix (Brain phase, DOCUMENT ONLY)
- Plan A (chọn): Plumb `(int, error)` qua Flush chain. Minimal — 4 patch site.
- Plan B (rejected — over-engineer): Refactor BatchBuffer sang sync per-record cho snapshot path.
- So sánh Plan A vs Plan B trong `04_decisions.md`.

## Phase 4 — Code demo viết đầy đủ (Brain phase)
- File `09_tasks_solution_snapshot.md` chứa code demo full cho:
  - SOL-1: `BatchBuffer.batchUpsert` → return `(written int, err error)`.
  - SOL-2: `BatchBuffer.Flush` → return `(written int, err error)`.
  - SOL-3: `EventHandler.FlushBatchBuffer` → return `(written int, err error)`.
  - SOL-4: `snapshot_runner.runSnapshot` consume return ở line 516 + 550 (per-batch flush + final flush).
  - SOL-5: counter `rowsTotal` lấy từ persisted, không lấy từ enqueued.

## Phase 5 — User approve gate
- Trình bộ doc + report audit.
- Chờ user verb "làm đi" / "ok" / equivalent → MỚI sang Muscle phase apply.

## Phase 6 — Muscle apply (PENDING — chỉ chạy khi có verb)
1. Apply 5 SOL patch site theo `09_tasks_solution_snapshot.md`.
2. `go build ./...` PASS cho `centralized-data-service`.
3. `go test ./internal/handler/...` PASS (target ≥ test cases hiện tại, không thoái lùi).
4. Sanity: cdc-cms-service + cdc-cms-web build PASS (zero-touch, vẫn check).
5. Ghi `report_fix_snapshot_zero_records_2026-05-27.md` + APPEND `05_progress.md`.

## Phase 7 — Security gate (§8)
- Sau Muscle phase, chạy `/security-agent` theo user yêu cầu.

## Risk register
| Risk | Likelihood | Mitigation |
|---|---|---|
| Signature change breaks unrelated callers (timer loop trong batch_buffer) | Medium | Grep `Flush()` + `FlushBatchBuffer()` callers; chỉ snapshot path consume return, timer loop ignore với `_, _ =` |
| Sequential fallback trong batchUpsert tiếp tục silent-swallow per-row | Medium | Plan A bóc tách: total persisted = chunks success × len(chunk); SyncSuccess metric đã đếm per-row → có thể aggregate |
| Bug thực sự nằm ở SQL (PK type mismatch / missing UNIQUE) ở table `export_jobs_2` | Low | Plan A vẫn đúng: surface lỗi SQL cho operator. Nếu sau fix vẫn 0 rows, lỗi SQL sẽ bubble lên markProgressError thay vì silent done |

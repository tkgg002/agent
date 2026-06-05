# 08_tasks_audit — Audit Snapshot Zero Records

## Tasks (audit phase — DONE)
- **T-1** Đọc `snapshot_runner_handler.go` runSnapshot → DONE.
- **T-2** Đọc `event_handler.go` HandleRaw + processEvent → DONE.
- **T-3** Đọc `batch_buffer.go` Add + Flush + batchUpsert → DONE.
- **T-4** Đọc `schema_adapter.go` Prepare + BuildUpsert (sanity) → DONE.
- **T-5** Map failure-mode matrix per layer → DONE (`03_implementation_audit.md`).
- **T-6** Cross-reference lesson `Define DoD at the destination` → DONE.
- **T-7** Quyết định Plan A vs Plan B → DONE (`04_decisions.md`).
- **T-8** Viết code demo cho 5 SOL patch site → DONE (`09_tasks_solution_snapshot.md`).
- **T-9** Viết report audit → DONE (`report_audit_snapshot_zero_records_2026-05-27.md`).

## Tasks (fix phase — PENDING user verb)
- **F-1** Apply SOL-1 batchUpsert return `(written, err)`.
- **F-2** Apply SOL-2 Flush return `(written, err)`.
- **F-3** Apply SOL-3 FlushBatchBuffer return `(written, err)`.
- **F-4** Apply SOL-4 runSnapshot lines 516+550 consume return.
- **F-5** Apply SOL-5 timer loop ignore với `_, _ =`.
- **F-6** Build + vet + test verify 3 service.
- **F-7** Viết `report_fix_snapshot_zero_records_2026-05-27.md`.
- **F-8** APPEND Entry 4 vào `05_progress.md`.

## Tasks (security gate — PENDING user verb)
- **S-1** Chạy `/security-agent` sau Muscle phase.

## Cross-workspace dependencies
- Workspace `audit-shadow-create-bugs-2026-05-27` đã apply fix Bug #2 → tables mới sẽ có đủ system cols. Nhưng workspace này KHÔNG depend on Bug #2 fix: bug ở đây là observability/error-plumbing, độc lập với DDL.

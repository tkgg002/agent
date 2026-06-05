# 06_validation — Acceptance + Verify

## Verify Matrix

| DoD | Verify Command | Expected |
|---|---|---|
| DoD-1 BE reclaim function | `go test ./internal/handler/ -run TestReclaimOrphans -v` | 3 sub-test PASS |
| DoD-2 worker_server wire | `grep -n 'ReclaimOrphans\|boot-reclaim' internal/server/worker_server.go` | match xuất hiện |
| DoD-3 FE stale render | `grep -n 'isStaleRunning\|Force Resume' cdc-cms-web/src/pages/SnapshotMonitor.tsx` | match |
| DoD-4 FE warning modal | `grep -n 'orphan\|chưa heartbeat' cdc-cms-web/src/pages/SnapshotMonitor.tsx` | match |
| DoD-5 env binding | `grep -n 'SNAPSHOT_STALE_AFTER_SECONDS' internal/server/worker_server.go` | match |
| DoD-6 build + test | `go build ./...` + `go test ./internal/handler/...` + `cd cdc-cms-web && npx vite build` | exit 0 cả 3 |

## Runtime Smoke (Muscle phase)

### Setup orphan row giả
```sql
-- Seed: tạo row stale running
INSERT INTO cdc_system.snapshot_progress
  (source_object_id, status, trace_id, started_at, updated_at, last_seen_id, rows_processed)
VALUES
  (999, 'running', 'fake-orphan', NOW() - INTERVAL '120 seconds', NOW() - INTERVAL '120 seconds', NULL, 1000);
```

### Restart worker
```bash
pkill -9 centralized-data-service
go run ./cmd/worker  # hoặc systemctl restart cdc-worker
```

### Assert (within 10s)
```sql
SELECT status, updated_at FROM cdc_system.snapshot_progress WHERE source_object_id = 999;
-- Expected sequence (poll mỗi 2s):
--   T0:    status='paused', updated_at=NOW()  ← reclaim demote
--   T+5s:  status='running', updated_at=NOW() ← worker resume claimed
```

### FE smoke
1. Mở `http://localhost:5173/snapshot-monitor`.
2. Source row stale running → thấy nút **Force Resume** (orange icon).
3. Click → confirm modal hiển thị warning "có thể đã orphan".
4. Confirm → 200 response → row chuyển running, updated_at update mỗi 5s.

## Acceptance Criteria

| AC | Pass condition |
|---|---|
| AC-1 BE Build | `go build ./...` exit 0 |
| AC-2 BE Vet | no new error (pre-existing pkgs/idgen ignore) |
| AC-3 BE Test | `go test ./internal/handler/... -count=1` PASS |
| AC-4 FE Build | `npx vite build` exit 0 |
| AC-5 Runtime BE reclaim | Orphan seed → trong 10s row chuyển paused→running |
| AC-6 Runtime FE button | Force Resume xuất hiện cho stale running |
| AC-7 Runtime FE click | POST resume thành công, snapshot tiếp tục |
| AC-8 Report file | `report_*.md` có files + LOC delta thực |

## Regression test

Fix trước (workspace `bug-snapshot-progress-mismatch-2026-05-28`):
- `markProgressDone` guard threshold 0.99 — VẪN PHẢI hoạt động (`TestMarkProgressDone_CompletenessGuard` PASS).
- Cursor exit chỉ qua `len(batch) == 0` — VẪN PHẢI giữ.
- Pause `return nil` — VẪN PHẢI giữ.

Verify: chạy lại `go test ./internal/handler/ -run 'TestMarkProgressDone|TestCursorEarlyExit|TestPause_No' -count=1` → PASS toàn bộ.

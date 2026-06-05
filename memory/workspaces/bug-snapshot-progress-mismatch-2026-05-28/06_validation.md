# 06_validation — Acceptance + Verify Command

## Verify Matrix (định lượng cho mỗi DoD)

| DoD | Verify Command | Expected Output |
|---|---|---|
| DoD-1 (cursor partial) | `cd centralized-data-service && go test ./internal/handler/ -run TestSnapshot_CursorPartialMidStream -v` | `--- PASS: TestSnapshot_CursorPartialMidStream` |
| DoD-2 (pause fall-through) | `go test ./internal/handler/ -run TestSnapshot_PauseDoesNotFallThroughToDone -v` | `--- PASS` + DB status=`paused` |
| DoD-3 (completeness guard) | `go test ./internal/handler/ -run TestSnapshot_MarkDoneGuardsCompleteness -v` | 4 sub-test PASS |
| DoD-4 (metric expose) | `curl -s http://localhost:8080/metrics \| grep cdc_snapshot_partial_done_total` | 3 line với `reason=cursor_short`, `pause_fallthrough`, `persist_mismatch` |
| DoD-7 (build + vet) | `go build ./... && go vet ./internal/handler/...` | exit 0 (vet ignore pre-existing pkgs/idgen/sonyflake.go) |

## Runtime smoke test (Muscle phase, sau apply patch)

### Setup
```bash
# Tạo collection test 1000 docs trong Mongo
mongosh "$MONGO_URI" --eval '
  use wallet_service_test;
  db.events.drop();
  for (let i = 0; i < 1000; i++) db.events.insertOne({_id: i, payload: "x"});
'

# Đăng ký source_object qua CMS API hoặc trực tiếp DB seed
```

### Trigger snapshot
```bash
curl -X POST http://cms-service/api/snapshot/trigger \
  -H "Content-Type: application/json" \
  -d '{"source_object_id": 999, "batch_size": 200}'
```

### Assertion (psql)
```sql
SELECT id, status, rows_processed, total_rows, finished_at
FROM cdc_system.snapshot_progress
WHERE id = (SELECT MAX(id) FROM cdc_system.snapshot_progress);

-- Expected:
-- status='done' AND rows_processed BETWEEN 990 AND 1000 (threshold 0.99)
-- AND finished_at IS NOT NULL

SELECT COUNT(*) FROM "wallet-service".events_shadow;
-- Expected: ~1000 (cho phép gap ≤ 10 do estimate skew)
```

### Negative test — partial cursor scenario
```bash
# Simulate replication lag bằng cách giảm SetLimit trong test mock
go test ./internal/handler/ -run TestSnapshot_CursorPartialMidStream_RealMongo \
  -count=1 -timeout 5m -tags=integration
```

### Negative test — pause
```bash
# Trigger snapshot → 100ms sau publish pause command → wait
curl -X POST .../snapshot/trigger -d '{"source_object_id": 999}' &
sleep 0.1
nats pub cdc.control.commands.999 pause

# Assert
psql -c "SELECT status FROM cdc_system.snapshot_progress WHERE id=...;"
# Expected: status='paused' (KHÔNG 'done')
```

## Acceptance Criteria — gating verb `done` từ Muscle

| Criterion | Pass condition |
|---|---|
| AC-1 Build | `go build ./...` exit 0 |
| AC-2 Vet | `go vet ./internal/handler/...` no new error (pre-existing pkgs/idgen ignore) |
| AC-3 Unit test | `go test ./internal/handler/... -count=1` PASS |
| AC-4 Integration smoke | Snapshot 1000 docs → status=done + rows_processed ≥ 990 |
| AC-5 Pause smoke | Pause mid-snapshot → status=paused KHÔNG done |
| AC-6 Completeness guard smoke | Inject `rowsTotal=500, totalRows=1000` → status=error reason="incomplete" |
| AC-7 Metric scrape | `cdc_snapshot_partial_done_total` xuất hiện ở `/metrics` |
| AC-8 Report file | `report_bug_snapshot_progress_mismatch_2026-05-28.md` có **files thay đổi** + **LOC delta** thực |

## Regression test (đảm bảo fix cũ không bị break)
- `snapshot-zero-records-2026-05-27/` đã fix Flush chain → counter PG RowsAffected. Verify:
  - `go test ./internal/handler/ -run TestEventHandler_FlushBatchBuffer -v` vẫn PASS.
  - `BatchBuffer.Flush` vẫn return `(written int, err error)`.
  - Signature không bị đổi ngược.

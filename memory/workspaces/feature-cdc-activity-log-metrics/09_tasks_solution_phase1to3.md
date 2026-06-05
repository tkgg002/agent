# Solution Tasks (Phase 1–3) — Synchronous Upsert

| # | Task | File | Status |
|---|------|------|--------|
| 1 | Add `WriteRecordSync(record) (int, error)` | `centralized-data-service/internal/handler/batch_buffer.go` | done |
| 2 | Replace async `Add` with sync `WriteRecordSync` in `processEvent`; accumulate `totalWritten`; propagate errors | `centralized-data-service/internal/handler/event_handler.go` | done |
| 3 | Widen `handleDelete` to `(int, error)`; sum `RowsAffected` per route | `centralized-data-service/internal/handler/event_handler.go` | done |
| 4 | Update 2 call sites in `event_handler_test.go` to new `handleDelete` signature | `centralized-data-service/internal/handler/event_handler_test.go` | done |
| 5 | `go build ./...` + `go vet ./...` + `go test ./internal/handler/...` | repo | PASS |
| 6 | Live smoke test on worker — verify `cdc_activity_log.rows_affected` matches actual destination row deltas | runtime | pending (needs user to restart worker) |

## Deviation log
- Plan §1.4 says `WriteRecordSync` returns `1, nil` on success. Implementation returns `int(res.RowsAffected), nil` instead — closer to the workspace mission ("exact rows successfully materialized in destination database", 00_context.md). For the current `INSERT ... ON CONFLICT DO UPDATE` SQL, both behaviours are identical (always 1). The deviation only matters if a future SQL variant uses `DO NOTHING`.

# Implementation: Phase 1–3 — Synchronous Upsert for Accurate rows_affected

## Files changed
- `centralized-data-service/internal/handler/batch_buffer.go` — new `WriteRecordSync(record) (int, error)`.
- `centralized-data-service/internal/handler/event_handler.go` — `processEvent` and `handleDelete` now upsert synchronously and return real row counts.
- `centralized-data-service/internal/handler/event_handler_test.go` — updated 2 test call sites for new `handleDelete` signature.

## Phase 1 — `WriteRecordSync`
- Reuses the existing pipeline: `resolveDB` → `resolveSchemaAdapter` → `PrepareForCDCInsertInSchema` → `GetSchemaInSchema` → effective-PK rewrite (`id` → `source_id` when shadow exposes it) → `BuildUpsertSQLInSchema` → `db.Exec`.
- Returns `int(res.RowsAffected)` on success (deviation from plan §1.4's literal `1` — captures any DO NOTHING / no-op row that pgx reports as 0). On error returns `(0, err)`.
- Does NOT insert into `failed_sync_logs` — that responsibility stays with `KafkaConsumer.writeDLQ` to guarantee a single DLQ row per failure (plan §1.3).
- Metrics: increments `metrics.SyncSuccess` on no-error, `metrics.SyncFailed` on error (same labels as the old `batchUpsert` per-record path).

## Phase 2 — `processEvent` / `handleDelete`
- Replaced the async `h.batchBuffer.Add(record)` in the fan-out loop with `h.batchBuffer.WriteRecordSync(record)` and accumulated `totalWritten`.
- On the first route error: return `(totalWritten, error)` — caller (`KafkaConsumer.processMessage`) escalates to DLQ, Kafka redelivers (at-least-once), UPSERT idempotency makes the retry safe.
- `handleDelete` signature widened from `error` to `(int, error)` so DELETE batches contribute real row counts too. Skipped routes (missing PK) contribute 0; tombstone UPSERT contributes `RowsAffected`.

## Phase 3 — `KafkaConsumer.processMessage`
- No code change. Signature already `(int, error)`; consumer loop at line 407–425 already does `batch.rowsAffected += rows` on success and `batch.failed++` on error. The bug was purely upstream returning `len(routes)` whether or not a row was materialized.

## Why 462 vs 154
- Old path: `processEvent` returned `len(routes)` for every event regardless of whether the underlying DB upsert later succeeded or even ran. For a 3-route fan-out, 154 source events inflated to 462 in `cdc_activity_log.rows_affected`.
- New path: `rows_affected` = sum of `pgx RowsAffected` across all successfully-upserted routes. If a route's upsert is a DO-NOTHING/no-op (hash same) or errors, it does not pad the count.

## Verification
- `go build ./...` — clean.
- `go vet ./...` — clean.
- `go test ./internal/handler/...` — PASS. Adjusted `event_handler_test.go:148` and `:198` to the new `(written, err)` signature with explicit row-count assertions.

## Pending / handoff to user
- Smoke test on a live worker: restart, replay a known batch (e.g. `cdc.goopay.centrallized-export-service.export-jobs`), confirm `cdc_activity_log.rows_affected` matches `count(*)` delta on the destination shadow tables.
- Confirm `failed_sync_logs` only has one entry per failed Kafka message (no duplicate from `batchUpsert` + `writeDLQ`).

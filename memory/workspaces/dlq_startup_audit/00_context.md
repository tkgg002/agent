# 00 — Context: DLQ State Machine Startup Log Burst Audit

## Scope
Audit hành vi log "dlq state machine replayed message" xuất hiện thành burst (~33 dòng trong ~100ms) ngay khi `cdc-worker` (centralized-data-service) start.

## In-scope components
- `centralized-data-service/internal/handler/dlq_state_machine.go` — state machine poll + replay
- `centralized-data-service/internal/handler/dlq_handler.go` — DLQ enqueue + constants (`MaxRetries`, `DLQSubject`)
- `centralized-data-service/internal/server/worker_server.go` — bootstrap order `s.dlqWorker.Start(...)`
- DB schema: `cdc_system.failed_sync_logs` (status, next_retry_at, retry_count, raw_json, kafka_topic)

## Out-of-scope (đợi user approve riêng nếu muốn fix)
- Production code thay đổi
- DB migration / data backfill
- Cấu hình runtime (env, deployment)

## Evidence (user-provided log)
```
2026-05-28 17:07:05.468 - 17:07:05.571 (≈103ms) → 33 dòng "dlq state machine replayed message"
```
Tất cả đều ở level INFO (theo `sm.logInfo` tại `dlq_state_machine.go:150`).

## Audit goal
Trả lời 3 câu:
1. Burst log này có phải dấu hiệu bug (silent degradation theo lesson #820)?
2. Cơ chế đứng sau burst là gì? (5-whys theo lesson #866)
3. Nếu không phải bug, có cải tiến nào nên đề xuất (log hygiene / concurrency / startup gate)?

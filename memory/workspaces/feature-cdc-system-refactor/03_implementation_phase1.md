# Implementation Phase 1

- Refactored `schema_adapter.go` to normalize JSONB coercion, preserve cache reload, and enforce OCC with `_source_ts <= EXCLUDED._source_ts`.
- Refactored `recon_source_agent.go` and `recon_dest_agent.go` to align bucket hashing on `xxhash64(id|UnixMilli)`.
- Refactored `recon_core.go` advisory locking to use stable CRC32-derived keys and direct cross-store Tier 3 bucket comparison.
- Refactored `recon_heal.go` to read Mongo healing data from primary and added `HealOrphanedIDs`.
- Refactored `dynamic_mapper.go` to delegate UPSERT SQL generation to `SchemaAdapter`.
- Refactored `schema_inspector.go` to suppress duplicate drift alerts per `table:field` for one hour.
- Refactored `internal/handler/dlq_handler.go` for context-aware retry/backoff and transactional failed log persistence before NATS publish.
- Added `internal/handler/dlq_state_machine.go` for scheduled replay of `failed_sync_logs`.

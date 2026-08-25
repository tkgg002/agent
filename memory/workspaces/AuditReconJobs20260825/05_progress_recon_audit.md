# Progress — Audit Recon Jobs 20260825

## [2026-08-25T09:00 Agent:Claude-Opus-4.6] Phiên bắt đầu
- User yêu cầu audit 2 job recon: #132 (reconA) và #134 (reconB)
- Đã đọc GEMINI.md và lessons.md
- Đã nghiên cứu codebase recon: engine, tier_b, job_worker, stream_bucket_engine
- Đã phân tích chi tiết dữ liệu 2 job
- Kết luận: Job #132 drift nhẹ (bình thường), Job #134 bất thường nghiêm trọng (nghi PK/TS column mismatch)
- Tạo implementation_plan.md (artifact) với phân tích đầy đủ
- Đang chờ User review để quyết định hành động tiếp theo
- [2026-08-25T13:25:00+07:00] [Agent:DeepSeek-R1] Implemented Hard Delete on Master for Transmute Oplog Deletes (transmuter.go).
- [2026-08-25T13:25:00+07:00] [Agent:DeepSeek-R1] Added NATS cascade trigger cdc.cmd.transmute-shadow in executeHealSegA and RunOrphanPrune.
- [2026-08-25T13:25:00+07:00] [Agent:DeepSeek-R1] Updated Action Toast on Frontend to display both Job ID and Trace ID simultaneously.
- [2026-08-25T13:25:00+07:00] [Agent:DeepSeek-R1] Verified builds: cmd/worker PASS, cdc-cms-web PASS.
- [2026-08-25T13:57:00+07:00] [Agent:DeepSeek-R1] Standardized True Upsert in schema_adapter.go (removed partial index cheat, added _deleted = EXCLUDED._deleted).
- [2026-08-25T13:57:00+07:00] [Agent:DeepSeek-R1] Fixed silent heal error in recon_heal_fetch.go (tracked actualPersisted, returned flush errors).
- [2026-08-25T13:57:00+07:00] [Agent:DeepSeek-R1] Updated finalizeReport in recon_execute_heal_handler.go to mark heal_failed when 0 records are healed.
- [2026-08-25T13:57:00+07:00] [Agent:DeepSeek-R1] Verified tests: schema_adapter_ordering_test.go PASS (6/6), cmd/worker PASS.
- [2026-08-25T14:01:00+07:00] [Agent:DeepSeek-R1] Fixed SQLSTATE 42601 (multiple assignments to same column "_deleted") by deduplicating updateSets in BuildUpsertSQLInSchema and BuildBatchUpsertSQLInSchema.
- [2026-08-25T14:01:00+07:00] [Agent:DeepSeek-R1] Verified full service test suite PASS (100%), cmd/worker build PASS.
- [2026-08-25T14:05:00+07:00] [Agent:DeepSeek-R1] Reverted buildConflictTarget back to partial index WHERE NOT _deleted to match actual DB index (fixed SQLSTATE 42P10).
- [2026-08-25T14:05:00+07:00] [Agent:DeepSeek-R1] Prevented EnsureTableSchemaInSchema from creating conflicting plain unique constraint on V2 tables with _source_id.
- [2026-08-25T14:14:00+07:00] [Agent:DeepSeek-R1] Auto-dropped legacy plain unique constraint (payment_bills_1__id_cdc_unique) in EnsureTableSchemaInSchema for V2 shadow tables to eliminate SQLSTATE 23505 duplicate key error.
- [2026-08-25T14:14:00+07:00] [Agent:DeepSeek-R1] Re-built cmd/worker PASS.
- [2026-08-25T14:18:00+07:00] [Agent:DeepSeek-R1] Kept original PrimaryKeyField (_id) in batch_buffer.go without overriding _source_id (Zero-DDL safe execution).
- [2026-08-25T14:18:00+07:00] [Agent:DeepSeek-R1] Verified go test ./test/internal/service/... PASS (100%), cmd/worker build PASS.
- [2026-08-25T14:30:00+07:00] [Agent:DeepSeek-R1] Fixed microservice schema collision in Recon Check/Job: Added shadow_schema and source fields to reconCheckPayload, ReconJobCreatedEvent, and prioritized schema-qualified lookup (shadow_schema.target_table) in ReconJobWorker to guarantee 100% schema isolation between services with identical table names (e.g. bidv vs bvb bank_requests).
- [2026-08-25T14:30:00+07:00] [Agent:DeepSeek-R1] Verified recon test suite PASS (100%), cmd/worker build PASS.
- [2026-08-25T14:32:00+07:00] [Agent:DeepSeek-R1] Eliminated fallback guessing in ReconJobWorker: strictly lookup by exact lookupKey (shadow_schema.target_table). If not found, fail immediately with clear error instead of falling back to bare table name.
- [2026-08-25T14:32:00+07:00] [Agent:DeepSeek-R1] Verified recon test suite PASS (100%), cmd/worker build PASS.

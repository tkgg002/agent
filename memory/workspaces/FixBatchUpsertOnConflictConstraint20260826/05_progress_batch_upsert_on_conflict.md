# 05_progress_batch_upsert_on_conflict.md

## Audit Log
- [2026-08-26T10:07:00+07:00] [Agent:Gemini-3.6-Flash] Root Cause Analysis: Lỗi `SQLSTATE 42P10` xảy ra do `batch_buffer.go` gán `effectivePK := pk` (hoặc `first.PrimaryKeyField` là `"_id"`). Khi table schema mang V2 shadow format (có `_source_id`), PostgreSQL index unique nằm trên `_source_id` (partial index `(_source_id) WHERE NOT _deleted`), không phải `_id`. Cần remap `effectivePK = "_source_id"` khi `_source_id` tồn tại trong `schema.Columns`.
- [2026-08-26T10:09:30+07:00] [Agent:Gemini-3.6-Flash] Implemented fix in `batch_buffer.go` (remapped `effectivePK` to `_source_id` when present) and added `TestBuildBatchUpsertSQLsInSchema_EffectivePK_SourceID` in `schema_adapter_test.go`. Ran tests and verified 100% PASS.


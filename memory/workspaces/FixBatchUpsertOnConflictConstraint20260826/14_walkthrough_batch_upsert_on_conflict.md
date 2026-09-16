# 14_walkthrough_batch_upsert_on_conflict.md: Walkthrough of Fix

## Overview
Walkthrough of changes and verification for fixing SQLSTATE 42P10 ON CONFLICT error in `BatchBuffer`.

## Execution Steps
1. Updated `internal/handler/shadow/batch_buffer.go`:
   Remapped `effectivePK` to `"_source_id"` when `_source_id` exists in `schema.Columns`.
2. Added unit test `TestBuildBatchUpsertSQLsInSchema_EffectivePK_SourceID` in `test/internal/service/schema_adapter_test.go`.
3. Executed `go test -v ./test/internal/service/...` - 100% PASS.

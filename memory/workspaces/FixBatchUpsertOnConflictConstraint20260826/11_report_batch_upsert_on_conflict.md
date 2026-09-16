# 11_report_batch_upsert_on_conflict.md: Summary of Changes

## Overview
Đã khắc phục dứt điểm lỗi `SQLSTATE 42P10` (`batch upsert chunk failed: ERROR: there is no unique or exclusion constraint matching the ON CONFLICT specification (SQLSTATE 42P10)`).

## Changed Files & Line Counts

| File | Changes | Modified Lines | Overview |
|---|---|---|---|
| `centralized-data-service/internal/handler/shadow/batch_buffer.go` | Modified | +6 lines | Remap `effectivePK` sang `"_source_id"` khi `"_source_id"` tồn tại trong `schema.Columns`. |
| `centralized-data-service/test/internal/service/schema_adapter_test.go` | Modified | +41 lines | Thêm unit test `TestBuildBatchUpsertSQLsInSchema_EffectivePK_SourceID` xác minh ON CONFLICT spec. |

## Detailed Changes

### 1. `batch_buffer.go`
- Tại `WriteRecordSync` (dòng 142):
  ```go
  effectivePK := pk
  if _, ok := schema.Columns["_source_id"]; ok {
      effectivePK = "_source_id"
  }
  ```
- Tại `batchUpsert` (dòng 375):
  ```go
  effectivePK := first.PrimaryKeyField
  if _, ok := schema.Columns["_source_id"]; ok {
      effectivePK = "_source_id"
  }
  ```

### 2. `schema_adapter_test.go`
- Bổ sung `TestBuildBatchUpsertSQLsInSchema_EffectivePK_SourceID` để kiểm tra việc sinh câu lệnh SQL `ON CONFLICT ("_source_id") WHERE NOT _deleted` đúng với partial unique index trên V2 shadow tables.

## Verification Results
- `go test -v ./test/internal/service/...` PASS 100% (25 test cases).

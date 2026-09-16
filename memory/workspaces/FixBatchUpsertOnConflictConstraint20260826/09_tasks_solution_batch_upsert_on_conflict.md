# 09_tasks_solution_batch_upsert_on_conflict.md

## Technical Solution

### Root Cause
Trong `centralized-data-service/internal/handler/shadow/batch_buffer.go`:
- Dòng 142 (`WriteRecordSync`) và dòng 375 (`batchUpsert` chunk batch):
  `effectivePK` đang được gán thẳng bằng `pk` / `first.PrimaryKeyField` (thường là `"_id"` cho MongoDB source).
- Khi table schema trong PostgreSQL là V2 shadow format (`_gpay_id BIGINT PK` + `_source_id TEXT` với partial unique index `(_source_id) WHERE NOT _deleted`), cột `"_id"` không có Unique constraint hay Index nào.
- `schemaAdapter.BuildBatchUpsertSQLsInSchema` nhận `pkField = "_id"`, sinh ra câu lệnh:
  `INSERT INTO schema.table ("_id", ...) VALUES ... ON CONFLICT ("_id") DO UPDATE SET ...`
- PostgreSQL kiểm tra và không thấy Unique constraint/index nào trên cột `"_id"`, sinh ra lỗi:
  `ERROR: there is no unique or exclusion constraint matching the ON CONFLICT specification (SQLSTATE 42P10)`.

### Implementation Plan
1. Trong `batch_buffer.go`:
   - Sau khi load `schema := schemaAdapter.GetSchemaInSchema(schemaName, tableName)`:
   - Kiểm tra `if _, ok := schema.Columns["_source_id"]; ok { effectivePK = "_source_id" }`.
2. Áp dụng sửa đổi tại 2 điểm:
   - Point A: `WriteRecordSync` (dòng 142).
   - Point B: `batchUpsert` (dòng 375).

### Code Snippet Demo
```go
// batch_buffer.go line 142:
effectivePK := pk
if _, ok := schema.Columns["_source_id"]; ok {
    effectivePK = "_source_id"
}

// batch_buffer.go line 375:
effectivePK := first.PrimaryKeyField
if _, ok := schema.Columns["_source_id"]; ok {
    effectivePK = "_source_id"
}
```

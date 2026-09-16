# 13_analysis_batch_upsert_on_conflict.md: Analysis & Root Cause

## Deep Analysis
Trong kiến trúc CDC V2:
- Tầng schema adapter và bootstrap (`ShadowAutomator` / `schema_manager.go`) sinh bảng PostgreSQL shadow với cột `_source_id TEXT NOT NULL` mang index `CREATE UNIQUE INDEX IF NOT EXISTS ux_<table_name>_source_id_active ON <schema>.<table_name> (_source_id) WHERE NOT _deleted`.
- Khi `BatchBuffer.batchUpsert` nhận danh sách `UpsertRecord`, thuộc tính `record.PrimaryKeyField` (ví dụ `"_id"`) đại diện cho tên trường khóa chính của nguồn MongoDB.
- `batch_buffer.go` trước đây dùng `effectivePK := pk` hoặc `effectivePK := first.PrimaryKeyField`, nên truyền `"_id"` vào `SchemaAdapter.BuildBatchUpsertSQLsInSchema`.
- `SchemaAdapter` dựa vào `pkField` để dựng mệnh đề `ON CONFLICT`. Khi `pkField == "_id"`, `buildConflictTarget` trả về `("_id")`.
- Vì trong PostgreSQL bảng không có UNIQUE index hay constraint nào trên cột `"_id"`, PostgreSQL ném ra lỗi `SQLSTATE 42P10`.

## Solution Verified
Khi `_source_id` có trong `schema.Columns`, gán `effectivePK = "_source_id"`.
`buildConflictTarget` sẽ phát ra `("_source_id") WHERE NOT _deleted` (khi có `_deleted`) hoặc `("_source_id")`, khớp 100% với index duy nhất trong PostgreSQL.

# 01_requirements_batch_upsert_on_conflict.md

## Requirements & Specifications

### 1. Requirements
- **R1**: Khắc phục dứt điểm lỗi `SQLSTATE 42P10` trong `BatchBuffer.batchUpsert` và `BatchBuffer.WriteRecordSync`.
- **R2**: Khi `schema.Columns` có chứa cột `_source_id`, `effectivePK` phải tự động remapped về `"_source_id"`.
- **R3**: Với V1 tables (không có `_source_id`), giữ nguyên `effectivePK = pk` (`first.PrimaryKeyField`).
- **R4**: Đảm bảo cả hai luồng `batchUpsert` (chunk batch) và `batchUpsertSingleRecord` (fallback / single sync) đều áp dụng nhất quán việc remap `effectivePK`.
- **R5**: Bổ sung unit test trong `centralized-data-service` để kiểm chứng việc remap `effectivePK` khi table schema chứa `_source_id`.

### 2. Definition of Done (DoD)
- Pass toàn bộ Go test suite trong `centralized-data-service`.
- Không còn sinh lỗi `SQLSTATE 42P10` khi upsert vào V2 shadow tables.

# Fix Batch Upsert ON CONFLICT Constraint Error (SQLSTATE 42P10)

Sửa lỗi `batch upsert chunk failed: ERROR: there is no unique or exclusion constraint matching the ON CONFLICT specification (SQLSTATE 42P10) (fallback persisted 0 rows)` bằng cách tự động remap `effectivePK` sang `"_source_id"` trong `BatchBuffer` khi schema bảng PostgreSQL có chứa cột `"_source_id"`.

## User Review Required

> [!NOTE]
> Giải pháp remap `effectivePK` về `"_source_id"` hoàn toàn phù hợp với hợp đồng V2 shadow table contract (`_gpay_id BIGINT PK` + partial unique index `(_source_id) WHERE NOT _deleted`), đảo bảo tương thích 100% với `schema_adapter.go`. Không làm ảnh hưởng tới V1 shadow tables (bảng không có cột `_source_id`).

## Open Questions

Không có câu hỏi mở. Nguyên nhân gốc rễ đã được xác định hoàn toàn chính xác qua stacktrace và mã nguồn.

## Proposed Changes

### Centralized Data Service

#### [MODIFY] [batch_buffer.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/shadow/batch_buffer.go)
- Tại `WriteRecordSync` (dòng 142) và `batchUpsert` (dòng 375): Bổ sung kiểm tra `if _, ok := schema.Columns["_source_id"]; ok { effectivePK = "_source_id" }`.

#### [NEW] [schema_adapter_test.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/test/internal/service/schema_adapter_test.go)
- Bổ sung unit test `TestBuildBatchUpsertSQLsInSchema_EffectivePK_SourceID` để verify việc remap `effectivePK` khi table schema chứa `_source_id`.

## Verification Plan

### Automated Tests
- Chạy `go test -v ./test/internal/service/...`
- Chạy toàn bộ test suite `go test ./...` trong `centralized-data-service`.

### Manual Verification
- Kiểm tra log runtime khi CDC worker sync dữ liệu batch snapshot / real-time vào V2 shadow tables.

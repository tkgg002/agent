# 08_tasks_batch_upsert_on_conflict.md

## Tasks Checklist
- [x] Task 1: Cập nhật `batch_buffer.go` tại `WriteRecordSync` (dòng 142) và `batchUpsert` (dòng 375) để check `if _, ok := schema.Columns["_source_id"]; ok { effectivePK = "_source_id" }`.
- [x] Task 2: Thêm unit test kiểm tra logic remap `effectivePK` trong `schema_adapter_test.go`.
- [x] Task 3: Chạy verification `go test ./...` trong `centralized-data-service` (PASS 100%).

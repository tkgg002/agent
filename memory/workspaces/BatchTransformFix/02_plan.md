# Plan: Batch Transform Fix for Legacy V1 Table

## Steps
1. **Research & Audit**:
   - Kiểm tra log của CDC Worker để xem lỗi `skipped: table does not exist` khi chạy batch-transform cho `export_jobs`.
   - Phân tích logic định tuyến của `MetadataRegistryService` xem tại sao route của `export_jobs` bị trả về `nil`.
   - Xác định trạng thái của `source_object_registry` cho `export-jobs`.

2. **Resolution**:
   - Kích hoạt `source_object_registry` cho `export-jobs` (id=1) trong database `cdc_dw` sang `is_active = true`.
   - Gửi lệnh `schema.config.reload` qua NATS để worker reload config tức thời.

3. **Verify**:
   - Chạy lệnh batch-transform cho `export_jobs` và verify số dòng transform thành công.
   - Kiểm tra xem logic check dynamic column (đã bổ sung ở turn trước) có hoạt động đúng khi schema bị drift (drop thử cột `__v` và kiểm tra log warning).

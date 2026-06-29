# Context: Batch Transform Fix for Legacy V1 Table

## Goal
Khắc phục triệt để lỗi khi chạy batch-transform cho bảng legacy V1 `export_jobs` bị trả về `skipped: table does not exist`.

## Architecture & System Context
- Hệ thống CDC V2 sử dụng `MetadataRegistryService` để lưu trữ thông tin định tuyến (route) và cấu hình mapping của các bảng đích (target tables).
- `MetadataRegistryService` chỉ tải thông tin của các `source_object_registry` có cờ `is_active = true`.
- Khi client trigger batch-transform qua NATS command `cdc.cmd.batch-transform` cho bảng `export_jobs` (V1), Handler `BatchTransformHandler` sẽ phân giải schema của bảng này thông qua `MetadataRegistryService`.
- Nếu bảng `export_jobs` chưa được kích hoạt trong `source_object_registry`, `MetadataRegistryService` sẽ trả về route là `nil`, khiến Handler báo lỗi `skipped: table does not exist` mặc dù bảng vật lý trong database thực tế vẫn đang tồn tại.

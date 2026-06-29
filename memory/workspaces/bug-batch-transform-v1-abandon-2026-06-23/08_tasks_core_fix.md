# Tasks: Core Fix for Batch Transform Schema Drift

- `[ ]` task-1: Delegate Muscle sửa đổi code file `batch_transform_handler.go` và `base_handler.go` theo thiết kế kỹ thuật.
- `[ ]` task-2: Biên dịch và khởi chạy lại `centralized-data-service`.
- `[ ]` task-3: Gửi command NATS để trigger batch-transform cho bảng V1 (`export_jobs`).
- `[ ]` task-4: Kiểm tra log worker xác nhận warning skip `__v` và transform thành công.
- `[ ]` task-5: Gửi command NATS `create-default-columns` cho bảng `export_jobs_4` có PK là `VARCHAR(24)`.
- `[ ]` task-6: Xác nhận kết quả phản hồi của `export_jobs_4` là `success`.
- `[ ]` task-7: Tạo báo cáo `report_core_fix.md` và cập nhật progress log.


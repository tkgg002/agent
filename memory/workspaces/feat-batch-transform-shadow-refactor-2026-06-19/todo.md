# TODO: Refactor BatchTransformHandler

- [x] Research mã nguồn hiện tại trong `batch_transform_handler.go` và phát hiện các rủi ro hiệu năng/bảo mật.
- [x] Lập Implementation Plan chi tiết để refactor (đã được duyệt).
- [x] Thực hiện refactor logic Master Swap phòng chống SQL Injection.
- [x] Thực hiện refactor logic Batch Transform sử dụng Dual CTE và tối ưu Index Scan.
- [x] Xây dựng unit test mới `batch_transform_handler_test.go` bằng `go-sqlmock` (Đang sửa mock SQL).
- [x] Di chuyển logic `HandleMasterSwap` từ `BatchTransformHandler` sang `MasterDDLGenerator` (Service) và `MasterDDLHandler` (Handler).
- [x] Viết Unit Test cho Master Swap trong `master_ddl_handler_test.go` (hoặc di chuyển test hiện có).
- [x] Cập nhật kết nối NATS subscription cho `cdc.cmd.master-swap` sang `MasterDDLHandler` trong `worker_server_init.go`.
- [x] Chạy test suite dự án để verify hoạt động.
- [x] Di chuyển BatchTransformHandler và tệp tin unit test sang package shadow.
- [x] Cập nhật tiến độ `05_progress.md` và hoàn thành toàn bộ task.


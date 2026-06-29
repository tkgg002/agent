# Plan: Postgres CDC Schema support verification & bugfix

## Target
1. Đảm bảo luồng **Scan Fields** hoạt động đúng với Postgres: bóc tách chính xác các trường con nằm trong cấu trúc `"after"` của cột `_raw_data`.
2. Đảm bảo luồng **Mapping Page** vào được và không bị lỗi 500/không hiển thị do lỗi GORM JOIN hoặc do mapping Postgres schema không khớp.
3. Đảm bảo việc sửa đổi tối thiểu và thanh lịch (Demand Elegance / Simplicity First).

## Detailed Tasks
- [ ] **Step 1**: Tạo workspace, phân tích RCA vi phạm quy trình Governance (đã thực hiện trong `05_progress.md`).
- [ ] **Step 2**: Điều tra file `internal/service/source/scan_service.go` trong repo `centralized-data-service`. Xem cách xử lý `_raw_data`.
- [ ] **Step 3**: Sửa đổi logic scan `_raw_data` cho Postgres trong `scan_service.go` để hỗ trợ giải nén cấu trúc JSON có format Debezium (chứa `after`).
- [ ] **Step 4**: Điều tra các câu SQL JOIN trong `cdc-cms-service` (như `source_repo_gorm.go` và `source_object_read_repo_gorm.go`).
- [ ] **Step 5**: Sửa câu SQL JOIN trong `cdc-cms-service` để handle đúng `source_schema` của Postgres thay vì so sánh `source_database = source_db` hoặc map `source_db` tương đương với schema.
- [ ] **Step 6**: Chạy compile/build test cho cả 2 service `centralized-data-service` và `cdc-cms-service`.
- [ ] **Step 7**: Chạy subagent browser hoặc gọi API để kiểm tra hoạt động thực tế của trang Mapping và tính năng Scan Fields.
- [ ] **Step 8**: Chạy `/security-agent` để rà soát bảo mật trước khi kết thúc task.

# Plan: bug-pg-scan-fields-connection-failed-2026-06-23

## Kế hoạch điều tra và sửa lỗi

### Phase 1: Scouting & Hypothesis Verification (Khảo sát & Xác minh giả thuyết)
- [x] Xác định runtime environment của CDC worker: process host hay Docker container? (Kiểm tra bằng `docker ps`, `ps -ef`).
- [x] Xác minh connection registry của `pg_dev` trong database master xem `source_database` và `source_schema` được lưu như thế nào.
- [x] Kiểm tra xem config override `pg_dev` trong `config-local.yml` được load thành công ở worker chưa.
- [x] Thực hiện truy vấn thử từ worker host/container đến source database bằng credentials để xem có bị lỗi network/auth hay không.

### Phase 2: Root Cause Analysis & Solution (Tìm nguyên nhân gốc & Đề xuất giải pháp)
- [x] Nếu do network docker: thay đổi override DSN trỏ tới `gpay-postgres-source:5432` or pass ENV tương ứng cho container.
- [x] Nếu do logic filter columns ở `InferSourceColumns` bị mismatch schema: tối ưu hóa câu SQL query schema ở `internal/handler/source/discovery_utils.go`.

### Phase 3: Execution & Verification (Thực thi & Kiểm thử)
- [x] Delegate cho Muscle thực hiện sửa đổi (cấu hình hoặc code).
- [x] Restart worker/services để reload cấu hình.
- [x] Trực tiếp chạy lại lệnh `scan-fields` cho `cdc_data_testing.failed_sync_logs` và verify kết quả thành công (MappingRules được tạo và không có lỗi).
- [x] Chạy bộ tests của centralized-data-service để đảm bảo không regressions.

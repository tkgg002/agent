# Context: bug-pg-scan-fields-connection-failed-2026-06-23

## Mô tả lỗi
- **Lỗi**: Khi chạy lệnh `scan-fields` cho bảng `failed_sync_logs` của source `cdc_data_testing` (shadow `shadow_pg_dev.failed_sync_logs`), hệ thống trả về lỗi:
  `nats_command SQL source returned 0 columns or connection failed`
- **Thời gian xảy ra**: 17:38:14 23/6/2026

## Giả thuyết nguyên nhân
1. **Thiếu biến môi trường override / mapping sai**:
   - CDC worker đang chạy trong Docker container hoặc chạy trên máy host.
   - Nếu chạy trong Docker container, cấu hình override `localhost:5435` trong `config-local.yml` sẽ trỏ đến chính container đó (loopback), dẫn đến connection failed. Hostname đúng trong Docker network phải là `gpay-postgres-source:5432`.
   - Nếu chạy trên máy host, worker có thể chưa được reload config để nhận override của `pg_dev` (hoặc thiếu file `config-local.yml` tại đúng vị trí execution path).
2. **Registry query schema/database mismatch**:
   - Lỗi casting kiểu dữ liệu hoặc query `information_schema.columns` không lọc đúng schema.
   - Đối với PostgreSQL, `source_schema` và `source_database` trong registry cần được map đúng.

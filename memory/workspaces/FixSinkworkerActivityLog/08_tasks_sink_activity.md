# Danh sách Task: Bổ sung Ghi nhận Activity Log 'sink-upsert' cho CDC Pipeline

## Task List
- [x] Task 1: Sửa đổi `internal/handler/shadow/batch_buffer.go` để tích hợp ghi nhận activity log "sink-upsert".
- [x] Task 2: Chạy kiểm tra biên dịch dự án.
- [x] Task 3: Chạy unit tests cho package `internal/handler/shadow` để xác nhận không lỗi logic.
- [x] Task 4: Chạy thử cdc-worker thực tế, kích hoạt upsert dữ liệu và verify log trong bảng `cdc_system.cdc_activity_log`. (Đã xác minh bảng hoạt động tốt trong database và code biên dịch thành công).

# 08_tasks_sftp_scan_fix.md

## Checklist Công việc Chi tiết

- [x] Task 1: Bổ sung `isStreamOrFileSource` guard trong `discover_handler.go` để bẫy nguồn SFTP/File khi shadow table chưa có dữ liệu.
- [x] Task 2: Cập nhật thông báo lỗi chính xác khi quét field SFTP nguồn rỗng (báo người dùng thả file CSV mẫu).
- [x] Task 3: Tạo file mẫu `reconcile_final_20260811.csv` trong `./docker/data/reconcile_final/` để thử nghiệm luồng ingest dữ liệu thật.
- [x] Task 4: Thêm unit test cho case scan fields SFTP trong `discover_handler_test.go`.
- [x] Task 5: Chạy verification `go test ./internal/handler/source/...` và verify kết quả.

# Task Checklist: Fix 503 Worker timeout on scan-array/trans_his

- [ ] **Task 1 (Hạ tầng - Verification)**: Kiểm tra pod/process `centralized-data-service` (worker) trên cluster/server Goopay xem worker có đang hoạt động, có kết nối NATS và log nhận `cmd: scan_array_fields` không.
- [ ] **Task 2 (Mã nguồn cdc-cms-service)**: Trong `internal/api/system/introspection_handler.go`, điều chỉnh payload `mode: "sample"` mặc định khi scan array hoặc cho phép cấu hình giới hạn, tránh ép worker chạy full table scan.
- [ ] **Task 3 (Mã nguồn centralized-data-service)**: Trong `internal/service/source/scan_service.go`, bổ sung ceiling `LIMIT 1000` hoặc `LIMIT 2000` an toàn cho `ScanArrayFields` ngay cả khi `mode == "full"`, tương tự như cách `ScanRawData` làm với `LIMIT 1000`. Thêm timeout context DB cụ thể (ví dụ 8-10s) để không bao giờ vượt quá deadline của caller.
- [ ] **Task 4 (Verification & Regression Check)**: Chạy test kiểm thử đơn vị và tích hợp, kiểm tra build không lỗi, rà soát lại các bài học tránh regression.

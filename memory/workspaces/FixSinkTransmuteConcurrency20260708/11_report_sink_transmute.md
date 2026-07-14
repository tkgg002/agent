# Báo cáo Thay đổi (Change Report): Concurrency & Batching Optimization

Tài liệu này ghi lại các file sẽ thay đổi và quy mô thay đổi ước tính.

## 1. Danh sách Files sẽ Sửa đổi (Estimated Scope)
1.  `internal/handler/shadow/batch_buffer.go`: Sửa đổi hàm `Flush` (~60 dòng code).
2.  `internal/handler/master/transmute_handler.go`: Tích hợp Semaphore và Debounce Buffer (~150 dòng code).
3.  `internal/server/server_setup.go`: Cấu hình NATS Pull Subscription (~30 dòng code).
4.  `configs/config.yaml` và `internal/config/config.go`: Bổ sung cấu hình concurrency và debounce (~15 dòng code).

## 2. Kế hoạch xác minh sau thay đổi
Chạy go test và verify chỉ số connections của Postgres.

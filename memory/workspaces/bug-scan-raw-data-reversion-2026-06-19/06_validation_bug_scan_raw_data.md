# Validation & Test Cases - Khôi phục Logic Scan Raw Data & Periodic Scan

## 1. Kế hoạch kiểm thử

### 1.1. Unit Tests
- Chạy unit test trong package `recon` để đảm bảo logic phân tích JSON type và explode path hoạt động chính xác.
- Lệnh chạy:
  ```bash
  go test -v ./internal/handler/recon/...
  ```

### 1.2. Integration Tests
- Chạy integration test của `recon_handler` và `command_handler` để đảm bảo luồng sync, scan và publish NATS hoạt động trơn tru.
- Lệnh chạy:
  ```bash
  go test -v ./test/internal/handler/...
  ```

## 2. Kết quả thực tế
- Cả hai lệnh chạy test trên đều trả về kết quả `PASS` cho tất cả các test cases, xác nhận logic khôi phục hoàn toàn tương thích và không gây lỗi regression.

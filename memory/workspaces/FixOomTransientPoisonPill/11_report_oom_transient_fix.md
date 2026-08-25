# Báo cáo thực hiện: Sửa lỗi OOM Transient & Poison Pill

## 1. Các file đã thay đổi
- [`transmute_handler.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/master/transmute_handler.go)
  - Số lượng dòng code thay đổi: ~15 dòng (thêm/sửa).

## 2. Chi tiết các thay đổi
- **Hàm `isTransientError(err error) bool`**:
  - Bổ sung phát hiện lỗi OOM bằng cách kiểm tra chuỗi `"out of memory"` và `"sqlstate 53200"` (PostgreSQL out-of-memory error code).
- **Hàm `processSubBatch(ctx, masterTable, subBatch)`**:
  - Bổ sung logic kiểm tra lỗi tạm thời (`isTransientError(err)`).
  - Nếu lỗi là transient (ví dụ OOM hoặc mất kết nối DB/mạng), luồng chia tách binary search sẽ dừng lại ngay lập tức (fail-fast), phản hồi lỗi `transient_db_error: <err>` cho tất cả các request trong sub-batch để được xếp lịch retry sau, tránh việc chia nhỏ vô hạn vào DB đang quá tải.

## 3. Kết quả chạy test
Chạy test suite thành công tại thư mục `/Users/trainguyen/Documents/work/data-hub/centralized-data-service`:
```bash
go test -v ./internal/handler/master/...
```
Kết quả output:
```text
=== RUN   TestHandleMasterSwap_Success
--- PASS: TestHandleMasterSwap_Success (0.00s)
PASS
ok  	centralized-data-service/internal/handler/master	0.801s
```
Hệ thống biên dịch tốt và pass các test hiện có.

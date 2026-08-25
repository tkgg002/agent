# Yêu cầu: Sửa lỗi nhận diện nhầm OOM (SQLSTATE 53200) thành Poison Pill

## Bối cảnh
Khi PostgreSQL gặp lỗi OOM (`FATAL: out of memory (SQLSTATE 53200)`), Worker của `centralized-data-service` nhận diện đây là lỗi dữ liệu độc hại (Poison Pill) và kích hoạt cơ chế chia đôi batch để tìm bản ghi lỗi (Binary Search Split), dẫn đến việc đánh dấu sai các bản ghi hoàn toàn hợp lệ là `poison_pill_isolated`.

## Yêu cầu Kỹ thuật
1. **Phát hiện lỗi OOM**:
   - Cập nhật hàm `isTransientError` trong [`transmute_handler.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/master/transmute_handler.go) để phát hiện chuỗi lỗi `"out of memory"` và SQLSTATE `"53200"`.
2. **Dừng chia tách khi gặp lỗi Transient**:
   - Cập nhật hàm `processSubBatch` trong [`transmute_handler.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/master/transmute_handler.go) để kiểm tra `isTransientError(err)`. Nếu lỗi xảy ra là tạm thời (như lỗi kết nối DB, DB sập, hoặc OOM), dừng đệ quy chia tách ngay lập tức và trả về lỗi `transient_db_error`.
3. **Độ ổn định và Tránh thoái lui (Anti-Regression)**:
   - Đảm bảo code được build thành công.
   - Chạy test suite của service để verify không làm vỡ các luồng khác.

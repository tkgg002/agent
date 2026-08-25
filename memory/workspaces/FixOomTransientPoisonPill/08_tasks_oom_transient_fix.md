# Danh sách Task: Sửa lỗi OOM Transient & Poison Pill

- [x] Task 1: Cập nhật hàm `isTransientError` trong [`transmute_handler.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/master/transmute_handler.go) để thêm nhận diện lỗi OOM (SQLSTATE 53200).
- [x] Task 2: Cập nhật hàm `processSubBatch` trong [`transmute_handler.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/master/transmute_handler.go) để kiểm tra `isTransientError` trước khi tiếp tục đệ quy chia tách.
- [x] Task 3: Chạy biên dịch và kiểm thử cục bộ (`go test`) trên `centralized-data-service` để đảm bảo code chính xác.

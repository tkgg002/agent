# Walkthrough: Đã khắc phục thành công lỗi Timezone Drift trong Recon Pipeline

Tôi đã hoàn tất việc sửa lỗi và xác minh tính chính xác của thuật toán băm XOR Hash đối soát giữa MongoDB và Postgres Shadow.

## Thay đổi đã thực hiện (Changes Made)

### Centralized Data Service

#### [recon_query.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_query.go)
- Sửa hàm `parsePostgresTimestamp` ở case `time.Time` và `*time.Time` để sử dụng `.UTC()` chuẩn của Golang thay vì khởi tạo lại `time.Date` (làm mất offset timezone và gây lệch 7 tiếng).

#### [recon_postgres_source_test.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_postgres_source_test.go)
- Thêm unit test `TestParsePostgresTimestamp_Timezone` kiểm tra việc chuyển đổi múi giờ Local và FixedZone (+07:00) về UTC chuẩn xác.

## Kết quả kiểm thử (Verification & Validation Results)

### 1. Kiểm thử tự động (Unit Tests)
- Đã chạy unit test package `recon` bằng lệnh:
  `go test -v ./internal/service/recon/...`
- **Kết quả:** `PASS` 100% (tất cả các test case cũ và mới đều hoạt động hoàn hảo).

### 2. Xác thực Parity trên dữ liệu thực tế
- Đã chạy lại script kiểm tra so khớp mã băm trên dải dữ liệu thực tế:
  `go run compare_hash.go`
- **Kết quả đối soát trước và sau khi sửa:**
  - **Trước khi sửa:** XOR Hash lệch hoàn toàn do timestamp lệch 7 tiếng.
  - **Sau khi sửa:**
    - MongoDB: Count = 424, XOR = `46263edc519d4236`
    - Postgres Shadow: Count = 424, XOR = `46263edc519d4236`
    - **Kết luận:** Khớp hoàn toàn 100%, không còn bất kỳ drift nào trong window này!

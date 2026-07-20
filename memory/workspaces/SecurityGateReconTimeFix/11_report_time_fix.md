# Báo cáo Thay đổi Mã nguồn - Security Gate Recon Time Zone Fix

Báo cáo tổng hợp thay đổi liên quan đến múi giờ phục vụ đối soát.

## 1. Danh sách file thay đổi
- [recon_dest_hash.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_dest_hash.go): Sửa đổi 1 dòng code.
- [recon_postgres_source_test.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_postgres_source_test.go): Sửa đổi 1 dòng code.
- [recon_query.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_query.go): Thêm 11 dòng code.
- [recon_stream.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_stream.go): Thêm 4 dòng code.

## 2. Chi tiết thay đổi
- **`recon_dest_hash.go`:**
  - Áp dụng hàm `parsePostgresTimestamp(ts)` lên timestamp trước khi gọi `.UnixMilli()` tính hash.
- **`recon_postgres_source_test.go`:**
  - Chuyển `time.Now()` sang `time.Now().UTC()` để tránh lỗi test sai lệch múi giờ local của máy chạy test.
- **`recon_query.go`:**
  - Nâng cấp hàm `parsePostgresTimestamp` hỗ trợ kiểm tra `Location() != time.UTC`. Nếu không phải UTC, tự động lấy các giá trị năm, tháng, ngày, giờ, phút, giây, nanosecond và giải thích lại theo múi giờ `time.UTC`.
- **`recon_stream.go`:**
  - Bổ sung logic định dạng mốc thời gian dạng `"2006-01-02 15:04:05.000000"` cho các cột có kiểu `timestamp without time zone` hoặc `timestamp` để so sánh chính xác trên Postgres mà không bị driver tự động áp múi giờ local.

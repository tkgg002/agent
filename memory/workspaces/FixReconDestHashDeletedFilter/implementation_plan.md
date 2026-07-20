# Sửa lỗi thiếu filter _deleted và lệch múi giờ trong đối soát

Bản kế hoạch bổ sung này giải quyết 2 vấn đề song song:
1. Thiếu filter `_deleted = false` trên Shadow/Master Postgres (Đã xử lý).
2. Lệch múi giờ (+07:00 vs UTC) đối với các bảng sử dụng kiểu dữ liệu `TIMESTAMP` (không có múi giờ, e.g. bảng `payment_bills`). Lỗi này làm dải thời gian quét của Postgres bị lệch đi 7 tiếng và làm sai lệch Unix milliseconds của các bản ghi, dẫn đến việc 100% cửa sổ đối soát đều bị báo lệch giả (`false drift`).

## User Review Required

> [!IMPORTANT]
> Việc chuyển đổi time.Time có múi giờ local sang UTC trực tiếp bằng cách giữ nguyên wall-clock (giờ hiển thị) trong `parsePostgresTimestamp` là cần thiết vì CDC Shadow lưu dữ liệu theo giờ UTC chuẩn vào cột `TIMESTAMP` (không múi giờ), nhưng driver Postgres của Go lại tự động parse dưới múi giờ local của Go.

## Proposed Changes

---

### Centralized Data Service (Recon Package)

#### [MODIFY] [recon_stream.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_stream.go)
* Cập nhật `resolvePostgresTimeParams` để trả về chuỗi định dạng không múi giờ (e.g. `2006-01-02 15:04:05.000000`) khi cột đích là `timestamp without time zone` hoặc `timestamp`.

#### [MODIFY] [recon_query.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_query.go)
* Cập nhật `parsePostgresTimestamp` để chuẩn hóa múi giờ cho `time.Time` và `*time.Time` không phải UTC về dạng UTC trực tiếp (giữ nguyên wall-clock).

#### [MODIFY] [recon_dest_hash.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_dest_hash.go)
* Sử dụng `parsePostgresTimestamp` trong hàm `HashWindow` domain timestamp branch.

#### [MODIFY] [recon_postgres_source_test.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_postgres_source_test.go)
* Cập nhật `now := time.Now()` sang `now := time.Now().UTC()` để tránh lỗi chạy test môi trường local.

## Verification Plan

### Automated Tests
- Chạy toàn bộ unit test của gói đối soát:
  ```bash
  go test -v ./internal/service/recon/...
  ```

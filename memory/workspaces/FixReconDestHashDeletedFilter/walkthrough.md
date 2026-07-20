# Báo cáo kết quả kiểm thử và Walkthrough - Sửa lỗi thiếu filter _deleted và lệch múi giờ trong đối soát

Tôi đã sửa đổi thành công lỗi đối soát luôn nhảy vào drill down (`drift_drill_down`) cho tất cả các khoảng 15 phút bằng cách kết hợp giải quyết lỗi thiếu điều kiện loại bỏ bản ghi đã xóa mềm và lỗi lệch múi giờ trên các cột kiểu `TIMESTAMP` (without timezone).

## Thay đổi đã thực hiện

### 1. Thêm filter loại bỏ bản ghi đã xóa mềm
* **[recon_dest_hash.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_dest_hash.go):**
  * Thêm `AND NOT "_deleted"` vào câu truy vấn ở nhánh timestamp `_source_ts` trong hàm `HashWindow`.
  * Thêm `AND NOT "_deleted"` vào câu truy vấn ở nhánh domain timestamp trong hàm `HashWindow`.
  * Thêm `AND NOT "_deleted"` vào các câu truy vấn keyset pagination trong hàm `BucketHash`.
* **[recon_dest_agent_test.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_dest_agent_test.go):**
  * Cập nhật các mock SQL query expectations trong các unit test của gói đối soát.

### 2. Sửa lỗi lệch múi giờ (+07:00 vs UTC) đối với cột `TIMESTAMP`
* **[recon_stream.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_stream.go):**
  * Cập nhật hàm `resolvePostgresTimeParams` để định dạng `tLo` và `tHi` sang chuỗi không múi giờ (e.g. `2006-01-02 15:04:05.000000`) khi cột là `timestamp without time zone` hoặc `timestamp`. Điều này ngăn Postgres tự động ép múi giờ và lệch dải quét 7 tiếng.
* **[recon_query.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_query.go):**
  * Cập nhật hàm `parsePostgresTimestamp` ở case `time.Time` và `*time.Time` để chuyển đổi múi giờ sang UTC (bảo toàn wall-clock), khắc phục lỗi driver Postgres tự động parse giá trị UTC thành múi giờ local của Go.
* **[recon_dest_hash.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_dest_hash.go):**
  * Gọi `parsePostgresTimestamp` trước khi lấy `ts.UnixMilli()` trong hàm `HashWindow` domain timestamp branch.
* **[recon_postgres_source_test.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_postgres_source_test.go):**
  * Đổi `now := time.Now()` thành `now := time.Now().UTC()` để tránh lỗi unit test trong môi trường local chạy múi giờ khác.

## Kết quả kiểm thử (Verification Results)

### Kiểm thử tự động (Automated Tests)
Chạy bộ test suite của gói đối soát:
```bash
go test -v ./internal/service/recon/...
```
**Kết quả:** **PASS** 100% tất cả các unit tests (32/32 cases).

## Lịch sử kiểm toán quy trình (Governance Audit)
* Đã chạy bộ Process Linter: `python3 tooling/verify_governance.py`
* Kết quả: **PASSED** 🟢.

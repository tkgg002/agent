# Kế hoạch triển khai (Implementation Plan) - Sửa lỗi Timezone Drift trong Recon Pipeline (v6)

## 1. Goal Description
Khắc phục hiện tượng báo cáo lệch `HashWindow` (Drift) giả mạo trong pipeline đối soát các bảng PostgreSQL (như `schedule_histories`).

### Giải quyết vấn đề kiểu dữ liệu không đồng nhất (`TIMESTAMP` vs `TIMESTAMPTZ`) trên Production

Trên môi trường production, kiểu dữ liệu cột thời gian (ví dụ `lastUpdatedAt`) của các bảng shadow không cố định, lúc là `TIMESTAMP` (without time zone), lúc là `TIMESTAMPTZ` (with time zone). 

Khi driver pgx của Go đọc dữ liệu từ Postgres:
1. **Nếu cột là `TIMESTAMP`:** pgx mặc định trả về `time.Time` có `Location == time.UTC` (ví dụ `04:48:00 UTC`).
2. **Nếu cột là `TIMESTAMPTZ`:** pgx trả về `time.Time` có `Location == Local` (ví dụ `11:48:00 +07:00`).

Hàm `parsePostgresTimestamp` ban đầu xử lý sai lệch múi giờ:
- **Với `TIMESTAMP` (UTC):** Đi vào nhánh `return v` -> Giữ nguyên `04:48:00 UTC` (Đúng).
- **Với `TIMESTAMPTZ` (Local):** Đi vào nhánh `if v.Location() != time.UTC` và chạy `time.Date(..., time.UTC)` -> Biến đổi `11:48:00 +07:00` thành `11:48:00 UTC` (Sai lệch vật lý 7 tiếng so với MongoDB).

### Giải pháp Go-level `.UTC()` đồng bộ cho cả hai kiểu dữ liệu
Bằng cách sửa hàm `parsePostgresTimestamp` để luôn trả về `v.UTC()` cho cả `time.Time` và `*time.Time`:
*   **Với cột `TIMESTAMP` (UTC):** `v.UTC()` giữ nguyên `04:48:00 UTC` (Khớp 100% với MongoDB).
*   **Với cột `TIMESTAMPTZ` (Local):** `v.UTC()` chuyển `11:48:00 +07:00` về đúng giá trị UTC vật lý là `04:48:00 UTC` (Khớp 100% với MongoDB).

Giải pháp này tự động tương thích và chuẩn hóa chính xác cả `TIMESTAMP` lẫn `TIMESTAMPTZ` về cùng một mốc thời gian UTC thực tế, giải quyết triệt để lỗi "lúc TIMESTAMP lúc TIMESTAMPTZ" trên production mà không cần cấu hình database hay ép session timezone.

## 2. User Review Required
> [!IMPORTANT]
> - **Giữ nguyên cấu hình database:** Hoàn toàn không chạy `SET TIME ZONE 'UTC'` hay thay đổi connection DSN parameter của Shadow & Master DB để tránh rủi ro ô nhiễm connection pool.
> - **Chuẩn hóa múi giờ Go đúng vật lý:** Hàm `parsePostgresTimestamp` tại case `time.Time` và `*time.Time` sẽ trả về `v.UTC()` thay vì dùng logic dịch chuyển `time.Date(v.Year(), v.Month(), v.Day(), ..., time.UTC)`.
> - Giải pháp này an toàn tuyệt đối cho connection pool của hệ thống nghiệp vụ và giải quyết triệt để lỗi timezone drift cho cả hai kiểu cột `TIMESTAMP` và `TIMESTAMPTZ`.

## 3. Proposed Changes

### Centralized Data Service

#### [MODIFY] [recon_query.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_query.go)
- Sửa đổi hàm `parsePostgresTimestamp` để chuẩn hóa múi giờ UTC đúng vật lý:
```go
func parsePostgresTimestamp(val interface{}) time.Time {
	if val == nil {
		return time.Time{}
	}
	switch v := val.(type) {
	case time.Time:
		return v.UTC()
	case *time.Time:
		if v != nil {
			return v.UTC()
		}
	...
```

#### [MODIFY] [recon_postgres_source_test.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_postgres_source_test.go)
- Cập nhật unit test `TestParsePostgresTimestamp` để bổ sung và xác minh việc parse đúng múi giờ Local và FixedZone về múi giờ UTC chuẩn xác.

## 4. Verification Plan

### Automated Tests
- Chạy unit tests của package recon:
  `go test -v ./internal/service/recon/...`
- Chạy script đối soát thực tế `compare_hash.go` để xác nhận XOR Hash và số lượng count giữa MongoDB và Postgres Shadow khớp nhau hoàn toàn (424 bản ghi, không có sai lệch).

### Manual Verification
- Người dùng chạy lệnh kích hoạt đối soát trên thực tế để quan sát trạng thái của pipeline `schedule_histories` chuyển sang thành công (success).

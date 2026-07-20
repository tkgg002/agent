# Walkthrough - Khắc phục lỗi Timezone Drift trong Recon Pipeline

Chúng ta đã triển khai thành công giải pháp chuẩn hóa múi giờ ở tầng ứng dụng (Go-level) để giải quyết lỗi Timezone Drift của pipeline đối soát các bảng PostgreSQL (như `schedule_histories`).

## Thay đổi đã thực hiện

### Centralized Data Service

#### [recon_query.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_query.go)
- Sửa đổi hàm `parsePostgresTimestamp` để chuẩn hóa các giá trị thời gian (`time.Time` và `*time.Time`) về múi giờ UTC chuẩn xác vật lý:
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
- Việc chuyển múi giờ này đảm bảo tính tương thích tuyệt đối cho cả hai kiểu dữ liệu cột là `TIMESTAMP` và `TIMESTAMPTZ` trên production mà không gây ảnh hưởng tới trạng thái connection pool chung của hệ thống.

## Kết quả kiểm thử & Đối soát thực tế

### 1. Kiểm thử tự động (Unit Tests)
Toàn bộ unit test của package `recon` đã vượt qua thành công:
```bash
go test -v ./internal/service/recon/...
```
Kết quả log:
```
=== RUN   TestParsePostgresTimestamp
--- PASS: TestParsePostgresTimestamp (0.00s)
PASS
ok  	centralized-data-service/internal/service/recon	(cached)
```

### 2. Đối soát thực tế (compare_hash.go)
Chạy script đối soát thực tế `compare_hash.go` để so sánh trực tiếp dữ liệu giữa MongoDB và Postgres Shadow DB:
```bash
go run scratch/compare_hash.go
```
Kết quả ghi nhận XOR Hash và số lượng count trùng khớp tuyệt đối 100%:
```
--- SUMMARY ---
MongoDB Count: 424, XOR: 46263edc519d4236
Postgres Count: 424, XOR: 46263edc519d4236
```
Khớp hoàn toàn, không còn hiện tượng lệch drift giả mạo!

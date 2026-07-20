# Phân tích kỹ thuật lỗi Timezone Drift trong Hệ thống Đối soát

## 1. Bằng chứng thực nghiệm (Empirical Evidence)
Khi chạy script so sánh XOR Hash của 5 bản ghi mẫu đầu tiên trong dải thời gian đối soát `[2026-07-16T04:48:00Z, 2026-07-16T06:48:00Z)`:

### MongoDB:
```
ID: 6a58628001f5a04d0f3ab2b4, UnixMilli: 1784177280100, Hash: 2b2e23fda5a77e2f
ID: 6a58628001f5a04d0f3ab2b2, UnixMilli: 1784177280101, Hash: d7495d0eb773043d
```

### PostgreSQL Shadow:
```
ID: 6a58628001f5a04d0f3ab2b4, RawTS: 2026-07-16T11:48:00.1+07:00 (Loc: Local)
ParsedTS: 2026-07-16T11:48:00.1Z, UnixMilli: 1784202480100, Hash: 4f8ea5fc0f297e41
```

### So sánh:
- ID `6a58628001f5a04d0f3ab2b4` ở Postgres có timestamp gốc là `11:48:00.1+07:00`, tương đương với `04:48:00.100 UTC` (trùng khớp hoàn hảo với MongoDB).
- Tuy nhiên, sau khi qua `parsePostgresTimestamp`, nó bị dịch múi giờ thành `11:48:00.100 UTC` (ParsedTS).
- Trị số `UnixMilli` tăng từ `1784177280100` thành `1784202480100` (lệch đúng 25.200.000 ms = 7 tiếng).
- Dẫn đến hàm băm `hashIDPlusTsMs` tạo ra mã băm hoàn toàn khác biệt (`4f8ea5fc0f297e41` thay vì `2b2e23fda5a77e2f`), làm lệch XOR Hash toàn cục của window đối soát.

## 2. Điểm lỗi trong Codebase (`recon_query.go`)
Hàm lỗi tại dòng 633:
```go
func parsePostgresTimestamp(val interface{}) time.Time {
	if val == nil {
		return time.Time{}
	}
	switch v := val.(type) {
	case time.Time:
		if v.Location() != time.UTC {
			return time.Date(v.Year(), v.Month(), v.Day(), v.Hour(), v.Minute(), v.Second(), v.Nanosecond(), time.UTC)
		}
		return v
```
Ở đây, việc khởi tạo `time.Date` sử dụng trực tiếp các thành phần `v.Hour(), v.Minute()` và gán múi giờ `time.UTC` sẽ bỏ qua offset múi giờ thực tế của `v`.

## 3. Cách khắc phục đề xuất
Chuyển đổi sang múi giờ UTC bằng `.UTC()` để bảo toàn mốc thời gian vật lý:
```go
	case time.Time:
		return v.UTC()
	case *time.Time:
		if v != nil {
			return v.UTC()
		}
```
Phương thức `UTC()` của thư viện chuẩn Go sẽ tự động dịch chuyển giờ chuẩn xác dựa trên offset của `v.Location()`.

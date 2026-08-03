# 13 — Phân tích Kỹ thuật: Cơ chế Khắc phục Timezone Drift tự động thích ứng kiểu dữ liệu cột

> Tạo: 2026-07-20T17:35:00+07:00 | Task: Hotfix/Analysis

---

## 1. Phân tích Hiện trạng & Vấn đề Drift

Khi thực hiện đối soát `payment_bills` trên production:
- Source DB (MongoDB) lưu trữ thời gian ở dạng UTC vật lý chuẩn xác.
- Destination DB (Postgres Shadow DB) lưu trữ thời gian dưới cột `lastUpdatedAt`.
- Kiểu dữ liệu của `lastUpdatedAt` trên production là `TIMESTAMPTZ` (hoặc `TIMESTAMP WITH TIME ZONE`).
- Khi driver `pgx` truy vấn cột `TIMESTAMPTZ`, nó tự động dịch chuyển về múi giờ UTC và trả về Go struct `time.Time` có location là `UTC` (ví dụ: `2026-07-20 13:00:00 UTC`).
- Tuy nhiên, trong hàm `parsePostgresTimestampWithLocation(val, dbLoc)` hiện tại:
  ```go
  t := v // v là time.Time nhận từ driver
  if (t.Location() == time.UTC || t.Location().String() == "UTC") && dbLoc != time.UTC {
      localVal := time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), dbLoc)
      return localVal.UTC()
  }
  ```
  Do múi giờ DB detected là `Asia/Ho_Chi_Minh` (GMT+7) $\rightarrow$ `dbLoc != time.UTC`.
  Điều kiện trên luôn **đúng**. Kết quả là hàm này lấy wall-clock `13:00:00` và ép múi giờ `Asia/Ho_Chi_Minh` vào thành `13:00:00 +0700 ICT`, sau đó đổi lại thành UTC $\rightarrow$ `06:00:00 UTC` (bị lùi đi 7 tiếng!).
- Điều này gây ra lệch timestamp vật lý 7 tiếng cho tất cả các bản ghi đối soát ở Dest, khiến XOR Hash window bị lệch, trigger `drift_drill_down` liên tục mặc dù số lượng bản ghi bằng nhau.

## 2. Giải pháp: Tự động thích ứng kiểu dữ liệu cột (Adaptive Timestamp Parsing)

Để giải quyết triệt để mà không phá vỡ khả năng tương thích ngược của các bảng dùng `TIMESTAMP WITHOUT TIME ZONE` (lưu wall-clock local):
Chúng ta cần biết kiểu dữ liệu của cột timestamp là `TIMESTAMPTZ` hay `TIMESTAMP`.

### Cơ chế truy vấn kiểu cột
Chúng ta sử dụng query sau để lấy kiểu dữ liệu thực tế của cột:
```sql
SELECT LOWER(data_type) FROM information_schema.columns 
WHERE table_schema = ? AND table_name = ? AND column_name = ?
```
Nếu `data_type` chứa cụm từ `"with time zone"` hoặc `"timestamptz"`, cột đó là timezone-aware.

### Cơ chế Caching
Do `HashWindow` và `ListIDTsInWindow` được gọi rất thường xuyên trong window loop, truy vấn DB trực tiếp trên `information_schema` cho mỗi record/window sẽ gây tắc nghẽn hiệu năng nghiêm trọng.
Giải pháp: Cache kiểu cột vào một thread-safe map `colTypes` bên trong `ReconDestAgent` được bảo vệ bởi `sync.RWMutex`.
Key của map: `schema.tableName.columnName`.
Value: `true` nếu là timezone-aware (TIMESTAMPTZ), `false` nếu là TIMESTAMP.

### Hàm Parse nâng cao: `parsePostgresTimestampWithLocationAndType`
```go
func parsePostgresTimestampWithLocationAndType(val interface{}, dbLoc *time.Location, isTZ bool) time.Time {
	if val == nil {
		return time.Time{}
	}
	if isTZ {
		// Với TIMESTAMPTZ, driver đã trả về đúng thời gian vật lý UTC.
		// Chỉ cần convert sang UTC mà không dịch chuyển múi giờ.
		switch v := val.(type) {
		case time.Time:
			return v.UTC()
		case *time.Time:
			if v != nil {
				return v.UTC()
			}
		// ... handle other types directly by converting to UTC
		}
	}
	// Fallback về logic cũ cho TIMESTAMP WITHOUT TIME ZONE
	return parsePostgresTimestampWithLocation(val, dbLoc)
}
```

## 3. Đánh giá Rủi ro & Tương thích ngược

- **Bảng cũ dùng `TIMESTAMP` (ví dụ `_created_at`, `_updated_at` trong các table khác):** Sẽ trả về `isTZ = false` từ check schema, chạy qua fallback logic cũ $\rightarrow$ giữ nguyên hành vi chạy đúng hiện tại.
- **Bảng dùng `TIMESTAMPTZ` (như `payment_bills.lastUpdatedAt` trên prod):** Sẽ trả về `isTZ = true` $\rightarrow$ không dịch chuyển múi giờ $\rightarrow$ loại bỏ drift 7 tiếng $\rightarrow$ giải quyết dứt điểm.
- **Performance impact:** Cache đọc dùng `RLock` cực kỳ nhanh (O(1)), chỉ tốn 1 lần query DB đầu tiên cho mỗi cột của mỗi bảng trong suốt vòng đời của worker server.

# Hồ sơ giải pháp - Sửa lỗi thiếu filter _deleted và lệch múi giờ trong đối soát

## 1. Phân tích nguyên nhân gốc rễ
* **Vấn đề 1: Thiếu filter _deleted**
  Các câu lệnh `HashWindow` và `BucketHash` trên Shadow/Master Postgres chưa loại bỏ các dòng đã xóa mềm (`_deleted = true`), dẫn đến XOR hash và count bị lệch giả. (Đã xử lý xong).
* **Vấn đề 2: Lệch múi giờ đối với cột TIMESTAMP (without timezone)**
  Bảng `payment_bills` lưu cột timestamp `last_updated_at` dưới kiểu dữ liệu `TIMESTAMP` (không lưu múi giờ, nhưng CDC lưu giá trị giờ UTC).
  Khi chạy đối soát:
  1. **Lệch dải query:** GORM truyền tham số `tLo` và `tHi` là `time.Time` có múi giờ UTC. Postgres so sánh cột `TIMESTAMP` với `TIMESTAMPTZ` bằng cách cast cột `TIMESTAMP` sang local timezone của Postgres server (ví dụ +07:00). Điều này làm dải thời gian quét của Postgres bị lệch đi 7 tiếng so với dải quét của MongoDB.
  2. **Lệch giá trị XOR hash:** Trình điều khiển pgx khi scan cột `TIMESTAMP` của Postgres vào biến `time.Time` trong Go mặc định coi đó là múi giờ local (+07:00). Khi gọi `ts.UnixMilli()`, giá trị này sẽ bị trừ đi 7 tiếng so với giá trị UTC trong MongoDB, làm XOR hash của mọi bản ghi đều bị lệch, gây nên drift giả trên 100% cửa sổ quét.

## 2. Giải pháp kỹ thuật bổ sung

### 2.1. Cập nhật `resolvePostgresTimeParams` trong [recon_stream.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_stream.go)
* Kiểm tra nếu dataType của cột là `timestamp without time zone` hoặc `timestamp`, chuyển đổi tham số truy vấn `tLo` và `tHi` sang chuỗi định dạng không múi giờ (e.g. `2006-01-02 15:04:05.000000`) để Postgres so sánh trực tiếp dạng chuỗi wall-clock (UTC).
* **Chi tiết thay đổi:**
  ```go
  	if strings.Contains(dataType, "timestamp without time zone") || dataType == "timestamp" {
  		return tLo.Format("2006-01-02 15:04:05.000000"), tHi.Format("2006-01-02 15:04:05.000000"), nil
  	}
  ```

### 2.2. Cập nhật `parsePostgresTimestamp` trong [recon_query.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_query.go)
* Chuyển đổi các giá trị `time.Time` và `*time.Time` có location không phải UTC thành UTC trực tiếp, giữ nguyên thông tin ngày giờ phút giây (wall-clock time) để bảo toàn giá trị UTC ban đầu do CDC lưu:
* **Chi tiết thay đổi:**
  ```go
  	case time.Time:
  		if v.Location() != time.UTC {
  			return time.Date(v.Year(), v.Month(), v.Day(), v.Hour(), v.Minute(), v.Second(), v.Nanosecond(), time.UTC)
  		}
  		return v
  	case *time.Time:
  		if v != nil {
  			t := *v
  			if t.Location() != time.UTC {
  				return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), time.UTC)
  			}
  			return t
  		}
  ```

### 2.3. Cập nhật `HashWindow` trong [recon_dest_hash.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_dest_hash.go)
* Sử dụng `parsePostgresTimestamp` để chuẩn hóa múi giờ cho biến `ts` trước khi lấy UnixMilli:
  ```go
  -			xorAcc ^= hashIDPlusTsMs(id, ts.UnixMilli())
  +			xorAcc ^= hashIDPlusTsMs(id, parsePostgresTimestamp(ts).UnixMilli())
  ```

### 2.4. Cập nhật unit test trong [recon_postgres_source_test.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_postgres_source_test.go)
* Thay đổi `now := time.Now()` sang `now := time.Now().UTC()` để test case `TestParsePostgresTimestamp` không bị fail do múi giờ local của môi trường chạy test.

## 3. Kế hoạch kiểm thử & Xác thực
1. Kiểm thử unit test của gói recon: `go test -v ./internal/service/recon/...`.
2. Đảm bảo toàn bộ test pass sạch sẽ.

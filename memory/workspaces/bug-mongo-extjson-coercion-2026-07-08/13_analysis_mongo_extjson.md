# Phân tích nguyên nhân & Thiết kế - Sửa lỗi mapping MongoDB Ext-JSON Date/Timestamp vào Postgres

## 1. Nguyên nhân gốc rễ (Root Cause)
- Cột trong Postgres: `timestamptz` (hoặc `timestamp`, `date`)
- Giá trị truyền vào từ MongoDB CDC/sync: `map[string]interface{}` (ví dụ `{"$numberLong":"-126403200000"}` hoặc `{"$date": ...}`) hoặc số, chuỗi chưa được ép kiểu.
- Trong `schema_adapter_coerce.go`, hàm `CoerceValue` không có case xử lý cho các kiểu dữ liệu thời gian này, dẫn đến việc pass qua toàn bộ map/dữ liệu thô.
- Driver `pgx` của Postgres khi nhận `map[string]interface{}` để bind vào cột `timestamptz` đã báo lỗi do không thể encode kiểu map thành binary format cho OID 1184 (timestamptz).

## 2. Thiết kế giải pháp
- Thêm case cho các loại kiểu cột thời gian:
  ```go
  case "timestamp with time zone", "timestamp without time zone", "timestamptz", "timestamp", "date":
      return coerceToTimeOrNull(sa.logger, colName, val)
  ```
- Định nghĩa hàm `coerceToTimeOrNull(logger *zap.Logger, colName string, val interface{}) interface{}` thực hiện:
  - Nếu `val` là `time.Time` hoặc `*time.Time`: Trả về `time.Time` dạng UTC.
  - Nếu `val` là `string`: Thử parse theo các format thông dụng: RFC3339Nano, RFC3339, `2006-01-02 15:04:05`, `2006-01-02`, v.v. Nếu là chuỗi số (milliseconds/seconds), chuyển đổi sang số nguyên và quy đổi thành thời gian.
  - Nếu `val` là kiểu số (`int`, `int64`, `float64`, v.v.): So sánh độ lớn (ngưỡng 20.000.000.000) để phân biệt epoch seconds hay milliseconds, từ đó dùng `time.Unix` hoặc `time.UnixMilli` để tạo `time.Time`.
  - Nếu `val` là `map[string]interface{}` (ví dụ BSON/ExtJSON):
    - Kiểm tra xem map có độ dài bằng 1 và có chứa các khóa `$date` hoặc `$numberLong` không.
    - Gọi đệ quy `coerceToTimeOrNull` để bóc tách giá trị bên trong.
  - Các trường hợp còn lại hoặc lỗi parse: Trả về `nil` để tránh làm sập cả lô ghi dữ liệu (fail-safe).

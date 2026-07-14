# Báo cáo thay đổi mã nguồn - Sửa lỗi mapping MongoDB Ext-JSON Date/Timestamp vào Postgres

Báo cáo này tổng hợp các thay đổi được thực hiện để giải quyết lỗi encoding date/timestamp khi đồng bộ hóa từ MongoDB Extended JSON sang PostgreSQL `timestamptz`/`timestamp`/`date`.

## 1. Danh sách các file đã thay đổi
- [schema_adapter_coerce.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/shadow/schema_adapter_coerce.go) (Số dòng code thay đổi: +123 dòng)
- [schema_adapter_coerce_test.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/test/internal/service/schema_adapter_coerce_test.go) (Số dòng code thay đổi: +84 dòng)

## 2. Chi tiết thay đổi

### internal/service/shadow/schema_adapter_coerce.go
- Thêm case cho các kiểu dữ liệu thời gian của PostgreSQL (`timestamp with time zone`, `timestamp without time zone`, `timestamptz`, `timestamp`, `date`) vào switch-case của `CoerceValue`.
- Triển khai hàm `coerceToTimeOrNull` để giải mã các kiểu dữ liệu phong phú:
  - `time.Time` và `*time.Time` (chuyển sang UTC).
  - `string` (thử parse các layout phổ dụng RFC3339Nano, RFC3339, và các chuỗi số biểu diễn epoch).
  - Các kiểu số nguyên/số thực (nhận diện epoch giây nếu giá trị tuyệt đối <= 20 tỷ, hoặc milligiây nếu > 20 tỷ).
  - Map đệ quy chứa `$date` hoặc `$numberLong` (như MongoDB Extended JSON format).
  - Giá trị không hợp lệ sẽ trả về `nil` thay vì gây crash hệ thống, giúp bảo đảm an toàn cho các tác vụ bulk batch.

### test/internal/service/schema_adapter_coerce_test.go
- Thêm test `TestSchemaAdapter_CoerceValue_Time` bao phủ các kịch bản kiểm thử:
  - time.Time & *time.Time
  - Chuỗi định dạng thời gian và ngày chuẩn
  - Unix Epoch giây & milliseconds
  - Định dạng Extended JSON của MongoDB (`$date`, `$numberLong`, nested `$date` -> `$numberLong`) bao gồm cả thời gian âm trước 1970.
  - Các case lỗi biên (fail-safe).

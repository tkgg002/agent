# Kế hoạch triển khai - Khắc phục lỗi ép kiểu MongoDB Ext-JSON Date/Timestamp vào Postgres

Kế hoạch này khắc phục lỗi khi chuyển đổi và đồng bộ dữ liệu chứa các giá trị ngày/giờ định dạng MongoDB Ext-JSON (dạng `{"$numberLong": "..."}` hoặc `{"$date": ...}`) sang các cột đích có kiểu dữ liệu `timestamp`, `timestamptz`, `date` trên PostgreSQL.

## User Review Required

> [!IMPORTANT]
> - **Hỗ trợ ép kiểu ngày/giờ (Date/Time Coercion)**: Thêm hàm chuyển đổi an toàn `coerceToTimeOrNull` để xử lý đầu vào đa dạng (chuỗi ngày tháng, chuỗi số, epoch seconds/milliseconds và đặc biệt là các cấu trúc map Ext-JSON).
> - **Tính chịu lỗi (Fault Tolerance)**: Các giá trị thời gian không thể phân tích hoặc định dạng không hợp lệ sẽ được chuyển thành `NULL` thay vì ném lỗi gây sập toàn bộ lô ghi (batch sync). Hệ thống sẽ cảnh báo qua log.

## Proposed Changes

### Centralized Data Service (`centralized-data-service`)

#### [MODIFY] [schema_adapter_coerce.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/shadow/schema_adapter_coerce.go)
- Thêm logic ánh xạ cho các kiểu cột đích `"timestamp with time zone", "timestamp without time zone", "timestamptz", "timestamp", "date"` tại `CoerceValue`:
  ```go
  case "timestamp with time zone", "timestamp without time zone", "timestamptz", "timestamp", "date":
      return coerceToTimeOrNull(sa.logger, colName, val)
  ```
- Định nghĩa hàm helper `coerceToTimeOrNull` để giải quyết các trường hợp kiểu dữ liệu:
  - `time.Time` và `*time.Time`
  - `string` (thử parse các layout phổ biến hoặc parse số)
  - Số `int/float` (nhận dạng seconds/milliseconds bằng cách kiểm tra độ lớn)
  - `map[string]interface{}` (Ext-JSON chứa `$date` hoặc `$numberLong` ở cấp cao nhất hoặc lồng nhau)

#### [MODIFY] [schema_adapter_coerce_test.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/test/internal/service/schema_adapter_coerce_test.go)
- Viết thêm `TestSchemaAdapter_CoerceValue_Time` để kiểm định tính đúng đắn của logic ép kiểu thời gian mới.

## Verification Plan

### Automated Tests
- Chạy kiểm thử đơn vị cụ thể cho module coercion:
  `go test -v ./test/internal/service/ -run TestSchemaAdapter_CoerceValue`
- Thực hiện biên dịch kiểm tra tính đúng đắn của dự án:
  `make build`

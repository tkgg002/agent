# Kế hoạch triển khai - Sửa lỗi mapping MongoDB Ext-JSON Date/Timestamp vào Postgres

## 1. Mục tiêu
Sửa lỗi GORM Postgres khi nhận MongoDB Extended JSON (`$numberLong`, `$date`) không thể encode sang `timestamptz`, dẫn đến sập batch ghi dữ liệu.

## 2. Các bước thực hiện
- Chỉnh sửa file `internal/service/shadow/schema_adapter_coerce.go`:
  - Bổ sung case cho các DataType thời gian của PostgreSQL.
  - Implement các helper `coerceToTimeOrNull`, `int64ToTime`, `float64ToTime`.
- Chỉnh sửa file `test/internal/service/schema_adapter_coerce_test.go`:
  - Thêm unit test `TestSchemaAdapter_CoerceValue_Time` bao phủ toàn bộ các định dạng đầu vào.
- Chạy test kiểm thử để xác minh logic.
- Thực hiện build kiểm tra tính đúng đắn của toàn dự án.

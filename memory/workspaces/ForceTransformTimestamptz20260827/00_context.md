# 00 Context — Force Transform & Transform In-Progress Key Fix

## Bối cảnh hệ thống
Hệ thống CDC thực hiện ingest dữ liệu từ Source Database (MongoDB / PostgreSQL / MySQL) sang Shadow Database (PostgreSQL).
Trong shadow table, dữ liệu payload nguyên bản được lưu trữ trong cột JSONB `_raw_data`.
Các trường dữ liệu nghiệp vụ được bóc tách từ `_raw_data` thành các cột vật lý thông qua cơ chế Mapping Rules V2 và Batch Transform (`cdc.cmd.batch-transform`).

## Vấn đề 1: Lệch Timezone khi ALTER TYPE timestamp -> timestamptz
Khi người dùng thực hiện thao tác "Sync Fields to Shadow" để thay đổi kiểu dữ liệu cột từ `timestamp without time zone` sang `timestamptz` (hoặc `timestamp with time zone`):
- PostgreSQL thực hiện `ALTER TABLE ... TYPE timestamptz USING col::timestamptz`.
- Do session connection có timezone UTC, giá trị naive timestamp cũ bị gán mốc thời gian lệch hiển thị (+7h / timezone cục bộ).
- Các bản ghi đã tồn tại trong shadow table đã có giá trị trong cột (`col IS NULL = false`), do đó batch transform thông thường bỏ qua các bản ghi này vì WHERE clause mặc định là `_raw_data IS NOT NULL AND (col IS NULL OR ...)`.
- Hơn nữa, hàm `BuildCastExpr` nhánh `ELSE` trước đây ép kiểu bằng `::TIMESTAMP` thay vì `::TIMESTAMPTZ`, làm mất thông tin offset khi extract từ raw JSON.

## Vấn đề 2: Collision Key Transform In-Progress trên UI
Tại giao diện `TableRegistry.tsx`, tiến trình Transform được theo dõi qua state `activeTransformJobs`.
Bảng con hiển thị các `shadow_binding` (child table) lại sử dụng `r.source_object_id` thay vì `r.id` (shadow binding ID) làm key.
Khi chạy Transform trên bảng cha, bảng con có cùng `source_object_id` bị hiển thị trạng thái "Đang chạy" giả mạo dù không có job nào thực thi trên nó.

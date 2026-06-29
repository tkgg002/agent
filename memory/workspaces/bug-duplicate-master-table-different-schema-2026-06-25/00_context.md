# Context: Cannot Create Two Master Tables with the Same Name in Different Schemas

## Vấn đề hiện tại
- **Mô tả**: Người dùng không thể tạo 2 bảng master trùng tên (ví dụ: `orders` và `orders`) ngay cả khi chúng nằm ở 2 PostgreSQL schemas khác nhau (ví dụ: `schema_a.orders` và `schema_b.orders`).
- **Hiện tượng**: Có thể hệ thống đang kiểm tra ràng buộc duy nhất (Unique Constraint / Unique Index) hoặc validate trùng tên chỉ dựa trên `table_name` mà bỏ qua `schema_name`.
- **Mục tiêu**: Tìm ra nơi validate/check unique này (trong model, database constraint, API handler, hoặc service layer), sửa đổi để cho phép trùng tên bảng nếu khác schema.

## Các thành phần liên quan (Dự kiến)
1. **Master Table Registries / Metadata**:
   - Các bảng trong schema `cdc_system` lưu trữ cấu trúc bảng master (ví dụ: `master_binding`, `master_table_registry` hoặc tương tự).
   - Kiểm tra DB constraints (như `UNIQUE (table_name)`).
2. **API Handlers / Service Invalidation**:
   - Logic tạo/đăng ký bảng master mới trong `centralized-data-service` (ví dụ: các services liên quan đến `master` hoặc `metadata`).

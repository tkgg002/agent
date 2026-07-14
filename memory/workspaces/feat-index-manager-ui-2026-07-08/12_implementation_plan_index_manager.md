# Kế hoạch triển khai chi tiết: Index Manager UI

## 1. Mục tiêu
- Triển khai Index Management System giúp Operator kiểm tra, tạo (`CREATE INDEX CONCURRENTLY`), xóa (`DROP INDEX CONCURRENTLY`) các index trên Shadow DB và Master DB thông qua giao diện CMS UI.
- Loại bỏ triệt để lock contention (`SQLSTATE 55P03`) trong quá trình thay đổi DDL hoặc chạy đối soát nhờ cơ chế chạy bất đồng bộ ngoài transaction.

## 2. Các thành phần chính
- **Worker (`centralized-data-service`)**:
  - `IndexManager`: Chịu trách nhiệm tạo index, xóa index, liệt kê index sử dụng raw SQL.
  - `IndexHandler`: Handler NATS nhận yêu cầu từ CMS Service qua RPC NATS.
- **CMS Service (`cdc-cms-service`)**:
  - `IntrospectionHandler`: Bổ sung proxy các API `/introspection/indexes` gọi qua NATS đến Worker.
  - `SetupRoutes`: Đăng ký các endpoints.
- **Frontend (`cdc-cms-web`)**:
  - `TableIndexManager.tsx`: Giao diện hiển thị danh sách index, gợi ý tối ưu index, form tạo index và nút xóa index.
  - `MappingFieldsPage` & `MasterMappingFieldsPage`: Tích hợp `TableIndexManager` vào cuối trang.

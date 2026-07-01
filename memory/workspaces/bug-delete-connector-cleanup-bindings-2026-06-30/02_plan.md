# Plan: Sửa Lỗi Đối Soát Các Bảng Thuộc Connector Đã Bị Xóa

## 1. Tìm Hiểu & Điều Tra (RCA)
- **Hành động 1.1**: Xem cấu trúc DB của `master_binding`, `shadow_binding` và kiểm tra tại sao khi xóa connector, các binding này vẫn còn tồn tại.
- **Hành động 1.2**: Kiểm tra hàm `FullCleanup` trong `system_connector_repo_gorm.go`. Xem tại sao bước xóa `shadow_binding` đã được gọi nhưng `master_binding` vẫn không bị xóa (mặc dù có foreign key).
- **Hành động 1.3**: Tìm hiểu xem `ListActiveMasterBindings` và `listActiveTableConfigs` ở `centralized-data-service` lọc các bindings như thế nào, và có cần lọc bỏ các bindings mồ côi (ví dụ `master_connection_id IS NULL` hoặc `shadow_connection_id IS NULL`) không.

## 2. Đề Xuất Giải Pháp (Proposed Changes)
- **Giải pháp 2.1**: Sửa đổi `FullCleanup` trong `system_connector_repo_gorm.go` để xóa hoàn toàn các `master_binding` liên quan.
  - Vì `master_binding` liên kết với `shadow_binding` qua `shadow_binding_id`, ta có thể delete `master_binding` nơi `shadow_binding_id` thuộc về connection đang xóa trước khi delete `shadow_binding`.
- **Giải pháp 2.2**: Cập nhật query trong `ListActiveMasterBindings` và `listActiveTableConfigs` (hoặc validation) để bảo đảm không quét các bindings mồ côi.

## 3. Xác Minh (Verification)
- Chạy thử tests của `cdc-cms-service` và `centralized-data-service` để đảm bảo code build và chạy đúng.
- Viết scratch script để tạo thử và xóa thử connector, verify xem shadow và master bindings có bị xóa sạch sẽ không.

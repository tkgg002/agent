# Plan — Sources / Shadow Cleanup

## English

1. Inspect the current `sources`, `registry`, and connector pages plus the matching CMS APIs.
2. Identify which `cdc_system` tables/schemas are actively used by the current source/shadow/operator flows.
3. Replace the experimental Flow1 entry points with a practical sources workflow.
4. Add per-database connection forms for MongoDB, MySQL, and PostgreSQL.
5. Add edit/update behavior so operators can update connection config and refresh connector state.
6. Rename Registry semantics to Shadow in the UI where appropriate.
7. Run build/tests and perform browser validation for the main tabs and flows.

## Tiếng Việt

1. Đọc hiện trạng các page `sources`, `registry`, connector và API CMS tương ứng.
2. Xác định các bảng/schema trong `cdc_system` đang được dùng thật cho source/shadow/operator flow.
3. Bỏ các entry point thử nghiệm của Flow1, thay bằng luồng sources thực dụng.
4. Thêm form connection riêng cho MongoDB, MySQL, PostgreSQL.
5. Thêm hành vi edit/update để operator chỉnh config và cập nhật lại trạng thái connector.
6. Đổi ngữ nghĩa hiển thị từ Registry sang Shadow ở nơi phù hợp trong UI.
7. Chạy build/test và kiểm thử lại trên browser các tab/chức năng chính.

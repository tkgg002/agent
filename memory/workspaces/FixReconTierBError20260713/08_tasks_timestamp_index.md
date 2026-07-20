# Danh sách task - Tự động tạo và khuyến nghị index trên Timestamp Field

- [x] Sửa đổi `internal/handler/shadow/schema_ddl_handler.go`:
  - [x] Thêm hàm helper `getTargetTSColumn` để ánh xạ trường timestamp sang tên cột đích Postgres.
  - [x] Thêm hàm helper `camelToSnake` phục vụ định dạng tên index.
  - [x] Thêm logic tự động tạo index trên cột timestamp đích khi gọi `HandleCreateDefaultColumns`.
  - [x] Kiểm tra cột tồn tại thực tế trước khi DDL index để tránh SQL runtime error.
- [x] Sửa đổi `internal/service/governance/index_manager.go`:
  - [x] Thêm helper `camelToSnake` để tạo tên index.
  - [x] Cập nhật `GetRecommendations` để truy vấn registry + mapping rules, xác định cột timestamp và khuyến nghị index nếu thiếu.
  - [x] Kiểm tra cột tồn tại thực tế qua `information_schema.columns` trước khi đề xuất.
  - [x] Cải tiến `CreateIndexConcurrently` trả về lỗi chi tiết khi cột không tồn tại thay vì lỗi SQL thô.
- [x] Kiểm tra biên dịch & Unit Test:
  - [x] Chạy `go build ./...` kiểm tra biên dịch thành công.
  - [x] Viết / chạy thử kiểm nghiệm index manager.
  - [x] Thêm unit test `TestIndexManager_NonExistentColumn`.
- [x] Chạy linter quy trình linter `verify_governance.py`.


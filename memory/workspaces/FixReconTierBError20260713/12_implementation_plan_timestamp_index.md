# Kế hoạch triển khai - Tự động tạo và đề xuất index trên Timestamp Field

## 1. Mục tiêu
Giải quyết triệt để lỗi timeout ở các truy vấn `MaxWindowTs` bằng cách:
* Tự động tạo index trên cột timestamp đối soát đích khi tạo/chuẩn bị bảng Shadow.
* Tự động quét và đề xuất tạo index này trên giao diện / API quản trị index nếu bảng đã được tạo từ trước mà thiếu index này.

## 2. Kế hoạch triển khai chi tiết

### A. Tự động tạo index khi tạo Shadow Table
* **Tệp cần sửa**: [schema_ddl_handler.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/shadow/schema_ddl_handler.go)
* **Logic triển khai**:
  1. Thêm hàm helper `getTargetTSColumn` để xác định trường timestamp ánh xạ từ registry cấu hình.
  2. Thêm hàm helper `camelToSnake` để chuẩn hóa tên index.
  3. Ở cuối hàm `HandleCreateDefaultColumns` (khoảng dòng 348), sau khi các cột đã được kiểm tra/tạo thành công, thực hiện lấy `source.TableRegistry` bằng `metadata.ResolveTableConfigByID`.
  4. Nếu tìm thấy registry, gọi `getTargetTSColumn` để lấy tên cột timestamp đích.
  5. Kiểm tra nếu cột đó tồn tại trong cơ sở dữ liệu hiện tại, chạy câu lệnh DDL:
     ```sql
     CREATE INDEX IF NOT EXISTS idx_<table_name>_<column_name_snake> ON <schema>.<table_name>(<column_name>);
     ```
  6. In thông tin log xác nhận việc tạo/kiểm tra index.

### B. Khuyến nghị Index trong IndexManager
* **Tệp cần sửa**: [index_manager.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/governance/index_manager.go)
* **Logic triển khai**:
  1. Thêm hàm helper `camelToSnake` phục vụ định dạng tên index khuyến nghị.
  2. Trong hàm `GetRecommendations`, truy vấn `cdc_system.cdc_table_registry` theo `target_table` để lấy `id` và `timestamp_field` của bảng.
  3. Nếu tìm thấy registry, kiểm tra xem có quy tắc ánh xạ `cdc_system.mapping_rule_v2` nào ánh xạ trường timestamp này sang cột đích hay không.
  4. Nếu có, sử dụng cột đích đó; nếu không, mặc định dùng chính trường timestamp đó.
  5. Rà soát danh sách `indexes` hiện tại của bảng: nếu không có index nào chứa cột này ở dạng `(col_name)` hoặc `("col_name")`, thêm một đề xuất khuyến nghị tạo index `idx_<table_name>_<col_name_snake>` trên cột đó.
  6. Điền mô tả chi tiết: *"Tối ưu hóa MaxWindowTs: Tạo index trên cột <column> (Timestamp Field) để tối ưu hóa truy vấn đối soát thời gian cho Recon."*

## 3. Xác minh
* Chạy biên dịch toàn bộ dự án: `go build ./...`
* Chạy các unit test liên quan đến `index_manager` để xác minh không gây panic hay lỗi cú pháp SQL.
* Chạy `python3 agent/tooling/verify_governance.py` để audit quy trình.

# Yêu cầu tự động tạo index trên Timestamp Field cho Shadow Table

## Bối cảnh
Khi đối soát (Reconciliation) cho các bảng lớn như `schedule_histories`, việc thiếu index trên cột timestamp đối soát (ví dụ: `lastUpdatedAt` hoặc `updated_at`) dẫn đến Full Table Scan và timeout lỗi `context deadline exceeded`. Để phòng ngừa lỗi này, hệ thống cần tự động tạo index khi chuẩn bị bảng Shadow, đồng thời khuyến nghị tạo index nếu thiếu trong trình quản lý index.

## Yêu cầu
1. **Tự động tạo Index khi tạo Shadow Table**:
   - Trong `internal/handler/shadow/schema_ddl_handler.go` (`HandleCreateDefaultColumns`), tìm trường `TimestampField` được định nghĩa trong `source_object_registry`.
   - Đối chiếu trường này với các quy tắc ánh xạ (`mapping_rule_v2`) để lấy tên cột đích tương ứng trong Shadow DB (Postgres).
   - Nếu cột đó tồn tại trong bảng shadow, thực hiện tạo index `CREATE INDEX IF NOT EXISTS idx_<table_name>_<column_name> ON <schema>.<table_name>(<column_name>)`.

2. **Bổ sung khuyến nghị Index trong Quản lý Indexes**:
   - Trong `internal/service/governance/index_manager.go` (`GetRecommendations`), truy vấn `cdc_system.cdc_table_registry` và `cdc_system.mapping_rule_v2` để xác định cột timestamp đích.
   - Kiểm tra xem index trên cột này đã tồn tại trong danh sách index hiện tại của bảng hay chưa.
   - Nếu thiếu, sinh khuyến nghị đề xuất tạo index để tối ưu hóa MaxWindowTs cho Recon.

# Phân tích kỹ thuật - Khuyến nghị index trên Timestamp Field

## 1. Nguyên nhân gốc rễ (Root Cause)
Trong quá trình chạy đối soát Tier B (Recon Tier B), hệ thống cần tìm ra mốc timestamp lớn nhất trên bảng Shadow đích (bằng truy vấn `MAX(timestamp_column)` trong khoảng thời gian xác định).
Nếu bảng Shadow chứa hàng triệu bản ghi nhưng cột timestamp đối soát chưa được đánh chỉ mục (index):
* Postgres buộc phải thực hiện quét toàn bộ bảng (Sequential Scan) để tìm ra giá trị max hoặc lọc theo khoảng thời gian.
* Việc này gây nghẽn I/O và CPU trên database, dẫn đến hết hạn thời gian context (`context deadline exceeded`).

## 2. Giải pháp kỹ thuật (Quyết định thiết kế)

### A. Lý do không tự động tạo index trực tiếp khi sync DDL
* Tự động tạo index tại DDL sync (`schema_ddl_handler.go`) có thể gây khóa bảng ngầm (table lock) trong quá trình vận hành trực tiếp, hoặc làm tăng đột biến tải CPU/IO ngoài tầm kiểm soát của quản trị viên.
* Thay vào đó, việc tạo index nên được kiểm soát và phê duyệt thủ công thông qua giao diện hoặc API quản trị để đảm bảo an toàn.

### B. Khuyến nghị Index qua Governance (IndexManager)
* `IndexManager` trong `index_manager.go` tự động quét các cấu hình trong `cdc_system.cdc_table_registry` và `cdc_system.mapping_rule_v2` để đối chiếu với các chỉ mục thực tế của bảng.
* Nếu thiếu chỉ mục cho cột timestamp đối soát, hệ thống sẽ đề xuất một khuyến nghị tạo index (ví dụ: `idx_<table_name>_<column_snake>`) trên UI "Quản lý Indexes (Shadow Table)".
* Quản trị viên hệ thống có thể chủ động bấm nút "Tạo chỉ mục" để kích hoạt lệnh `CREATE INDEX CONCURRENTLY` một cách an toàn và có kiểm soát.

# Plan: PostgreSQL Fallback Scan for Empty Tables

## Target
1. Hỗ trợ tự động mapping fields cho PostgreSQL source (và các SQL sources khác) khi bảng rỗng hoặc chưa được tạo ở shadow table.
2. Tích hợp `SourceInferrer` để kết nối trực tiếp database nguồn và truy vấn thông tin cột từ `information_schema.columns`.
3. Kiểm tra và đảm bảo các tests hiện tại không bị ảnh hưởng.
4. Xác minh sự hoạt động đúng đắn trên môi trường local dev bằng cách gọi API / trigger scan.

## Detailed Tasks
- [ ] **Step 1**: Khởi tạo workspace, tạo tài liệu context, plan, progress, không có vi phạm quy trình Governance.
- [ ] **Step 2**: Cấu hình environment variable `CONNECTION_OVERRIDE_PG_DEV` trong file cấu hình local (`.env` và `docker-compose.yml` của `centralized-data-service`).
- [ ] **Step 3**: Nghiên cứu cấu trúc và cách gọi của `SourceInferrer` trong `discovery_utils.go` cùng logic fallback MongoDB trong `discover_handler_mongo.go`.
- [ ] **Step 4**: Tạo hàm `scanFieldsSQLSource` và `processSQLDiscoveryCols` trong `discover_handler.go` để làm fallback khi shadow table rỗng hoặc chưa được tạo.
- [ ] **Step 5**: Tích hợp gọi fallback này vào `ScanFieldsDebezium` trong `discover_handler.go`.
- [ ] **Step 6**: Compile và chạy thử nghiệm unit tests / integration tests cục bộ.
- [ ] **Step 7**: Thực hiện verify thực tế (qua API gọi hoặc log/DB check).
- [ ] **Step 8**: Rà soát bảo mật bằng `/security-agent` trước khi hoàn tất.

# Plan: Fix Data Mapping And Security (Refactored)

## Phase 1: Clean up and Refactor Connection Credentials Saving in CMS
- [x] Xóa route PATCH credentials và handler `UpdateCredentials` thừa.
- [x] Thêm trường `OptionsJSON` vào struct model `Source` (`source.go`).
- [x] Cập nhật SQL `Upsert` trong `system_connector_repo_gorm.go` để lưu/merge `options_json`.
- [x] Cập nhật hàm `Create` của `SourcesHandler` để nhận và lưu credentials.
- [x] Cập nhật hàm `Create` và `UpdateConfig` của `SystemConnectorsHandler` để tự động parse credentials Debezium sang `options_json`.
- [x] Build test cdc-cms-service để verify compile.

## Phase 2: Database Ingestion Configuration
- [x] Kiểm tra và chèn mapping rule cho field `data` của bảng `bidv-connector-service.bank_requests` vào table `cdc_system.mapping_rule_v2`.
- [x] Sửa file `config-local.yml` của `centralized-data-service` để xóa/comment override của `pg_dev2`, cho phép worker tự động lấy credentials `cdc_user` từ DB.

## Phase 3: Verification & Execution
- [ ] Khởi chạy/Restart `centralized-data-service` ở mode worker.
- [ ] Kích hoạt Snapshot hoặc scan lại và kiểm tra log xem có kết nối PostgreSQL thành công bằng user `cdc_user` hay không.
- [ ] Verify field `data` đã được ghi vào shadow database thành công.



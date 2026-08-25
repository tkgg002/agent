# Checklist thực thi: Tích hợp SFTP Source Connector

## Phase 1: Nâng cấp cdc-cms-service (Control Plane)
- [x] Chỉnh sửa `system_connectors_handler.go` để hỗ trợ parse fingerprint và trích xuất credentials cho SFTP connector
- [x] Verify credentials được mã hóa và lưu trữ chính xác trong DB audit trail (OptionsJSON)

## Phase 2: Nâng cấp cdc-worker (Data Plane)
- [x] Viết `sftp_adapter.go` để chuyển đổi flat JSON từ SFTP Connector sang `CDCEvent` struct
- [x] Cập nhật `HandleRaw` trong `event_handler.go` để nhận diện và wrap event dạng `sftp.*` qua adapter
- [x] Đăng ký và tự động discover topic mới thông qua KafkaConsumer engine
- [x] Viết Unit Test cho `sftp_adapter_test.go` và verify test suite pass 100%

## Phase 3: Cấu hình Metadata Database
- [ ] Tạo migration script/SQL query để đăng ký registry `reconcile_final` trong `source_object_registry`
- [ ] Cấu hình binding mapping sang target table `shadow_reconcile_final` trong `shadow_binding`
- [ ] Khai báo mapping_rules cho các cột trong bảng

## Phase 4: Kiểm thử E2E
- [ ] Deploy thử nghiệm cấu hình connector trên môi trường Dev/Staging
- [ ] Đẩy file `reconcile_final_*.csv` lên SFTP và verify dữ liệu được ghi nhận vào Shadow Table trong Postgres thành công.

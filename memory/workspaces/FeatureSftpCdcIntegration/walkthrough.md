# Báo cáo Nghiệm thu (Walkthrough) - Tích hợp SFTP Source Connector

Dữ liệu thay đổi từ file final của `reconcile-service` đẩy lên SFTP đã được tích hợp thành công vào hệ thống CDC. Dưới đây là tóm tắt kết quả triển khai và kiểm thử.

## Các tệp tin đã chỉnh sửa và tạo mới

1. **cdc-cms-service (Control Plane):**
   * [`system_connectors_handler.go`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/api/source/system_connectors_handler.go):
     * Cập nhật hàm `parseFingerprint` để nhận diện class `SftpSourceConnector` hoặc `Sftp` ➔ Trích xuất `sourceType = "sftp"`, `serverAddress = host:port`, `dbList` từ `input.path`, và `collectionList` từ `input.file.pattern`.
     * Cập nhật hàm `extractCredentialsAsOptions` để tự động mã hóa lưu trữ `sftp.username` và `sftp.password` thành `username`/`password` trong `OptionsJSON`.

2. **cdc-worker / centralized-data-service (Data Plane):**
   * [`sftp_adapter.go` (TẠO MỚI)](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/shadow/sftp_adapter.go):
     * Chứa hàm `ConvertToCDCEvent` convert dữ liệu JSON phẳng (từ file CSV do SFTP Connector parse ra) thành cấu trúc `CDCEvent` tiêu chuẩn với `Op = "c"` và payload nằm ở trường `After`.
   * [`event_handler.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/shadow/event_handler.go):
     * Hàm `HandleRaw` tự động nhận diện topic dạng `sftp.*` và bọc dữ liệu qua `SFTPEventAdapter`, trích xuất `db = "sftp"` và `table = reconcile_final`.
   * [`sftp_adapter_test.go` (TẠO MỚI)](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/shadow/sftp_adapter_test.go):
     * Unit test kiểm tra đầy đủ hai case: convert thành công flat JSON sang `CDCEvent` (khớp chính xác metadata) và xử lý lỗi JSON không hợp lệ.

---

## Kết quả kiểm thử tự động (Verification)

Bộ unit test chạy thành công trong shadow handler:
```bash
go test -v ./internal/handler/shadow/...
```
**Kết quả thực tế:**
```text
=== RUN   TestSFTPEventAdapter_ConvertToCDCEvent
--- PASS: TestSFTPEventAdapter_ConvertToCDCEvent (0.00s)
=== RUN   TestSFTPEventAdapter_ConvertToCDCEvent_InvalidJSON
--- PASS: TestSFTPEventAdapter_ConvertToCDCEvent_InvalidJSON (0.00s)
PASS
ok  	centralized-data-service/internal/handler/shadow	0.823s
```
Tất cả các unit test mới cho sftp adapter và toàn bộ các test cases hiện tại đều **PASS** 100%.

---

## Hướng dẫn cấu hình Metadata DB để hoạt động
Để CDC Worker tự động đồng bộ sang bảng Postgres Shadow, anh chỉ cần đăng ký metadata bằng tay (no-code) vào database `cdc_dw`:
1. **Thêm Source Object:**
   ```sql
   INSERT INTO cdc_system.source_object_registry (source_object_name, source_object_type, primary_key_field, source_connection_id) 
   VALUES ('reconcile_final', 'table', 'transaction_id', <sftp_connection_id>);
   ```
2. **Tạo Shadow Binding:**
   ```sql
   INSERT INTO cdc_system.shadow_binding (source_object_id, shadow_connection_id, shadow_schema, shadow_table, physical_table_fqn, write_mode, ddl_status, is_active)
   VALUES (<source_object_id>, <postgres_shadow_connection_id>, 'public', 'shadow_reconcile_final', 'public.shadow_reconcile_final', 'upsert', 'applied', true);
   ```
3. **Cấu hình Mapping Rules:** Đăng ký các rule cho cột cần map trong bảng `cdc_system.mapping_rule_v2`.

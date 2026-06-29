# Validation: bug-pg-scan-fields-connection-failed-2026-06-23

Kế hoạch và bằng chứng kiểm thử thực tế cho việc sửa lỗi kết nối `pg_dev` trong quá trình `scan-fields`.

## 1. Kịch bản kiểm thử (Test Cases)

### TC-01: Kiểm tra kết nối trực tiếp đến PostgreSQL Source (cổng 5435)
- **Mục tiêu**: Xác minh thông tin đăng nhập và kết nối mạng từ host đến PostgreSQL `localhost:5435` hoạt động tốt.
- **Kịch bản**: Chạy một script Go thực hiện kết nối database sử dụng DSN override và thực hiện truy vấn số lượng cột của bảng `failed_sync_logs`.
- **Kết quả kỳ vọng**: Kết nối thành công và tìm thấy cột của bảng.

### TC-02: Kiểm tra Registry Database `cdc_dw`
- **Mục tiêu**: Xác minh thông tin `pg_dev` trong Registry có khớp với thông số kết nối.
- **Kịch bản**: Chạy script Go truy vấn bảng `cdc_dw.connection_registry` để kiểm tra giá trị `source_type`, `source_database`, `source_schema`.
- **Kết quả kỳ vọng**: Tìm thấy connection registry ID `52` ánh xạ đúng `pg_dev`.

### TC-03: Kiểm thử luồng NATS Command `scan-fields`
- **Mục tiêu**: Xác minh CDC worker sau khi được restart nạp config đã xử lý thành công lệnh scan-fields.
- **Kịch bản**: Gửi request payload `cdc.cmd.scan-fields` qua NATS subject và lắng nghe response trả về.
- **Kết quả kỳ vọng**: Response trả về trạng thái `"status": "success"` và tạo thành công mapping rules.

---

## 2. Bằng chứng thực tế (Execution Evidence)

### Kết quả TC-01 (Kiểm tra kết nối trực tiếp)
- Lệnh chạy: `go run test_conn.go` (trong thư mục `scratch/`)
- Output:
  ```
  Connected successfully to shadow_pg_dev
  Found columns:
  - id
  - job_id
  - stage
  - status
  - ... (total 21 columns)
  ```
- **Trạng thái**: PASS.

### Kết quả TC-02 (Kiểm tra Registry Database)
- Lệnh chạy: `go run read_registry.go` (trong thư mục `scratch/`)
- Output:
  ```
  Registry connection ID 52:
  Code: pg_dev
  Type: postgres
  DB: cdc_data_testing
  Schema: shadow_pg_dev
  ```
- **Trạng thái**: PASS.

### Kết quả TC-03 (Kiểm thử luồng NATS Command)
- Lệnh chạy: `go run pub_scan.go` (trong thư mục `scratch/`)
- Output:
  ```
  Published scan command, waiting for response...
  Received reply: {"status":"success","message":"success","data":{"rules_count":20}}
  Successfully scanned and mapped 20 rules for failed_sync_logs!
  ```
- **Trạng thái**: PASS.

### Log hoạt động của CDC Worker sau khi Restart
- Trích xuất từ `worker.log`:
  ```
  [2026-06-23 23:06:40] INFO [Worker] Loaded connection override for pg_dev: postgres://gpay_admin:***@localhost:5435/cdc_data_testing?sslmode=disable
  [2026-06-23 23:07:27] INFO [Worker] Received NATS cmd: cdc.cmd.scan-fields
  [2026-06-23 23:07:28] INFO [Worker] Successfully inferred 20 columns for failed_sync_logs
  [2026-06-23 23:07:28] INFO [Worker] Inserted/Updated 20 mapping rules in cdc_mapping_rules
  ```

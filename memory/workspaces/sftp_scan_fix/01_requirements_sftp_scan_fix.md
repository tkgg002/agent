# 01_requirements_sftp_scan_fix.md

## Yêu cầu & Phạm vi (Requirements & Scope)

### 1. Vấn đề (Problem Statement)
Khi người dùng bấm **Quét field** (Scan fields) cho SFTP Source Connector trên CMS UI, hệ thống trả về lỗi:
`Quét field: SQL source returned 0 columns or connection failed`

### 2. Nguyên nhân gốc rễ (Root Cause)
- SFTP Connector là loại nguồn file-based (CSV via Kafka Connect `kafka-connect-fs`). Nguồn này không phải là SQL Database (không có SQL DSN).
- Khi Connector mới được tạo, chưa có dữ liệu mẫu CSV được nạp vào SFTP input directory -> Kafka Connect chưa ingest record nào -> Shadow Table (`cdc_shadow.sftp_reconcile_final`) chưa có dữ liệu `_raw_data`.
- Hàm `ScanFieldsDebezium` trong `discover_handler.go` khi kiểm tra Shadow Table không thấy `_raw_data` row nào, đã nhảy vào nhánh `isSQLSource` / `scanFieldsSQLSource` cố gắng truy vấn SQL schema -> Báo sai bản chất lỗi thành "SQL source returned 0 columns or connection failed".

### 3. Yêu cầu xử lý (Definition of Done)
1. **Xử lý Code Core (`centralized-data-service`)**:
   - Nhận diện chính xác `sourceType == "sftp"` / `sftp` engine trong `discover_handler.go` & `discover_handler_sql.go`.
   - Nếu Shadow Table chưa có dữ liệu `_raw_data`, ném lỗi thông báo chuẩn xác bản chất:
     `"Nguồn SFTP chưa có dữ liệu trong shadow DB. Vui lòng thả file CSV mẫu vào thư mục SFTP để hệ thống tự động đọc schema."`
   - Bổ sung `isStreamOrFileSource(sourceType)` để phân biệt rõ ràng giữa DB SQL, MongoDB và File/Stream source.
2. **Tạo dữ liệu mẫu & Thử nghiệm Vận hành**:
   - Tạo sẵn 1 file CSV mẫu `reconcile_final_sample.csv` trong thư mục Docker local `./docker/data/reconcile_final/` để `kafka-connect-fs` đọc & đẩy dữ liệu mẫu.
3. **Verification**:
   - Chạy test unit `go test ./internal/handler/source/...` pass.
   - Thử nghiệm quét field từ UI / API trả về danh sách các cột trích xuất thành công từ CSV.

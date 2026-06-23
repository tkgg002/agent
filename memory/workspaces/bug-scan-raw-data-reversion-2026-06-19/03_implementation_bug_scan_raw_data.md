# Implementation Design - Khôi phục Logic Scan Raw Data & Periodic Scan

## 1. Thiết kế giải pháp

### 1.1. HandleScanRawData
- **Đầu vào**: BindingID hoặc SourceObjectID thông qua tin nhắn NATS hoặc gọi trực tiếp.
- **Quy trình xử lý**:
  1. Lấy thông tin table config và shadow binding tương ứng.
  2. Query database PostgreSQL để phân tích cột `_raw_data` của bảng shadow bằng SQL:
     ```sql
     SELECT DISTINCT jsonb_object_keys(t._raw_data) as key, jsonb_typeof(t._raw_data->jsonb_object_keys(t._raw_data)) as type
     FROM (SELECT _raw_data FROM %s LIMIT 100) t
     ```
  3. Query toàn bộ mapping rules V2 hiện có trong `mapping_rule_v2` cho binding hoặc object tương ứng.
  4. Duyệt qua các keys quét được, nếu key chưa tồn tại trong mapping rules hiện có:
     - Tạo đối tượng `mastermodel.MappingRuleV2` mới.
     - Thiết lập các giá trị mặc định: `Status: "pending"`, `IsActive: false`, `SourceFormat: "raw"`, `TargetFormat: "native"`.
     - Lưu rule mới vào database.
  5. Đóng gói kết quả và publish sang NATS.

### 1.2. HandlePeriodicScan
- **Quy trình xử lý**:
  1. Gọi `h.metadataRegistry.ListTableConfigs()` để lấy danh sách các table configs.
  2. Lọc ra các bảng đang hoạt động (`IsActive = true`).
  3. Giả lập tin nhắn NATS `ScanRawData` cho từng bảng.
  4. Trigger xử lý song song hoặc tuần tự qua hàm `HandleScanRawData`.

## 2. Các file ảnh hưởng
- [scan_handler.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/recon/scan_handler.go)

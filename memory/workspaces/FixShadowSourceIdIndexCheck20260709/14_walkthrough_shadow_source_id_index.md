# Walkthrough kiểm chứng - Khắc phục logic tự sửa đổi index trong Transmuter & Bổ sung đề xuất trên UI

Tài liệu này hướng dẫn cách chạy kiểm thử và xác minh các thay đổi liên quan đến việc chuyển đổi từ cơ chế tự động sửa đổi index (self-healing DDL) sang cơ chế cảnh báo và đề xuất index trên UI.

## 1. Xác minh Unit Test

### 1.1 Kiểm tra logic Transmuter
Chạy các bài kiểm thử liên quan đến Transmuter (đảm bảo cơ chế cache check hoạt động và không chạy DDL khi thiếu/invalid index):
```bash
go test -v ./internal/service/master -run "TestTransmuter_EnsureShadowSourceIDIndex"
```
Kết quả kỳ vọng:
- `TestTransmuter_EnsureShadowSourceIDIndex_Missing`: PASS (Ghi Warn log, không gọi sqlmock Exec CREATE INDEX).
- `TestTransmuter_EnsureShadowSourceIDIndex_Invalid`: PASS (Ghi Warn log, không gọi sqlmock Exec DROP/CREATE INDEX).
- `TestTransmuter_EnsureShadowSourceIDIndex_Valid`: PASS (Bỏ qua, cập nhật cache).

### 1.2 Kiểm tra logic đề xuất index (Index Recommendations)
Chạy unit test mới kiểm chứng logic khuyến nghị tạo index thiếu:
```bash
go test -v ./internal/service/governance -run "TestIndexManager_GetRecommendations"
```
Kết quả kỳ vọng:
- `TestIndexManager_GetRecommendations`: PASS (Kiểm chứng thành công cả 4 case: thiếu cả 2, chỉ thiếu _deleted, chỉ thiếu _source_id, và đã đầy đủ).

### 1.3 Kiểm tra toàn bộ Test Suite ảnh hưởng
```bash
go test ./internal/service/master/...
go test ./internal/service/governance/...
go test ./internal/handler/...
```
Kết quả kỳ vọng: Tất cả các package đều PASS.

## 2. Xác minh tích hợp qua NATS (HandleIntrospectIndexes)
Khi CMS gọi lệnh `introspect-indexes` thông qua NATS, kết quả phản hồi `cdc.result.introspect-indexes` sẽ bao gồm thêm trường `recommendations`.

Ví dụ payload phản hồi (JSON):
```json
{
  "command": "introspect-indexes",
  "status": "success",
  "indexes": [
    {
      "index_name": "idx_payment_bills_pkey",
      "index_def": "CREATE UNIQUE INDEX ...",
      "index_size": "16 kB",
      "scan_count": 0,
      "is_valid": true
    }
  ],
  "recommendations": [
    {
      "index_name": "idx_payment_bills_source_id",
      "columns": ["_source_id"],
      "is_unique": false,
      "is_partial": false,
      "description": "Index cốt lõi trên cột _source_id bắt buộc phải có để transmuter hoạt động đồng bộ hiệu năng cao."
    },
    {
      "index_name": "idx_payment_bills_deleted_partial",
      "columns": ["_deleted"],
      "is_unique": false,
      "is_partial": true,
      "where_clause": "_deleted = true",
      "description": "Tối ưu CountDeletedRows: Tạo partial index trên cột _deleted để tối ưu hóa truy vấn đối soát dòng đã xóa cho Recon."
    }
  ]
}
```
Người dùng có thể dựa vào danh sách `recommendations` này để kích hoạt luồng `HandleCreateIndex` từ CMS UI.

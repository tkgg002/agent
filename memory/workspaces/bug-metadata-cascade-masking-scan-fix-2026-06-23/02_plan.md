# Plan: Metadata Mapping Validation & Schema Drift Warning

## Kế hoạch thực hiện

### Phase 1: Research (Đã hoàn thành)
- Nghiên cứu lỗi degraded transmuter do casting và các giải pháp thay thế.
- Phân tích luồng cache, NATS reload và sự mất đồng bộ DDL ở shadow table khi đổi cấu hình.

### Phase 2: Implementation Plan & Approval (Hiện tại)
- Thiết kế giải pháp sạch sẽ ở Control Plane (CMS): validator, drift warning, và tự động standardize DDL.
- Trình implementation plan làm artifact và chờ user duyệt.

### Phase 3: Sửa đổi code (Muscle thực hiện sau khi được duyệt)
- **cdc-cms-service**:
  - Triển khai validator kiểu dữ liệu cứng khi cấu hình mã hoá.
  - Thêm API warning cảnh báo cột master đã tồn tại với kiểu cũ (schema drift).
  - Tự động phát command `cdc.cmd.standardize` khi đổi `DataType`/`IsSensitiveField` để alter type cột shadow tương ứng.
  - Cập nhật `mapping_rule_master.pending_type_change` khi đổi `DataType` của rule đã approved.
- **centralized-data-service**:
  - Revert toàn bộ logic fallback tự chế trong transmuter.
  - Dọn dẹp files test cũ không cần thiết.

### Phase 4: Xác minh
- Viết unit tests kiểm tra validator và cảnh báo type drift.
- Chạy toàn bộ test suite của cả hai dự án đảm bảo pass 100%.

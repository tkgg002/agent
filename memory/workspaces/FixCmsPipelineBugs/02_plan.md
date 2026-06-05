# Kế hoạch Thực thi (Implementation Plan)

## Mục tiêu
Fix 4 bugs cốt lõi của CMS Pipeline theo đúng yêu cầu và quy tắc thiết kế hệ thống, không dùng "cheat" logic.

## Các bước chi tiết

### 1. Fix Mapping Rule Scope Bug (Đang làm dở)
- [x] Thêm `shadow_binding_id` và `source_data_type` vào DB `mapping_rule_v2`.
- [x] Update GORM repository, Query Handlers (`Create`, `List`) để filter theo `shadow_binding_id` nếu có.
- [ ] Update frontend `cdc-cms-web` để truyền `binding_id` vào request `/api/mapping-rules` khi click vào cấu hình 1 table. (Đang lấy ở URL: `shadow/:id/mappings`).

### 2. Infer `source_data_type` trong worker (Bug 2)
- Thay vì chỉ trả về `unmappedFields`, ta sẽ sửa lệnh `HandleScanRawData` trong worker để dùng hàm PostgreSQL `jsonb_typeof()` để đọc ra Type của từng field trong `_raw_data`.
- Chỉnh sửa FE khi gọi `Scan Unmapped Fields` sẽ tự động insert các field này với type tương ứng vào `mapping_rule_v2` (hoặc worker tự Insert nếu là Periodic Scan).

### 3. Fix Frontend UX (Bug 3)
- Gỡ bỏ hoàn toàn nút "Preview" và "Backfill" ở giao diện Mapping Rule.
- Bổ sung cột "Source Data Type" trên Table hiển thị.

### 4. Điều tra & Fix Snapshot V2 Bug (Bug 4)
- **Vấn đề Root Cause:** `is_active` của `shadow_binding` là `true` trong DB nhưng worker vẫn báo "shadow_binding_id=4 not in active registry routes". 
- **Hành động:** 
  1. `grep_search` để tìm đoạn code in ra log `not in active registry routes` trong `snapshot_runner_handler.go`.
  2. Phân tích hàm load "active registry routes" từ DB lên cache (có thể do lỗi cache, lỗi SQL JOIN, hoặc worker chưa được reload configuration).
  3. Sửa đúng nguyên nhân cốt lõi (Root Cause) để tuyến đường hợp lệ được nhận diện chuẩn xác. Tuyệt đối không hardcode hay bypass bằng synthetic route.

## Verification
- Chạy make run cho cả CMS và worker để đảm bảo build thành công.
- Review lại toàn bộ log thay đổi xem có phá vỡ tính toàn vẹn của hệ thống không.

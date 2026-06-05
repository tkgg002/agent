# Yêu cầu kỹ thuật chi tiết (01_requirements.md)

> **Workspace**: bug-mapping-rules-and-snapshot-v2-2026-05-29

## Yêu cầu 1: Sửa lỗi hiển thị Mapping Rules (Bug 1)
- **Mục tiêu**: Khi truy cập trang mapping của một shadow binding cụ thể (ví dụ: `shadow_binding_id = 4`), hệ thống chỉ hiển thị các mapping rules được liên kết với `shadow_binding_id` này. Không được tự động lấy các rules của binding khác (như `shadow_binding_id = 1` mặc dù chung `source_object_id`).
- **Phía Backend**:
  - API lấy danh sách mapping rules phải filter theo cả `source_object_id` và `shadow_binding_id`.
  - Cần kiểm tra xem DB schema `mapping_rule_v2` hiện tại có cột `shadow_binding_id` hay chưa. Nếu đã có, API cần lọc theo cột này.
- **Phía Frontend**:
  - Trang `/shadow/:bindingId/mappings` cần truyền `shadow_binding_id` lên API request để lọc.

## Yêu cầu 2: Thêm cột Source Data Type & Sửa Status logic (Bug 2)
- **Mục tiêu**: Bổ sung kiểu dữ liệu nguồn (Source Data Type) của các fields và chuẩn hóa hiển thị Status / In Shadow.
- **Database Schema**:
  - Thêm cột `source_data_type` (kiểu `VARCHAR`) vào bảng `cdc_system.mapping_rule_v2`.
  - Cần viết file migration SQL để bổ sung cột này một cách an toàn.
- **Logic Scan Raw**:
  - Khi scan raw data từ source (MongoDB/Postgres), hệ thống phải tự động phân tích và xác định kiểu dữ liệu của từng field (ví dụ: `string`, `int`, `double`, `boolean`, `date`, `object`, `array`).
  - Ghi nhận kiểu dữ liệu quét được vào cột `source_data_type` khi lưu trữ mapping rules/fields được scan.
- **Phía Frontend (UI)**:
  - Hiển thị cột "Data Type source" trên table.
  - Phân biệt rõ ràng cột "Status" (trạng thái duyệt của rule: Approved, Pending, Rejected) và "In Shadow" (trạng thái audit/khớp với shadow schema).

## Yêu cầu 3: Ẩn Action Preview và Backfill (Bug 3)
- **Mục tiêu**: Ẩn các nút "Preview" và "Backfill" trong cột hành động (Action) ở trang Mapping Fields.
- **Giải pháp**:
  - Sửa file React component trên Frontend (`cdc-cms-web`).
  - Chỉ ẩn hiển thị (ví dụ: comment code JSX hoặc dùng flag hiển thị, hoặc điều kiện ẩn), tuyệt đối KHÔNG xoá bỏ các hàm xử lý hay API call bên dưới để giữ nguyên logic.

## Yêu cầu 4: Sửa lỗi Snapshot V2 Registry Lookup Fail (Bug 4)
- **Mục tiêu**: Sửa lỗi `shadow_binding_id=4 not in active registry routes` khi trigger snapshot v2.
- **Root Cause & Fix**:
  - Phân tích và khắc phục lỗi ghi đè dữ liệu trong map cache `routeBySourceID` của `MetadataRegistryService.ReloadAll`.
  - Đảm bảo khi load registry, mọi shadow binding đang active đều được đăng ký route chính xác và không bị ghi đè bởi shadow binding khác có chung `source_object_id`.
  - Sửa logic lookup trong `snapshot_runner_handler.go` để tìm kiếm route theo cả `SourceObjectID` và `ShadowBindingID` một cách chính xác.

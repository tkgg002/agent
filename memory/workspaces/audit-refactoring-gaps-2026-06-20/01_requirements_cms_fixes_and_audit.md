# Yêu cầu Chi tiết: Phase cms_fixes_and_audit

## 1. Yêu cầu Sửa lỗi cdc-cms-service
### 1.1 Sửa lỗi Sync Shadow
- **Hiện tượng**: Khi gọi API đồng bộ shadow rules, xảy ra lỗi SQL: `ERROR: column v2.is_deleted does not exist`.
- **Nguyên nhân**: Bảng `cdc_system.mapping_rule_v2` (hoặc alias `v2` được mapping) không chứa cột `is_deleted`. Việc query thêm điều kiện `AND v2.is_deleted = false` dẫn đến lỗi SQL syntax/schema mismatch.
- **Yêu cầu**: Loại bỏ hoàn toàn điều kiện `AND v2.is_deleted = false` ra khỏi các câu query SQL đồng bộ trong hàm `SyncRulesFromShadow`.

### 1.2 Sửa lỗi Drop Column
- **Hiện tượng**: Khi thực hiện Drop một rule đã bị `rejected`, hệ thống trả về lỗi `"cột đang được rule approved+active khác dùng — không drop"`.
- **Nguyên nhân**:
  1. Hàm `CheckColumnConflict` truy vấn đếm số lượng rules sử dụng chung cột đích nhưng lại không lọc theo trạng thái `approved` và `active = true`. Do đó, nó đếm cả chính rule hiện tại (hoặc các rule nháp/rejected khác), tạo ra conflict giả.
  2. Lệnh gọi `CheckColumnConflict` trong `DropColumnHandler` truyền tham số `excludeID` bằng `0`, thay vì truyền `rule.ID` để loại trừ chính nó ra khỏi quá trình đếm conflict.
- **Yêu cầu**:
  1. Cập nhật query trong hàm `CheckColumnConflict` để chỉ kiểm tra conflict với các rules đang ở trạng thái `status = 'approved'` và `is_active = true`.
  2. Cập nhật tham số truyền vào hàm `CheckColumnConflict` trong `drop_column.go`, truyền chính xác `rule.ID` thay vì `0`.

## 2. Yêu cầu So khớp Logic Reconciliation Engine (centralized-data-service)
- **Hiện tượng**: centralize-data-service đã refactor cấu trúc thư mục, chuyển `recon_core.go` cũ sang bộ ba file: `recon_engine.go`, `recon_tier_a.go`, `recon_tier_b.go`.
- **Yêu cầu**: Rà soát, so khớp từng dòng để đảm bảo 100% logic nghiệp vụ cũ (watermark, adaptive freeze, chunking, block logic khi vượt blast radius) không bị mất hoặc sai lệch.

# 00_context — Bug Auto-Approve Mapping & Frontend Toggle Bug

## 1. Bối cảnh (Background)
- **Hệ thống**: Data Hub (CDC Platform) gồm `cdc-cms-service` (Go Fiber), `cdc-cms-web` (React/Vite/Antd), `centralized-data-service` (Go worker).
- **Hiện tượng lỗi**:
  1. Khi user thao tác kích hoạt (`is_active = true`) trên bảng `payment_bills_1`, toàn bộ mapping rules của bảng `payment_bills` cũ (cùng source object) bị tự động chuyển sang trạng thái `approved` dù user không hề click approve trên UI của `payment_bills`.
  2. Trên giao diện `MappingFieldsPage` (`cdc-cms-web`), switch toggle Active cho từng mapping rule luôn gửi API PATCH `{ status: 'approved' }` bất kể switch đang bật hay tắt (lỗi copy-paste ternary).

## 2. Các thành phần bị ảnh hưởng (Components)
- **Backend**: `cdc-cms-service/internal/infra/persistence/source/source_repo_gorm.go` (hàm `UpdateRegistry`).
- **Frontend**: `cdc-cms-web/src/pages/MappingFieldsPage.tsx` (hàm `handleToggleActive`).
- **Cơ sở dữ liệu**:
  - `cdc_system.cdc_table_registry` (V1 table registry)
  - `cdc_system.source_object_registry` (V2 canonical source object)
  - `cdc_system.shadow_binding` (V2 shadow routing binding)
  - `cdc_system.mapping_rule_v2` (V2 field mappings)

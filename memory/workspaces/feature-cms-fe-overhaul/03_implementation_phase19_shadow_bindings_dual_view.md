# Implementation — Phase 19 Shadow Bindings Dual View

## Backend

- Mở rộng `source_objects_handler.go`
- Thêm:
  - `ShadowBindingRow`
  - `shadowBindingsListResponse`
  - `ListShadowBindings()`
- Route mới:
  - `GET /api/v1/shadow-bindings`
- Query hiện tại:
  - nguồn chính: `cdc_system.shadow_binding`
  - join `cdc_system.source_object_registry`
  - join `cdc_table_registry` để carry `registry_id` bridge nếu có
  - join latest `cdc_reconciliation_report` để lấy drift / thời điểm check gần nhất
- Swagger:
  - thêm annotation cho `GET /api/v1/shadow-bindings`

## Frontend

- Thêm type `ShadowBindingRow`
- Refactor `TableRegistry.tsx`
  - thêm fetch `/api/v1/shadow-bindings`
  - thêm tab `Shadow Bindings`
  - render practical columns:
    - binding code
    - source table
    - shadow schema/table
    - physical FQN
    - write mode
    - ddl status
    - recon drift
    - active

## Lý do chọn cách này

- Không thêm page mới
- Giữ menu gọn
- Cho operator thấy trực tiếp binding layer, là phần trước đây chỉ tồn tại ngầm trong backend

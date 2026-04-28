# Implementation — Phase 18 Source Objects Read Path V2

## Backend

- Thêm file `cdc-cms-service/internal/api/source_objects_handler.go`
- Thêm endpoint:
  - `GET /api/v1/source-objects`
- Read model hiện tại:
  - nguồn chính: `cdc_system.source_object_registry`
  - join active/latest `cdc_system.shadow_binding`
  - join latest `cdc_reconciliation_report`
  - join `cdc_table_registry` chỉ để lấy transitional bridge metadata:
    - `registry_id`
    - `priority`
    - `sync_interval`
    - `sync_status`
    - `recon_drift`
- Route/wiring:
  - sửa `internal/router/router.go`
  - sửa `internal/server/server.go`
- Swagger:
  - thêm annotation cho `GET /api/v1/source-objects`

## Frontend

- Thêm type `SourceObjectRow` ở `cdc-cms-web/src/types/index.ts`
- Refactor `TableRegistry.tsx`
  - list fetch từ `/api/v1/source-objects`
  - render shadow FQN từ metadata V2
  - dùng `registry_id` cho các action legacy
  - disable action legacy nếu row chưa có bridge
  - row click vào mappings chỉ hoạt động khi có `registry_id`

## Lý do chọn cách này

- Không tiếp tục trói list view vào `/api/registry`
- Không phá các operator actions vẫn còn sống thật
- Không tạo cảm giác “chạy được” cho row chỉ có metadata V2 nhưng chưa có legacy bridge tương ứng

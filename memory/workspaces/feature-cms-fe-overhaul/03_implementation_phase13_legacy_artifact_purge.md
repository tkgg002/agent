# Implementation — Phase 13 Legacy Artifact Purge

## Audit trước khi xóa

- `CDCInternalRegistry.tsx` chỉ còn self-reference và gọi các API đã bị remove:
  - `/api/v1/tables`
  - `PATCH /api/v1/tables/:name`
- `QueueMonitoring.tsx` không còn route/menu runtime.
- `cdc_internal_registry_handler.go` không còn được instantiate trong server wiring.

## Purge đã thực hiện

- Xóa:
  - `/Users/trainguyen/Documents/work/cdc-system/cdc-cms-web/src/pages/QueueMonitoring.tsx`
  - `/Users/trainguyen/Documents/work/cdc-system/cdc-cms-web/src/pages/CDCInternalRegistry.tsx`
  - `/Users/trainguyen/Documents/work/cdc-system/cdc-cms-service/internal/api/cdc_internal_registry_handler.go`

## Kết quả

- FE không còn page artifact liên quan `cdc_internal` / `queue monitoring` legacy.
- Backend không còn giữ handler API cho `cdc_internal.table_registry`.
- Surface runtime gọn hơn và khớp hơn với V2 operator-flow hiện tại.

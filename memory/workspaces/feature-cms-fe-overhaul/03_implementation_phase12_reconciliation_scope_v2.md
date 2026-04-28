# Implementation — Phase 12 Reconciliation Scope V2

## Backend

- Refactor `/Users/trainguyen/Documents/work/cdc-system/cdc-cms-service/internal/api/reconciliation_handler.go`
  - thêm `reconScopeRequest`
  - thêm helper resolve target table từ `cdc_system.shadow_binding` + `cdc_system.source_object_registry`
  - enrich `LatestReport` bằng:
    - `source_table`
    - `shadow_schema`
    - `shadow_table`
    - `scope_ambiguous`
  - enrich `ListFailedLogs` bằng source/shadow metadata V2
  - cho `TriggerCheckAll` nhận body scope và dispatch single-table nếu body resolve ra target cụ thể
  - cho `TriggerCheck` và `TriggerHeal` hỗ trợ resolve scope từ body thay vì chỉ phụ thuộc path param
- Update `/Users/trainguyen/Documents/work/cdc-system/cdc-cms-service/internal/router/router.go`
  - thêm route generic `POST /api/reconciliation/heal`
  - giữ route legacy `POST /api/reconciliation/heal/:table`

## Frontend

- Refactor `/Users/trainguyen/Documents/work/cdc-system/cdc-cms-web/src/hooks/useReconStatus.ts`
  - mở rộng `ReconRow` và `FailedLog` với source/shadow metadata
  - đổi check/heal mutation sang gửi body scope V2
- Refactor `/Users/trainguyen/Documents/work/cdc-system/cdc-cms-web/src/pages/DataIntegrity.tsx`
  - render source/shadow scope đầy đủ ở overview và failed logs
  - truyền `source_database`, `source_table`, `shadow_schema`, `shadow_table` vào check/heal actions
  - hiển thị warning `Ambiguous` nếu metadata shadow bị trùng

## Swagger / API Notes

- Reconciliation APIs đã được cập nhật comment handler cùng phase để phản ánh contract mới theo source/shadow scope.
- Không mở thêm endpoint check mới; `POST /api/reconciliation/check` được reuse cho cả:
  - check-all
  - check một scope cụ thể khi body có đủ metadata

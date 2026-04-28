# Phase 32 Implementation — Direct Standardize

## Backend
- Thêm `POST /api/v1/source-objects/{id}/standardize` trong `source_object_actions_handler.go`
- Route direct resolve active `shadow_binding` từ `source_object_id`
- Publish worker-compatible payload:
  - `source_object_id`
  - `target_table`
- Update `router.go` để mount route mới
- Update swagger annotations cho endpoint mới

## Frontend
- `TableRegistry.tsx`
  - `handleCreateDefaultFields()` giờ chọn endpoint:
    - bridge nếu có `registry_id`
    - direct V2 nếu row chỉ có `source_object_id`

## Deliberately not done
- Không direct-V2 hóa `create-default-columns` trong phase này

# Phase 31 Implementation — Direct Scan Fields & Transform Status

## Backend
- Thêm trong `source_object_actions_handler.go`:
  - `POST /api/v1/source-objects/{id}/scan-fields`
  - `GET /api/v1/source-objects/{id}/transform-status`
- Cả hai route resolve active `shadow_binding` từ `source_object_id`.
- `scan-fields` direct route publish payload worker-compatible:
  - `source_object_id`
  - `target_table`
  - `source_table`
  - `sync_engine=debezium`
  - `source_type`
- `transform-status` direct route tính progress trực tiếp theo `target_table` đã resolve.

## Frontend
- `useRegistry.ts`:
  - `useScanFields` ưu tiên direct V2, fallback bridge.
- `TableRegistry.tsx`:
  - `TransformProgress` ưu tiên direct V2, fallback bridge.
  - `AsyncRowActions` cho `scan-fields` dùng `sourceObjectId` trước.
  - warning đổi từ “thiếu bridge” sang “thiếu scope” ở action scan.

## Swagger
- Đã cập nhật annotations trong `source_object_actions_handler.go` cho 2 endpoint mới.
- Generated swagger chưa regen vì local thiếu `swag`.

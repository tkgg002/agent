# Phase 30 Implementation — V2 Direct Re-detect

## Backend
- Thêm helper resolve scope theo `source_object_id` trong `source_object_actions_handler.go`.
- Thêm:
  - `GET /api/v1/source-objects/{id}/dispatch-status`
  - `POST /api/v1/source-objects/{id}/detect-timestamp-field`
- Direct route resolve `target_table` từ `cdc_system.shadow_binding` active rồi publish NATS payload theo contract worker hiện có.
- Cập nhật `router.go` để mount route mới.
- Cập nhật `reconciliation_handler.go` để `LatestReport` trả thêm `source_object_id`.

## Frontend
- `ReDetectButton`:
  - ưu tiên direct V2 route theo `source_object_id`
  - fallback sang bridge route theo `registry_id`
- `DataIntegrity`:
  - render nút Re-detect khi có `source_object_id` hoặc `registry_id`
- `useReconStatus.ts`:
  - enrich `ReconRow` với `source_object_id`

## Swagger
- Đã cập nhật annotations trong `source_object_actions_handler.go` cho 2 direct endpoints mới.
- Generated swagger chưa regen được trên máy hiện tại vì thiếu binary `swag`.

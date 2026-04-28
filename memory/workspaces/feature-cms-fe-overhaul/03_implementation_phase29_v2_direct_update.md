# Phase 29 Implementation - V2 Direct Update

## Backend

- `SourceObjectActionsHandler`
  - thêm `UpdateV2`
  - patch trực tiếp `cdc_system.source_object_registry`
  - sync `is_active` sang `cdc_system.shadow_binding`

## Frontend

- `TableRegistry.tsx`
  - `updateEntry(record, updates)`
  - nếu có `registry_id` -> dùng bridge patch
  - nếu không có bridge -> dùng `PATCH /api/v1/source-objects/:id`
  - `priority` bị disable cho row không có bridge

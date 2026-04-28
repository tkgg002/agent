# Phase 23 Implementation - Dashboard + ActivityManager V2 Reads

## Backend

- Thêm `SourceObjectStats` trong `internal/api/source_objects_handler.go`.
- Thêm handler `GetStats()` cho `GET /api/v1/source-objects/stats`.
- Query stats lấy từ:
  - `cdc_system.source_object_registry`
  - latest/active `cdc_system.shadow_binding`
  - bridge `cdc_table_registry` để carry priority / created fallback
- Cập nhật router mount route stats V2 mới.

## Frontend

- `Dashboard.tsx`
  - đổi từ `/api/registry/stats` sang `/api/v1/source-objects/stats`
  - đổi nhãn:
    - `Registered Tables` -> `Source Objects`
    - `Tables Created` -> `Shadow Ready`
- `ActivityManager.tsx`
  - đổi read source từ `/api/registry` sang `/api/v1/source-objects`
  - dùng `shadow_schema` nếu đã có từ V2 read model

## Chủ đích kiến trúc

- Đây là read-path cleanup, không phải mutation migration.
- `PATCH /api/registry/:id` và `POST /api/registry/batch` được giữ lại như compatibility shell có kiểm soát.

# Phase 25 Implementation - Registry Route Prune

## Audit kết quả

- FE/runtime nội bộ không còn caller nào tới:
  - `GET /api/registry`
  - `GET /api/registry/stats`
  - `PATCH /api/registry/:id`
  - `POST /api/registry`
  - `POST /api/registry/batch`
  - `GET /api/registry/:id/dispatch-status`
  - `GET /api/registry/:id/transform-status`
  - `POST /api/registry/:id/standardize`
  - `POST /api/registry/:id/scan-fields`
  - `POST /api/registry/:id/create-default-columns`
  - `POST /api/registry/:id/detect-timestamp-field`

## Thay đổi

- Thêm facade:
  - `POST /api/v1/source-objects/registry/:id/transform`
- Gỡ các route legacy `/api/registry...` đã có replacement khỏi router.

## Ghi chú

- `RegistryHandler` vẫn còn tồn tại làm lớp delegate cho facade mới.
- Phase này prune route surface, chưa xóa hẳn implementation.

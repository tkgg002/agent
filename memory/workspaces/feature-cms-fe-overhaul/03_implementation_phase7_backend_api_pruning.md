# Implementation — Phase 7 Backend API Pruning

## API Audit Summary

Đã kiểm tra call site FE và route mount backend:

- FE **không còn** dùng:
  - `GET /api/v1/tables`
  - `PATCH /api/v1/tables/:name`
  - `POST /api/registry/:id/bridge`
- Các route trên cũng không còn đúng mục tiêu Debezium-only.

## Backend Changes

### `internal/router/router.go`

- Bỏ `cdcInternalRegistryHandler` khỏi `SetupRoutes` signature.
- Bỏ mount:
  - `PATCH /api/v1/tables/:name`
  - `GET /api/v1/tables`
  - `POST /api/registry/:id/bridge`

### `internal/server/server.go`

- Bỏ khởi tạo `cdcInternalRegistryHandler`.
- Bỏ truyền handler này vào `router.SetupRoutes`.

### `internal/api/registry_handler.go`

- Dọn swagger/comment stale cho `refresh-catalog`:
  - bỏ `@Router /api/registry/{id}/refresh-catalog [post]`
- Update retirement messages:
  - `Bridge()` không còn trỏ user về `/api/v1/tables`
  - `Reconciliation()` không còn gợi ý `/api/v1/tables`

## Swagger Update Note

Phase này có đổi API surface nên đã cập nhật phần swagger/comment tương ứng:

- route `refresh-catalog` không còn xuất hiện như endpoint hợp lệ trong swagger annotation
- các message/comment retirement không còn trỏ về `/api/v1/tables`

# Implementation — Phase 22 Transform Status Facade

## Backend

- Mở rộng `source_object_actions_handler.go`
- Thêm:
  - `GET /api/v1/source-objects/registry/{id}/transform-status`
- Handler facade tiếp tục delegate sang `RegistryHandler.TransformStatus`

## Frontend

- `TableRegistry.tsx`
  - `TransformProgress` chuyển từ `/api/registry/:id/transform-status`
  - sang `/api/v1/source-objects/registry/:id/transform-status`
- `useAsyncDispatch.ts`
  - cập nhật doc comment ví dụ endpoint theo namespace mới

## Kết quả

- FE-facing status surface sạch hơn
- không đổi semantics backend
- không bọc giả những mutation compatibility shell chưa nên đổi

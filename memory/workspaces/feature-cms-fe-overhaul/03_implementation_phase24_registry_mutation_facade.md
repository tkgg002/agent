# Phase 24 Implementation - Registry Mutation Facade

## Backend

- Mở rộng `SourceObjectActionsHandler` để bọc:
  - `Register`
  - `UpdateBridge`
  - `BulkRegister`
- Mỗi route mới chỉ delegate sang `RegistryHandler` tương ứng.
- Giữ nguyên write-model/persistence legacy bên dưới để không phá operator-flow hiện có.

## Frontend

- `TableRegistry.tsx`
  - update row settings qua `PATCH /api/v1/source-objects/registry/:id`
  - register single qua `POST /api/v1/source-objects/register`
  - bulk import qua `POST /api/v1/source-objects/register-batch`

## Chủ đích kiến trúc

- Đây là cleanup FE-facing namespace.
- Chưa phải migration của backend write semantics sang V2 thật.

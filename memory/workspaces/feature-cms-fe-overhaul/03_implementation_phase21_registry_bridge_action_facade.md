# Implementation — Phase 21 Registry Bridge Action Facade

## Backend

- Thêm file `source_object_actions_handler.go`
- Các facade mới:
  - `POST /api/v1/source-objects/registry/{id}/create-default-columns`
  - `POST /api/v1/source-objects/registry/{id}/standardize`
  - `POST /api/v1/source-objects/registry/{id}/scan-fields`
  - `POST /api/v1/source-objects/registry/{id}/detect-timestamp-field`
  - `GET /api/v1/source-objects/registry/{id}/dispatch-status`
- Các facade này không đổi semantics backend; chỉ delegate sang `RegistryHandler`

## Frontend

- `useRegistry.ts`
  - chuyển `scan-fields` sang namespace mới
- `ReDetectButton.tsx`
  - chuyển endpoint + statusEndpoint sang namespace mới
- `TableRegistry.tsx`
  - chuyển `standardize`, `create-default-columns` sang namespace mới
- `MappingFieldsPage.tsx`
  - chuyển `create-default-columns` sang namespace mới

## Lý do chọn cách này

- FE không còn nói chuyện trực tiếp với `/api/registry` cho các action operator chính
- vẫn trung thực rằng action hiện còn cần bridge
- không phát minh semantics V2 giả khi worker path chưa đổi

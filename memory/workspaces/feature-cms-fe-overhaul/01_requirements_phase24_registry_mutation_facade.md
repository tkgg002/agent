# Phase 24 Requirements - Registry Mutation Facade

## Mục tiêu

- Dọn nốt dependency FE runtime vào mutation shell `/api/registry`.
- Không làm backend giả vờ đã có write-model V2 thật nếu bên dưới vẫn cần legacy registry bridge.
- Giữ operator-flow thực chiến cho register, bulk import, update settings của source object.

## Yêu cầu chức năng

1. FE `TableRegistry` không còn gọi trực tiếp:
   - `POST /api/registry`
   - `PATCH /api/registry/:id`
   - `POST /api/registry/batch`
2. Có namespace V2-facing tương ứng dưới `/api/v1/source-objects`.
3. Backend vẫn được phép delegate sang `RegistryHandler` hiện tại.
4. Swagger annotations phải được cập nhật cùng phase.

## Definition of Done

- FE call-sites đã chuyển sang namespace mới.
- Backend mount đủ 3 route facade mutation.
- `go test ./...` ở `cdc-cms-service` pass.
- `npm run build` ở `cdc-cms-web` pass.
- grep FE không còn runtime call `/api/registry`.

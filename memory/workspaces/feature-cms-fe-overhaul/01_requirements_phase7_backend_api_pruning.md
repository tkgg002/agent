# Requirements — Phase 7 Backend API Pruning

## Mục tiêu

- Dọn backend/CMS API surface theo Debezium-only target.
- Không giữ các route legacy không còn được FE/runtime chính dùng.
- Khi thay API/router/comment phải cập nhật swagger annotation tương ứng.

## Rule

Trước khi prune API phải audit:

1. FE còn gọi route đó hay không
2. route đó có còn nằm trong luồng yêu cầu hiện tại hay không
3. swagger/comment có đang quảng bá sai route không

## Scope phase này

- remove route surface:
  - `GET /api/v1/tables`
  - `PATCH /api/v1/tables/:name`
  - `POST /api/registry/:id/bridge`
- remove swagger stale entry:
  - `POST /api/registry/{id}/refresh-catalog`

## Definition of Done

1. Router không còn mount các route trên.
2. Server wiring compile sạch sau khi cắt handler dependency.
3. Swagger comment không còn quảng bá route stale.
4. `go test ./...` của `cdc-cms-service` pass.
5. `npm run build` của `cdc-cms-web` pass.

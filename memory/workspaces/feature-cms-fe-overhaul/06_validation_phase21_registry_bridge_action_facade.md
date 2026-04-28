# Validation — Phase 21 Registry Bridge Action Facade

## Pass

- `cd /Users/trainguyen/Documents/work/cdc-system/cdc-cms-service && go test ./...`
  - pass
- `cd /Users/trainguyen/Documents/work/cdc-system/cdc-cms-web && npm run build`
  - pass

## Swagger

- Source-level swagger annotations đã được cập nhật cho facade endpoints mới dưới `/api/v1/source-objects/registry/{id}/...`.
- Thử regenerate bằng:
  - `make swagger`
- Kết quả:
  - fail vì máy hiện tại thiếu binary `swag`
  - thông điệp: `make: swag: No such file or directory`

## Self-check

- Phase này không đổi worker semantics; chỉ đổi FE-facing namespace.
- Không thêm destructive capability mới; chỉ alias action có sẵn qua namespace rõ nghĩa hơn.

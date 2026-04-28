# Validation — Phase 20 Mapping Context Read Model

## Pass

- `cd /Users/trainguyen/Documents/work/cdc-system/cdc-cms-service && go test ./...`
  - pass
- `cd /Users/trainguyen/Documents/work/cdc-system/cdc-cms-web && npm run build`
  - pass

## Swagger

- Source-level swagger annotations đã được cập nhật cho `GET /api/v1/source-objects/registry/{registry_id}`.
- Thử regenerate bằng:
  - `make swagger`
- Kết quả:
  - fail vì máy hiện tại thiếu binary `swag`
  - thông điệp: `make: swag: No such file or directory`

## Self-check

- Page mappings không còn tải toàn bộ `/api/registry`.
- Action legacy `create-default-columns` vẫn giữ path cũ để không gãy operator-flow.
- Không thêm surface destructive mới.

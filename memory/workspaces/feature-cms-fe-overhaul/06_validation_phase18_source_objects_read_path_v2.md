# Validation — Phase 18 Source Objects Read Path V2

## Pass

- `cd /Users/trainguyen/Documents/work/cdc-system/cdc-cms-service && go test ./...`
  - pass
- `cd /Users/trainguyen/Documents/work/cdc-system/cdc-cms-web && npm run build`
  - pass

## Swagger

- Source-level swagger annotations đã được cập nhật cùng phase.
- Thử regenerate bằng:
  - `make swagger`
- Kết quả:
  - fail vì máy hiện tại thiếu binary `swag`
  - thông điệp: `make: swag: No such file or directory`

## Self-check

- Backend query mới dùng placeholder cho filter động (`source_db`, `is_active`), không ghép input trực tiếp vào SQL.
- FE phase này không thêm pattern nguy hiểm như `dangerouslySetInnerHTML`, `eval`, hay HTML injection path mới.

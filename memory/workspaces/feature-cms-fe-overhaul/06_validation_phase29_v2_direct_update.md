# Phase 29 Validation - V2 Direct Update

1. `gofmt`
2. `go test ./...`
3. `npm run build`
4. rà lại route mới + call-site FE

## Kết quả thực tế

- `gofmt`: pass
- `go test ./...` trong `cdc-cms-service`: pass
- `npm run build` trong `cdc-cms-web`: pass
- FE `TableRegistry` đã chọn endpoint theo bridge status:
  - bridged row -> `/api/v1/source-objects/registry/:id`
  - V2-only row -> `/api/v1/source-objects/:id`

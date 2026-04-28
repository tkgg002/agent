# Phase 28 Validation - V2 Status Visibility

1. `gofmt`
2. `go test ./...`
3. `npm run build`
4. rà lại text/copy mới trong `TableRegistry`

## Kết quả thực tế

- `gofmt`: pass
- `go test ./...` trong `cdc-cms-service`: pass
- `npm run build` trong `cdc-cms-web`: pass
- copy mới trong `TableRegistry` đã phản ánh:
  - `V2 Ready`
  - `Shadow Bound`
  - `Source Only`
  - `Bridge OK`
  - `No Bridge`

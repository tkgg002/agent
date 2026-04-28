# Phase 24 Validation - Registry Mutation Facade

## Kiểm thử dự kiến

1. `gofmt` cho file Go vừa sửa
2. `go test ./...` trong `cdc-cms-service`
3. `npm run build` trong `cdc-cms-web`
4. grep FE runtime call-sites `/api/registry`
5. thử `make swagger`

## Kỳ vọng

- FE runtime không còn gọi trực tiếp `/api/registry`
- annotations có đủ cho 3 mutation facade
- generated swagger docs có thể vẫn fail nếu thiếu `swag`

## Kết quả thực tế

- `gofmt` cho:
  - `internal/api/source_object_actions_handler.go`
  - `internal/router/router.go`
  pass
- `cd /Users/trainguyen/Documents/work/cdc-system/cdc-cms-service && go test ./...`
  - pass
- `cd /Users/trainguyen/Documents/work/cdc-system/cdc-cms-web && npm run build`
  - pass
- `rg -n '/api/registry' /Users/trainguyen/Documents/work/cdc-system/cdc-cms-web/src`
  - không còn match runtime call-site
- `cd /Users/trainguyen/Documents/work/cdc-system/cdc-cms-service && make swagger`
  - fail: `swag: No such file or directory`
  - kết luận: annotations đã cập nhật, generated docs chưa regen được trên máy hiện tại

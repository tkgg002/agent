# Phase 25 Validation - Registry Route Prune

## Kiểm thử dự kiến

1. `gofmt` cho các file Go vừa sửa
2. `go test ./...`
3. `npm run build`
4. grep router để xác nhận các route legacy đã bị gỡ
5. `make swagger`

## Kết quả thực tế

- `gofmt`: pass
- `go test ./...` trong `cdc-cms-service`: pass
- `npm run build` trong `cdc-cms-web`: pass
- grep router legacy `/registry...`: rỗng
- `make swagger`: fail do thiếu `swag`

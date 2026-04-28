# Phase 26 Validation - Legacy Swagger Cleanup

- `rg -n "@Router\\s+/api/registry" internal/api/registry_handler.go`
- `gofmt`
- `go test ./...`
- `make swagger`

## Kết quả thực tế

- grep `@Router /api/registry` trong `registry_handler.go`: rỗng
- `gofmt`: pass
- `go test ./...` trong `cdc-cms-service`: pass
- `make swagger`: fail do thiếu `swag`

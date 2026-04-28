# Phase 26 Plan - Legacy Swagger Cleanup

1. Audit `registry_handler.go` để tìm các `@Router /api/registry...` còn sót.
2. Đổi các godoc blocks cũ thành comment delegate nội bộ.
3. Verify bằng grep, `gofmt`, `go test ./...`, `make swagger`.

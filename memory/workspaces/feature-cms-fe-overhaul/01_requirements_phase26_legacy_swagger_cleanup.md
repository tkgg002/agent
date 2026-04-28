# Phase 26 Requirements - Legacy Swagger Cleanup

## Mục tiêu

- Đồng bộ source-level swagger annotations với router đã prune.
- Không để `make swagger` trong tương lai sinh lại route `/api/registry...` đã bị xóa khỏi router.

## Definition of Done

- `registry_handler.go` không còn `@Router /api/registry...` cho các route đã bị gỡ.
- Có note rõ các method này chỉ còn là delegate nội bộ/facade backend.
- `go test ./...` pass.

# Implementation Plan: Refactor Common Handler to Base Handler

## Checklist

- [x] Tạo base handler mới ở `internal/handler/base/base_handler.go` với code do User yêu cầu.
- [x] Cập nhật các references trong handlers:
  - [x] `internal/handler/source/sync_handler.go`
  - [x] `internal/handler/master/batch_transform_handler.go`
  - [x] `internal/handler/orchestration/mongo_discover_handler.go`
  - [x] `internal/handler/orchestration/scan_handler.go`
  - [x] `internal/handler/orchestration/discover_handler.go`
  - [x] `internal/handler/master/master_ddl_handler.go`
  - [x] `internal/handler/master/schema_ddl_handler.go`
- [x] Cập nhật server wiring:
  - [x] `internal/server/worker_server_init.go`
- [x] Cập nhật và sửa lỗi các test files:
  - [x] `test/internal/handler/command_handler_activity_integration_test.go`
  - [x] `test/internal/handler/command_handler_test.go`
  - [x] `test/internal/handler/command_handler_cast_expr_test.go`
- [x] Xóa file cũ `internal/handler/common/common.go`.
- [x] Verify build (`go build ./...`) và chạy test suite (`go test ./...`).

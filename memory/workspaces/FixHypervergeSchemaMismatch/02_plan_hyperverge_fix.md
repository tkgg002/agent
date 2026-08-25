# High-level Plan: Remove Forced "_id" -> "id" Overrides in Handlers

## Objective
Xóa bỏ hoàn toàn các câu lệnh ép đè cứng cưỡng chế `_id` thành `"id"` tại `event_handler.go` và `bridge_handler.go` để bảo toàn giá trị `PrimaryKeyField = "_id"` cho các bảng MongoDB.

## Implementation Steps

1. **Refactor `internal/handler/shadow/event_handler.go`:**
   - Xóa bỏ block `if pgPKField == "_id" { pgPKField = "id" }` ở luồng fallback delete.
   - Xóa bỏ block `if !mappedPK && pkField == "_id" { pgPKField = "id" }` ở luồng process event chung.

2. **Refactor `internal/handler/source/bridge_handler.go`:**
   - Xóa bỏ `if resolved.pgPKField == "" || resolved.pgPKField == "_id" { resolved.pgPKField = "id" }`.

3. **Verify:**
   - Chạy `go test ./internal/handler/shadow/...` và `go test ./internal/handler/source/...` để đảm bảo không văng lỗi regression.

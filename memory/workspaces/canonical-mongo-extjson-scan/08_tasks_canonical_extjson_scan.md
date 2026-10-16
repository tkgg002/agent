# 08 Tasks: Canonical MongoDB Extended JSON Scan & Flatten Fix

- [x] Task 1: Cập nhật `flattenJSONWithTypes` và thêm `isCanonicalMongoExtJSON` trong `scan_service.go`
- [x] Task 2: Cập nhật `flattenJSONWithTypes` và thêm `isCanonicalMongoExtJSON` trong `scan_handler.go`
- [x] Task 3: Cập nhật `InferTypeFromRawData` trong `source_router.go`
- [x] Task 4: Cập nhật `unwrapExtJSON` trong `transmute/flatten.go`
- [x] Task 5: Viết Unit Test cho `flattenJSONWithTypes` và ExtJSON wrappers
- [x] Task 6: Chạy kiểm thử tự động, build verification (`go test`, `go build`)
- [x] Task 7: Rà soát Security và nghiệm thu DoD

### Phase 2: Refactor DRY & Unwrapping Coercion
- [x] Task 8: Tái cấu trúc DRY: Export `IsCanonicalMongoExtJSON` từ package `source`, loại bỏ code lặp trong `scan_handler.go` và rút gọn `source_router.go`
- [x] Task 9: Nâng cấp `unwrapExtJSON` trong `transmute/flatten.go`: Bổ sung `mongoNumberToGo`, `strconv.ParseFloat`, và bóc tách `$date` / `$timestamp`
- [x] Task 10: Cập nhật unit tests `strategy_test.go`, `scan_array_path_test.go`
- [x] Task 11: Chạy kiểm thử hồi quy và verify build toàn bộ hệ thống


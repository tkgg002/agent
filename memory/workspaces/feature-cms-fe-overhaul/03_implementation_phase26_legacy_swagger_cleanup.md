# Phase 26 Implementation - Legacy Swagger Cleanup

- Dọn các swagger blocks cũ trong `internal/api/registry_handler.go` cho:
  - `List`
  - `Register`
  - `Update`
  - `BulkRegister`
  - `GetStats`
  - `Standardize`
  - `ScanFields`
  - `DispatchStatus`
- Đổi thành comment mô tả chúng là compatibility delegates cho namespace V2.

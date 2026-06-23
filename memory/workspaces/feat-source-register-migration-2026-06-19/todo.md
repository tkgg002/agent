# TODO: Source Register Migration & Refactor

- [x] Create `internal/handler/source/source_register.go` with `RegisterHandler` and helpers.
- [x] Delete `internal/admin/source_register.go`.
- [x] Delete `internal/admin/types.go`.
- [x] Clean up registration helpers from `internal/admin/helpers.go`.
- [x] Modify `internal/admin/server.go` to inject and route `source.RegisterHandler`.
- [x] Update `internal/admin/server_test.go` to import `source` and use its DTOs.
- [x] Build project and run tests.

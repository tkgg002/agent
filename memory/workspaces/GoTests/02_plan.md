# Execution Plan: GoTests

## Checklist
- [x] Sửa và bổ sung test cho `internal/service/shadow/` (`child_explode_test.go`, `enrichment_service_test.go`, `type_resolver_test.go`, `schema_adapter_coerce_test.go`).
- [ ] Bổ sung unit test cho tầng Repository (Mock qua sqlmock):
  - [ ] `shadow_repo_test.go`
  - [ ] `source_repo_test.go`
  - [ ] `master_repo_test.go`
  - [ ] `recon_repo_test.go`
- [ ] Bổ sung unit test cho tầng Service (Nghiệp vụ Recon & Master):
  - [ ] `master_service_test.go`
  - [ ] `source_service_test.go`
  - [ ] `recon_service_detail_test.go`
- [ ] Kiểm tra và chạy thành công toàn bộ test suite dự án bằng `go test ./...`.

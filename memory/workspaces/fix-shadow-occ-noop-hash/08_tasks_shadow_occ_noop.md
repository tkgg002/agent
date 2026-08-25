# Tasks: Fix Shadow Table OCC No-Op Hash Gate

## Checklist
- [x] Task 1: Nâng cấp hàm `buildOCCWhereClause` trong `centralized-data-service/internal/service/shadow/schema_adapter.go`.
- [x] Task 2: Thêm unit test `TestEventOrdering_SameDataSnapshot_NoOp` trong `centralized-data-service/test/internal/service/schema_adapter_ordering_test.go`.
- [x] Task 3: Chạy test suite `go test ./test/internal/service/...` để kiểm chứng không regression (18/18 PASS).
- [x] Task 4: Chạy kiểm toán chuyên sâu và lưu file vật lý `audit_report_shadow_occ_noop.md`.

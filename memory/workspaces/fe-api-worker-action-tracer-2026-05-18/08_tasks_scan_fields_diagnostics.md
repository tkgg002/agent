# Tasks — scan-fields Diagnostics

**Phase**: fe-api-worker-action-tracer-2026-05-18 / scan_fields_diagnostics

## Checklist

- [x] T1 — Đọc handler + service + override + model (Bash + Read trên 6 file)
- [x] T2 — Viết `01_requirements_scan_fields_diagnostics.md`
- [x] T3 — Viết `02_plan_scan_fields_diagnostics.md`
- [x] T4 — Implement `SanitizeMongoDSN` + `IntrospectDiagnosis` + `IntrospectCollectionDiagnose` trong `internal/service/mongo_introspection.go`
- [x] T5 — Implement test cho `SanitizeMongoDSN` trong `internal/service/mongo_introspection_test.go`
- [x] T6 — Refactor `scanFieldsMongoSource` trong `internal/handler/command_handler.go` để dùng `IntrospectCollectionDiagnose` + log INFO upfront + 5-case switch
- [x] T7 — `go build ./...` (worker) PASS
- [x] T8 — `go vet ./...` (worker) PASS
- [x] T9 — `go test -count=1 ./...` (worker) PASS — service 1.338s, handler 3.332s, tất cả package khác PASS
- [x] T10 — Viết `03_implementation_scan_fields_diagnostics.md`
- [x] T11 — Viết `09_tasks_solution_scan_fields_diagnostics.md`
- [x] T12 — Viết `report_scan_fields_diagnostics.md`
- [x] T13 — APPEND `05_progress.md` (5+ dòng timestamped + agent + model)
- [x] T14 — APPEND `agent/memory/global/lessons.md` (lesson "Generic Empty Error Hides Multi-Cause Failure")
- [ ] T15 — User restart worker `tty003` → `go run cmd/worker/main.go` (chờ)
- [ ] T16 — User click Scan Fields lại trên `export-jobs` → đọc log mới để xác định root cause thật (chờ)

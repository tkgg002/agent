# Task List: Phase 1.10 Execution

## Status: COMPLETED

- [x] Research codebase (TableRegistry, QueueMonitoring, SchemaChanges, router, models)
- [x] Tạo 03_implementation_phase_1.10.md trong workspace
- [x] Cập nhật 05_progress.md
- [x] B1: Migration SQL — thêm `status` + `rule_type` vào `cdc_mapping_rules`
- [x] B2: Update `mapping_rule.go` — thêm 2 field mới
- [x] B3: `registry_handler.go` — thêm `ScanSource` handler
- [x] B4: `registry_handler.go` — thêm `ScanFields` handler
- [x] B5: `router.go` — đăng ký 2 routes mới
- [x] B6: Build verify `go build ./...`
- [x] F1: `QueueMonitoring.tsx` — fix crash (division by zero)
- [x] F2: `TableRegistry.tsx` — title align-left + full columns + Scan Fields btn
- [x] F3: `SchemaChanges.tsx` — đổi endpoint sang `/api/mapping-rules?status=pending`

# 08_tasks_invalid_json_syntax.md: Danh sách Task chi tiết

- [ ] Task 1: Document Root Cause & Technical Solution in `09_tasks_solution_invalid_json_syntax.md` & `12_implementation_plan_invalid_json_syntax.md` (Brain)
- [ ] Task 2: Standardize `loadSchemaInSchema` & `IsJSONB` in `schema_adapter.go` and `schema_adapter_coerce.go` (Muscle)
- [ ] Task 3: Harden `decodeBase64JSON` and `CoerceValue` for `JSON`/`JSONB` columns (Muscle)
- [ ] Task 4: Add fallback guard for `_raw_data` JSONB metadata column (Muscle)
- [ ] Task 5: Harden `coerceForColumn` in `transmuter_utils.go` for Master Table transmute path (Muscle)
- [ ] Task 6: Add Unit Tests in `schema_adapter_coerce_test.go` and run verification suite (`go test ./...`) (Muscle)
- [ ] Task 7: Run Security Auto-Check (`/security-agent`) & Verification Audit (Muscle)

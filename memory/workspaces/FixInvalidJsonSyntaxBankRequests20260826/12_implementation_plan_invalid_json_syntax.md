# 12_implementation_plan_invalid_json_syntax.md: Kế hoạch Triển khai Chi tiết AI

## Phase 1: Preparation & Doc Alignment
- Khởi tạo workspace & documents theo chuẩn Governance v1.10 (Done).
- Đọc `lessons.md` & `GEMINI.md` (Done).

## Phase 2: Execution (Muscle Role)
1. **Edit `internal/service/shadow/schema_adapter.go`**:
   - Call `strings.ToLower(strings.TrimSpace(r.DataType))` trong `loadSchemaInSchema`.
   - Protect `_raw_data` trong `getMetadataInsertPlaceholdersAndValues` để fallback về `{}` nếu rỗng/invalid.
2. **Edit `internal/service/shadow/schema_adapter_coerce.go`**:
   - Enhance `IsJSONB` to check normalized `strings.HasPrefix(dt, "json")`.
   - Update `decodeBase64JSON` to avoid returning un-marshaled raw `[]byte`.
   - Fix `CoerceValue` to guarantee 100% valid JSON string outputs for all JSON/JSONB columns.
3. **Edit `internal/service/master/transmuter_utils.go`**:
   - Harden `coerceForColumn` for JSON/JSONB master columns.
4. **Add Unit Tests**:
   - Add test cases covering string ID `6a8e3fc31b3d2729eb078a60`, empty string `""`, map/slice, unwrapped Mongo Extended JSON, and invalid base64 string inputs to `schema_adapter_coerce_test.go`.
5. **Run Verification**:
   - Execute `go test ./...` in `centralized-data-service`.

## Phase 3: Post-Execution Audit & Reporting
- Audit against DOD Gates (G1 - G8).
- Append progress log into `05_progress_invalid_json_syntax.md`.
- Report solution & results clearly to User.

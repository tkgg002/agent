# 05 Progress Log — Mapping Fields: Add "Data Type Current" Column

> Append-only. KHÔNG xóa, sửa nội dung cũ.

---

## [2026-08-27T09:07] [Brain:Claude-Sonnet-4.6] SESSION START

### Task
Thêm cột "Data Type Current" lên trang `/shadow/72/mappings?binding_id=190`  
URL: `MappingFieldsPage.tsx`

### Research Done
- Đọc `MappingFieldsPage.tsx` (765 lines): hiểu cấu trúc UI, columns table hiện tại
- Đọc `MappingRule` TypeScript interface: không có field nào về "current type"
- Đọc `MappingRuleRow` DTO (Go): không có field về actual shadow column type
- Đọc `ShadowColumns` API handler: trả về `[]string` column names, không có data_type
- Đọc `GetColumns` impl: `SELECT column_name FROM information_schema.columns` — chỉ lấy tên, không lấy type
- **Kết luận:** Backend cần được mở rộng để lấy `data_type` từ `information_schema.columns`

### Plan
Xem `implementation_plan.md`

### Status: PLANNING

---

## [2026-08-27T10:47] [Muscle:Claude-Sonnet-4.6] EXECUTION

### Files Modified
1. `cdc-cms-service/internal/app/ports/repository.go` — thêm `GetColumnsWithTypes` vào `ShadowSchemaReader` interface
2. `cdc-cms-service/internal/infra/persistence/shadow/shadow_schema_reader_gorm.go` — implement `GetColumnsWithTypes`, query `information_schema.columns` lấy `data_type + udt_name`
3. `cdc-cms-service/internal/api/system/introspection_handler.go` — thêm handler `ShadowColumnsWithTypes`
4. `cdc-cms-service/internal/router/router.go` — đăng ký route `GET /introspection/shadow-columns-with-types/:table`
5. `cdc-cms-web/src/pages/MappingFieldsPage.tsx` — thêm state `shadowColumnTypes`, cập nhật `fetchShadowColumns` gọi endpoint mới, thêm cột "Data Type Current" với drift detection

### Verification
- `go build ./internal/... ./cmd/...` → EXIT CODE 0 ✅
- `npx tsc --noEmit` → EXIT CODE 0 ✅

### Status: DONE ✅

---

## [2026-08-27T13:10] [QA/Brain:Gemini-3.7-Flash] ADVERSARIAL AUDIT & REMEDIATION

### Audit Findings & Self-Improvement
1. **Bug #1 (Critical - Fixed):** `information_schema.columns.data_type` trả về ANSI standard names (`"character varying"`, `"timestamp with time zone"`). Bổ sung `normalizePostgresType` để map về `"VARCHAR"`, `"TIMESTAMPTZ"`, `"CHAR"`, `"TIMESTAMP"`.
2. **Bug #2 (Precision Normalization - Fixed):** FE drift comparison đã thêm regex normalization `replace(/\(\d+(?:,\s*\d+)?\)/, '')` để so sánh chuẩn giữa `VARCHAR(255)` vs `VARCHAR` mà không bị false positive drift.
3. **Bug #3 (Key Normalization - Fixed):** Backend chuẩn hóa map key lowercase `strings.ToLower(row.ColumnName)`. FE lookup chuẩn hóa lowercase.
4. **Bug #4 (Code Style - Fixed):** Dọn dẹp blank line dư trong `introspection_handler.go`.
5. **Lesson Added:** Đã ghi lesson mới vào `lessons.md`: *[2026-08-27] Không normalize kiểu dữ liệu từ information_schema.columns dẫn đến so sánh drift sai lệch hoàn toàn*.
6. **Audit Report Created:** Lưu báo cáo chi tiết tại `audit_report_2026-08-27.md`.

### Final Verification
- `go build ./internal/... ./cmd/...` → PASS (EXIT CODE 0)
- `go test ./internal/...` → PASS (EXIT CODE 0)
- `npx tsc --noEmit` → PASS (EXIT CODE 0)
- Governance KPI script `governance_metrics.sh` → PASS (100% format-compliant)

### Status: AUDIT PASSED & REMEDIATED ✅


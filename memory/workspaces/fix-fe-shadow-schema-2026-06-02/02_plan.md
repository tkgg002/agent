# 02_plan — Fix FE shadow_schema (Architectural)

## Triết lý
- **BE owns naming**. FE chỉ pass-through.
- Xoá hết logic FE compose schema → khi DB có schema lạ (multi-tenant `shadow_<conn>_<db>`), FE vẫn đúng.
- Không patch fallback — **delete the fallback entirely**.

## Phase 0 — Audit (đã xong, xem 00_context)

## Phase 1 — BE đảm bảo `shadow_schema` luôn present
### T1: Sửa SQL `GetMappingContextByRegistryID` để `shadow_schema` non-null
- File: `cdc-cms-service/internal/infra/persistence/source_object_read_repo_gorm.go:189-284`.
- Đổi `sb.shadow_schema,` → `COALESCE(sb.shadow_schema, '') AS shadow_schema,`.
- Đổi `sb.physical_table_fqn,` → `COALESCE(sb.physical_table_fqn, '') AS physical_table_fqn,` (consistency).
- **Lý do**: nếu LATERAL không match → schema NULL → JSON omit → FE undefined → fallback bậy. COALESCE đảm bảo luôn có string trong response.

### T2: Sửa model JSON tag để KHÔNG omitempty
- File: `cdc-cms-service/internal/app/queries/source_objects_read_models.go:70-71`.
- Đổi:
  ```go
  ShadowSchema     *string   `json:"shadow_schema,omitempty"`
  PhysicalTableFQN *string   `json:"physical_table_fqn,omitempty"`
  ```
  → 
  ```go
  ShadowSchema     string    `json:"shadow_schema"`
  PhysicalTableFQN string    `json:"physical_table_fqn"`
  ```
- **Lý do**: `string` luôn marshal (kể cả ""). Loại bỏ pointer + omitempty → FE luôn nhận field.

### T3: Verify BE build + smoke
- `cd cdc-cms-service && go build ./...` EXIT=0.
- `curl http://localhost:8083/api/v1/source-objects/registry/15?binding_id=59 | jq .shadow_schema` → string non-empty.

## Phase 2 — FE cleanup: xoá `normalizeShadowSchema`

### T4: Xoá định nghĩa function ở 4 file
- `src/pages/TableRegistry.tsx:75-81` — xoá.
- `src/pages/MappingFieldsPage.tsx:14-17` — xoá.
- `src/pages/DataIntegrity.tsx:56-62` — xoá.
- `src/pages/ActivityManager.tsx:65` — xoá (inline arrow trong component).

### T5: Xoá `|| normalizeShadowSchema(...)` ở 13 callsite
Thay `record.shadow_schema || normalizeShadowSchema(record.source_db)` → `record.shadow_schema || ''` (giữ empty string fallback cho UI render).

**Site list**:
- TableRegistry.tsx: 85, 780, 884
- MappingFieldsPage.tsx: 24, 112, 198, 213, 357, 372, 572
- DataIntegrity.tsx: 65, 81
- ActivityManager.tsx: 135, 138, 176

### T6: Cập nhật type
- `src/types/index.ts:67,134` — `shadow_schema?: string | null` giữ nguyên (compat với row chưa register).
- `src/types/index.ts:112` (SourceObjectMappingContext) — `shadow_schema: string` (required).

### T7: UI fallback rỗng
- `MappingFieldsPage.tsx:572` Descriptions item:
  - Nếu `registry.shadow_schema` empty → render `<Text type="secondary">(chưa có)</Text>`.
- `MappingFieldsPage.tsx:213` `fetchShadowColumns`:
  - Nếu schema empty → early return + `setShadowColumns(new Set())`, không call API.

### T8: Verify FE
- `cd cdc-cms-web && npx tsc --noEmit -p tsconfig.app.json` EXIT=0.
- `grep -r normalizeShadowSchema src/` → empty.

## Phase 3 — Smoke test end-to-end

### T9: Manual smoke
- Reload `/shadow/15/mappings?binding_id=59`.
- DevTools Network → verify URL `?schema=shadow_goopay_test_local_as_auth_service`.
- Verify response `columns` non-null.
- Click sang `/shadow` → cột Shadow Schema hiển thị đúng giá trị DB.

### T10: Report
- Tạo `report_2026-06-02_fix-fe-shadow-schema.md`:
  - LOC delta: số dòng xoá ở 4 file FE + 2 file BE.
  - Output `go build` + `tsc`.
  - A1-A5 PASS/FAIL.

## Quyết định (Decisions)

### D1: Tại sao COALESCE thay vì FIX JOIN?
- JOIN logic đúng — nếu user truyền `binding_id=59` đúng → LATERAL match.
- Tuy nhiên có khả năng row không có binding (v2_source_only) → LATERAL miss legitimate.
- COALESCE bảo vệ contract: API luôn trả field, FE không phải đoán.

### D2: Tại sao xoá `*string`?
- `*string` + `omitempty` = source of bug. Nil → key omitted → FE thấy undefined → bị bypass null-check.
- `string` empty = explicit signal. FE check `if (!shadow_schema)` rõ ràng.

### D3: Tại sao `|| ''` thay vì xoá hoàn toàn?
- Một số site (display Descriptions) cần string để render. `|| ''` an toàn cho render.
- Site critical (fetch URL) sẽ guard bằng early-return.

### D4: Có cần migrate type field thành required `string`?
- Có. `SourceObjectMappingContext.shadow_schema: string` → buộc compiler check tất cả call site.
- Field khác (`ReconReport.shadow_schema`) giữ `string | null` vì có thể null hợp lệ.

### D5: Có break consumer khác?
- Endpoint chỉ phục vụ MappingFieldsPage + TableRegistry. Grep verified không có consumer khác.
- BE response thêm key (thay vì omit) → backward compat OK.

## Rollback
1. Revert commit BE → JSON lại omitempty → FE đọc undefined → nhưng không còn fallback → render `(chưa có)` → user thấy UI rỗng. Không crash.
2. Revert commit FE → restore `normalizeShadowSchema` → trở về bug cũ. Acceptable rollback.

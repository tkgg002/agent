# 08_tasks — Fix FE shadow_schema (Architectural)

> Plan-only. KHÔNG set `in_progress` cho đến khi user duyệt.

## T1 — BE: COALESCE shadow_schema trong SQL
- **Owner**: Muscle
- **File**: `cdc-cms-service/internal/infra/persistence/source_object_read_repo_gorm.go`
- **DoD**: 
  - SQL `sb.shadow_schema,` đổi thành `COALESCE(sb.shadow_schema, '') AS shadow_schema,`.
  - Cùng pattern cho `physical_table_fqn`.
  - `go build ./...` EXIT=0.

## T2 — BE: Đổi `*string` → `string` ở model
- **Owner**: Muscle
- **Phụ thuộc**: T1
- **File**: `cdc-cms-service/internal/app/queries/source_objects_read_models.go:70-71`
- **DoD**:
  - `ShadowSchema string`, `PhysicalTableFQN string`, bỏ `omitempty`.
  - `go build ./...` EXIT=0.
  - `grep -rn "\.ShadowSchema\|\.PhysicalTableFQN" cdc-cms-service/internal/` → kiểm tra không có site deref `*model.ShadowSchema`.

## T3 — BE: Smoke endpoint
- **Owner**: Muscle
- **Phụ thuộc**: T2
- **DoD**:
  - Restart cdc-cms-service.
  - `curl -H "Authorization: Bearer $JWT" "http://localhost:8083/api/v1/source-objects/registry/15?binding_id=59" | jq .shadow_schema` → `"shadow_goopay_test_local_as_auth_service"`.

## T4 — FE: Xoá `normalizeShadowSchema` definition (4 file)
- **Owner**: Muscle
- **DoD**: 
  - Xoá khối function tại TableRegistry.tsx:75-81, MappingFieldsPage.tsx:14-17, DataIntegrity.tsx:56-62, ActivityManager.tsx:65.
  - `grep -rn "function normalizeShadowSchema\|const normalizeShadowSchema" cdc-cms-web/src/` → empty.

## T5 — FE: Thay 13 callsite `|| normalizeShadowSchema(...)` → `|| ''`
- **Owner**: Muscle
- **Phụ thuộc**: T4 (delete định nghĩa trước để compiler báo lỗi callsite còn sót)
- **DoD**:
  - TableRegistry.tsx: 85, 780, 884 fixed.
  - MappingFieldsPage.tsx: 24, 112, 198, 213, 357, 372, 572 fixed.
  - DataIntegrity.tsx: 65, 81 fixed.
  - ActivityManager.tsx: 135, 138, 176 fixed.
  - `grep -r normalizeShadowSchema cdc-cms-web/src/` → empty.

## T6 — FE: Hardening `fetchShadowColumns` early-return
- **Owner**: Muscle
- **Phụ thuộc**: T5
- **File**: `src/pages/MappingFieldsPage.tsx:210-220`
- **DoD**:
  - Nếu `registry.shadow_schema` empty → `setShadowColumns(new Set())` + return.
  - Không gọi API với schema rỗng.

## T7 — FE: UI fallback rỗng ở Descriptions
- **Owner**: Muscle
- **Phụ thuộc**: T5
- **File**: `src/pages/MappingFieldsPage.tsx:572`
- **DoD**:
  - Ternary render `<Text code>{value}</Text>` vs `<Text type="secondary">(chưa có)</Text>`.

## T8 — FE: Type-check
- **Owner**: Muscle
- **Phụ thuộc**: T7
- **DoD**:
  - `cd cdc-cms-web && npx tsc --noEmit -p tsconfig.app.json` EXIT=0.

## T9 — Smoke test end-to-end
- **Owner**: Muscle
- **Phụ thuộc**: T8 + T3
- **DoD**:
  - Reload `/shadow/15/mappings?binding_id=59` → DevTools Network shows `?schema=shadow_goopay_test_local_as_auth_service`.
  - Response `columns` non-null.
  - `/shadow` row hiển thị đúng schema column.
  - A1-A5 PASS.

## T10 — Report
- **Owner**: Muscle
- **DoD**:
  - Tạo `report_2026-06-02_fix-fe-shadow-schema.md`.
  - LOC delta thực tế.
  - 5 acceptance criteria PASS/FAIL.

## Anti-tasks (KHÔNG làm)
- ❌ KHÔNG fix BE `normalizeShadowSchemaWithConnection` (V2 register naming) — đó là source of truth.
- ❌ KHÔNG migrate dữ liệu shadow_binding.
- ❌ KHÔNG đụng worker `provisioning_step_handlers.go` (pattern khác — V1 legacy).
- ❌ KHÔNG add new fallback function ở FE.

## Escalation
- Nếu T3 smoke fail (BE vẫn trả null) → dừng, ghi 05_progress, debug query LATERAL bằng `EXPLAIN ANALYZE` trên Postgres production.
- Nếu T8 type-check fail vì grep miss site → restore 1 file, type-check lại để Identify site, fix lại.

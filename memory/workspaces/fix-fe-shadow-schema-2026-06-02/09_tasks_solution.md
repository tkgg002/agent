# 09_tasks_solution — Fix FE shadow_schema (Solution dossier)

## S1 — Edge cases BE

### S1.1 — Registry tồn tại nhưng không có shadow_binding (v2_source_only)
- `sb` LATERAL không match → COALESCE trả `""`.
- Response: `{shadow_schema: "", ...}` — không omit.
- FE: `registry.shadow_schema = ""` → guard `if (!schema)` → không call API → UI hiển thị `(chưa có)`.

### S1.2 — Registry có binding nhưng shadow_schema NULL trong DB
- Edge case rất hiếm (migration cũ). COALESCE → `""`.
- Cùng UX với S1.1.

### S1.3 — `binding_id` truyền sai (không tồn tại)
- LATERAL ưu tiên `sb.id = bindingID` → không match → `sb.*` NULL → COALESCE → `""`.
- Đây là contract: caller truyền `binding_id` sai → page hiển thị "(chưa có)" → user biết deep-link broken.

### S1.4 — Multi-binding source: cùng `source_object_id`, khác `shadow_schema`
- LATERAL pick theo `binding_id` cụ thể → trả đúng schema của binding đó. ✓

## S2 — Edge cases FE

### S2.1 — Reload page khi BE chưa deploy version mới
- BE vẫn omit `shadow_schema` (null) → FE đọc `undefined` → `|| ''` → `""` → guard → `(chưa có)`.
- Không crash. UX: nút mapping không hoạt động. User reload sau khi BE deploy → OK.

### S2.2 — DataIntegrity row có `shadow_schema = null`
- `record.shadow_schema || ''` → empty string.
- `getResolvedShadowFqn` trả `".target_table"` (leading dot) → cosmetic issue.
- Fix: trong `DataIntegrity.tsx`, nếu schema empty → trả chỉ `target_table` không có dot.

### S2.3 — `AddMappingModal` initialValues có `shadow_schema: ""`
- AntD Form sẽ submit empty string → BE validate fail nếu required.
- Acceptable: user nhập tay nếu thực sự cần (hiếm), hoặc disable button "Thêm" khi schema empty.

### S2.4 — ActivityManager `meta` không có field `shadow_schema`
- Cần type guard. Nếu `meta` interface không có → thêm `shadow_schema?: string` ở interface declaration cùng file.
- Đọc kỹ ActivityManager.tsx:25-40 (interface) trước khi sửa T5.

## S3 — Verify checklist (S → Smoke)

| ID | Step | Expected |
|----|------|----------|
| TC1 | Reload `/shadow/15/mappings?binding_id=59` | Network: `shadow-columns/tokens?schema=shadow_goopay_test_local_as_auth_service` |
| TC2 | Same → response columns | `{columns: [...], schema: "shadow_...", table: "tokens"}` non-null |
| TC3 | Open `/shadow` page → row id=15 | Cột "Shadow Schema" hiển thị `shadow_goopay_test_local_as_auth_service` |
| TC4 | Click row chưa có binding (v2_source_only) | Cột "Shadow Schema" hiển thị empty / `(chưa có)`, không crash |
| TC5 | DevTools console | KHÔNG có warning `normalizeShadowSchema is not defined` |
| TC6 | `grep -r normalizeShadowSchema cdc-cms-web/src/` | Empty |
| TC7 | `curl registry/15?binding_id=59` | `shadow_schema` field present, string |
| TC8 | `curl registry/15?binding_id=999` (sai) | `shadow_schema: ""` |

## S4 — Lý do KHÔNG fix theo cách khác

### S4.1 — Tại sao không "fix algorithm `normalizeShadowSchema` cho khớp BE"?
- BE V2 dùng `shadow_<conn>_<db>` (cần biết `connection_code` từ DB).
- FE không có `connection_code` trong context của TableRegistry row.
- Reproduce algorithm BE ở FE = **leaky abstraction**, tightly couple FE với BE naming convention. Mỗi lần BE đổi naming → FE phải sync. Anti-pattern.

### S4.2 — Tại sao không "fetch connection_code rồi FE compose"?
- Vẫn duplicate logic. Naming có thể đổi (e.g. thêm env prefix, hash). BE owns naming.

### S4.3 — Tại sao COALESCE thay vì redesign endpoint?
- Minimal change. Endpoint contract sửa thêm field "always present" là backward compat.
- Redesign = scope creep.

## S5 — Rollback plan

### Bước 1 (revert BE)
- Revert SQL COALESCE + struct field type.
- Effect: API lại omit `shadow_schema` khi null.
- FE đã xoá fallback → render `(chưa có)` cho row null → UX degrade nhưng không crash.

### Bước 2 (revert FE)
- Revert 4 file FE.
- Effect: trở lại bug cũ (fallback bậy). Acceptable rollback.

### Cả hai bước an toàn vì:
- Không có data destructive.
- Không có migration DB.
- Chỉ thay đổi API contract (thêm field) + FE rendering.

## S6 — Verification commands

```bash
# BE build
cd /Users/trainguyen/Documents/work/data-hub/cdc-cms-service && go build ./...

# BE smoke
JWT="<token>"
curl -H "Authorization: Bearer $JWT" \
  "http://localhost:8083/api/v1/source-objects/registry/15?binding_id=59" \
  | jq '{shadow_schema, physical_table_fqn, source_db, target_table}'

# FE type-check
cd /Users/trainguyen/Documents/work/data-hub/cdc-cms-web && npx tsc --noEmit -p tsconfig.app.json

# FE grep verify
grep -rn "normalizeShadowSchema" cdc-cms-web/src/ ; echo "exit=$?"

# DB sanity (đã verify)
psql -h localhost -U cdc -d cdc_db -c "
  SELECT sb.id, sb.shadow_schema, sb.shadow_table
  FROM cdc_system.shadow_binding sb
  WHERE sb.id = 59
"
```

## S7 — Lessons tiềm năng (ghi sau khi xong T10)

- **Lesson candidate**: "FE compose domain identifier (schema/table/key) duplicates BE source-of-truth → khi BE đổi naming, FE phá vỡ. Đúng: BE owns naming, FE chỉ render. Cụ thể: pattern `record.X || frontendCompute(record.Y)` luôn là smell."
- **Lesson candidate**: "JSON `omitempty` + `*pointer` cho field critical (UI dependency) tạo `undefined` ở FE → bypass null check. Đúng: dùng value type + `string`/`int` zero value + non-omit tag để contract explicit."
- **Lesson candidate**: "Khi user reject phương án 'patch N callsite' và demand architectural fix → phải xoá root cause (function), không patch usage. `delete the dependency, not the symptoms`."

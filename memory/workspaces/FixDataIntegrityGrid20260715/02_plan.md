# 02 — Plan: FixDataIntegrityGrid20260715

## Tổng quan

Fix 2 bugs trên trang `/data-integrity`:
1. `— — 2,718,739`: source/shadow count hiển thị dashes
2. Pipelines xoá source object vẫn hiện trên grid

---

## Bug 1 Analysis: `— — 2,718,739`

**Root Cause:**
- `DISTINCT ON (shadow_schema, shadow_table, ...)` trong inner UNION bị split:
  - Nhánh `cdc_reconciliation_report`: records cũ có `shadow_schema = NULL`
  - Nhánh `cdc_recon_smoke_result`: records mới có `shadow_schema = 'shadow_testss'`
  - PostgreSQL: NULL ≠ 'shadow_testss' → 2 DISTINCT ON groups riêng
- FE `buildPipelines` dedup dùng `checked_at DESC` → record NULL-schema (mới hơn vài phút) thắng, hiện `— —`

**Những gì đã làm (trước khi có plan):**
- Backend: Thêm LATERAL JOIN `sb_norm` + COALESCE shadow_schema → chưa resolve (sb_norm trả NULL vì shadow_table của record cũ cũng có thể NULL)
- FE: Sửa dedup ưu tiên row có active counts → đây là fix đúng hướng, là safety net

**Plan fix cuối cùng (2 lớp):**

### Lớp 1: FE dedup (đã implement, cần verify)
```js
const rHasActive = r.source_active != null || r.shadow_active != null || r.master_active != null;
const exHasActive = ...;
const shouldReplace = !existing
  || (rHasActive && !exHasActive)
  || (rHasActive === exHasActive && new Date(r.checked_at) > new Date(existing.checked_at));
```
→ Ưu tiên row có data, không để record "trống" thắng

### Lớp 2: Backend SQL cleanup
Thêm điều kiện vào WHERE của nhánh `cdc_reconciliation_report` để **loại hẳn records cũ không có shadow_schema**:
```sql
WHERE (r.shadow_schema IS NOT NULL OR sb_norm.shadow_schema IS NOT NULL)
```
Hoặc đơn giản hơn, thêm filter vào outer query:
```sql
WHERE COALESCE(r.shadow_schema, sb.shadow_schema) IS NOT NULL
```

---

## Bug 2 Analysis: Pipelines xoá source object vẫn hiện

**Root Cause:**
- Khi user xoá source object trong CMS → `source_object_registry.is_active = FALSE`
- `shadow_binding` có thể vẫn `is_active = TRUE` (không cascade deactivate ngay)
- `listLatestPrimary` dùng `LEFT JOIN source_object_registry so` → **không filter `so.is_active`**
- Rows trong `cdc_recon_smoke_result` / `cdc_reconciliation_report` vẫn tồn tại
- → Pipeline vẫn hiện

**Fix: Thêm filter `so.is_active = TRUE` hoặc dùng INNER JOIN**

Thay dòng 179:
```sql
-- Cũ:
LEFT JOIN cdc_system.source_object_registry so ON so.id = sb.source_object_id
-- Mới: Chỉ giữ pipeline nếu source object còn active
LEFT JOIN cdc_system.source_object_registry so ON so.id = sb.source_object_id AND so.is_active = TRUE
```

Đồng thời, nếu `sb` null (không có active shadow_binding) thì `so` cũng null → pipeline sẽ có `source_connection_code = null`. Cần thêm điều kiện thứ 2:

**Option A (backend):** Thêm `WHERE (sb.shadow_table IS NULL OR so.is_active = TRUE)` vào outer WHERE — nhưng phức tạp.

**Option B (backend + FE — đơn giản, ít rủi ro):** 
- Backend: Đổi thành `INNER JOIN shadow_binding sb ON ... AND sb.is_active = TRUE` (hiện là LEFT JOIN LATERAL → đổi cách JOIN)
- Hoặc thêm WHERE filter ở ngoài cùng: `AND (sb.source_object_id IS NULL OR so.is_active = TRUE)`

**Chọn: Option B — thêm WHERE condition ở cuối outer query**:
```sql
-- Sau ORDER BY r.shadow_table, thêm:
AND (sb.source_object_id IS NULL OR so.is_active = TRUE)
```

Điều này có nghĩa: 
- Nếu không có shadow_binding → vẫn hiện (giữ behavior cũ)
- Nếu có shadow_binding → chỉ hiện nếu source object còn active

---

## Definition of Done

- [ ] API `/api/reconciliation/report` trả đúng số rows, `source_active` != null cho schedule_histories
- [ ] Grid không hiện `— —` cho các pipeline có smoke data
- [ ] Pipelines có source object `is_active = FALSE` không xuất hiện trên grid
- [ ] TypeScript check pass
- [ ] `go vet` pass

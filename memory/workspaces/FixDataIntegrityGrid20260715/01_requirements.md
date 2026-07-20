# 01 — Yêu cầu: Fix Data Integrity Grid

**Task:** Fix 2 bugs trên trang `http://localhost:5173/data-integrity`

---

## Bug 1: Hiển thị `— — 2,718,739` (source/shadow = dashes, master có số)

**Mô tả:** Một số pipeline rows trên grid hiển thị `—` cho source count và shadow count dù master count hiển thị đúng số.

**Root Cause đã xác định:**
- `listLatestPrimary` dùng `DISTINCT ON (shadow_schema, shadow_table, master_schema, master_table, segment)` trên UNION ALL của 2 bảng.
- Nhánh `cdc_reconciliation_report`: records cũ có `shadow_schema = NULL` trong DB.
- Nhánh `cdc_recon_smoke_result`: records mới có `shadow_schema = 'shadow_testss'`.
- PostgreSQL coi `NULL ≠ 'shadow_testss'` → 2 group riêng → cả 2 pass DISTINCT ON.
- FE `buildPipelines` dedup dùng `checked_at DESC` → record NULL-schema thắng vì mới hơn → hiện `— —`.

**DoD:**
- API `GET /api/reconciliation/report` trả đúng 2 rows per pipeline (1 source_shadow + 1 shadow_master).
- `source_active` và `shadow_active` có giá trị số, không phải null.
- Grid hiển thị số đúng, không có `— —`.

---

## Bug 2: Pipelines đã xoá connector vẫn hiện trên list

**Mô tả:** Các pipeline mà connector Debezium đã bị xoá vẫn còn xuất hiện trên data-integrity grid.

**Cần điều tra thêm:**
- User clarified: "xoá source object trong CMS" = deactivate/delete source object trong source_object_registry (is_active = FALSE).
- `listLatestPrimary` có filter `INNER JOIN cdc_table_registry reg ON reg.is_active = TRUE` — nhưng đây là shadow table registry, không phải source object.
- Cần kiểm tra outer query có join `source_object_registry` với `is_active = TRUE` không.
- Nếu không có → pipeline vẫn hiện vì records trong cdc_reconciliation_report và cdc_recon_smoke_result vẫn tồn tại.

**DoD:**
- Pipelines có source object bị xoá (is_active = FALSE trong source_object_registry) không xuất hiện trên grid.

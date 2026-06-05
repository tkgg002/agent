# 09_solution — FE Humanize API Errors

## Approach
**Minimal-impact (§6)**: tạo 1 helper centralize, sweep call sites đổi sang gọi helper. Không sửa BE.

## Files changed

### NEW: `cdc-cms-web/src/utils/apiError.ts` (+144 LOC)
Helper `humanizeApiError(err, fallback)`:
- Bắt `ECONNABORTED` / `ERR_NETWORK` riêng.
- Trích raw text từ `response.data.error|detail|message` rồi tới `err.message`.
- Match regex `SQLSTATE (\w+)` → dispatch theo SQLSTATE map (14 codes).
- 23505/23503/23514: trích thêm constraint name → tra `CONSTRAINT_HUMAN` map (6 constraint cụ thể tới dự án).
- 23502/42703/42P01: trích column/table name → inject vào message.
- Generic match: `timeout|context deadline` → "Server phản hồi quá lâu". `permission denied` → "Không đủ quyền".
- Fallback HTTP status (400-504): map sang câu tiếng Việt.
- Cuối cùng `trimRaw(raw)` cắt prefix `ERROR:`, hậu tố `(SQLSTATE …)`, clamp 250 ký tự.

### SWEEP: 14 file gọi `humanizeApiError`
- `pages/TableRegistry.tsx` (3 sites: update, register, snapshot).
- `pages/MappingFieldsPage.tsx` (7 sites: batch update, sync fields, toggle, mask, datatype, scan-fallback, backfill, reload).
- `pages/SensitiveFieldsPage.tsx` (4 sites: fetch, add, update strategy, delete).
- `pages/SourceToMasterWizard.tsx` (2 sites: save, execute).
- `pages/ActivityManager.tsx` (2 sites: update schedule, create).
- `pages/SchemaChanges.tsx` (2 sites: approve, reject).
- `pages/SourceConnectors.tsx` (4 sites: create, update, op, delete).
- `pages/Login.tsx` (1 site).
- `pages/DataIntegrity.tsx` (1 site).
- `pages/MasterRegistry.tsx` (2 sites).
- `pages/TransmuteSchedules.tsx` (3 sites: create, toggle, run-now).
- `pages/SchemaProposals.tsx` (1 site).
- `components/MappingRuleList.tsx` (4 sites: fetch, scan, backfill, reload).
- `components/AddMappingModal.tsx` (1 site).

### Localize English fallbacks → Vietnamese
- `MappingFieldsPage.tsx:197` "Failed to fetch mapping rules" → "Không tải được mapping rules."
- `SourceToMasterWizard.tsx:82` "Session not found" → "Không tìm thấy wizard session."
- `SourceToMasterWizard.tsx:101` "Cannot create wizard session" → "Không tạo được wizard session."
- `MasterRegistry.tsx:199` "Swap failed" → "Swap master thất bại".

## Result với lỗi user báo
- Input: SQLSTATE 23505 + constraint `cdc_table_registry_conn_source_db_table_target_key`.
- Output mới: **"Cặp (source DB + source table + target table) đã được đăng ký rồi."**

## Verify
- `npx tsc --noEmit -p tsconfig.app.json` EXIT=0 (toàn bộ project pass).
- `grep -r "humanizeApiError" src` → 14 file + helper.

## Out-of-scope
- BE chuẩn hoá error envelope (`code`/`detail`/`hint`) — nếu sau này muốn cấu trúc hoá thì sửa BE; helper FE đã đủ tốt cho hiện trạng.
- i18n: hiện tại hard-code tiếng Việt. Nếu mở rộng sang en/vi sẽ chuyển sang key + dict.

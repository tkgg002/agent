# 02_plan

## Strategy
FE-only patch — gom logic vào 4 chỗ trong `cdc-cms-web/src/pages/TableRegistry.tsx`:

1. **P-1**: Compute `firstRowIndexBySource: Map<sourceObjectId, index>` qua `useMemo([data])`. Nguồn truth cho "first occurrence" check.
2. **P-2**: Trong column "Thao tác" render (`render(_, record, index)`), conditional show Edit button khi `firstRowIndexBySource.get(record.id) === index`.
3. **P-3**: Snapshot + Manage Masters button: `disabled={!record.is_active}` + Tooltip hint.
4. **P-4**: AsyncRowActions: compute `effectiveActive` (per-binding hoặc per-source fallback), pass vào Quét field button `disabled={!canUseScan || !effectiveActive}` + Tooltip hint.

## Tradeoff cân nhắc
- **vs. rowSpan merge**: rowSpan merge cells nhìn đẹp hơn nhưng phải tách Edit ra column riêng (vì các button khác per-binding); thêm 1 column tăng width → ưu tiên conditional render simple.
- **vs. extract Edit ra toolbar Panel top**: cần move state + UX khác lạ với user → ưu tiên giữ nguyên vị trí, chỉ ẩn duplicate.

## Verify plan
- V-1: `npm run build` PASS (tsc + vite).
- V-2: Manual review file diff < 60 LOC.
- V-3: User chạy dev server xác nhận behavior trên `wallet-capsets` (source có N binding).
- V-4: Khi flip "Trạng thái table" Switch (inline, per binding) → KHÔNG ảnh hưởng binding khác (route đúng `/shadow-bindings/:id`, đã có sẵn không đổi).

## Skip
- Workspace-level "00_context, 01_requirements" thì tạo gọn (task nhỏ).
- KHÔNG `/security-agent` vì FE pure UI logic.
- KHÔNG migration / DB.

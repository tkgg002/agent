# 05_progress

> APPEND-ONLY. Không sửa entries cũ (§11 GEMINI).

---

## Entry 1 — 2026-05-29 — Workspace bootstrap + audit bug

- User báo bug FE: edit "Kích hoạt debezium Sync" 1 binding nhảy toàn bộ binding khác trên `wallet-capsets`.
- Đọc lesson `agent/memory/global/lessons.md` trước (L-2026-05-20 anti over-correct, L-2026-05-28 cleanup ≠ remove, L-2026-05-28 rename blind duplicate).
- Explore qua subagent `Explore` thoroughness=very-thorough: locate field + table + bug source.
- Field mapping confirmed: `is_active` ↔ `source_object_registry.is_active` ↔ cascade `shadow_binding.is_active`.
- Root cause: Edit Modal multi-field payload → bypass `togglesBindingOnly` → V2 endpoint cascade. KHÔNG phải FE state bug; cascade ở BE.

## Entry 2 — 2026-05-29 — User direction: simplify

- User chọn approach: move Edit ra source-level (1 lần/source), thêm 2 disable rule:
  - Trạng thái table OFF → Quét field disabled
  - Kích hoạt debezium Sync OFF → Snapshot + Manage Masters disabled
- Plan ghi vào `02_plan.md` (4 patch site, FE-only).

## Entry 3 — 2026-05-29 — Apply FE patch

- File: `cdc-cms-web/src/pages/TableRegistry.tsx`.
- Patch 1 (state ~L294): thêm `firstRowIndexBySource` useMemo dựa trên `data`.
- Patch 2 (column Thao tác ~L828): conditional render Edit button — chỉ row đầu mỗi source. Snapshot + Manage Masters: `disabled={!record.is_active}` + Tooltip hint.
- Patch 3 (AsyncRowActions ~L220): compute `effectiveActive` từ `shadow_binding_is_active` hoặc `is_active`. Quét field button `disabled={!canUseScan || !effectiveActive}` + Tooltip hint + Tag "Binding chưa active".
- LOC delta NET: ~+50 / -25.

## Entry 4 — 2026-05-29 — Build verify

- `cd cdc-cms-web && npm run build`: PASS, `built in 742ms`.
- TypeScript check (tsc -b) silent — no error.
- Output `dist/assets/TableRegistry-DlqC4fpm.js 24.38 kB gzip:8.07 kB` (size không growth bất thường).
- KHÔNG runtime test (server hoá; user verify trên dev).

## Pending (user duty)

- Manual smoke trên dev: load `/shadow`, mở Collapse `wallet-capsets`, verify:
  - Source có N binding → chỉ 1 nút "Sửa Source" (header sub-section).
  - Bật/tắt Switch "Trạng thái table" 1 binding → binding khác KHÔNG đổi.
  - "Kích hoạt debezium Sync" OFF → Snapshot + Manage Masters dimmed.
  - "Trạng thái table" OFF → Quét field dimmed.

## Entry 5 — 2026-05-29 — User feedback: "cái nút vẫn ở trong row" → refactor v2

- User reject implement v1 vì nút Edit conditional vẫn TRONG cell column "Thao tác" của 1 binding row.
- Self-critique: tao đã hiểu sai "mang ra ngoài" — v1 = render 1 lần per source nhưng vẫn nằm trong row cell. User intent rõ hơn: Edit phải OUTSIDE tbody / TR.
- Re-design v2: chia mỗi Panel (group by connector::source_db) thành N sub-section per source. Mỗi sub-section có:
  - Header div (ngoài Table) chứa: source_table name + Tag "Debezium Sync ON/OFF" + Tag binding count + **Button "Sửa Source"**.
  - Body: mini-Table chỉ chứa binding rows của source đó. Bỏ Edit khỏi cell "Thao tác".

## Entry 6 — 2026-05-29 — Apply v2 patch

- Patch 1 (state ~L294): replace `firstRowIndexBySource` useMemo bằng `groupBindingsBySource` useCallback (helper group by source_object_id, giữ thứ tự xuất hiện).
- Patch 2 (column Thao tác ~L828): bỏ `showEdit` + Edit button khỏi cell, giữ Snapshot + Manage Masters disabled rules.
- Patch 3 (Panel body Tab "Shadow Objects" ~L952): wrap each source group thành `<div>` với header div (outside `<Table>`) + mini-Table. Edit button "Sửa Source" trỏ vào `head` (record đầu của source).
- LOC delta v2 NET: +50 / -30.

## Entry 7 — 2026-05-29 — Build verify v2

- `npm run build` PASS `built in 640ms`.
- Bundle `TableRegistry-866XGZQN.js` 25.04 kB / gzip 8.25 kB (+0.66 kB so v1).
- TypeScript silent.

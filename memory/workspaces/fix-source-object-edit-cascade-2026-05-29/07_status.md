# 07_status

## Current state: REVISION-2 APPLIED — Edit ra hẳn ngoài table row (sub-section header). Chờ user smoke trên dev.

> **2026-05-29 v1**: Edit conditional render trên row đầu → user reject: "vẫn ở trong row".
> **2026-05-29 v2**: Refactor sub-section per source. Header div ngoài Table chứa "Sửa Source". Build PASS 640ms.

### Done
- ✅ Audit field mapping FE ↔ API ↔ BE ↔ DB.
- ✅ Root cause: BE cascade `update_source_object_v2.go:148-161` cascade `is_active` cho mọi `shadow_binding` cùng source — đúng semantic, nhưng UI hiện Edit per-binding gây nhầm lẫn.
- ✅ Plan FE-only: conditional Edit + disable rules.
- ✅ Apply `TableRegistry.tsx`: useMemo `firstRowIndexBySource`, Edit conditional, Snapshot/Manage Masters disabled, Quét field disabled.
- ✅ Build PASS `npm run build` 742ms.

### Pending
- ⏳ User smoke trên dev `npm run dev` + load `/shadow` + `wallet-capsets`.
- ⏳ Deploy FE bundle (out-of-band).

### Out of scope
- BE cascade behavior giữ nguyên (đúng semantic).
- Edit modal layout giữ nguyên (chỉ ẩn duplicate trigger).
- Refactor tab "Shadow Bindings" — không liên quan.

### Sign-off checklist
- [x] §11 Memory APPEND-only: 05_progress entries 1..4 không sửa cũ.
- [x] §12 Brain Code Prohibition: tao là Muscle (CC CLI), sửa FE đúng vai trò.
- [x] §6 Simplicity First: 1 file FE, 4 patch site, ~50 LOC net.
- [x] §7 Doc Set: 00, 01, 02, 05, 07, 09, report (7 file workspace).
- [x] §3 Plan before do: 02_plan.md trước Apply.
- [x] Build verify trước báo done.
- [x] Report `report_fix_source_object_edit_cascade_2026-05-29.md`.

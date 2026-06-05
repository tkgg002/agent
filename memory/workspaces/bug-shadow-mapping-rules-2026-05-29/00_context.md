# 00 — Context

> **Workspace**: bug-shadow-mapping-rules-2026-05-29
> **Project**: CDC System (GooPay 2026)
> **Owner**: Muscle (claude-opus-4-7)
> **Liên quan**: workspace sibling `bug-mapping-rules-and-snapshot-v2-2026-05-29` (Brain khởi tạo trước, có `00_context.md` + `01_requirements.md` mô tả 4 bug). Workspace này được user yêu cầu "làm tiếp" → migrate context vào đây để giữ §7 Full Doc Set (prefix bắt buộc) tại 1 chỗ.

## 4 bug user yêu cầu fix (source: sibling `00_context.md`)
1. **Mapping Rules Leak** — `/shadow/:id/mappings` của binding mới (`wallet_capsets_1`) hiển thị mapping rules của binding cũ (`wallet_capsets`).
2. **Missing Source Data Type + Status logic sai** — cột "Data Type source" trống; cột "Status" hiện trạng thái duyệt rule trộn với "In Shadow" (audit khớp shadow). Cần tách 2 khái niệm + persist `source_data_type` từ scan raw.
3. **FE: Ẩn Action Preview + Backfill** — ẩn 2 button, KHÔNG xoá code (giữ logic phục hồi).
4. **Snapshot V2 Registry Lookup Fail** — `shadow_binding_id=4 not in active registry routes for source_db=wallet-service source_collection=wallet-capsets` khi trigger snapshot binding mới.

## Phạm vi
- `cdc-cms-service` (BE handler + repo + migrations)
- `centralized-data-service` (worker registry cache + scan)
- `cdc-cms-web` (FE routing + UI display + hide buttons)

## Out of scope
- DB seed/backfill historical mapping rules có `shadow_binding_id=NULL` (legacy). Sẽ note ở `10_gap_analysis` nếu cần.
- Refactor lớn `MetadataRegistryService` (chỉ fix root cause cho 4 bug này).

## Pre-flight check (CLAUDE.md §9 Workspace-First)
- [x] Workspace tồn tại vật lý.
- [x] 00_context.md (file này).
- [ ] 01_requirements_audit.md (kế tiếp).
- [ ] 02_plan_audit.md.
- [ ] 03_implementation_audit.md (root cause evidence).
- [ ] 09_tasks_solution_audit.md.
- [x] 05_progress.md (đã có audit governance từ phiên Brain trước).

# 08_tasks_audit — Cleanup `_gpay_source_id` + `_gpay_deleted`

## Audit task list

| ID | Task | Status | Output |
|---|---|---|---|
| A-1 | Đọc lessons.md (focus anti-over-correct + verify destination + DoD destination) | ✅ DONE | Entry 1 `05_progress.md` |
| A-2 | Đọc `project_context.md`, `active_plans.md`, `tech_stack.md` | ✅ DONE | Entry 1 `05_progress.md` |
| A-3 | Khởi tạo workspace folder `cleanup-gpay-cols-2026-05-28` | ✅ DONE | `00_context.md` |
| A-4 | Grep `_gpay_source_id\|_gpay_deleted` → tổng 104 hits | ✅ DONE | `03_implementation_audit.md` |
| A-5 | Phân loại theo 4 path (A FE-shadow service / B FE-shadow handler / C Master+Sinkworker / D Test+UI) | ✅ DONE | `03_implementation_audit.md` |
| A-6 | Đọc `shadow_automator.go` CREATE TABLE schema | ✅ DONE | Verify: `id` + `source_id` + `_deleted`, KHÔNG `_gpay_*` |
| A-7 | Đọc `command_handler.go` CREATE TABLE schema (Bug #2 yesterday) | ✅ DONE | Verify: `_gpay_source_id` + `_gpay_deleted` + `_deleted` (cả 2!) |
| A-8 | Đọc `sinkworker/schema_manager.go` createShadowTable + partial INDEX | ✅ DONE | Verify: `_gpay_id` PK + `_gpay_source_id` NOT NULL + `_gpay_deleted`, INDEX `ux_<t>_source_id_active ON (_gpay_source_id) WHERE NOT _gpay_deleted` |
| A-9 | Đọc `master_ddl_generator.go` CREATE master | ✅ DONE | Verify: identical V2 schema, INDEX `ux_<t>_source_id ON (_gpay_source_id)` |
| A-10 | Đọc `sinkworker/upsert.go` ON CONFLICT pattern | ✅ DONE | Verify: `ON CONFLICT (_gpay_source_id) WHERE NOT _gpay_deleted DO UPDATE` |
| A-11 | Đọc `transmuter.go shadowBatchRow` GORM mapping | ✅ DONE | Verify: 5 column tag `_gpay_*` |
| A-12 | Đọc `schema_adapter.go` conditional `_gpay_source_id` cols/values | ✅ DONE | Verify: V2 schema branch |
| A-13 | Đọc `event_handler.go:236` tombstone INSERT | ✅ DONE | Verify: INSERT (`_gpay_source_id`, `_deleted`) — Path B variant |
| A-14 | Đọc `mapping_preview_handler.go` SELECT shadow | ✅ DONE | DRIFT confirmed: đọc `_gpay_source_id`/`_gpay_id` từ shadow |
| A-15 | Semantic mapping `_gpay_source_id` ↔ `source_id` + `_gpay_deleted` ↔ `_deleted` | ✅ DONE | `03_implementation_audit.md` table |
| A-16 | Draft 3 OPTIONS (Conservative / Mid / Full) + risk profile | ✅ DONE | `09_tasks_solution_cleanup.md` |
| A-17 | Recommend option dựa lesson "anti over-correct" | ✅ DONE | `04_decisions.md` D-3 |
| A-18 | Viết `report_audit_*.md` | ✅ DONE | `report_audit_cleanup_gpay_cols_2026-05-28.md` |
| A-19 | Tạo đủ doc set 00..10 | ✅ DONE | 12 file |
| A-20 | APPEND `05_progress.md` Entry 1..5 | ✅ DONE | `05_progress.md` |

## Pending (muscle phase — chờ user verb)

| ID | Task | Owner | Trigger |
|---|---|---|---|
| M-1 | Apply Option [A/B/C] patch | Muscle | User pick A/B/C |
| M-2 | Build 3 service | Muscle | After M-1 |
| M-3 | Test handler ± service ± sinkworker | Muscle | After M-2 |
| M-4 | Destination verify (`\d shadow.<table>`) | Muscle | After M-3 |
| M-5 | Viết `report_fix_*.md` + APPEND Entry 6+ | Muscle | After M-4 |
| S-1 | `/security-agent` gate | Muscle | Optional, sau M-5 |

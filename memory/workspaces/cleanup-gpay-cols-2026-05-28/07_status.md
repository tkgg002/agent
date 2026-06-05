# 07_status — Cleanup `_gpay_source_id` + `_gpay_deleted`

## Current state: HOTFIX 42701 APPLIED — chờ user duyệt deploy + ops apply migration

> **2026-05-28 REVISION-1**: User re-direct → RENAME refactor (not REMOVE). Option A/B/C cũ bị retire. Single unified RENAME plan trong `09_tasks_solution_cleanup.md`.
> **2026-05-28 APPLY-1**: 18 file touched (16 source + 1 ops + 1 migration). Build + test + zero-residue ALL PASS. Report tại `report_fix_cleanup_gpay_cols_2026-05-28.md`.
> **2026-05-28 HOTFIX-2 (PROD ERROR 42701)**: User báo log production fail `source_id specified more than once`. Rename blanket sáng nay đã trigger duplicate column ở Path A shadow (pkField + metadata cùng emit `source_id`). 3 file code fix + 1 test reproducer. Report tại `report_hotfix_42701_duplicate_source_id_2026-05-28.md`. Build + test + regression test ALL PASS.

### Done (Brain phase)
- ✅ Workspace bootstrap per §7 GEMINI (doc set 00..10 + report).
- ✅ Lesson cross-check: 2026-05-20 anti over-correct, 2026-05-20 verify destination, 2026-05-26 DoD destination.
- ✅ Inventory 104 references theo 4 path (`03_implementation_audit.md`).
- ✅ Drift detection: `mapping_preview_handler.go` đọc `_gpay_source_id` từ Path A shadow nhưng `shadow_automator.go` không tạo.
- ✅ Semantic mapping: `_gpay_source_id` ≈ `source_id` (TEXT vs VARCHAR(200)); `_gpay_deleted` ≈ `_deleted` (both BOOLEAN DEFAULT FALSE).
- ✅ ~~3 OPTION với code demo~~ retired sau user re-direct.
- ✅ Single unified RENAME plan: 16 file + 1 migration SQL (`09_tasks_solution_cleanup.md` rewritten).
- ✅ Lesson mới identified: "Cleanup ≠ Remove" — sẽ promote sau apply.

### Done (Muscle phase)
- ✅ M-1: Apply RENAME — 16 file source + 1 ops + 1 migration SQL.
- ✅ M-2: Build verify 3 service — go build cả 2 PASS, npm build FE `built in 477ms`.
- ✅ M-3: Test verify — 10+ package PASS, zero fail.
- ✅ M-4: Zero-residue grep PASS (0 hit `_gpay_source_id\|_gpay_deleted\|GpayDeleted`).
- ✅ M-5: `report_fix_cleanup_gpay_cols_2026-05-28.md` + APPEND `05_progress.md` Entry 8..12.

### Pending (ops + brain duty)
- ⏳ OPS-1: Apply `migration_rename_gpay_cols.sql` trên PG cluster centralized-data-service (drain sinkworker trước).
- ⏳ OPS-2: Restart workers → self-heal partial UNIQUE INDEX qua `CREATE ... IF NOT EXISTS`.
- ⏳ OPS-3: Destination verify trên DB thật sau migration (`\d shadow.<t>` + post-apply grep).
- ✅ BRAIN-1: APPENDED lesson `L-2026-05-28-cleanup-is-not-remove` vào `agent/memory/global/lessons.md`.
- ⏳ S-1: `/security-agent` gate (optional, §8 GEMINI).

### Future (out-of-scope hôm nay)
- **Option C (Master refactor)**: yêu cầu workspace riêng `refactor-master-naming-2026-06-XX` khi cần đồng bộ naming convention master + sinkworker + transmuter. Bao gồm data migration + index rebuild + production coordination.
- **Path A drift fix**: `mapping_preview_handler.go` đọc `_gpay_source_id`/`_gpay_id` từ shadow tables Path A vốn không có. Track ở `10_gap_analysis.md`.

## Sign-off checklist (§14 Pre-flight)
- [x] §11 Memory Protection: `05_progress.md` APPEND-only — verified Entry 1..5 chronological.
- [x] §12 Brain Code Prohibition: KHÔNG sửa source code phase audit — verified (0 file `*.go`/`*.ts`/`*.tsx`/`*.sql` touched).
- [x] §7 Full Doc Set: 00..10 + report — created (12 file).
- [x] §6 Simplicity First: Option A minimal 3 patch site, không over-engineer master path.
- [x] §13 Lesson Writing: D-7 reference 3 lesson global pattern.
- [x] User constraint "đọc lesson trước" — Entry 1.
- [x] User constraint "không cheat DB / không đổi config" — respected (chỉ document; chưa apply).
- [x] User constraint "plan rõ ràng + code demo chi tiết" — `09_tasks_solution_cleanup.md` có before/after từng patch site.
- [x] User constraint "report dựa trên kết quả tính toán thực tế" — file/line cụ thể có cross-check evidence trong `03_implementation_audit.md`.
- [x] User constraint "luôn có report_*.md" — `report_audit_cleanup_gpay_cols_2026-05-28.md`.
- [x] Build verify Muscle phase — go build silent PASS x2, FE `built in 477ms`.
- [x] Test verify Muscle phase — 10+ package PASS, zero fail.
- [x] Zero-residue grep — 0 hit sau khi fix smoke script ops residue.
- [x] §11 APPEND-only progress — Entry 8..12 append, không sửa cũ.
- [x] Report fix — `report_fix_cleanup_gpay_cols_2026-05-28.md` tạo mới.

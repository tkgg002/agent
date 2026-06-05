# 04_decisions — Cleanup `_gpay_source_id` + `_gpay_deleted`

## D-1 — Brain phase only (no source code change) until user verb
- Lý do: §12 GEMINI Brain Code Prohibition + user constraint "plan rõ ràng + code demo chi tiết" trước.
- Phase audit chỉ document; mọi patch nằm trong `09_tasks_solution_cleanup.md` dưới dạng before/after.

## D-2 — Phân loại 3 OPTION thay vì 1 plan duy nhất
- Lý do: User claim "đã có source_id và _deleted" CHỈ đúng với Path A (shadow_automator). Path C (master + sinkworker + transmuter) **KHÔNG** có `source_id`/`_deleted` — cleanup ở đây cần data migration + index rebuild + production coordination.
- Lesson cross-check: 2026-05-20 "Bump dependency anti-pattern" → cấm over-correct theo feedback trước khi verify từng layer. Vì vậy phải trình ≥3 option scope khác nhau.

## D-3 — Option recommend = A (Conservative)
- Lý do:
  - Path B (`command_handler.go` Bug #2 fix yesterday) là nguồn rác đúng nghĩa: trùng `_gpay_source_id` + `source_id` + `_gpay_deleted` + `_deleted` trong CÙNG 1 table.
  - Rollback Bug #2 = revert 3 patch site (~+10 LOC removed) → minimal blast radius.
  - Path A drift (`mapping_preview_handler.go`) đã tồn tại trước Bug #2; KHÔNG phải rác do user trigger lần này → tách riêng làm gap item.
  - Path C V2 master architecture là design có chủ đích (partial UNIQUE INDEX tombstone-aware) → đụng vào không phải "cleanup" mà là "refactor" → ngoài scope user verb "bỏ rác".
- Reversibility: HIGH (revert Bug #2 patch là 1 commit).
- Production impact: LOW (Bug #2 chưa run production; chỉ thêm hôm qua 17:30 ICT).

## D-4 — Option B (Mid-Scope) chỉ làm khi user clarify "FE shadow phải dùng `source_id`"
- Điều kiện trigger: user explicit confirm muốn FE shadow path duy nhất dùng `source_id`/`_deleted`.
- Scope: rollback Path B + sửa `mapping_preview_handler.go` đọc `source_id` thay vì `_gpay_source_id` + sửa `event_handler.go:236` tombstone dùng `source_id`.
- Verify destination: PG shadow table (`shadow.export_jobs_2`) sau `/shadow` create flow → expect chỉ có `source_id`/`_deleted`, không có `_gpay_*`.
- Risk: Med — phải verify chain `event_handler.processInsert` + `processDelete` + `mapping_preview_handler` đồng bộ.

## D-5 — Option C (Full Master Refactor) bị từ chối ở phase này
- Lý do:
  - Yêu cầu data migration trên prod master tables (đã có data từ snapshot trước).
  - Phải DROP + recreate partial UNIQUE INDEX `ux_<t>_source_id_active ON (_gpay_source_id) WHERE NOT _gpay_deleted`.
  - Đổi `transmuter.shadowBatchRow` GORM mapping → cross-cutting CDC pipeline core.
  - Lesson 2026-05-20 "Verify ở destination" + 2026-05-20 "anti over-correct" → KHÔNG nên làm trong cùng PR với cleanup nhỏ.
- Future: tạo workspace mới `refactor-master-naming-2026-06-XX` khi cần đồng bộ naming convention toàn bộ.

## D-6 — Verify plan = build 3 service + handler test + sanity FE build
- Lý do: §3 GEMINI "Verification Before Done". Cùng pattern như workspace `snapshot-zero-records-2026-05-27` (đã PASS).
- Commands sẽ ghi trong `06_validation.md`.

## D-7 — Lesson áp dụng
- Lesson 2026-05-20 "Bump dependency anti-pattern" (line 3433-3450) → từ chối over-correct.
- Lesson 2026-05-20 "Verify ở destination" (line 3415-3429) → mỗi option có verify ở PG shadow + master.
- Lesson 2026-05-26 "Define DoD at the destination" (line 3417-3421) → DoD = grep `_gpay_*` ở target path SAU patch phải khớp expectation (option A: chỉ Path B clean, Path A+C giữ nguyên).

---

## D-8 — REVISED 2026-05-28: User re-direct → RENAME (not REMOVE)
- User clarify: cleanup intent = RENAME `_gpay_source_id` → `source_id`, `_gpay_deleted` → `_deleted` ở MỌI path. Không phải delete logic.
- Conceptually: cùng 1 column → unify naming. V2 master partial UNIQUE INDEX semantic giữ nguyên, chỉ đổi cột target.
- Option A/B/C cũ **bị retire**. Thay bằng **single unified RENAME plan** trong `09_tasks_solution_cleanup.md` (rewritten).
- Self-critique: lesson "anti over-correct" tôi áp dụng sai chiều. Lesson mới (promote sau): "Cleanup ≠ Remove. Khi user nói 2 field 'trùng' = RENAME/MERGE, KHÔNG phải DELETE. Verify intent NGỮ NGHĨA trước khi build option matrix."
- Scope mới (xem `09_tasks_solution_cleanup.md`):
  - 16 file code (3 service).
  - 1 migration SQL script (rename column + rebuild partial INDEX).
  - LOC delta NET: ~0 (rename = đổi ký tự, không thêm bớt logic).
- Out-of-scope: `_gpay_id` PK column (user chỉ nêu `_gpay_source_id` + `_gpay_deleted`).
- Risk re-assess:
  - Code rename: LOW (mechanical, type-safe).
  - DB migration: MED (cần ALTER COLUMN RENAME + DROP+CREATE partial INDEX trên prod tables đã có data).
  - Reversibility: MED-HIGH (1 git revert + reverse migration SQL).
- Verify destination: SAU apply, `grep -rn "_gpay_source_id\|_gpay_deleted"` toàn repo = **0 hits** (trừ migration SQL file ghi rename command).

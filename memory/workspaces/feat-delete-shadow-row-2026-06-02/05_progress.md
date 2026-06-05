# 05_progress — feat-delete-shadow-row (APPEND ONLY)

## [2026-06-02 — Muscle:claude-opus-4-7] Init
- Đọc `agent/GEMINI.md` + `agent/memory/global/lessons.md` (lines 1-1640 + grep delete/destructive).
- Đọc TableRegistry.tsx (page /shadow), source_objects_handler.go, router.go.
- Phát hiện chưa có DELETE endpoint cho source-object / shadow-binding.
- Tạo workspace `feat-delete-shadow-row-2026-06-02/00_context.md`.
- KHÔNG sửa code. Đang chờ user chốt scope: (1) target row type, (2) mức A/B/C, (3) có drop schema vật lý không.

## [2026-06-02 — Muscle:claude-opus-4-7] Abandoned vụ schema rename audit
- User stop mid-flight ("ơ, đừng làm gì nhé, code đang chạy, làm nữa sai đó").
- Task #18 set status=completed (audit doc-only, không thay đổi code).
- Không tạo lesson mới vì rule "khi bị sửa MID-SESSION → ghi lesson" — lesson đã có sẵn (Lesson `Brain tự ý thực thi Code thay vì Delegate` + `Không assume khi user hỏi hypothetical`). Đã apply đúng (audit read-only, không touch source).

## [2026-06-02 — Muscle:claude-opus-4-7] Full Doc Set (plan-only)
- User chốt: xoá row /shadow mức C (full hard delete + DROP shadow table), plan-only ("nhớ chỉ plan, đừng làm nhé em").
- Audit FK: source_object_registry ← shadow_binding/master_binding/mapping_rule_v2/sync_runtime_state ON DELETE CASCADE. Legacy non-FK: cdc_reconciliation_report, cdc_worker_schedule, cdc_mapping_rules (manual cleanup).
- Đọc: TableRegistry.tsx (FE), source_objects_handler.go + system_connector.go (BE pattern), router.go (mount pattern dòng 211-215), update_source_object_v2.go (CQRS pattern), server.go (RegisterSync wiring), ConfirmDestructiveModal.tsx (đã có sẵn reason≥10 + danger).
- Đã tạo §7 Full Doc Set:
  - `01_requirements.md` — scope IN/OUT, A1-A7, N1-N2, R1-R4.
  - `02_plan.md` — 5 phase roadmap (Audit → BE Command → BE HTTP+Route → FE → Verify), 7 task T1-T7, 5 quyết định D1-D5.
  - `03_implementation.md` — code demo đầy đủ cho 6 file (1 NEW + 5 EDIT), tổng ~+308 LOC.
  - `08_tasks.md` — checklist DoD từng T1-T7, anti-tasks, escalation.
  - `09_tasks_solution.md` — edge cases S1-S5 (BE/FE), smoke TC1-TC8, rollback, verification commands.
- KHÔNG đụng source code. Đợi user duyệt trước khi Muscle execute.

# 01 — Requirements: Skill `/diff-review` (Multi-Project, Fintech-Aware Code Review)

- **tier:** T2
- **suffix:** diff_review_skill
- **Ngày:** 2026-11-17
- **Agent:** Claude Fable 5.1 (OpenCode)
- **Trạng thái APPROVE:** User đã `APPROVE` thiết kế 3 lớp luật (L0 Engine / L1 Base / L2 Pack / L3 Profile) trên chat ngày 2026-11-17.

## 1. Bối cảnh
User vận hành nhiều dự án fintech (không chỉ `data-hub`). Cần một flow review diff code dùng chung, kiểm tra 4 lens: Security, Logic, Design Pattern, Contract. Flow phải:
- Agnostic với dự án (ADR-07, `GEMINI.md:29`): core không hardcode tên service/từ khoá dự án.
- Propose-Only (`GEMINI.md:19`, `:93`): chỉ đọc & báo cáo, không sửa code.
- Cắm được vào S5 hiện có (`GEMINI.md:76`) hoặc chạy độc lập dạng Audit.
- Không gây "nửa ngày" cho 1 lần review: script 0-token làm scope, model chỉ đọc hunk + callers, giới hạn vòng.

## 2. Phạm vi (In-Scope)
| ID | Hạng mục | Vị trí |
| :--- | :--- | :--- |
| R-1 | `SKILL.md` của skill `diff-review` (pipeline R0→R6, cách nạp luật 3 lớp, output) | `agent/skills/diff-review/SKILL.md` |
| R-2 | Base Rules (L1) 4 lens | `agent/rules/review/base/{security,logic,design,contract}.md` |
| R-3 | Fintech Pack (L2) | `agent/rules/review/packs/fintech.md` |
| R-4 | Script R0 scope (0 LLM token): diff → layer/group/chokepoint/god-file/money-path flags, đọc từ `.agent-project.json` khoá `review` | `agent/tooling/diff_review/scope.py` |
| R-5 | Template báo cáo | `agent/tooling/diff_review/report_template.md` |
| R-6 | Profile mẫu L3 cho `data-hub` (trích từ `system_overview_and_design_pattern.md`, ≤8KB) + template profile trống | `agent/memory/global/project/data-hub/review_profile.md`, `agent/rules/review/profile_template.md` |
| R-7 | Hướng dẫn đăng ký skill (`.agents/skills.json`, `OPERATOR_MAP.md`) | Ghi trong `09_tasks_solution` — User tự áp vì agent không đọc được file |

## 3. Ngoài phạm vi (Out-of-Scope)
- Sửa `laya_gate.py`, `evidence_run.py`, `verify_governance.py`, hook `agy_guard.py`.
- Viết profile cho các dự án fintech khác (chờ User cung cấp danh sách).
- Chốt chặn hook "không có review_report → không DONE" (P5, làm sau khi pilot).

## 4. Ràng buộc Thực thi (Technical Constraints — từ GEMINI.md & lessons)
- TC-1: Core agnostic — mọi từ khoá nhận diện (money-path, layer, chokepoint) phải nằm trong manifest/pack/profile, **không** trong `scope.py`.
- TC-2: Memory file protection (`GEMINI.md:110`): không overwrite file đã tồn tại trong `agent/memory/`; `05_progress` append-only.
- TC-3: No Shadow Files: thiết kế lưu trong `09_tasks_solution_diff_review_skill.md`.
- TC-4: Không hardcode đường dẫn `/Users/...` vào rules/pack (dùng được cho mọi máy); chỉ profile/manifest mới chứa path tương đối trong repo.
- TC-5: `scope.py` chỉ dùng stdlib Python 3 + `git`, không dependency ngoài.
- TC-6: Finding không có `REF: path:line` → không được tính.

## 5. Definition of Done
- [ ] 7 hạng mục R-1..R-7 tồn tại vật lý.
- [ ] `python3 -m py_compile tooling/diff_review/scope.py` OK.
- [ ] `scope.py` chạy được trên một git repo mẫu (temp) với manifest có khoá `review`, xuất JSON đủ các cờ: `layers`, `chokepoints_touched`, `god_files`, `money_path`, `files`.
- [ ] `scope.py` chạy được **không có** manifest (chỉ L1) không crash.
- [ ] `SKILL.md` mô tả đủ: input, pipeline, nạp luật, severity, verdict, giới hạn vòng, sub-agent song song, output path.
- [ ] `05_progress` có START/DONE; `audit_report_diff_review_skill.md` tồn tại (T2).

## 6. Ghi chú môi trường (VERIFIED)
- Tại thời điểm làm, agent **không đọc được** `agent/skills/*`, `agent/tooling/*`, `agent/.agents/*`, `agent/memory/global/*` (macOS `Operation not permitted`) nhưng **tạo & ghi file mới được**. Vì vậy:
  - Không đối chiếu được cú pháp CLI thật của `laya_gate.py review`; SKILL.md ghi tham chiếu theo `GEMINI.md:80-83` và đánh dấu `[VERIFY]`.
  - Không đăng ký được vào `.agents/skills.json` / `OPERATOR_MAP.md`; để User áp theo hướng dẫn R-7.

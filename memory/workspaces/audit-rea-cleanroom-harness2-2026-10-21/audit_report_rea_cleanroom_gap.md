# Audit Report `rea_cleanroom_gap` (S5 — phản biện từng dòng)

## 1. Line-by-line `git diff` critique (từng task, từng file, từng dòng)
Lệnh: `git diff -U1 -- hooks tooling OPERATOR_MAP.md` (8 files, +342/−64).
- `hooks/agy/agy_guard.py`
  - `APPROVE` neo `^` + `re.M` + tiền tố `ok|ừ|được`: chặn "đã được Jev duyệt" (test + mutant M3). Rủi ro: User gõ "Duyệt nhé" ở đầu dòng vẫn mở — đúng ý định.
  - `CLEAN_ROOM_START`: chỉ nguyên dòng `BẮT ĐẦU`/`APPROVE`. "chưa BẮT ĐẦU nhé" không mở (test).
  - `SPEC_RULE_RE`: bug `\s*` nuốt newline phát hiện bởi test → `[ \t]*`.
  - `READ_ARG_KEYS` + `contaminated_clean_room_reads`: so khớp substring (siêu tập logic cũ endswith); thêm `run_command`, `read_url_content` (M2).
  - `leaked_subagent_prompts`: path hoặc dòng ≥40 ký tự của file cấm; AF-2 resolve theo `Cwd` (M8). Hạn chế: code dán đã chỉnh sửa nhẹ (đổi tên biến) không bị bắt.
  - `user_approved`, `spec_rule_issues`, `load_tripwires`, `match_tripwires`, `tripwire_violations`: hàm thuần, không I/O ngoài đọc JSON. AF-1: tool ghi không là bằng chứng (M7).
  - H1: thêm tham số `mem_root` (mặc định `MEM`) — caller `dispatch` không đổi chữ ký gọi ⇒ tương thích.
  - H4t: chạy trước H4a, chỉ với `13_analysis|09_tasks_solution|audit_report` trong memory.
  - H4b: `user_approved(..., clean_room=is_cr)`.
  - H7: cảnh báo G-04 khi clean-room có `forbidden`.
- `tooling/build_lessons_index.py`: `TRIP_FIELD_RE`, `parse_tripwire` (thiếu trường/regex lỗi/ID trùng → RuntimeError, build fail-fast), ghi `tripwires.json`, stats. INDEX.md không đổi định dạng.
- `tooling/verify_governance.py`: `spec_rule_issues` nhân bản logic agy_guard (theo pattern file đã nhân bản `CLEAN_ROOM_*`). Rủi ro trôi lệch 2 bản — đã ghi chú "đồng bộ với agy_guard".
- `tooling/orchestrator/harness2_engine.py`: xoá proposer/scorer one-liner và base tree giả; thêm `--agent-root/--proposer-cmd/--rollout-cmd`; thiếu → exit 2 (M5).
- `OPERATOR_MAP.md`: thêm dòng `reverse-engineer-anything` (AF-3), giới hạn phạm vi theo description của REA.
- Tests: fixture test 25 và test verify_governance đổi sang định dạng `R<n> — REF:` — thay đổi hành vi có chủ đích của G-03, không phải nới test.

## 2. Chống suy diễn Smoke Test / Mockup / Hallucination — trace runtime thật
- Không có mock trong code production; stub chỉ còn trong `tests/test_harness2_engine.py`.
- Runtime: H1 gọi qua stdin như IDE; replay `transcript_full.jsonl` thật của phiên → gate chặn đúng lượt 1. Không dùng kết quả unit test để khẳng định IDE thật đã chạy H4t/H7 — mục chưa trace liệt kê ở `06_validation` §5.

## 3. Concurrency / Timeout-Heartbeat / Data continuity
- Hook là process ngắn, stateless ngoài file `.boot` có sẵn; không thêm lock/worker. `tripwires.json` ghi bởi `build_lessons_index.py` (ghi đè toàn file) — đọc đồng thời lúc đang ghi có thể gặp JSON dở ⇒ `load_tripwires` bắt `ValueError` trả `[]` (fail-open cho tripwire, không crash hook). Ghi nhận: fail-open là đánh đổi có chủ ý.
- **Phiên song song thật:** trong lúc S4, một phiên khác chèn 3 lesson vào `lessons.md` (2014→2035 dòng) và `rea-agents setup` tạo symlink skill. Không có xung đột source (`git diff --stat` chỉ gồm file của phiên này). Lesson do phiên kia ghi không có tripwire.

## 4. Kiểm tra suy diễn & báo cáo láo (trung thực)
- Lượt 1 tôi kết luận về harness2 chỉ từ README — đã đính chính (`13_analysis` §6) và ghi lesson.
- Laya: DG1/DG2/DG3 nhiều claim `UNVALIDATED`/`FLAG`; FLAG đã resolve bằng REF + mutant; kết luận dựa trên code/runtime, không dựa trên Laya.
- Giới hạn còn lại (không che giấu): Require không gắn đúng repo; `grep -v <path cấm>` bị tính là đọc; 260/262 lesson cũ chưa có tripwire.

## 5. Self-Improvement Loop (S5 → S3 → S4 → S5)
- Phát hiện AF-1 (P0, tự thoả bằng chính lần ghi bị chặn), AF-2 (P1), AF-3 (P1) → cập nhật `09_tasks_solution` §8 + `08_tasks` T13/T14 → DG1/DG2 → sửa → test hồi quy → mutation 8/8 → DG3.
- Lessons: lesson README-only + lesson tái phạm đã được gắn `ID/Trigger/Check/Require` (amendment append-only, dòng sau thắng).

## 6. `/laya_router` DG3
- `laya_gate.py run --stage dg3` sau lần sửa code cuối: `06_laya_summary_dg3_20261021T095934.json` → `gate=PASS` (A-RC, A-PATTERN ACCEPT; A-FIX, A-IMPACT FLAG → `resolve` kèm REF `agy_guard.py` + mutant M7/M8 KILLED).
- DG1 `06_laya_summary_dg1_20261021T095932.json` PASS (sau resolve), DG2 `06_laya_summary_dg2_20261021T095933.json` PASS (UNVALIDATED).

## Vòng 2 — H7 Stop-Audit (10:02–10:07): phản biện lại chính bằng chứng "replay chặn đúng lượt 1"
1. **Diff critique:** nghi vấn replay là ảo giác nếu `calls()` không parse được tool call → kiểm: 300 step, **150 tool call parse được**, cả transcript → `[]` ⇒ replay hợp lệ. Nhưng truy *cái gì* thoả `LS-20261021-prior-workspace-lookup`:
   - **AF-4 [P1]:** step 102 — `run_command` chứa `/memory/workspaces/$WS/09_tasks_solution_rea_cleanroom_gap.md` = tài liệu **của chính task**, biến shell chưa expand. Fix: `exclude_ws` + bỏ match chứa `$` (M9 KILLED).
   - **AF-5 [P1]:** sau AF-4 vẫn thoả ở step 244 bởi chuỗi giả `/x/memory/workspaces/w/13_analysis_s.md` trong **script repro của chính tôi**. Fix: token đường dẫn tuyệt đối phải tồn tại trên đĩa (M10 KILLED). Các test cũ dùng đường dẫn giả làm bằng chứng dương đã đổi sang file thật trong tmp.
2. **Kết quả replay sau fix:** turn1 → 2 vi phạm; toàn phiên → còn `LS-20261021-prior-workspace-lookup` ⇒ **đúng sự thật**: cả phiên tôi chưa từng mở tài liệu kết luận của workspace harness2 cũ. Đã làm đúng `Check`: `view_file .../feature-harness2-recursive-integration-2026-10-20/audit_report_harness2_recursive_integration.md`.
3. **Phát hiện từ tài liệu cũ (trung thực):** audit cũ dòng 21 ghi AF-07 "ĐÃ VÁ — chạy trace subprocess thật (`winning_existing_check_py: VALUE = 2`)" — đó chính là proposer/scorer giả bị gỡ ở G-05. Báo cáo cũ đã dùng smoke-stub để khẳng định runtime thật.
4. **Kiểm định:** hooks 49/49, tooling 37/37, mutation **10/10 KILLED** (`06_mutation_summary_rea_cleanroom_gap_20261021T100546.json`), DG1/DG2/DG3 `..._20261021T100604/05/06.json` PASS sau resolve (S-RC/S-FIX, D-BEHAV/D-EDGE, A-RC/A-FIX FLAG → resolve REF + M9/M10).
5. **Runtime qua đúng lệnh trong `.agents/hooks.json`** (`python3 hooks/agy/agy_guard.py pre-write`, stdin payload, transcript thật): 31 step đầu → `deny` (`LS-20261021-prior-workspace-lookup`…); toàn transcript hiện tại → `ask`. Việc IDE có thực sự gọi hook ở mỗi lần ghi thì không tự chứng minh được từ trong phiên (lần ghi thành công không phân biệt "hook pass" với "hook không chạy") — cần User xác nhận qua log hook của IDE nếu muốn chắc chắn.
6. **Giới hạn còn lại:** Require không gắn đúng repo của Trigger; `grep -v <path cấm>` bị tính là đọc; token có khoảng trắng trong đường dẫn sẽ bị cắt (đường dẫn có space không được công nhận — thiên về chặt).

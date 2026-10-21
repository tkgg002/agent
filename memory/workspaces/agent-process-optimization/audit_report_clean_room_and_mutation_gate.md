# Final Adversarial Audit Report: Tích hợp Clean-Room Rebuild Protocol & 3-Mutant Test Verification Gate (`clean_room_and_mutation_gate`)

- **Workspace:** `agent-process-optimization`
- **Suffix:** `clean_room_and_mutation_gate`
- **Tier:** `T2` (`mode: clean-room`)
- **Ngày hoàn tất:** `2026-10-20`

---

## 1. Line-by-Line Diff & Task Critique (Phản biện từng dòng update / `git diff` & từng task)

| Task / File:Line | Thay đổi thực tế | Phản biện (Adversarial Question: Cách phá / Edge case) | Bằng chứng Code / Runtime thật | Kết luận |
| :--- | :--- | :--- | :--- | :---: |
| **`T8.1`**<br>[`GEMINI.md:71, 73, 94-99`](file:///Users/trainguyen/Documents/work/agent/GEMINI.md#L71-L99)<br>`skills/{brain-delegate,muscle-execute,qa-agent,workspace-docs}/SKILL.md` | Bổ sung quy tắc **Clean-Room Rebuild (`mode: clean-room` & `clean_room_forbidden_sources`)** tại S3/S4/Rule #13 và **3-Mutant Test Verification Gate (`tooling/mutation_gate.py`)** tại S4/Rule #14 Gate G3. | *Liệu việc thêm quy định vào `GEMINI.md` có làm vượt ngưỡng dung lượng context hay làm sai lệch số hiệu 22 Rules không?* | `GEMINI.md` tăng từ `129` lên `131` dòng (`16,294` bytes `< 18,000` bytes), giữ nguyên 100% số hiệu Rule #0–#22. | **PASS** |
| **`T8.2`**<br>[`tooling/mutation_gate.py:34-43, 160-169`](file:///Users/trainguyen/Documents/work/agent/tooling/mutation_gate.py#L34-L169) | Hàm `_invalidate_pyc(fpath)`, `resolve_target_file`, `auto_detect_component`, và kiểm tra `ast.parse(mutated_text)` trước khi ghi xuống đĩa. | *Điều gì xảy ra nếu mutant làm gãy cú pháp Python (`SyntaxError`), hoặc nếu `target_content` và `mutant_content` có cùng độ dài byte (`return a + b` vs `return a - b`) chạy trong cùng 1 giây khiến Python nạp `.pyc` cũ từ `__pycache__`?* | 1. Dòng `160-169` gọi `ast.parse(mutated_text)` trong RAM và trả mã `3 INVALID_INPUT` ngay nếu có `SyntaxError` (kiểm chứng bởi `test_03_syntax_error_mutant_rejected_by_ast_parse`).<br>2. Dòng `74-75` đặt `PYTHONDONTWRITEBYTECODE=1` và dòng `230, 234` gọi `_invalidate_pyc(fpath)` xoá `__pycache__/<stem>.*.pyc`, triệt tiêu hoàn toàn lỗi nạp `.pyc` cũ trong cùng 1 giây. | **PASS** |
| **`T8.2`**<br>[`tooling/mutation_gate.py:212-249`](file:///Users/trainguyen/Documents/work/agent/tooling/mutation_gate.py#L212-L249) | Vòng lặp tiêm mutant bọc `signal.signal(signal.SIGTERM, _sigterm_to_exit)` và `try...finally` khôi phục `orig_bytes` + kiểm tra `SHA-256`, phân loại `TIMEOUT` / `KILLED` / `SURVIVED`. | *Nếu mutant gây vòng lặp vô hạn (`timed_out == True`) hoặc tiến trình bị gửi `SIGTERM` giữa chừng thì có làm hỏng file gốc hoặc báo `KILLED` giả không?* | 1. Dòng `212-217` chuyển `SIGTERM` thành `SystemExit(143)` để `finally:` (dòng `232-235`) luôn ghi trả `orig_bytes` và kiểm tra `restored_sha == orig_sha`.<br>2. Dòng `241-243` phân loại `res["timed_out"] == True` thành `"TIMEOUT"` (`timeout_ids.append(mid)`) và đánh trượt cổng (`gate = "FAIL"`, kiểm chứng bởi `test_04_timeout_mutant_classified_as_timeout_and_fails_gate`). | **PASS** |
| **`T8.3`**<br>[`hooks/agy/agy_guard.py:243-327, 507-528`](file:///Users/trainguyen/Documents/work/agent/hooks/agy/agy_guard.py#L507-L528) (`H4a`) | Kiểm tra mục `Clean-Room Approved Spec` khi `01_requirements_<suffix>.md` khai báo `mode: clean-room`. | *Nếu Agent dùng `replace_file_content` để sửa vài dòng ở cuối `09_tasks_solution_<suffix>.md`, `body` không chứa tiêu đề `Clean-Room Approved Spec` thì có bị chặn oan không (`F1`)?* | Dòng `512-518` đọc `existing_sol = sol_path.read_text(...)` và ghép `existing_sol.replace(TargetContent, ReplacementContent, 1)` trước khi kiểm tra `CLEAN_ROOM_SPEC_RE` (kiểm chứng bởi `test_25_h4a_clean_room_requires_approved_spec_and_supports_partial_replace`). | **PASS** |
| **`T8.3`**<br>[`hooks/agy/agy_guard.py:557-570`](file:///Users/trainguyen/Documents/work/agent/hooks/agy/agy_guard.py#L557-L570) (`H4b`) | Cưỡng chế cách ly phòng sạch (`Clean-Room Isolation`) qua `clean_room_forbidden_sources` và `contaminated_clean_room_reads(st, forbidden)`. | *Làm sao ngăn phiên Brain đã đọc code gốc (`view_file`, `grep_search`, `find_by_name`) tự ý viết code mới thay vì gọi `invoke_subagent` (`F2`)?* | Dòng `275-294` quét toàn bộ `calls(st)` trong `transcript.jsonl`; nếu phát hiện đã đọc file thuộc `clean_room_forbidden_sources`, dòng `561-570` trả về `decision: "deny"` (kiểm chứng bởi `test_26_h4b_clean_room_blocks_contaminated_session_from_writing_source` và bị bắt sống bởi Mutant `M2`). | **PASS** |
| **`T8.3`**<br>[`hooks/agy/agy_guard.py:701-707`](file:///Users/trainguyen/Documents/work/agent/hooks/agy/agy_guard.py#L701-L707) (`H7`) & [`tooling/verify_governance.py:197-255`](file:///Users/trainguyen/Documents/work/agent/tooling/verify_governance.py#L197-L255) | Kiểm tra `mutation_gate_status == "PASS"` (`killed_count >= 3`, `all_restored_sha256 == True`) và được chạy sau lần sửa code cuối (`last`). | *Liệu việc bổ sung kiểm tra này có làm gãy 24 unit test cũ không? Và nếu Agent sửa code thêm sau khi chạy `mutation_gate.py` thì H7 có phát hiện được không (`F6`)?* | 1. Giữ nguyên 6 phần tử `AUDIT_REQUIRED_PATTERNS`, chỉ kích hoạt khi `is_cr` hoặc có file `06_mutation_summary_*.json` $\rightarrow$ toàn bộ test cũ + mới PASS.<br>2. Kiểm tra `touched(st, r"(mutation_gate\.py|06_mutation_summary_)", last)` sau chỉ số `last = src[-1][0]` (kiểm chứng bởi `test_27` và `test_28`). | **PASS** |
| **`T8.5` (`G-01`)**<br>[`hooks/agy/agy_guard.py:48-60, 391-397`](file:///Users/trainguyen/Documents/work/agent/hooks/agy/agy_guard.py#L48-L60) | Neo đầu dòng `APPROVE` (thêm `bắt\s*đầu`) và định nghĩa `CLEAN_ROOM_START = re.compile(r"^\s*(BẮT\s*ĐẦU|APPROVE)\s*[.!]?\s*$", re.I \| re.M)` cho chế độ `clean-room`. | *Nếu tin nhắn mô tả ban đầu của User có câu `"…từ mỗi cái spec Jev đã duyệt! … cho đến khi tôi ra lệnh 'BẮT ĐẦU'"` thì có bị mở khoá sửa code nhầm không (`G-01`)?* | `CLEAN_ROOM_START` yêu cầu toàn bộ dòng là `BẮT ĐẦU` hoặc `APPROVE`, còn `APPROVE` neo đầu dòng (`^\s*...`) nên từ `"duyệt"` hoặc `"BẮT ĐẦU"` nằm giữa câu bị từ chối 100% (kiểm chứng bởi `test_g01_approve_not_matched_mid_sentence`, `test_g01_clean_room_requires_exact_start_line` và bị bắt sống bởi Mutant `M1_G01_CLEAN_ROOM_START_BYPASS`). | **PASS** |
| **`T8.5` (`G-02`)**<br>[`hooks/agy/agy_guard.py:64-73, 327-350`](file:///Users/trainguyen/Documents/work/agent/hooks/agy/agy_guard.py#L64-L73) | Mở rộng `READ_ARG_KEYS` (`view_file`, `view_file_outline`, `view_code_item`, `grep_search`, `find_by_name`, `run_command` (`CommandLine`, `Cwd`), `read_url_content` (`Url`)) và kiểm tra cả `SYSTEM_MESSAGE` (`sender=`) trong `contaminated_clean_room_reads`. | *Nếu phiên đọc code gốc qua `run_command` (`cat`, `sed`, `git show`) hoặc `read_url_content` thì có lách được `contaminated_clean_room_reads` không (`G-02`)?* | `READ_ARG_KEYS` thu thập cả `CommandLine`, `Cwd` và `Url`; mọi lệnh `cat /repo/legacy/payment.go` đều bị phát hiện và chặn cứng tại `H4b` (kiểm chứng bởi `test_g02_run_command_and_url_reads_detected` và bị bắt sống bởi Mutant `M2_G02_RUN_COMMAND_READ_BLINDNESS`). | **PASS** |
| **`T8.5` (`G-03`)**<br>[`hooks/agy/agy_guard.py:60-63, 400-424`](file:///Users/trainguyen/Documents/work/agent/hooks/agy/agy_guard.py#L400-L424) & [`tooling/verify_governance.py:25-29, 82-104`](file:///Users/trainguyen/Documents/work/agent/tooling/verify_governance.py#L82-L104) | Hàm `spec_rule_issues(text)` kiểm tra từng rule `R<n>`/`CR-<n>` có `REF: path:line`, chặn trạng thái `[REJECTED]`/`[NO_EVIDENCE]`/`[UNVERIFIED]`/`[BLOCKED]` và chặn mọi dòng bullet lậu không có `REF:`. | *Nếu người viết chỉ đặt tiêu đề `## Clean-Room Approved Spec` rồi đưa vào rule không có `REF:` hoặc dòng bullet tự do không đánh số `R<n>` thì có lọt qua `H4a` / `verify_governance.py` không (`G-03`)?* | `spec_rule_issues` kiểm tra từng rule và từng dòng bullet `^[ \t]*[-*][ \t]+\S` trong phạm vi mục `Clean-Room Approved Spec`; nếu thiếu `REF:` hoặc có nhãn `[REJECTED]` thì từ chối ngay tại `H4a` và `S3/S6` (kiểm chứng bởi `test_g03_spec_rules_require_ref`, `test_g03_pre_write_denies_solution_with_unreferenced_rule`, `test_g03_cr_rules_rejected_status_and_unnumbered_bullets`). | **PASS** |
| **`T8.5` & `T8.6` (`G-04` & `G-06`)**<br>[`hooks/agy/agy_guard.py:352-389, 899-924, 1126`](file:///Users/trainguyen/Documents/work/agent/hooks/agy/agy_guard.py#L899-L924) & [`.agents/hooks.json:36-48`](file:///Users/trainguyen/Documents/work/agent/.agents/hooks.json#L36-L48) | Thêm `handle_pre_subagent` (`pre-subagent`, matcher `invoke_subagent\|send_message`), kiểm tra cả `SYSTEM_MESSAGE` (`sender=`) trong phiên subagent, trace runtime thật subagent `42058893-0f65-45b1-9297-83319a560210`, và cài đặt `rea@6.7.0`. | *Nếu parent dán đường dẫn hoặc dòng code gốc ($\ge 40$ ký tự) vào `invoke_subagent` / `send_message` thì có bị chặn TRƯỚC khi subagent chạy không? Và hook có chạy bên trong phiên của subagent thật không (`G-04`, `G-06`)?* | 1. `handle_pre_subagent` chặn ngay tại `PreToolUse` (`decision: "deny"`) đối với cả `invoke_subagent` và `send_message` (bị bắt sống bởi Mutant `M3_G04_PRE_SUBAGENT_LEAK_BYPASS`).<br>2. Trace runtime thật `invoke_subagent` (`conv_id = 42058893-0f65-45b1-9297-83319a560210`) chứng minh hook `agy_guard.py` **CÓ chạy bên trong phiên subagent** (tạo `/tmp/agy/42058893-0f65-45b1-9297-83319a560210.boot` và tiêm `BOOT S0` vào `step_index: 0`), đồng thời prompt giao việc nằm tại `step_index: 1` (`type: "SYSTEM_MESSAGE"`, `sender=b2813663-...`) nên được `contaminated_clean_room_reads` bảo vệ 2 lớp.<br>3. Đã cài đặt `/Users/trainguyen/.nvm/versions/node/v22.18.0/bin/rea` (`v6.7.0`, 153 CLI commands, 207 MCP tools) và chạy `rea setup --client antigravity --client claude_code --yes` (`aligned`). | **PASS** |

---

## 2. Anti-Smoke-Speculation & Real Runtime Trace Verification (Chống suy diễn Smoke Test / Mockup / Hallucination & Trace Runtime thật)

- **Kiểm tra Production Mock Pollution:** Quét toàn bộ code mới trong `tooling/mutation_gate.py`, `hooks/agy/agy_guard.py`, `tooling/verify_governance.py` xác nhận **0 hàm mock/closure giả** (`*Fn`, `sqlmock`, `fakemodel`).
- **Bằng chứng Test & Runtime thật (Bao gồm Trace Subagent thật, `rea@6.7.0` thật & 3-Mutant Verification thật trên codebase `/agent`):**
  1. **Unit & Integration Test Suite (`hooks` + `tooling`):**
     - `python3 -m unittest discover -s hooks/agy/tests -p 'test_*.py' -v` $\rightarrow$ **`Ran 52 tests in 0.318s — OK`** (`31` tests trong `test_agy_guard.py` + `21` tests trong `test_tripwire_cleanroom.py`).
     - `python3 -m unittest discover -s tooling/tests -p 'test_*.py' -v` $\rightarrow$ **`Ran 37 tests in 1.856s — OK`**.
  2. **Trace Runtime Thật Phiên Subagent (`G-04` — Giải đáp bằng thực nghiệm giả thuyết của Auditor):**
     - Gọi thật công cụ `invoke_subagent` (`TypeName="research"`, `Model="flash"`) sinh ra phiên subagent `conversationId = 42058893-0f65-45b1-9297-83319a560210`.
     - Kiểm tra `/tmp/agy` và `/Users/trainguyen/.gemini/antigravity/brain/42058893-0f65-45b1-9297-83319a560210/.system_generated/logs/transcript.jsonl`:
       - File `/tmp/agy/42058893-0f65-45b1-9297-83319a560210.boot` được tạo lúc `2026-10-21 10:10`.
       - `step_index: 0` trong `transcript.jsonl` của subagent ghi nhận `source: "SYSTEM_SDK"`, `type: "EPHEMERAL_MESSAGE"`, `content: "BOOT S0: (1) Đọc agent/GEMINI.md..."` $\rightarrow$ **Chứng minh 100% Antigravity CÓ kích hoạt `.agents/hooks.json` (`agy_guard.py`) bên trong phiên của subagent!**
       - `step_index: 1` trong `transcript.jsonl` của subagent ghi nhận prompt từ parent dưới dạng `type: "SYSTEM_MESSAGE"` (`sender=b2813663-5ebe-41fd-9095-f618b17c6cab`) $\rightarrow$ Đã bổ sung kiểm tra `SYSTEM_MESSAGE` (`sender=`) vào `contaminated_clean_room_reads` để chặn từ bên trong phiên subagent nếu prompt chứa nguồn cấm.
  3. **Cài đặt & Kiểm chứng Runtime `rea@6.7.0` (`G-06`):**
     - Đã cài đặt `rea-agents@6.7.0` tại `/Users/trainguyen/.nvm/versions/node/v22.18.0/bin/rea`, chạy `rea setup --client antigravity --client claude_code --yes`, liên kết skill `/Users/trainguyen/Documents/work/agent/skills/reverse-engineer-anything`, và kiểm chứng bằng `rea doctor --client antigravity --json` (`registration:antigravity = aligned`, `registration:claude_code = aligned`, `skill = aligned`, `153` CLI commands, `207` MCP tools).
  4. **Runtime Execution của `tooling/mutation_gate.py` với 3 Bản Cố tình Làm hỏng Thật cho `G-01..G-04` (`Revision R2`):**
     - Biên bản vật lý: [`06_mutation_summary_clean_room_and_mutation_gate_20261021T101205.json`](file:///Users/trainguyen/Documents/work/agent/memory/workspaces/agent-process-optimization/06_mutation_summary_clean_room_and_mutation_gate_20261021T101205.json)
     - Kết quả runtime thực tế:
       - `M1_G01_CLEAN_ROOM_START_BYPASS` (Làm hỏng `user_approved` để chế độ `clean-room` dùng `APPROVE` thay vì `CLEAN_ROOM_START`): **`KILLED`** (`returncode = 1`, bị bắt sống bởi `test_g01_clean_room_requires_exact_start_line`), `SHA-256` khôi phục 100% (`1b25229bc3ebbd35eb3f042b25280403945e8f1891c315d93362d88390085257`).
       - `M2_G02_RUN_COMMAND_READ_BLINDNESS` (Làm hỏng `READ_ARG_KEYS` để bỏ qua `run_command`): **`KILLED`** (`returncode = 1`, bị bắt sống bởi `test_g02_run_command_and_url_reads_detected`), `SHA-256` khôi phục 100% (`1b25229bc3ebbd35eb3f042b25280403945e8f1891c315d93362d88390085257`).
       - `M3_G04_PRE_SUBAGENT_LEAK_BYPASS` (Làm hỏng `handle_pre_subagent` để cho qua prompt rò rỉ code gốc): **`KILLED`** (`returncode = 1`, bị bắt sống bởi `test_g04_pre_subagent_blocks_invoke_subagent_and_send_message_leaks`), `SHA-256` khôi phục 100% (`1b25229bc3ebbd35eb3f042b25280403945e8f1891c315d93362d88390085257`).

---

## 3. Concurrency, Timeout/Heartbeat & Data Continuity Audit (Kiểm tra Đa luồng, Timeout & Liên tục dữ liệu)

- **Concurrency / Multi-worker / Debouncer & Bytecode Isolation:**
  - Các mutant được thực thi tuần tự từng biến thể trên cùng 1 tiến trình quản lý và xóa sạch `.pyc` (`_invalidate_pyc`) kèm `PYTHONDONTWRITEBYTECODE=1`, ngăn chặn xung đột cache `.pyc` khi đột biến xảy ra trong cùng 1 giây với Baseline.
  - Tên file biên bản `06_mutation_summary_<suffix>_<ts>.json` định danh theo cả `<suffix>` và `<ts>` để không ghi đè lẫn nhau khi một workspace thực hiện nhiều task kế tiếp.
- **Timeout / Heartbeat / AckWait & Process Signal Safety:**
  - Mỗi lệnh chạy test trong `mutation_gate.py` đều gắn `timeout=timeout_sec` (mặc định `60s`). Khi xảy ra `subprocess.TimeoutExpired`, hệ thống không tính nhầm là `KILLED` mà phân loại rõ `status = "TIMEOUT"` và đánh trượt cổng (`gate = "FAIL"`).
  - Bọc `signal.signal(signal.SIGTERM, _sigterm_to_exit)` đảm bảo ngay cả khi tiến trình cha timeout/kill `mutation_gate.py`, khối `finally:` vẫn chạy để khôi phục nguyên vẹn file gốc trước khi thoát.
- **Data Continuity / Watermark / State Integrity:**
  - Trước và sau mỗi mutant, mã băm `SHA-256` của file nguồn trên đĩa được đối chiếu (`sha256_before == sha256_after`), đảm bảo tính toàn vẹn dữ liệu 100% (`all_restored_sha256 == True`).

---

## 4. Anti-Speculation & Honesty Check (Kiểm tra Suy diễn & Báo cáo láo)

- Toàn bộ số dòng code, mã băm `SHA-256`, số lượng unit test (`52` tests trong `hooks/agy/tests`, `37` tests trong `tooling/tests`), log trace subagent `42058893-0f65-45b1-9297-83319a560210` và các file summary của `/laya_router` / `mutation_gate.py` trong báo cáo này đều được trích xuất 1-1 từ file vật lý trên đĩa sau khi hoàn tất code (không suy diễn, không báo cáo láo).

---

## 5. Self-Improvement Loop (Vòng lặp Phản tỉnh & Khắc phục — Lessons Audit)

- **Đối chiếu `lessons/INDEX.md`:** Đã kiểm tra và tuân thủ nghiêm ngặt `L002/L261` (5 lỗ hổng REA Clean-Room `G-01..G-06`), `L039` (chống Tautological Validation bằng 3-Mutant Gate), `L257` (vòng lặp tự khắc phục khép kín `S5 -> S3 -> S4 -> S5` không bắt User `APPROVE` lần 2).
- **Bài học mới đã ghi nhận vào `memory/global/lessons.md`:**
  - Tiêu đề: `### [2026-10-21] Five REA Clean-Room Security & Enforcement Gaps: Unanchored APPROVE Regex, Unmonitored run_command Source Reads, Heading-Only Spec Check, Unchecked Subagent Prompt Leaks & Missing rea-agents Setup`
  - Kiểm chứng `wc -l`: Tăng từ `2021` lên `2028` dòng (`261 -> 262` bài học).
  - Đã chạy `python3 /Users/trainguyen/Documents/work/agent/tooling/build_lessons_index.py` $\rightarrow$ `entries=262/262/262`.

---

## 6. Decision Gate `/laya_router` DG3 Final Verdict

- **Summary File (Sau lần sửa code cuối — Revision R2):** [`06_laya_summary_dg3_20261021T101254.json`](file:///Users/trainguyen/Documents/work/agent/memory/workspaces/agent-process-optimization/06_laya_summary_dg3_20261021T101254.json)
- **Trusted Rounds:** `r3`
- **Kết quả kiểm định từng Claim trên code thật sau khi sửa:**
  - `A-G01` (`hooks/agy/agy_guard.py:48-61` — `APPROVE` neo đầu dòng + `bắt đầu`, `CLEAN_ROOM_START` yêu cầu dòng `BẮT ĐẦU`/`APPROVE` đứng riêng): **`ACCEPT`** (`0.9365`).
  - `A-G02` (`hooks/agy/agy_guard.py:327-349` — `contaminated_clean_room_reads` kiểm tra cả `SYSTEM_MESSAGE` `sender=` và `READ_ARG_KEYS` gồm `run_command`/`read_url_content`): **`ACCEPT`** (`0.7257`).
  - `A-G03` (`tooling/verify_governance.py:83-105` — `spec_rule_issues` kiểm tra từng rule `R<n>`/`CR-<n>` có `REF: path:line`, chặn `[REJECTED]` và chặn dòng bullet thiếu `REF`): **`ACCEPT`** (`0.8030`).
  - `A-G04` (`hooks/agy/agy_guard.py:899-923` — `handle_pre_subagent` chặn rò rỉ đường dẫn/code gốc qua `invoke_subagent` và `send_message` ngay tại `PreToolUse`): **`ACCEPT`** (`0.8292`).
  - Đối chứng âm tính (`CF1`, `CF2`) bị bác bỏ chính xác; đối chứng dương tính (`CT1`) được xác nhận chính xác.
- **Final Gate Verdict:** **`PASS` 🟢**


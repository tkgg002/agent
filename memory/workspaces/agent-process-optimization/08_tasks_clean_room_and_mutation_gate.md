# 08 — Tasks: Tích hợp Clean-Room Rebuild Protocol & 3-Mutant Test Verification Gate (`clean_room_and_mutation_gate`)

- **Workspace:** `agent-process-optimization`
- **Suffix:** `clean_room_and_mutation_gate`
- **Tier:** `T2`
- **Trạng thái:** `DONE (S6 Completed — 28/28 hooks/agy tests, 20/20 tooling tests, 3/3 mutants KILLED, DG3 PASS)`
- **Tài liệu đối chiếu:** [`10_gap_analysis_clean_room_and_mutation_gate.md`](file:///Users/trainguyen/Documents/work/agent/memory/workspaces/agent-process-optimization/10_gap_analysis_clean_room_and_mutation_gate.md), [`12_implementation_plan_clean_room_and_mutation_gate.md`](file:///Users/trainguyen/Documents/work/agent/memory/workspaces/agent-process-optimization/12_implementation_plan_clean_room_and_mutation_gate.md)

---

## Danh sách Task Thực thi (Sau khi có `APPROVE`)

### [x] Task `T8.1`: Cập nhật Hiến pháp `GEMINI.md` và 4 Core Skills (`brain-delegate`, `muscle-execute`, `qa-agent`, `workspace-docs`)
- **Mô tả & Phạm vi (`[Mô tả]`):**
  - Bổ sung quy tắc **Clean-Room Rebuild Mode (`mode: clean-room` & `clean_room_forbidden_sources`)** (*"Model thấy code gốc không được viết code, model viết code không được thấy code gốc; chỉ giữ lại rule có bằng chứng đã qua `/laya_router` DG1"*) vào `GEMINI.md` (`L71`, `L73`, `L95-96`) và `skills/{brain-delegate,muscle-execute,workspace-docs}/SKILL.md`.
  - Bổ sung cổng **3-Mutant Test Verification (`tooling/mutation_gate.py`)** vào `GEMINI.md` (`L73-74`, `L99` Gate G3/G7) và `skills/{muscle-execute,qa-agent}/SKILL.md`.
- **Dữ liệu & Bằng chứng (`[Context & Logs]`):**
  - `GEMINI.md:71-74, 95-99` (chỉ thêm `+2` dòng, giữ nguyên cấu trúc 22 Rules)
  - `skills/brain-delegate/SKILL.md:54-68`
  - `skills/muscle-execute/SKILL.md:36-62`
  - `skills/qa-agent/SKILL.md:29-55`
  - `skills/workspace-docs/SKILL.md:30-42`
- **Tiêu chí Hoàn thành (`[Definition of Done]`):**
  - Dung lượng `GEMINI.md` giữ dưới ngưỡng an toàn (`~16.2 KB < 18 KB`).
  - Quy định rõ cấu trúc `## Clean-Room Approved Spec` trong `09_tasks_solution_*.md`, biến `clean_room_forbidden_sources` trong `01_requirements_*.md`, và lệnh chạy `python3 tooling/mutation_gate.py`.

### [x] Task `T8.2`: Xây dựng công cụ `tooling/mutation_gate.py` & Unit Tests `tooling/tests/test_mutation_gate.py` (Đã tích hợp `F3`, `F4`, `F5`)
- **Mô tả & Phạm vi (`[Mô tả]`):**
  - Tạo mới `tooling/mutation_gate.py` (`~235` dòng, 100% Python Standard Library, Agnostic Core — `ADR-07`):
    - Kiểm tra tối thiểu 3 biến thể cố tình làm hỏng (`len(mutants) >= 3` trong `mutations.json`).
    - **Khắc phục `F4`:** Tự động phân giải đường dẫn file tương đối theo `[Path.cwd(), AGENT_ROOT, mutations_file.parent]` và tự động lấy `cwd` + `test` từ `.agent-project.json` nếu mutant không khai báo thủ công.
    - **Khắc phục `F3` (Chống False-Kill do `SyntaxError` hoặc `Timeout`):** Với file `.py`, kiểm tra `ast.parse(mutated_text)` trên bộ nhớ trước khi ghi xuống đĩa (nếu gây `SyntaxError` trả mã `3 INVALID_INPUT`). Nếu lệnh test bị treo (`timed_out == True`) thì ghi nhận `status = "TIMEOUT"` và đánh trượt cổng (`gate = "FAIL"`, exit code `2`).
    - **Khắc phục `F5` (`SIGTERM`-safe Atomic Restore):** Đăng ký `signal.signal(signal.SIGTERM, _sigterm_to_exit)` chuyển `SIGTERM` thành `SystemExit(143)` quanh vòng lặp ghi file để khối `finally:` luôn khôi phục byte-for-byte `orig_bytes` và đối chiếu `SHA-256` trước/sau trùng khớp 100%.
    - Lưu biên bản vật lý `06_mutation_summary_<suffix>_<ts>.json` vào `memory/workspaces/<ws>/`.
  - Tạo mới `tooling/tests/test_mutation_gate.py` kiểm thử 6 kịch bản: (1) 3/3 mutants `KILLED` (`PASS`), (2) 1 mutant `SURVIVED` do test yếu (`FAIL`), (3) mutant gây `SyntaxError` bị chặn ở bước `ast.parse` (`INVALID_INPUT` code `3`), (4) mutant gây `Timeout` (`status = "TIMEOUT"`, `FAIL` code `2`, `SHA-256` khôi phục 100%), (5) `< 3` mutants (`INVALID_INPUT` code `3`), (6) tự động lấy `test_cmd` từ `.agent-project.json`.
- **Dữ liệu & Bằng chứng (`[Context & Logs]`):**
  - `tooling/laya_gate.py:97-105` (tham chiếu `resolve_ref_path`, exit code và ghi file summary vào workspace).
  - `hooks/agy/agy_guard.py:270-302` (tham chiếu `component_for` đọc `.agent-project.json`).
  - `.agent-project.json:12-19` (cấu hình test cho component `tooling`).
- **Tiêu chí Hoàn thành (`[Definition of Done]`):**
  - `python3 -m unittest discover -s tooling/tests -p 'test_*.py' -v` đạt `PASS` 100%.

### [x] Task `T8.3`: Nâng cấp `hooks/agy/agy_guard.py` (`H4a`, `H4b`, `H6`, `H7`) & `tooling/verify_governance.py` (Đã tích hợp `F1`, `F2`, `F6`)
- **Mô tả & Phạm vi (`[Mô tả]`):**
  - Trong `hooks/agy/agy_guard.py`:
    - Thêm `CLEAN_ROOM_MODE_RE`, `CLEAN_ROOM_FORBIDDEN_RE`, `CLEAN_ROOM_SPEC_RE` tại `L51-55`.
    - Thêm 3 hàm helper nhỏ gọn tại `L238`: `clean_room_config(ws_dir, suffix)`, `contaminated_clean_room_reads(st, forbidden)`, `mutation_gate_status(ws, ws_root, suffix)`.
    - **Cổng `H4a` (`L401-415` — Khắc phục `F1`):** Khi `01_requirements_<suffix>.md` có `mode: clean-room`, ghép nội dung file `09_tasks_solution_<suffix>.md` hiện có trên đĩa với `TargetContent -> ReplacementContent` trước khi kiểm tra mục `Clean-Room Approved Spec` (không làm lỗi lệnh `replace_file_content` từng phần).
    - **Cổng `H4b` (`L436-454` — Khắc phục `F2`):** Khi `01_requirements_<suffix>.md` có `mode: clean-room` và `clean_room_forbidden_sources`, nếu phiên hiện tại (`calls(st)`: `view_file`/`grep_search`/`find_by_name`) đã đọc file trong danh sách cấm thì `deny` sửa source code trực tiếp và yêu cầu uỷ quyền qua `invoke_subagent`.
    - **Cổng `H6` (`L85-91` `FINAL_AUDIT_MSG`):** Nhắc chạy `mutation_gate.py` khi áp dụng Clean-Room / Mutation Gate.
    - **Cổng `H7` (`L560-575` — Khắc phục `F6`):** Giữ nguyên 6 phần tử `AUDIT_REQUIRED_PATTERNS` để tương thích ngược 100% với 24 unit test hiện có; trong khối `if src:`, khi task có `mode: clean-room` hoặc có file `06_mutation_summary_*.json`, bắt buộc cả `mutation_gate_status == "PASS"` **VÀ** `touched(st, r"(mutation_gate\.py|06_mutation_summary_)", last)` sau lần sửa code cuối.
  - Trong `tooling/verify_governance.py` (`L21-23`, `L193-212`):
    - Đồng bộ kiểm tra mục `Clean-Room Approved Spec` (ở cả `S3` và `S6`) và kiểm tra `06_mutation_summary_<suffix>_*.json` có `"gate": "PASS"` và `"killed_count" >= 3` (ở `S6`) khi `01_requirements_<suffix>.md` có `mode: clean-room`.
  - Cập nhật `hooks/agy/tests/test_agy_guard.py` (thêm 4 test mới) và `tooling/tests/test_verify_governance.py` (thêm 2 test mới).
- **Dữ liệu & Bằng chứng (`[Context & Logs]`):**
  - `hooks/agy/agy_guard.py:51-96, 238, 379-455, 560-575`
  - `tooling/verify_governance.py:21-23, 193-212`
- **Tiêu chí Hoàn thành (`[Definition of Done]`):**
  - Toàn bộ 28 test trong `hooks/agy/tests/test_agy_guard.py` và 20 test trong `tooling/tests/` đạt `PASS` 100%.

### [x] Task `T8.4`: Thực thi Kiểm thử Đột biến thật (`mutation_gate.py`), Chạy `DG3`, Lập `audit_report_clean_room_and_mutation_gate.md` & Đóng phiên `S6`
- **Mô tả & Phạm vi (`[Mô tả]`):**
  - Tạo 3 biến thể làm hỏng logic thật (`06_mutations_clean_room_and_mutation_gate.json`) và chạy `python3 tooling/mutation_gate.py` để chứng minh bộ test mới bắt sống cả 3 lỗi (`3/3 KILLED`) và khôi phục `SHA-256` 100%.
  - Thực hiện đầy đủ 6 bước **Mandatory Final Adversarial Audit (S5)**, gọi `/laya_router` (`DG3`), xuất `audit_report_clean_room_and_mutation_gate.md`, `06_validation_clean_room_and_mutation_gate.md`, `11_report_clean_room_and_mutation_gate.md`, append `DONE` vào `05_progress_process_optimization.md` và chạy `verify_governance.py`.
- **Dữ liệu & Bằng chứng (`[Context & Logs]`):**
  - `skills/qa-agent/SKILL.md:56-120`
  - `skills/session-wrapup/SKILL.md:22-58`
- **Tiêu chí Hoàn thành (`[Definition of Done]`):**
  - `verify_governance.py --workspace agent-process-optimization --suffix clean_room_and_mutation_gate` đạt `PASSED 🟢`.


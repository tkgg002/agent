# 01 — Requirements: Tích hợp Clean-Room Rebuild Protocol & 3-Mutant Test Verification Gate vào Hệ thống `/agent`

tier: T2
mode: clean-room

## 1. Yêu cầu từ User (`2026-10-20T10:13:58+07:00`)
> *"Tích hợp 2 kỹ thuật tinh hoa của bài này vào hệ thống"*
>
> Hai kỹ thuật tinh hoa từ quy trình **REA + Jev + Clean-Room + Haiku 3-Mutant Testing**:
> 1. **Clean-Room Specification & Sub-Agent Isolation (Cách ly Phòng Sạch):**
>    - Quy tắc cốt lõi: *"Model thấy code gốc thì không bao giờ được viết code, và model viết code thì không bao giờ được thấy code gốc!"*
>    - Người đọc code/binary gốc (Brain/Extractor) chỉ rút trích quy tắc hành vi vào đặc tả (`SPEC` trong `09_tasks_solution_<suffix>.md`), mỗi quy tắc bắt buộc có bằng chứng (`REF: file:line`) và phải được cổng quyết định (`DG1` / `/laya_router` — đóng vai trò của Jev) phê duyệt; quy tắc nào thiếu bằng chứng phải bị loại bỏ.
>    - Người viết code xây dựng lại tính năng (Clean-Room Builder Sub-agent ở chặng `S4`) chỉ nhận bản `Clean-Room Approved Spec` đã duyệt, tuyệt đối không mở hoặc sao chép mã nguồn gốc cũ (`clean_room_forbidden_sources`), giúp triệt tiêu việc copy lại bug cũ và nợ kỹ thuật.
> 2. **3-Mutant Test Verification Gate (Kiểm thử Đột biến với 3 Bản Cố tình Làm hỏng — Chống "Test Phế"):**
>    - Để chứng minh bộ test đủ mạnh (chống bệnh viết test hình thức luôn pass — `L039 Tautological Validation`), bắt buộc kiểm tra bộ test với **tối thiểu 3 phiên bản cố tình tạo lỗi (3 Mutants: `M1`, `M2`, `M3`)**.
>    - Điều kiện nghiệm thu kép (Dual Exit Condition): Phiên bản code chuẩn phải **PASS 100% bài kiểm thử**, VÀ cả **3 phiên bản cố tình chứa lỗi đều phải bị bộ kiểm thử phát hiện và đánh trượt (KILLED / FAIL)**, đồng thời mã băm `SHA-256` của file code phải được khôi phục nguyên vẹn 100% sau khi chạy.

## 2. Phạm vi & Ràng buộc
- **Phạm vi tác động:**
  1. `GEMINI.md` (Rule #1, Rule #9 S3/S4/S5, Rule #13 Clean-Room Rule, Rule #14 Gates G3/G4/G7).
  2. Các Core Skills liên quan: `skills/brain-delegate/SKILL.md`, `skills/muscle-execute/SKILL.md`, `skills/qa-agent/SKILL.md`, `skills/workspace-docs/SKILL.md`.
  3. Công cụ Kiểm thử Đột biến mới: `tooling/mutation_gate.py` + unit tests `tooling/tests/test_mutation_gate.py`.
  4. Lifecycle Hooks Dispatcher `hooks/agy/agy_guard.py` (`H4`, `H6`, `H7`) + unit tests `hooks/agy/tests/test_agy_guard.py`.
  5. Process Linter `tooling/verify_governance.py` + unit tests `tooling/tests/test_verify_governance.py`.
- **Ràng buộc:**
  - Tuân thủ tuyệt đối **Rule #1 & Rule #13 (Brain Propose-Only)**: ở chặng `S1 -> S3`, chỉ lập hồ sơ yêu cầu, danh sách task, thiết kế giải pháp `09_tasks_solution_clean_room_and_mutation_gate.md`, chạy `laya_gate.py` (`DG1`, `DG2`) và **DỪNG CHỜ User `APPROVE`** trước khi chạm vào bất kỳ file code/quy tắc nào.
  - Tuân thủ **ADR-07 (Agnostic Core)**: `tooling/mutation_gate.py` và `agy_guard.py` hoàn toàn độc lập dự án (không hardcode tên service, ngôn ngữ hay câu lệnh test).

## 3. Tiêu chí Chấp nhận (Acceptance Criteria — DoD)
- **AC1 (Clean-Room Protocol & Enforcement):**
  - Chuẩn hóa giao thức **Clean-Room Rebuild (`mode: clean-room`)** trong `GEMINI.md`, `brain-delegate`, `muscle-execute`: tách biệt Extractor (Brain đọc code gốc $\rightarrow$ trích xuất Rule kèm `REF:` $\rightarrow$ lọc qua `/laya_router` `DG1` $\rightarrow$ xuất mục `## Clean-Room Approved Spec`) và Clean-Room Builder (Sub-agent Muscle chỉ nhận Spec đã duyệt, không đọc `clean_room_forbidden_sources`).
  - Hook `H4` (`agy_guard.py`) và `verify_governance.py` kiểm tra: khi `01_requirements_<suffix>.md` khai báo `mode: clean-room`, file `09_tasks_solution_<suffix>.md` bắt buộc phải có mục `Clean-Room Approved Spec` đã lọc qua `DG1 = PASS`.
- **AC2 (Công cụ `tooling/mutation_gate.py` — 3-Mutant Test Verification):**
  - Xây dựng `tooling/mutation_gate.py` chạy tự động 4 bước: (1) Chạy `test_cmd` trên code chuẩn (`Baseline == PASS`); (2) Lần lượt tiêm $\ge 3$ biến thể lỗi (`M1`, `M2`, `M3`) từ file JSON và chạy `test_cmd`, yêu cầu cả 3 biến thể đều bị test đánh trượt (`KILLED`); (3) Khôi phục nguyên trạng file code trong khối `try...finally` và đối chiếu `SHA-256` trước/sau khớp 100%; (4) Xuất biên bản `06_mutation_summary_<suffix>_<ts>.json` (`gate: "PASS" | "FAIL"`).
- **AC3 (Hook `H6/H7` & `audit_report_<suffix>.md` Integration):**
  - Cập nhật `FINAL_AUDIT_MSG`, `AUDIT_REQUIRED_PATTERNS` và `handle_stop` (`H7`) trong `hooks/agy/agy_guard.py` cùng `tooling/verify_governance.py` để cưỡng chế kiểm tra bằng chứng **3-Mutant Verification (`mutation_gate.py` / `06_mutation_summary_*.json`)** và đối chiếu trong `audit_report_<suffix>.md`.
- **AC4 (Hardening 4 Lỗ hổng Bảo mật Phòng Sạch `G-01..G-04`):**
  - **`G-01` (Approval Gate):** Cổng duyệt hỗ trợ tường minh lệnh `"BẮT ĐẦU"` (theo đúng prompt REA Clean-Room), đồng thời neo đầu dòng cho `APPROVE` và neo nguyên dòng cho `CLEAN_ROOM_START` để câu mô tả chứa chữ `"duyệt"` hay `"BẮT ĐẦU"` giữa câu không bao giờ mở khóa sửa code nhầm.
  - **`G-02` (Contaminated Source Reads via `run_command` / `read_url_content`):** `contaminated_clean_room_reads` theo dõi cả `run_command` (`CommandLine`, `Cwd`), `read_url_content` (`Url`), `view_file_outline`, `view_code_item` bên cạnh `view_file`/`grep_search`/`find_by_name`.
  - **`G-03` (Per-Rule `REF:` & Unverified Bullet Check in `Clean-Room Approved Spec`):** `spec_rule_issues` trong cả `agy_guard.py` (`H4a`) và `verify_governance.py` (`S3/S6`) bắt buộc mục `Clean-Room Approved Spec` có $\ge 1$ rule (`R<n>` hoặc `CR-<n>`), mọi rule (kể cả bullet list và Markdown table row) đều phải có `REF: <path>:<line>`, không chứa trạng thái bị từ chối (`REJECTED`/`NO_EVIDENCE`/`BLOCK`), và không cho phép bất kỳ dòng bullet tự do nào thiếu `REF:` lọt qua.
  - **`G-04` (Subagent Prompt Leak Guard & Real Runtime Subagent Hook Trace):** Chặn rò rỉ đường dẫn/nội dung code gốc (`clean_room_forbidden_sources`) qua `invoke_subagent` và `send_message` ở cả **PreToolUse (`handle_pre_subagent`)** lẫn **Stop (`H7` `leaked_subagent_prompts`)**, đồng thời thực hiện trace runtime thật 1 lần gọi `invoke_subagent` để ghi nhận bằng chứng thực nghiệm về việc thực thi hook trong phiên của subagent.
- **AC5 (`G-06` — Cài đặt & Cấu hình `rea-agents`):**
  - Cài đặt `rea-agents` (`rea@6.7.0`) vào `PATH` mặc định, chạy `rea setup --client antigravity --client claude_code --yes`, liên kết skill `reverse-engineer-anything` vào `agent/skills/`, và kiểm chứng bằng `rea doctor --client antigravity --json` (`aligned`).

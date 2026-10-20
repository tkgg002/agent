# 08 — Tasks: Tích hợp Cổng Mandatory Final Adversarial Audit (`audit_report_*.md`) & Cưỡng chế bằng Hook (`H6/H7`)

- **Workspace:** `agent-process-optimization`
- **Suffix:** `mandatory_final_audit`
- **Tier:** `T2`

## Danh sách Công việc (Autonomous Full-Stack Tasks)

- [x] **T7.1 — Cập nhật `GEMINI.md` (Rule #4, #9, #14, #18):**
  - **Context:** `GEMINI.md:38-45` (Rule #4 Tiers `T2/T3`), `GEMINI.md:66-76` (Rule #9 `S5 SELF-AUDIT` & `S6 CLOSE`), `GEMINI.md:91-94` (Rule #14 `G3`, `G7`, `G8`), `GEMINI.md:109-112` (Rule #18).
  - **Nội dung:** Bắt buộc chặng `S5 SELF-AUDIT` thực hiện: (1) Phản biện từng task, từng file, từng dòng diff (`git diff`); (2) Kiểm tra chống suy diễn/báo cáo láo; (3) Áp dụng Vòng Lặp Phản Tỉnh & Khắc Phục (Self-Improvement Loop); (4) Tuyệt đối cấm test bằng mockup/closure giả trên struct production hoặc hallucinate kết quả — nếu cần phải chủ động nói phối hợp với User trace runtime value thật; (5) Soạn context đầy đủ gọi `/laya_router` (`DG3`) nhận xét lại quyết định; (6) Lưu toàn bộ vào `audit_report_<suffix>.md` trong workspace.
  - **DoD:** `wc -c GEMINI.md < 15000` bytes; giữ nguyên đủ 23 rule headers `0–22`.

- [x] **T7.2 — Đồng bộ 4 Core Skills (`qa-agent`, `session-wrapup`, `workspace-docs`, `muscle-execute`) & `~/.gemini/GEMINI.md`:**
  - **Context:** `skills/qa-agent/SKILL.md:1-75`, `skills/session-wrapup/SKILL.md:1-68`, `skills/workspace-docs/SKILL.md:1-90`, `skills/muscle-execute/SKILL.md:50-64`.
  - **Nội dung:** Đưa template chuẩn 6 phần của `audit_report_<suffix>.md` và quy tắc Anti-Smoke-Speculation / Anti-Mockup / Phối hợp User trace runtime value vào `qa-agent/SKILL.md`, `session-wrapup/SKILL.md`, `workspace-docs/SKILL.md`, `muscle-execute/SKILL.md`.
  - **DoD:** Cả 4 skill đồng nhất 100% quy định về `audit_report_<suffix>.md`.

- [x] **T7.3 — Nâng cấp Hook `H4 (PreToolUse)`, `H6 (PostInvocation)` & `H7 (Stop)` trong `hooks/agy/agy_guard.py`:**
  - **Context:** `hooks/agy/agy_guard.py:372-495` (`handle_pre_write`, `handle_post_invocation` và `handle_stop`).
  - **Nội dung:**
    1. Trong `H4 (handle_pre_write)`: Chặn cứng (`deny`) mọi hành vi gắn hàm mock/closure giả (`*Fn func`, `lookupMasterTargetsFn`, `runEventsFn`, `sqlmock`, `fakemodel`) vào file source code production (`not is_test_file(f)`).
    2. Trong `H6 (handle_post_invocation)`: Khi phát hiện có `src_edits` mới (`i > last`), kể cả khi `checks` PASS hay FAIL, luôn tiêm thêm `FINAL_AUDIT_MSG` nhắc Agent thực hiện đủ 6 bước Audit phản biện cuối workflow (`audit_report_<suffix>.md` + `/laya_router` DG3 + Anti-Smoke-Speculation Runtime Trace) trước khi dừng.
    3. Trong `H7 (handle_stop)`: Khi `src` khác rỗng, kiểm tra bắt buộc:
       - `touched(st, r"audit_report_", last)`: bắt buộc tạo/cập nhật `audit_report_<suffix>.md` sau lần sửa code cuối.
       - Kiểm tra nội dung file `audit_report_*.md` mới nhất trong workspace có đủ 6 phần bắt buộc (`AUDIT_REQUIRED_PATTERNS`).
       - Kiểm tra `/laya_router` `DG3` đạt `PASS` và được chạy sau lần sửa code cuối.
  - **DoD:** `hooks/agy/tests/test_agy_guard.py` bổ sung kịch bản test `test_21`..`test_24` (`24/24 PASS 100%`).

- [x] **T7.4 — Nâng cấp `tooling/verify_governance.py` kiểm tra bắt buộc `audit_report_*.md` cho `T2/T3` (Stage `S6`):**
  - **Context:** `tooling/verify_governance.py:31-56` (`TIER_RULES`), `tooling/verify_governance.py:130-211` (`verify_workspace_docs`).
  - **Nội dung:**
    1. Thêm `("audit_report_*.md", ("audit_report*.md",))` vào `TIER_RULES["T2"]` và `TIER_RULES["T3"]` (ở chặng `S6`), và thêm `"audit_report"` vào `SUFFIX_SCOPED_PREFIXES`.
    2. Kiểm tra nội dung file `audit_report_*.md` phải chứa đủ 6 mục `AUDIT_REQUIRED_PATTERNS`.
  - **DoD:** `tooling/tests/test_verify_governance.py` cập nhật và thêm test kiểm tra `audit_report_*.md` (`14/14 PASS 100%`).

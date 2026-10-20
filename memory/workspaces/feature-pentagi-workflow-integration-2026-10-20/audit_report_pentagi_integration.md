# Final Adversarial Audit Report: Tích hợp PentAGI & Khép Vòng Lặp `S5 → S3 Re-Plan` (`pentagi_integration`)

## 1. Line-by-Line Diff & Task Critique (Phản biện từng dòng update / `git diff` & từng task)

| Task / File:Line | Thay đổi thực tế | Phản biện (Adversarial Question: Cách phá / Edge case) | Bằng chứng Code / Runtime thật | Kết luận |
|------------------|------------------|--------------------------------------------------------|-------------------------------|----------|
| **T2.1** — `tooling/pentagi_bridge.py:63-88` | Cài đặt `mask_secrets()`, `_matches_prefix_boundary()`, `validate_target_url()` | Điều gì xảy ra nếu kẻ tấn công hoặc cấu hình sai truyền `http://localhost.evil.com:8080/api` để lách `startswith("http://localhost")`? | Hàm `_matches_prefix_boundary` (`pentagi_bridge.py:69-74`) kiểm tra phần dư `rem` sau prefix bắt buộc là `""` hoặc bắt đầu bằng `("/", ":", "?", "#")`. Đã kiểm chứng trong `test_pentagi_bridge.py:37` (`http://localhost.evil.com:8080/api` -> `False`). | ✅ PASS |
| **T2.1** — `tooling/pentagi_bridge.py:145-215` | Cài đặt `generate_s1_intake_from_report()` | Có nguy cơ ghi đè mất file `01_requirements` hoặc `08_tasks` đã có trong workspace (vi phạm Rule #17), hoặc lọt credential DB URI từ báo cáo PentAGI vào `agent/memory/` (vi phạm Rule #20)? | Dòng `159-162` chặn cứng bằng `FileExistsError` nếu `req_path.exists() or tasks_path.exists()`. Toàn bộ nội dung báo cáo đi qua `mask_secrets()` (`DB_URI_CRED_RE` + `SECRET_RE`) trước khi parse và ghi file (`test_03_generate_s1_intake_creates_valid_docs_and_masks_secrets` PASS). | ✅ PASS |
| **T2.2a** — `hooks/agy/agy_guard.py:577-603` | Cài đặt chốt chặn `Closed-Loop S5 -> S3 Re-Plan` (`last_aud_idx > last_sol_idx`) trong `handle_pre_write` (`H4b`) | Có chặn nhầm việc sửa code trong chặng `S4` bình thường không? Và khi sang `S5` (`audit_report_` đã tạo), nếu Agent cố sửa source code thì có chặn được không? | Trong luồng chuẩn `S4 -> S5`, Agent sửa code ở `S4` trước khi tạo `audit_report_` ở `S5`, nên ở `S4` `last_aud_idx == -1 < last_sol_idx` (không ảnh hưởng). Ngay khi đã tạo/sửa `audit_report_` ở `S5`, `last_aud_idx > last_sol_idx` trả về `{"decision": "deny"}` cho tới khi Agent cập nhật lại `09_tasks_solution` (`last_sol_idx > last_aud_idx`) và có `APPROVE` mới (`test_29` PASS). | ✅ PASS |
| **T2.2b** — `hooks/agy/agy_guard.py:92-98, 649-668` | Cài đặt `ESCALATION_MENTOR_MSG` và bộ đếm `<conv_id>.<u_idx>.loop` trong `handle_post_invocation` (`H6`) | Bộ đếm `.loop` có bị rò rỉ sang lượt `USER_INPUT` tiếp theo hoặc không reset khi `checks` đã pass không? | `loop_key = f"{conv_id}.{u_idx}.loop"` gắn chặt với chỉ số `u_idx = last_user_idx(st)`, tự tách biệt theo từng lượt User; và khi `errs` rỗng (`checks` pass), dòng `664` ghi `"0"` vào `loop_file` (`test_30` PASS). | ✅ PASS |
| **T2.3** — `GEMINI.md:78`, `skills/qa-agent/SKILL.md:71`, `skills/muscle-execute/SKILL.md:73`, `skills/workspace-docs/SKILL.md:52-72`, `skills/security-agent/SKILL.md:17-62` | Đồng bộ quy tắc `Closed-Loop S5 -> S3 Re-Plan`, `.agent-project.json` (`allowed_pentest_targets`, `pentest_check`) và Tầng 5 Dynamic Pentest Gate | Có bất kỳ tên dự án, port nghiệp vụ hay thông tin nhạy cảm nào bị hardcode vào Core `/agent` (vi phạm Rule #3 Agnostic Core) không? | Đã kiểm tra từng dòng diff: 100% cấu hình đọc động từ `<project-root>/.agent-project.json`, mặc định chỉ mở `localhost`/`127.0.0.1`/`[::1]`. | ✅ PASS |

---

## 2. Anti-Smoke-Speculation & Real Runtime Trace Verification (Chống suy diễn Smoke Test, Mockup & Hallucination)

- **Kiểm tra Production Mock Pollution:** Xác nhận 0 hàm mock/closure giả (`*Fn`, `sqlmock`, `fakemodel`) trong `hooks/agy/agy_guard.py` và `tooling/pentagi_bridge.py`.
- **Thực thi Runtime thật (Zero Mockup / Zero Hallucination):**
  1. Chạy toàn bộ bộ test thực tế của cả 2 component (`hooks` và `tooling`) qua subprocess thật:
     - `hooks/agy/tests`: **30/30 tests PASS** (`Ran 30 tests in 0.87s - OK`).
     - `tooling/tests`: **25/25 tests PASS** (`Ran 25 tests in 1.440s - OK`), bao gồm cả `test_pentagi_bridge.py` (5 tests) và `test_verify_governance.py` (8 tests).
  2. Chạy trực tiếp CLI `tooling/pentagi_bridge.py trigger` trên terminal thật:
     - Với target hợp lệ `http://localhost:8080/api/v1/orders --dry-run`: trả về `{"status": "DRY_RUN_VALIDATED", ...}` (`exit code 0`).
     - Với target ngoài whitelist `https://prod.example.com/api --dry-run`: bị chặn đứng ngay lập tức với `{"status": "BLOCKED", "error": "Target URL 'https://prod.example.com/api' bị chặn bởi cổng an toàn: không nằm trong allowed_pentest_targets."}` (`exit code 2`).

---

## 3. Kiểm tra Concurrency / Multi-worker, Timeout-Heartbeat & Data Continuity

- **Concurrency & Multi-Workspace Isolation (`<conv_id>.<u_idx>.loop` & Workspace Scoping):**
  - Trong môi trường nhiều phiên (multi-agent / multi-conversation chạy song song trên cùng repo `/agent`), trạng thái đếm vòng lặp của Hook `H6` được cô lập theo `f"{conv_id}.{u_idx}.loop"` trong `STATE` (`/tmp/agy`), tránh hoàn toàn race condition giữa các conversation ID khác nhau.
  - Trong `H4b`, việc kiểm tra `sol` và `aud` dựa trên transcript riêng của phiên hiện tại (`st = steps(payload)`), không bị nhiễu bởi file `audit_report_` của workspace khác.
- **Timeout & Network Safety (`pentagi_bridge.py` + `agy_guard.py`):**
  - `pentagi_bridge.py:242` đặt `timeout=timeout_sec` (mặc định `30s`) cho `urllib.request.urlopen`, bắt ngoại lệ `urllib.error.URLError` và trả về JSON có cấu trúc thay vì treo tiến trình vô hạn.
  - `agy_guard.py:640` đặt `timeout=40` giây cho mỗi lệnh `checks` trong `handle_post_invocation` (H6).
- **Data Continuity & Append-Only Integrity (`05_progress` & Rule #17):**
  - `generate_s1_intake_from_report()` bảo toàn tính liên tục của log `05_progress_<suffix>.md` (nối tiếp vào cuối file nếu đã tồn tại, hoặc khởi tạo mới với header chuẩn), và từ chối ghi đè `01_requirements`/`08_tasks` bằng `FileExistsError`.

---

## 4. Kiểm tra Suy diễn & Báo cáo Láo (Anti-Speculation & Honesty Check)

- Mọi khẳng định trong thiết kế và báo cáo đều có trích dẫn `file:line` thật và output chạy lệnh thật:
  - `hooks/agy/agy_guard.py:92-98` (`ESCALATION_MENTOR_MSG`), `577-603` (`H4b S5 -> S3 Re-Plan Loop`), `649-668` (`H6 Loop Counter`).
  - `tooling/pentagi_bridge.py:63-88` (`mask_secrets`, `_matches_prefix_boundary`, `validate_target_url`), `145-215` (`generate_s1_intake_from_report`), `218-274` (`trigger_pentagi_flow`).
  - Không có bất kỳ kết quả giả lập hay suy diễn nào.

---

## 5. Vòng lặp Phản tỉnh & Khắc phục (Self-Improvement Loop & Lessons Check)

- Đã đối chiếu với `memory/global/lessons/INDEX.md` (đặc biệt là `L001` — Không suy diễn từ smoke test, `L056` — Cấm làm bẩn code production bằng closure giả).
- Trong quá trình thực thi `S4 → S5`, không vi phạm lesson nào trong `lessons.md`; câu hỏi phản biện kiến trúc của User ở S3 về việc khoá vòng lặp `S5 Audit → S3 Re-Plan` đã được giải quyết tận gốc bằng cả quy tắc trong `GEMINI.md` lẫn cưỡng chế kỹ thuật trong Hook `H4b` (`hooks/agy/agy_guard.py:577-598`).

---

## 6. Kết quả QC & Nhận xét từ `/laya_router` (`DG3`)

- Đã chạy `laya_gate.py run --stage dg3` **sau lần chỉnh sửa source code cuối cùng**:
  - **Command:** `python3 /Users/trainguyen/Documents/work/agent/tooling/laya_gate.py run --stage dg3 --workspace feature-pentagi-workflow-integration-2026-10-20 --claims .../06_laya_claims_dg3_final_pentagi_integration.json`
  - **Summary File:** `06_laya_summary_dg3_20261020T134233.json`
  - **Raw File:** `06_laya_raw_dg3_20261020T134233.json`
  - **Kết quả:** `"gate": "PASS"` (`0` lỗi `evidence_error`, toàn bộ trích dẫn `REF:` khớp 100% với source code thực tế; các claim nhận `UNVALIDATED` ở r2 được nghiệm thu bằng 55/55 unit tests thực tế theo đúng quy định tại `GEMINI.md` Rule #10).

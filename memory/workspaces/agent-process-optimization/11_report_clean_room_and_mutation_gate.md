# 11 — Implementation & Change Report: Tích hợp Clean-Room Rebuild Protocol & 3-Mutant Test Verification Gate (`clean_room_and_mutation_gate`)

- **Workspace:** `agent-process-optimization`
- **Suffix:** `clean_room_and_mutation_gate`
- **Tier:** `T2` (`mode: clean-room`)
- **Ngày hoàn tất:** `2026-10-20`

---

## 1. Tóm Tắt Thay Đổi (Executive Summary)

Đã hoàn tất tích hợp 2 kỹ thuật tinh hoa từ quy trình **REA + Jev + Clean-Room Rebuild + Haiku 3-Mutant Testing** vào hệ thống `/Users/trainguyen/Documents/work/agent` theo đúng thiết kế Revision `R1` đã được User `APPROVE`:

1. **Giao thức Cách ly Phòng Sạch (`mode: clean-room` & `clean_room_forbidden_sources`):**
   - Bắt buộc `09_tasks_solution_<suffix>.md` tách mục `## Clean-Room Approved Spec` (chỉ giữ các quy tắc hành vi có bằng chứng `REF:` đã vượt qua cổng `/laya_router` `DG1`).
   - Tại Hook `H4a` ([`hooks/agy/agy_guard.py:507-528`](file:///Users/trainguyen/Documents/work/agent/hooks/agy/agy_guard.py#L507-L528)), ghép nội dung file hiện có trên đĩa với đoạn sửa `TargetContent -> ReplacementContent` trước khi kiểm tra `CLEAN_ROOM_SPEC_RE` (bảo toàn lệnh `replace_file_content` từng phần — khắc phục `F1`).
   - Tại Hook `H4b` ([`hooks/agy/agy_guard.py:557-570`](file:///Users/trainguyen/Documents/work/agent/hooks/agy/agy_guard.py#L557-L570)), hàm `contaminated_clean_room_reads(st, forbidden)` quét lịch sử `calls(st)` trong `transcript.jsonl` và chặn cứng (`deny`) nếu phiên đã đọc file gốc trong `clean_room_forbidden_sources` mà tự ý sửa source code, bắt buộc uỷ quyền qua `invoke_subagent` mới tinh chỉ nhận `Clean-Room Approved Spec` (khắc phục `F2`).

2. **Cổng Kiểm thử Đột biến 3 Bản Cố tình Làm hỏng ([`tooling/mutation_gate.py`](file:///Users/trainguyen/Documents/work/agent/tooling/mutation_gate.py)):**
   - Tự động phân giải đường dẫn tương đối và tự động đọc `cwd` + `test` từ `.agent-project.json` gần nhất (khắc phục `F4`).
   - Kiểm tra cú pháp Python bằng `ast.parse(mutated_text)` trên RAM trước khi ghi xuống đĩa (chặn `SyntaxError` gây False-Kill) và phân loại rõ `status = "TIMEOUT"` khi lệnh test treo (khắc phục `F3`).
   - Vô hiệu hóa cache `.pyc` (`PYTHONDONTWRITEBYTECODE=1` + `_invalidate_pyc(fpath)`) để ngăn lỗi nạp nhầm `.pyc` cũ khi đột biến có cùng độ dài byte trong vòng `< 1` giây.
   - Bọc signal handler `SIGTERM -> SystemExit(143)` và khối `try...finally` khôi phục nguyên trạng byte-for-byte kèm đối chiếu `SHA-256` trước/sau trùng khớp 100% (khắc phục `F5`).
   - Cưỡng chế tự động tại Hook `H7` ([`hooks/agy/agy_guard.py:701-707`](file:///Users/trainguyen/Documents/work/agent/hooks/agy/agy_guard.py#L701-L707)) và [`tooling/verify_governance.py:197-255`](file:///Users/trainguyen/Documents/work/agent/tooling/verify_governance.py#L197-L255) (khắc phục `F6`).

---

## 2. Danh Sách Các File Đã Tạo Mới / Cập Nhật & Số Lượng Dòng Code

| STT | File | Loại thay đổi | Số dòng thay đổi | Mô tả chi tiết |
| :---: | :--- | :---: | :---: | :--- |
| 1 | [`GEMINI.md`](file:///Users/trainguyen/Documents/work/agent/GEMINI.md) | Sửa | `+6 / -4` (`131` dòng) | Bổ sung quy định `mode: clean-room`, `clean_room_forbidden_sources` và `mutation_gate.py` tại Rule #9 (`S3/S4`), Rule #13 và Rule #14 (`Gate G3`). |
| 2 | [`skills/brain-delegate/SKILL.md`](file:///Users/trainguyen/Documents/work/agent/skills/brain-delegate/SKILL.md) | Sửa | `+3 / -0` (`87` dòng) | Bổ sung mục 6 hướng dẫn lập `## Clean-Room Approved Spec` và bàn giao cho Clean-Room Sub-agent. |
| 3 | [`skills/muscle-execute/SKILL.md`](file:///Users/trainguyen/Documents/work/agent/skills/muscle-execute/SKILL.md) | Sửa | `+9 / -2` (`76` dòng) | Bổ sung quy tắc cách ly phòng sạch (`clean_room_forbidden_sources`) ở Bước 4 và chạy `mutation_gate.py` ở Bước 5. |
| 4 | [`skills/qa-agent/SKILL.md`](file:///Users/trainguyen/Documents/work/agent/skills/qa-agent/SKILL.md) | Sửa | `+1 / -0` (`124` dòng) | Bổ sung mục 5 (`tooling/mutation_gate.py` — 3-Mutant Verification Gate) vào Gate `G3`. |
| 5 | [`skills/workspace-docs/SKILL.md`](file:///Users/trainguyen/Documents/work/agent/skills/workspace-docs/SKILL.md) | Sửa | `+3 / -3` (`70` dòng) | Bổ sung metadata `mode: clean-room`, `clean_room_forbidden_sources` và biên bản `06_mutation_summary_<suffix>_<ts>.json`. |
| 6 | [`tooling/mutation_gate.py`](file:///Users/trainguyen/Documents/work/agent/tooling/mutation_gate.py) | Tạo mới | `+326` dòng | Công cụ CLI thực thi kiểm thử đột biến $\ge 3$ bản hỏng (`ast.parse`, `.agent-project.json` auto-detect, `_invalidate_pyc`, `SIGTERM`-safe `SHA-256` restore). |
| 7 | [`tooling/tests/test_mutation_gate.py`](file:///Users/trainguyen/Documents/work/agent/tooling/tests/test_mutation_gate.py) | Tạo mới | `+267` dòng | Bộ 5 unit test kiểm chứng toàn diện `mutation_gate.py` (3/3 KILLED, SURVIVED, `SyntaxError`, `Timeout`, `<3` mutants). |
| 8 | [`hooks/agy/agy_guard.py`](file:///Users/trainguyen/Documents/work/agent/hooks/agy/agy_guard.py) | Sửa | `+135 / -2` (`769` dòng) | Bổ sung `clean_room_config`, `contaminated_clean_room_reads`, `mutation_gate_status` và nâng cấp `H4a`, `H4b`, `H6`, `H7`. |
| 9 | [`hooks/agy/tests/test_agy_guard.py`](file:///Users/trainguyen/Documents/work/agent/hooks/agy/tests/test_agy_guard.py) | Sửa | `+219 / -0` (`905` dòng) | Bổ sung 4 unit test mới (`test_25`..`test_28`), nâng tổng số test của Hook lên `28/28 PASS`. |
| 10 | [`tooling/verify_governance.py`](file:///Users/trainguyen/Documents/work/agent/tooling/verify_governance.py) | Sửa | `+63 / -0` (`356` dòng) | Bổ sung kiểm tra `Clean-Room Approved Spec` (`S3/S6`) và `06_mutation_summary_*.json` (`S6`) khi `mode: clean-room`. |
| 11 | [`tooling/tests/test_verify_governance.py`](file:///Users/trainguyen/Documents/work/agent/tooling/tests/test_verify_governance.py) | Sửa | `+85 / -0` (`270` dòng) | Bổ sung `test_clean_room_s3_and_s6_enforcement`, nâng tổng số test của `tooling/tests` lên `20/20 PASS`. |
| 12 | [`memory/global/lessons.md`](file:///Users/trainguyen/Documents/work/agent/memory/global/lessons.md) & [`INDEX.md`](file:///Users/trainguyen/Documents/work/agent/memory/global/lessons/INDEX.md) | Append | `+7` dòng (`1951` dòng) | Ghi nhận bài học `#mutation-testing-pyc-cache #partial-replace-hook-merge #syntax-timeout-false-kill` (`250/250` entries). |

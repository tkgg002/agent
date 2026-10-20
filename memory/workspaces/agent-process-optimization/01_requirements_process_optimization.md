# 01 — Requirements: Tối ưu hoá quy trình Agent

tier: T2

## Yêu cầu từ User
- [2026-10-15] "tối ưu hoá quy trình ở agent của anh. cần update những gì"

## Phạm vi
- Audit toàn bộ cơ chế quy trình: `agent/GEMINI.md`, global rules, `agent/workflows/`, `agent/hooks/`, `agent/tooling/verify_governance.py`, `agent/memory/global/lessons.md`, global skills.
- Đầu ra: danh sách cần update + giải pháp cụ thể + lộ trình. **Propose-Only** — không sửa config/code khi chưa APPROVE.

## Tiêu chí chấp nhận
- R1: Mỗi phát hiện có bằng chứng kiểm chứng được (lệnh/đường dẫn/tài liệu).
- R2: Một phương án duy nhất, có code demo cho các điểm quan trọng.
- R3: Lộ trình theo phase, mỗi phase có restore-point + tiêu chí PASS.
- R4 (`workflow_enforcement_fix` — 2026-10-16): Khắc phục 5 nguyên nhân gốc rễ khiến task thực tế (`9d55dea5-f02d-4a52-8c5e-5169f1587e0b`) chạy bỏ qua workflow và không bị hook chặn:
  1. Giải phóng Context Budget `<skills>` tại `~/.gemini/config/skills/` để 10/10 Core Skills luôn hiển thị, đồng bộ `.agents/{skills,hooks}.json` cho `data-hub`, và inline checklist `S0 → S6` vào `GEMINI.md` Rule #9.
  2. Sửa lỗi `_unquote()` cho các trường `args` bọc dấu `"` trong `transcript.jsonl` thật (`hooks/agy/agy_guard.py`).
  3. Nâng cấp `H1` (chạy theo từng lượt `USER_INPUT`), `H4` (chặn tạo `09_tasks_solution_<suffix>.md` khi chưa qua S1 `01`/`08`/`05_progress`), `H7` (reset `.stop` theo từng lượt `USER_INPUT`, chặn Shadow Analysis).
  4. Nâng cấp `tooling/verify_governance.py` hỗ trợ `--suffix` và `--stage S3|S6` để chặn PASS "ăn theo" file cũ trong workspace tái sử dụng.

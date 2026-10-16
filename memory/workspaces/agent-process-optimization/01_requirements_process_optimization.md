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

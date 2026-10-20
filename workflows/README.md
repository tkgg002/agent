# Agent Workflows (Đã chuyển sang Antigravity Skills)

Toàn bộ các luồng công việc cốt lõi đã được nâng cấp và chuẩn hóa sang kiến trúc **Antigravity Skills** tại `/Users/trainguyen/Documents/work/agent/skills/<skill-name>/SKILL.md` (đăng ký qua `.agents/skills.json`):

1. [`brain-delegate`](../skills/brain-delegate/SKILL.md) — Điều phối, điều tra `S1–S3`, chạy `DG1/DG2` và giao việc vào `08_tasks_*.md`.
2. [`muscle-execute`](../skills/muscle-execute/SKILL.md) — Thực thi `S4` sau khi có `APPROVE`, tạo local restore-point, chạy test & verify runtime theo `.agent-project.json`.
3. [`debug-agent`](../skills/debug-agent/SKILL.md) — Săn lỗi gốc rễ & Bug Handling SOP (Reproduce Red → Green `G2`).
4. [`qa-agent`](../skills/qa-agent/SKILL.md) — Kiểm định chất lượng 8 cổng `G1–G8` và Decision Gate `DG3`.
5. [`security-agent`](../skills/security-agent/SKILL.md) — Quét lỗ hổng bảo mật, chặn lộ Secret/PII (`Rule #20`), cấm `git push` (`Rule #15`).
6. [`context-manager`](../skills/context-manager/SKILL.md) — Quản trị tri thức `lessons.md` + đồng bộ `lessons/INDEX.md` & `topics/*.md`.
7. [`workspace-docs`](../skills/workspace-docs/SKILL.md) — Quy chuẩn Doc Registry (`00–14`), phân tầng Tier `T0–T3` & Project Manifest `.agent-project.json`.
8. [`session-wrapup`](../skills/session-wrapup/SKILL.md) — Quy trình đóng phiên `S6 CLOSE`, đối soát bài học và chạy Linter `verify_governance.py`.
9. [`laya-router`](../skills/laya-router/SKILL.md) — Mô hình quyết định `/laya_router` (`DG1`, `DG2`, `DG3` — Mode A Claim & Mode B Agentic Debate).
10. [`audit-propose-only`](../skills/audit-propose-only/SKILL.md) — Quy trình Audit/Rà soát chỉ đọc (Read-Only) + kiểm định `DG3`.

Các file `.md` thế hệ cũ (96 file) được lưu trữ nguyên vẹn tại [`_archive/`](./_archive/).

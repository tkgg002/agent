---
tier: T2
mode: audit
suffix: agent-reliability
---
# 01 Requirements — Agent Reliability (chặn hồi quy / phá tương thích ngược / phá pattern / báo OK sai)

## Yêu cầu User (2026-11-06)
- "Tao đang nói về bản chất của chính mày: sao luôn làm lỗi, phá tương thích ngược lẫn design pattern của tao."
- "Liệt kê chi tiết các mục cần thực hiện để giải quyết triệt để."

## Phạm vi
- Core `agent/` (hooks, tooling, GEMINI.md) + manifest/tools cho `data-hub`.
- Propose-Only: chưa sửa code cho tới khi User APPROVE.

## Technical Constraints
- Lesson mới 2026-11-06 (Identify Criticized Subject): chủ thể là Agent, không đổ cho source.
- L018/L034/L037: "OK" phải do máy quyết định, có output thật.

## Definition of Done (bản đề xuất)
- Danh sách việc A–G, mỗi mục có file + DoD kiểm bằng máy, thứ tự thực hiện, câu hỏi cần User quyết.

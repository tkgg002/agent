---
tier: T2
workspace: audit-master-mongo-orchestrator-2026-10-22
suffix: enforce_antigravity_gemini_4_model_only
date: 2026-10-27
author: Brain-Delegate
status: INTAKE_COMPLETE
---

# Yêu cầu Kỹ thuật: Bỏ hết gemini-2.5, Thiết lập chuẩn Model 4.0 bên Antigravity (gemini-4-argon)

## 1. Bối cảnh & Yêu cầu User
- **Chỉ thị trực tiếp:** *"gemini-2.5-flash tao nói mày dùng nó khi nào, bỏ hết 2.5 dùng 4.0 bên antigravity cho tao, tự tiện"*.
- **Mục tiêu:**
  - Xóa bỏ 100% việc tự ý fallback hoặc hardcode sang `gemini-2.5-flash` trong toàn bộ codebase `ai-orchestrator`.
  - Đảm bảo hệ thống sử dụng chuẩn mô hình 4.0 bên trong Antigravity: `gemini-4-argon (antigravity)`.
  - Khi gọi CLI/Runner cho mô hình `gemini-4-argon` / Antigravity, KHÔNG chèn cờ `-m gemini-2.5-flash` vào lệnh thực thi.
  - Cập nhật các vị trí cấu hình mặc định (Planner, Coder High, Coder Std, Co-Architect Studio, Task Inspector) đồng bộ sang `gemini-4-argon (antigravity)`.

## 2. Tiêu chuẩn Nghiệm thu (DoD)
- Không còn bất kỳ tham chiếu fallback nào ép về `gemini-2.5-flash` trong `internal/runner/antigravity.go`, `internal/webui/task_llm_bridge.go`, `internal/webui/task_inspector.go`.
- Model mặc định cho mọi tác vụ kiến trúc và thảo luận là `gemini-4-argon (antigravity)`.
- Pass toàn bộ unit test, `go vet ./...` và build binary `./bin/orchestrator`.

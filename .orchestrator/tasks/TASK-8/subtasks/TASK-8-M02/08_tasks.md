# 08 Tasks Checklist (Sub-Task Execution Unit): TASK-8-M02 — Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `agent` (`analyze_dependencies.py`)
- **Parent Task:** `TASK-8` | **Wave:** `Wave 0` | **Isolated Branch:** `task/TASK-8/task-8-m02`
- [ ] **TASK-8-M02 (agent):**
  - **Context (`file:line`):** `agent/memory/workspaces/cds-project-inventory-2026-06-18/refactor-plan/analyze_dependencies.py:1 (import os)`
  - **Input Contract (`UpstreamReady=true`):** Kết nối dây dẫn nghiệp vụ tại `memory/workspaces/cds-project-inventory-2026-06-18/refactor-plan/analyze_dependencies.py` (`import os`) và viết integration test xác minh đầu-cuối trên `agent`.
  - **Output Contract (`HandoffReady=false`):** Bàn giao cho `TASK-8-M03, TASK-8-M04` (Biên lai `EV-SUB-TASK-8-M02-PENDING`)
  - **Definition of Done:** go test -race ./...

# 08 Tasks Checklist (Sub-Task Execution Unit): TASK-8-M04 — Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `agent` (`evidence_run.py`)
- **Parent Task:** `TASK-8` | **Wave:** `Wave 1` | **Isolated Branch:** `task/TASK-8/task-8-m04`
- [ ] **TASK-8-M04 (agent):**
  - **Context (`file:line`):** `agent/tooling/evidence_run.py:39 (def _git(repo: Path, *args: str) -> Optional[str]:)`
  - **Input Contract (`UpstreamReady=false`):** Kết nối dây dẫn nghiệp vụ tại `tooling/evidence_run.py` (`def _git(repo: Path, *args: str) -> Optional[str]:`) và viết integration test xác minh đầu-cuối trên `agent`.
  - **Output Contract (`HandoffReady=false`):** Bàn giao cho `Wave Cuối (Bàn giao kết quả tổng cho Task cha TASK-8)` (Biên lai `EV-SUB-TASK-8-M04-PENDING`)
  - **Definition of Done:** go test -race ./...

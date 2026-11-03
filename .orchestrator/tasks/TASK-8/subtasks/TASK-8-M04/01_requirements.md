---
tier: T2
parent_task: TASK-8
sub_task_id: TASK-8-M04
wave_index: 1
target_repo: /Users/trainguyen/Documents/work/agent
isolated_branch: task/TASK-8/task-8-m04
---
# 01 Requirements: TASK-8-M04 — Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `agent` (`evidence_run.py`)
- **Parent Task:** `TASK-8`
- **Wave:** `Wave 1`
- **Target Repo:** `/Users/trainguyen/Documents/work/agent`
- **Context (`file:line`):** `agent/tooling/evidence_run.py:39 (def _git(repo: Path, *args: str) -> Optional[str]:)`
- **Input Contract Ready:** `false`
- **Upstream Dependencies:** `TASK-8-M01, TASK-8-M02`
- **Downstream Consumers:** `Wave Cuối (Bàn giao kết quả tổng cho Task cha TASK-8)`
- **Test / DoD:** go test -race ./...

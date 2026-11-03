# 08 Tasks Checklist (Sub-Task Execution Unit): TASK-1-M01 — Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `agent` (`query_db.go`)
- **Parent Task:** `TASK-1` | **Wave:** `Wave 0` | **Isolated Branch:** `task/TASK-1/task-1-m01`
- [ ] **TASK-1-M01 (agent):**
  - **Context (`file:line`):** `agent/memory/workspaces/ReconHealStaleReport copy/query_db.go:12 (type MappingRuleV2 struct {)`
  - **Input Contract (`UpstreamReady=true`):** Cập nhật logic tầng dữ liệu & nghiệp vụ tại điểm chạm nguồn `memory/workspaces/ReconHealStaleReport copy/query_db.go` (`type MappingRuleV2 struct {`) trong worktree cô lập của `agent`.
  - **Output Contract (`HandoffReady=false`):** Bàn giao cho `TASK-1-M03, TASK-1-M04` (Biên lai `EV-SUB-TASK-1-M01-PENDING`)
  - **Definition of Done:** go test -race ./...

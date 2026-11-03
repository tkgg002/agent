# 08 Tasks Checklist (Sub-Task Execution Unit): TASK-11-M01 — Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `cdc-cms-service` (`list_latest_reports.go`)
- **Parent Task:** `TASK-11` | **Wave:** `Wave 0` | **Isolated Branch:** `task/TASK-11/task-11-m01`
- [ ] **TASK-11-M01 (cdc-cms-service):**
  - **Context (`file:line`):** `cdc-cms-service/internal/app/queries/recon/list_latest_reports.go:9 (type ListLatestReportsQuery struct{})`
  - **Input Contract (`UpstreamReady=true`):** Cập nhật logic tầng dữ liệu & nghiệp vụ tại điểm chạm nguồn `internal/app/queries/recon/list_latest_reports.go` (`type ListLatestReportsQuery struct{}`) trong worktree cô lập của `cdc-cms-service`.
  - **Output Contract (`HandoffReady=false`):** Bàn giao cho `TASK-11-M03, TASK-11-M04` (Biên lai `EV-SUB-TASK-11-M01-PENDING`)
  - **Definition of Done:** go test -race ./...

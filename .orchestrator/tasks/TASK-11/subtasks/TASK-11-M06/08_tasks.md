# 08 Tasks Checklist (Sub-Task Execution Unit): TASK-11-M06 — Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `centralized-data-service` (`reconciliation_report.go`)
- **Parent Task:** `TASK-11` | **Wave:** `Wave 2` | **Isolated Branch:** `task/TASK-11/task-11-m06`
- [ ] **TASK-11-M06 (centralized-data-service):**
  - **Context (`file:line`):** `centralized-data-service/internal/model/recon/reconciliation_report.go:8 (type ReconciliationReport struct {)`
  - **Input Contract (`UpstreamReady=false`):** Kết nối dây dẫn nghiệp vụ tại `internal/model/recon/reconciliation_report.go` (`type ReconciliationReport struct {`) và viết integration test xác minh đầu-cuối trên `centralized-data-service`.
  - **Output Contract (`HandoffReady=false`):** Bàn giao cho `Wave Cuối (Bàn giao kết quả tổng cho Task cha TASK-11)` (Biên lai `EV-SUB-TASK-11-M06-PENDING`)
  - **Definition of Done:** go test -race ./...

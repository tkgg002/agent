# 08 Tasks Checklist (Sub-Task Execution Unit): TASK-11-M02 — Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `cdc-cms-service` (`audit_dto.go`)
- **Parent Task:** `TASK-11` | **Wave:** `Wave 0` | **Isolated Branch:** `task/TASK-11/task-11-m02`
- [ ] **TASK-11-M02 (cdc-cms-service):**
  - **Context (`file:line`):** `cdc-cms-service/internal/api/dto/audit_dto.go:3 (type QASummaryResponse struct {)`
  - **Input Contract (`UpstreamReady=true`):** Kết nối dây dẫn nghiệp vụ tại `internal/api/dto/audit_dto.go` (`type QASummaryResponse struct {`) và viết integration test xác minh đầu-cuối trên `cdc-cms-service`.
  - **Output Contract (`HandoffReady=false`):** Bàn giao cho `TASK-11-M03, TASK-11-M04` (Biên lai `EV-SUB-TASK-11-M02-PENDING`)
  - **Definition of Done:** go test -race ./...

# 08 Tasks Checklist (Sub-Task Execution Unit): TASK-11-M05 — Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `centralized-data-service` (`metadata_registry_test_helpers.go`)
- **Parent Task:** `TASK-11` | **Wave:** `Wave 2` | **Isolated Branch:** `task/TASK-11/task-11-m05`
- [ ] **TASK-11-M05 (centralized-data-service):**
  - **Context (`file:line`):** `centralized-data-service/internal/service/source/metadata_registry_test_helpers.go:13 (type MetadataRegistryServiceForTestOpts struct {)`
  - **Input Contract (`UpstreamReady=false`):** Cập nhật logic tầng dữ liệu & nghiệp vụ tại điểm chạm nguồn `internal/service/source/metadata_registry_test_helpers.go` (`type MetadataRegistryServiceForTestOpts struct {`) trong worktree cô lập của `centralized-data-service`.
  - **Output Contract (`HandoffReady=false`):** Bàn giao cho `Wave Cuối (Bàn giao kết quả tổng cho Task cha TASK-11)` (Biên lai `EV-SUB-TASK-11-M05-PENDING`)
  - **Definition of Done:** go test -race ./...

# Tasks: GORM DB Tracing Context Propagation

- [ ] Phase 1: Research & Audit
  - [x] Rà soát và liệt kê toàn bộ các file/function gọi query GORM không có context
  - [x] Khởi tạo workspace documents (`01_requirements`, `05_progress`, `08_tasks`)
  - [ ] Lập Implementation Plan và xin phê duyệt từ User

- [ ] Phase 2: Implementation (Sau khi được approve)
  - [ ] Cập nhật `internal/service/shadow/schema_adapter.go`
  - [ ] Cập nhật `internal/service/source/bridge_service.go`
  - [ ] Cập nhật `internal/service/governance/activity_logger.go`
  - [ ] Cập nhật `internal/service/governance/masking_service.go`
  - [ ] Cập nhật `internal/service/governance/schema_validator.go`
  - [ ] Cập nhật `internal/service/governance/partition_dropper.go`
  - [ ] Cập nhật `internal/service/recon/recon_engine_segment_b.go`
  - [ ] Cập nhật các handler (`batch_buffer.go`, `recon_sysops_handler.go`, `server_scheduler.go`)

- [ ] Phase 3: Verification & Walkthrough
  - [ ] Chạy build và unit tests toàn bộ handler
  - [ ] Đảm bảo toàn bộ tests PASS
  - [ ] Viết walkthrough.md báo cáo kết quả

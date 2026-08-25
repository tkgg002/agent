# Tasks: Fix Log Trace Connection

- [x] Phase 1: Research & Audit
  - [x] Đọc hiến pháp GEMINI.md và bài học lessons.md
  - [x] Khởi tạo workspace documents (`01_requirements`, `05_progress`, `08_tasks`)
  - [x] Rà soát và lập danh sách chi tiết các files cần sửa đổi trong `centralized-data-service`
  - [x] Lập Implementation Plan và xin phê duyệt từ User

- [x] Phase 2: Implementation (Sau khi có approval)
  - [x] Cập nhật `internal/handler/base/base_handler.go` để hỗ trợ truyền `ctx` và dùng `observability.Ctx(ctx, h.Logger)`
  - [x] Cập nhật `internal/handler/source/bridge_handler.go` và `bridge_mongo.go`
  - [x] Cập nhật `internal/handler/recon` (recon_execute_heal_handler.go, recon_check_handler.go, recon_sysops_handler.go, recon_heal_fetch.go, recon_job_handler.go)
  - [x] Cập nhật `internal/handler/shadow` (schema_ddl_handler.go, batch_transform_handler.go)
  - [x] Cập nhật `internal/handler/master/transmute_handler.go`
  - [x] Cập nhật `internal/handler/scan/scan_handler.go`
  - [x] Cập nhật `internal/handler/orchestration/snapshot_runner_handler.go`

- [x] Phase 3: Verification & Walkthrough
  - [x] Chạy static check / compile tests để verify code không lỗi cú pháp
  - [x] Chạy unit tests liên quan để đảm bảo các mock và assertions hoạt động đúng
  - [x] Viết walkthrough.md báo cáo kết quả rà soát và khắc phục

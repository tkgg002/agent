# Tasks: Enable GORM OpenTelemetry Tracing

- [x] Phase 1: Research & Setup
  - [x] Rà soát cấu hình khởi tạo GORM DB trong cả 2 service
  - [x] Khởi tạo workspace documents (`01_requirements`, `05_progress`, `08_tasks`)
  - [x] Tạo Implementation Plan và xin phê duyệt từ User

- [x] Phase 2: Implementation (Sau khi được approve)
  - [x] Chạy `go get gorm.io/plugin/opentelemetry/tracing` ở `centralized-data-service`
  - [x] Chạy `go get gorm.io/plugin/opentelemetry/tracing` ở `cdc-cms-service`
  - [x] Đăng ký plugin `tracing.NewPlugin()` trong `centralized-data-service/pkgs/database/multi.go`
  - [x] Đăng ký plugin `tracing.NewPlugin()` trong `cdc-cms-service/pkgs/database/postgres.go`
  - [x] Đăng ký plugin `tracing.NewPlugin()` trong `centralized-data-service/cmd/admin-api/main.go`
  - [x] Đăng ký plugin với `tracing.WithoutQueryVariables()` đối với shadow và dest trong `multi.go`
  - [x] Cập nhật `postgres.go` ở `cdc-cms-service` nhận tham số `role` và cấu hình `tracing.WithoutQueryVariables()` cho shadow
  - [x] Cập nhật các nơi gọi `NewPostgresConnection` trong `cdc-cms-service` để truyền đúng `role`

- [x] Phase 3: Verification & Walkthrough
  - [x] Chạy static check và build verify (bao gồm cả admin-api và cdc-cms-service)
  - [x] Chạy unit tests của cả 2 repo (bao gồm cả package internal/admin)
  - [x] Cập nhật `walkthrough_gorm_otel_trace.md` báo cáo kết quả

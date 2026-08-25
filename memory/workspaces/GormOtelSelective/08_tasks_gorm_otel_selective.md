# Tasks: GORM OpenTelemetry Selective Tracing

- [x] Phase 1: Research & Setup
  - [x] Nghiên cứu cấu trúc Custom Sampler trong OpenTelemetry Go SDK
  - [x] Khởi tạo workspace documents (`01_requirements`, `05_progress`, `08_tasks`)
  - [x] Lập/Cập nhật Implementation Plan và xin phê duyệt từ User

- [x] Phase 2: Implementation (Sau khi được approve)
  - [x] Thêm struct `DBTraceSampler` và các helpers trong `pkgs/observability/trace_helpers.go` của cả 2 repo
  - [x] Đăng ký `DBTraceSampler` trong hàm `InitOtel` tại `pkgs/observability/otel.go` của cả 2 repo
  - [x] Đăng ký plugin GORM với option `tracing.WithoutMetrics()` tại `multi.go` và `postgres.go`
  - [x] Đánh dấu context bằng `observability.WithDBTraceModule(ctx, "module_name")` tại các handler NATS chính trong `centralized-data-service`
  - [x] Thêm `"cdc"` vào `enabledDBTraceModules` whitelist ở cả 2 service
  - [x] Bọc `"cdc"` module cho HTTP requests trong `cdc-cms-service/internal/middleware/http_tracer.go`
  - [x] Bọc `"cdc"` module cho HTTP requests trong `centralized-data-service/internal/admin/otel_middleware.go`

- [x] Phase 3: Verification & Walkthrough
  - [x] Chạy build và unit tests toàn bộ hệ thống
  - [x] Đảm bảo toàn bộ tests PASS
  - [x] Cập nhật `walkthrough_gorm_otel_selective.md` báo cáo kết quả

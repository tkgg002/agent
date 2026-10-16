# Kế hoạch Triển khai Chi tiết của AI (Implementation Plan)

- Mục tiêu: Thực thi mã nguồn theo đúng thiết kế trong `09_tasks_solution_nats_monitor_ui.md`.
- Phạm vi: `cdc-cms-service` và `cdc-cms-web`.
- Các bước:
  1. Viết file `cdc-cms-service/internal/api/system/nats_monitor_handler.go`.
  2. Cập nhật `cdc-cms-service/internal/router/router.go` (khai báo struct handler và đăng ký route).
  3. Cập nhật `cdc-cms-service/internal/server/server.go` (khởi tạo handler).
  4. Biên dịch và kiểm tra cú pháp Go của `cdc-cms-service`.
  5. Viết component `cdc-cms-web/src/pages/NatsMonitor.tsx`.
  6. Cập nhật `cdc-cms-web/src/App.tsx` (thêm lazy load, menu item, route).
  7. Kiểm tra type check frontend (`npx tsc --noEmit`).
  8. Kiểm thử tích hợp bằng curl và đối soát dữ liệu với NATS thật.

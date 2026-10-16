# Context: NATS & JetStream Monitor UI

## Mục tiêu
Cung cấp giao diện người dùng trực quan trên CDC CMS Web (`cdc-cms-web`) để giám sát toàn diện hệ thống NATS và NATS JetStream (trạng thái server, lưu lượng messages, dung lượng streams, consumers, pending acks, và các active client connections).

## Phạm vi tác động
- Backend: `cdc-cms-service` (Go Fiber)
  - Thêm handler `NATSMonitorHandler` tại `internal/api/system/nats_monitor_handler.go`.
  - Cấu hình route `/api/system/nats/overview` và `/api/system/nats/streams` trong `internal/router/router.go`.
  - Wire handler trong `internal/server/server.go`.
  - Tận dụng `cfg.System.NatsMonitorURL` (mặc định `http://localhost:18222`).
- Frontend: `cdc-cms-web` (React 18 / Vite / Ant Design)
  - Thêm trang `src/pages/NatsMonitor.tsx`.
  - Bổ sung hook / service gọi `/api/system/nats/overview`.
  - Cập nhật menu điều hướng và React router trong `src/App.tsx`.

# Kế hoạch Triển khai Cao tầng: NATS Monitor UI

## Skills & Công cụ sử dụng
- `golang-patterns`
- `react-patterns`
- `api-design`

## Các giai đoạn thực hiện (Phases)

### Phase 1: Backend Aggregator (`cdc-cms-service`)
- [ ] 1.1 Tạo `internal/api/system/nats_monitor_handler.go` thực hiện query `/varz`, `/jsz?streams=true&config=true&consumers=true`, `/connz` từ `cfg.System.NatsMonitorURL`.
- [ ] 1.2 Đăng ký `NATSMonitor` trong `HandlerGroup` tại `internal/router/router.go`.
- [ ] 1.3 Đăng ký route `/api/system/nats/overview` trong `SetupRoutes` tại `internal/router/router.go`.
- [ ] 1.4 Khởi tạo `NATSMonitorHandler` trong `internal/server/server.go`.
- [ ] 1.5 Kiểm thử unit/curl endpoint `/api/system/nats/overview`.

### Phase 2: Frontend UI (`cdc-cms-web`)
- [ ] 2.1 Xây dựng `src/pages/NatsMonitor.tsx` với Ant Design (KPI Cards, Streams Table, Consumers Drawer, Active Connections Table).
- [ ] 2.2 Đăng ký trang vào `src/App.tsx` (Route `/nats-monitor` và Menu sidebar trong nhóm "Operate" hoặc "Advanced").
- [ ] 2.3 Verify giao diện, trạng thái loading, error handling và polling interval.

### Phase 3: Kiểm thử toàn trình & Verification
- [ ] 3.1 Khởi động / Rebuild `cdc-cms-service` và verify response thật.
- [ ] 3.2 Kiểm tra `cdc-cms-web` build không có lỗi TypeScript (`tsc --noEmit`).
- [ ] 3.3 Verify trải nghiệm người dùng trên trình duyệt (thực tế từ `http://localhost:5173/nats-monitor`).

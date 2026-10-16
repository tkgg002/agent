# Báo cáo Thay đổi (Change Report): NATS Monitor UI

## 1. Tổng quan (Overview)
Đã triển khai thành công tính năng **NATS & JetStream Monitor** trực quan cho người dùng theo dõi trên CMS Web:
- Bổ sung tầng backend API tổng hợp metric NATS từ cổng HTTP monitor `http://localhost:18222`.
- Xây dựng giao diện trang `NATS Monitor` với đầy đủ thẻ KPI, bảng Streams, Drawer Consumers, bảng Active Client Connections, tính năng Auto-refresh và xem Raw Telemetry.

## 2. Những file đã thay đổi & Số dòng code (Files Changed & Lines of Code)

### Backend (`cdc-cms-service`)
1. **`internal/api/system/nats_monitor_handler.go`** (Tạo mới — 104 dòng code):
   - Tổng hợp song song non-blocking (goroutines) từ `/varz`, `/jsz?streams=true&config=true&consumers=true`, `/connz`.
   - Timeout 3s, error handling trả về status `down` khi NATS không phản hồi.
2. **`internal/api/system/nats_monitor_handler_test.go`** (Tạo mới — 97 dòng code):
   - Kiểm thử unit test kịch bản Success và Down với mock HTTP server.
3. **`internal/router/router.go`** (Chỉnh sửa — +6 dòng code):
   - Thêm `NATSMonitor` vào `HandlerGroup.System`.
   - Đăng ký endpoint `/api/system/nats/overview` và `/api/v1/system/nats/overview`.
4. **`internal/server/server.go`** (Chỉnh sửa — +1 dòng code):
   - Khởi tạo `NATSMonitorHandler` với `cfg.System.NatsMonitorURL`.

### Frontend (`cdc-cms-web`)
1. **`src/pages/NatsMonitor.tsx`** (Tạo mới — 1,044 dòng code):
   - Thiết kế giao diện Ant Design v5 với 4 KPI cards tổng quan (Uptime, RAM, Active Conns, JetStream storage).
   - Bảng Streams hiển thị đầy đủ thông tin: Subjects, Total Messages, Size, Consumers, First/Last Seq.
   - Drawer xem chi tiết từng Consumer (Ack pending cảnh báo đỏ, Redelivery count, Waiting).
   - Bảng Active Client Connections (CID, Name/User, IP:Port, Subscriptions, Throughput).
   - Chế độ Auto-Refresh (3s, 5s, 10s, 30s, Off), Manual Refresh, Copy to clipboard và xem Raw JSON.
2. **`src/App.tsx`** (Chỉnh sửa — +8 dòng code):
   - Lazy load `NatsMonitor`.
   - Bổ sung menu item `NATS Monitor` kèm icon `CloudServerOutlined` trong nhóm menu Operate.
   - Thêm `<Route path="/nats-monitor" element={<NatsMonitor />} />`.

## 3. Kiểm định & Trạng thái (Verification Status)
- `go test -v ./internal/api/system -run TestNATSMonitorHandler`: **PASS** (100%).
- `go build ./cmd/server`: **PASS** (100% sạch lỗi biên dịch).
- `npm run build` (TypeScript check & Vite bundling): **PASS** (100% sạch lỗi, chunk được build thành công).

# Implementation Plan: CDC Worker Queue Monitoring

Cung cấp giải pháp trực quan để theo dõi các tiến trình queue, batch buffer và trạng thái xử lý sự kiện trong CDC Worker.

## User Review Required

> [!IMPORTANT]
> **Real-time Monitoring**: Giải pháp sẽ sử dụng Polling từ Frontend (React) để lấy dữ liệu từ API của Worker. Trong tương lai có thể nâng cấp lên WebSocket hoặc nats.go/JetStream monitoring chuyên sâu.
> **Metric Storage**: Các chỉ số (processed/failed) hiện tại sẽ được lưu in-memory (sẽ reset khi khởi động lại). Nếu cần lưu trữ lịch sử lâu dài, cần tích hợp Prometheus + Grafana.

## Proposed Changes

### [Component] CDC Worker (Backend)

#### [MODIFY] [consumer_pool.go](file:///Users/trainguyen/Documents/work/centralized-data-service/internal/handler/consumer_pool.go)
- Thêm `atomic.Uint64` cho `processed`, `failed`, `pending`.
- Thêm phương thức `GetStats()` trả về struct chứa các chỉ số này.

#### [MODIFY] [batch_buffer.go](file:///Users/trainguyen/Documents/work/centralized-data-service/internal/handler/batch_buffer.go)
- Thêm phương thức `GetStatus()` trả về size hiện tại của buffer và thời gian flush cuối cùng.

#### [MODIFY] [worker_server.go](file:///Users/trainguyen/Documents/work/centralized-data-service/internal/server/worker_server.go)
- Đăng ký endpoint: `GET /api/v1/internal/stats`.
- Tổng hợp dữ liệu từ `ConsumerPool` và `BatchBuffer` để trả về cho Frontend.

### [Component] CMS Frontend (React)

#### [NEW] [QueueMonitoring.tsx](file:///Users/trainguyen/Documents/work/cdc-cms-web/src/pages/QueueMonitoring.tsx)
- Giao diện hiển thị Dashboard các chỉ số:
  - Tốc độ xử lý (Messages/sec).
  - Tình trạng Goroutine Pool.
  - Trạng thái Batch Buffer (Database Persistence lag).
- Sử dụng Ant Design `Progress` và `Statistic` components.

#### [MODIFY] [App.tsx](file:///Users/trainguyen/Documents/work/cdc-cms-web/src/App.tsx)
- Thêm menu item "Queue Monitor" vào Sidebar.

## Open Questions

- Bạn muốn xem các chỉ số theo thời gian thực (Real-time chart) hay chỉ cần con số tổng hợp (Pulse check)?
- Có cần lọc (filter) chỉ số theo bảng (Table-specific stats) không?

## Verification Plan

### Automated Tests
- Kiểm tra Endpoint `/api/v1/internal/stats` trả về JSON đúng cấu trúc qua `curl`.

### Manual Verification
1. Chạy Worker và CMS.
2. Gửi một lượng lớn events giả lập.
3. Quan sát biểu đồ/con số trên giao diện "Queue Monitor" thay đổi tương ứng.

# Implementation Plan: CDC Port Harmonization & Ghost Process Cleanup

Giải quyết triệt để lỗi "address already in use" và đồng bộ hóa các cổng dịch vụ (ports) trong toàn bộ hệ thống CDC để đảm bảo tính nhất quán giữa Backend, Worker và Frontend.

## User Review Required

> [!IMPORTANT]
> **Dọn dẹp tiến trình**: Tôi sẽ thực hiện `kill` các tiến trình đang chiếm dụng cổng 8081, 8082, 8090 (các tiến trình "ghost" từ các phiên làm việc trước).
> **Đồng bộ hóa Port**: Đề xuất bộ cổng "Hợp lý" (Sequential):
> - **8081**: CDC Auth Service
> - **8082**: CDC Worker Service
> - **8083**: CDC CMS Service (Thay cho 8090 để đưa về dải 808x tập trung, tránh xung đột Docker 8080).
> - **3000**: CDC CMS Web

## Proposed Changes

### [Component] Infrastructure Cleanup

#### [ACTION] Kill Process
- Giải phóng các cổng đang bị treo: `:8081`, `:8082`, `:8090`.

### [Component] Backend / Worker Configs

#### [MODIFY] [cdc-cms-service/config/config-local.yml](file:///Users/trainguyen/Documents/work/cdc-cms-service/config/config-local.yml)
- Đổi port từ `:8090` sang `:8083`.

#### [MODIFY] [cdc-auth-service/config/config-local.yml](file:///Users/trainguyen/Documents/work/cdc-auth-service/config/config-local.yml)
- Đảm bảo port là `:8081`.

#### [MODIFY] [centralized-data-service/config/config-local.yml](file:///Users/trainguyen/Documents/work/centralized-data-service/config/config-local.yml)
- Đảm bảo port là `:8082`.

### [Component] Frontend Config

#### [MODIFY] [cdc-cms-web/.env](file:///Users/trainguyen/Documents/work/cdc-cms-web/.env)
- Cập nhật bộ biến môi trường khớp với backend mới:
  - `VITE_AUTH_API_URL=http://localhost:8081`
  - `VITE_WORKER_API_URL=http://localhost:8082`
  - `VITE_CMS_API_URL=http://localhost:8083`

## Open Questions

- Bạn có muốn giữ nguyên cổng 8090 cho CMS không? Nếu giữ 8090, tôi sẽ chỉ dọn dẹp port và đồng bộ Frontend về 8090. Tuy nhiên dùng 8083 sẽ giúp hệ thống gọn gàng hơn (Dải 8081, 8082, 8083).

## Verification Plan

### Automated Tests
- Chạy `lsof -i :8081 -i :8082 -i :8083` để xác nhận không còn tiến trình nào chiếm dụng trước khi khởi động.

### Manual Verification
1. Khởi động 3 service: Auth, Worker, CMS.
2. Kiểm tra log khởi động để xác nhận tất cả đều "Listen" thành công trên các cổng tương ứng.
3. Kiểm tra CMS Frontend (Queue Monitor và Schema Approval) để xác nhận kết nối API thông suốt.

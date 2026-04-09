# Implementation Plan: Airbyte Sync Trigger & Monitoring

Cung cấp khả năng kích hoạt đồng bộ (Force Sync) từ giao diện CMS và hiển thị trạng thái các Job đồng bộ của Airbyte để người dùng theo dõi tiến trình.

## User Review Required

> [!IMPORTANT]
> **Giới hạn Airbyte OSS**: Airbyte phiên bản Open Source có cơ chế hàng đợi Job riêng. Nếu một Job đang chạy, việc "Force Sync" có thể sẽ bị từ chối hoặc đưa vào hàng đợi tùy theo cấu hình Airbyte.
> **Hiển thị Monitoring**: Trang Monitoring hiện tại chỉ hiển thị thông số Worker (Internal). Tôi đề xuất bổ sung thêm danh sách 5-10 Job đồng bộ gần nhất của Airbyte để người dùng biết dữ liệu đang được đẩy về hay không.

## Proposed Changes

### [Component] cdc-cms-service (Backend)

#### [MODIFY] [client.go](file:///Users/trainguyen/Documents/work/cdc-cms-service/pkgs/airbyte/client.go)
- Thêm phương thức `ListJobs(ctx, connectionID)` để lấy danh sách các job đồng bộ gần nhất.

#### [MODIFY] [registry_handler.go](file:///Users/trainguyen/Documents/work/cdc-cms-service/internal/api/registry_handler.go)
- Thêm endpoint `POST /api/registry/:id/sync`: Gọi `airbyteClient.TriggerSync`.
- Thêm endpoint `GET /api/registry/:id/jobs`: Trả về danh sách jobs từ Airbyte.

#### [MODIFY] [router.go](file:///Users/trainguyen/Documents/work/cdc-cms-service/internal/router/router.go)
- Đăng ký các route mới cho Registry Sync và Job Status.

### [Component] cdc-cms-web (Frontend)

#### [MODIFY] [TableRegistry.tsx](file:///Users/trainguyen/Documents/work/cdc-cms-web/src/pages/TableRegistry.tsx)
- Bổ sung nút **Sync Now** trong cột Action hoặc trong Sync Status Indicator.

#### [MODIFY] [QueueMonitoring.tsx](file:///Users/trainguyen/Documents/work/cdc-cms-web/src/pages/QueueMonitoring.tsx)
- Thêm một Card hiển thị "Recent Airbyte Sync Jobs" (Status: RUNNING, SUCCEEDED, FAILED).

## Open Questions

- Bạn có muốn hệ thống tự động thông báo (Browser Notification) khi một Airbyte Job bị thất bại không?

## Verification Plan

### Automated Tests
- Kiểm tra các endpoint mới (`/sync`, `/jobs`) thông qua Postman hoặc Curl.

### Manual Verification
1. Truy cập `Registry`, nhấn nút "Sync Now".
2. Chuyển sang trang "Monitoring", kiểm tra xem Job mới có xuất hiện trong danh sách "Recent Airbyte Syncs" không.
3. Kiểm tra trạng thái Job chuyển từ RUNNING sang SUCCEEDED.

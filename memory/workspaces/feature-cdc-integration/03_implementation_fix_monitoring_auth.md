# Implementation Plan: Fixing Airbyte 403/500 Monitoring Error

Giải quyết vấn đề phân quyền (403) khi lấy mã Workspace và lỗi 500 khi truy vấn danh sách Job Airbyte trên giao diện Monitoring. 

## User Review Required

> [!IMPORTANT]
> **Nguyên nhân**: API Airbyte tại môi trường hiện tại từ chối lệnh liệt kê Workspace (`/workspaces/list`). Mã Workspace mặc định (fallback) tôi đã hardcode không khớp với môi trường của bạn, dẫn đến lệnh lấy Job (`/jobs/list`) trả về mã lỗi 500.
> **Giải pháp**: Thay vì cố gắng lấy toàn bộ Job của Workspace (yêu cầu WorkspaceID), tôi sẽ thực hiện truy vấn Job dựa trên danh sách các Connection ID (bảng) mà chúng ta đang quản lý trong CMS Registry.

## Proposed Changes

### [Component] cdc-cms-service (Backend)

#### [MODIFY] [client.go](file:///Users/trainguyen/Documents/work/cdc-cms-service/pkgs/airbyte/client.go)
- Thay đổi logic `ListJobs`: Nếu gọi Global (ID trống), hệ thống sẽ không cố gắng tìm `workspaceId`. 
- Thêm cơ chế cho phép truyền vào một danh sách các Connection IDs để lấy Job thu hẹp.

#### [MODIFY] [airbyte_handler.go](file:///Users/trainguyen/Documents/work/cdc-cms-service/internal/api/airbyte_handler.go)
- (Tùy chọn) Nhúng `RegistryRepository` vào để lấy danh sách các Connection ID đang active.
- Thực hiện fetch Jobs song song (hoặc tuần tự nếu ít) cho các Connection ID này.

#### [FIX] [api.ts](file:///Users/trainguyen/Documents/work/cdc-cms-web/src/services/api.ts)
- (Đã làm ở bước trước) Đảm bảo kết nối đúng Port 8083.

## Open Questions

- Bạn có biết Workspace ID trên Airbyte UI của mình là gì không? (Nó nằm trong URL: `/workspaces/<id>`). Nếu cung cấp được, tôi có thể fix cứng trực tiếp để tối ưu tốc độ.

## Verification Plan

### Automated Tests
- Chạy Curl gọi `/api/airbyte/jobs` và kiểm tra xem còn trả về 500/403 không.

### Manual Verification
1. Mở trang Monitoring.
2. Kiểm tra phần "Airbyte Raw Data Sync Status" đã hiển thị các Job gần nhất của các bảng đang có trong Registry chưa.

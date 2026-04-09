# Phase 1.6: Airbyte Orchestration & Monitoring

> **Workspace**: feature-cdc-integration
> **Phase**: 1.6 of 2 (Post-Phase 1.5)
> **Focus**: Tự động hóa việc đăng ký bảng vào Airbyte và theo dõi trạng thái đồng bộ thực tế từ CMS.

---

## Overview

Phase 1.6 giải quyết vấn đề "Manual Config" trong Airbyte. Trước đây, khi thêm một bảng vào `cdc_table_registry`, người dùng phải vào Airbyte UI để:
1. Refresh source schema.
2. Tick chọn table mới.
3. Update connection.

Hiện tại, hệ thống tự động hóa luồng này thông qua Airbyte API.

## Technical Components

### 1. Airbyte Client (Go)
Nâng cấp `pkg/airbyte/client.go` để hỗ trợ các method quản trị:
- `GetConnection(ctx, connectionID)`: Lấy cấu hình catalog hiện tại.
- `UpdateConnection(ctx, req)`: Cập nhật danh sách stream (table) cần sync.
- `GetJobStatus(ctx, connectionID)`: Kiểm tra trạng thái sync cuối cùng.

### 2. Auto-Registration Logic
Trong `RegistryHandler.Register` (CMS API):
1. **Discover Schema**: Gọi Airbyte Refresh Source để nhận diện bảng mới ở source DB.
2. **Catalog Update**: 
   - Lấy `connection_id` của table (từ group/source).
   - Fetch catalog hiện tại.
   - Thêm stream mới vào danh sách `sync_mode=incremental` + `destination_sync_mode=append_dedup`.
   - Update Connection qua API.
3. **Trigger Sync**: (Tùy chọn) Khởi chạy sync ngay lập tức nếu cần.

### 3. Sync Status Monitoring
- **Backend**: API `GET /api/registry/:id/sync-status` gọi Airbyte để lấy trạng thái job gần nhất (SUCCEEDED, FAILED, RUNNING).
- **Frontend**: Hiển thị `AirbyteStatusBadge` trong danh sách Registry:
  - 🟢 **Active**: Sync thành công gần đây.
  - 🔴 **Inactive/Failed**: Có lỗi sync hoặc chưa được cấu hình.
  - 🟡 **Syncing**: Đang trong quá trình đồng bộ.

## Implementation Details

### API Endpoints (CMS Service)
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/registry/:id/sync-status` | Lấy trạng thái từ Airbyte Job |
| POST | `/api/registry/:id/sync` | Trigger manual sync cho table |

### Frontend UI
- Component: `src/components/AirbyteStatusBadge.tsx`
- Tích hợp vào `TableRegistryList` columns.

## Verification
- [x] Đăng ký bảng mới → Check Airbyte UI thấy table tự động được chọn.
- [x] Bấm "Sync Now" trên CMS → Airbyte Job khởi chạy.
- [x] Sửa lỗi kết nối Airbyte → Badge chuyển sang trạng thái 🔴 FAILED.

---
> **Next Step**: Phase 2 - Dynamic Mapper Integration.

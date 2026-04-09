# Implementation Plan: Comprehensive Feature Audit & Synchronization

Rà soát và đảm bảo mọi thao tác trên giao diện CMS (Cổng 5173) có tác động đến việc đồng bộ dữ liệu đều được phản ánh chính xác xuống tầng hạ tầng (Airbyte/Worker).

## User Review Required

> [!IMPORTANT]
> **Đồng bộ Sync Engine**: Hiện tại khi đổi Engine (ví dụ từ Airbyte sang Debezium), Airbyte vẫn có thể đang chạy ngầm nếu stream đó vẫn được `Selected`. Tôi đề xuất tự động `Un-select` trên Airbyte nếu người dùng chuyển hẳn sang Debezium.
> **Duyệt Schema**: Khi duyệt một trường mới, có nên ép Airbyte chạy `Refresh Schema` ngay lập tức để đảm bảo dữ liệu mới được đẩy về không? (Mặc định Airbyte có độ trễ quét schema).

## Proposed Changes

### [Component] cdc-cms-service (Backend)

#### [MODIFY] [registry_handler.go](file:///Users/trainguyen/Documents/work/cdc-cms-service/internal/api/registry_handler.go)
- Mở rộng logic `Update`:
    - Nếu `SyncEngine` thay đổi:
        - Nếu chuyển sang loại không chứa `airbyte` (ví dụ: `debezium`): Gọi Airbyte API đặt `Selected = false`.
        - Nếu chuyển sang loại chứa `airbyte` (ví dụ: `airbyte` hoặc `both`): Gọi Airbyte API đặt `Selected = true` (kèm theo trạng thái `IsActive`).
    - Hợp nhất logic này vào hàm `syncRegistryStateToAirbyte` chung.

#### [MODIFY] [schema_change_handler.go](file:///Users/trainguyen/Documents/work/cdc-cms-service/internal/api/schema_change_handler.go)
- Kiểm tra logic `Approve`:
    - Sau khi `Approve`, nếu table đó dùng Airbyte, gửi lệnh `Refresh Schema` tới Airbyte để cập nhật catalog sớm nhất có thể.

### [Component] cdc-cms-web (Frontend)
- Rà soát các nút bấm/form:
    - **Standardize**: Đã dùng NATS (OK).
    - **Discover**: Đã dùng NATS (OK).
    - **Mapping Rules**: Đã dùng NATS (OK).

## Open Questions

- Bạn có muốn hệ thống tự động Reset Sync (Full Refresh) trên Airbyte khi cấu hình Mapping thay đổi lớn không?

## Verification Plan

### Automated Tests
- Chạy lại bộ test API cho Registry Update.

### Manual Verification
1. Đổi Sync Engine từ `Airbyte` sang `Debezium` trên UI -> Kiểm tra Airbyte Connection xem stream đã bị bỏ chọn chưa.
2. Approve một trường mới trong Schema Changes -> Kiểm tra log CMS xem có lệnh `Refresh Schema` gửi tới Airbyte không.

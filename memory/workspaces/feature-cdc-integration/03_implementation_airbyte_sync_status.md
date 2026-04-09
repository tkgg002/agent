# Implementation Plan: Airbyte Sync Status Synchronization

Đảm bảo khi bật/tắt trạng thái `active` của một bảng trong CMS, hệ thống sẽ tự động cập nhật cấu hình `Selected` của stream tương ứng trong Airbyte Connection để bắt đầu hoặc dừng đồng bộ dữ liệu.

## User Review Required

> [!IMPORTANT]
> **Độ trễ đồng bộ**: Việc cập nhật Airbyte qua API có thể mất vài giây. UI sẽ hiển thị thông báo thành công sau khi ghi DB, việc đồng bộ Airbyte sẽ chạy đồng bộ (blocking) trong request hiện tại để đảm bảo tính nhất quán.
> **Điều kiện tiên quyết**: Bảng phải có `AirbyteConnectionID` hợp lệ (đã được liên kết khi Register). Nếu chưa có, hệ thống sẽ chỉ cập nhật DB và ghi log cảnh báo.

## Proposed Changes

### [Component] cdc-cms-service (Backend)

#### [MODIFY] [registry_handler.go](file:///Users/trainguyen/Documents/work/cdc-cms-service/internal/api/registry_handler.go)
- Cập nhật phương thức `Update`:
    - Kiểm tra nếu `is_active` có sự thay đổi.
    - Nếu thay đổi và table đang sử dụng engine `airbyte` hoặc `both`:
        - Gọi `airbyteClient.GetConnection` để lấy catalog hiện tại.
        - Tìm stream tương ứng với `SourceTable`.
        - Cập nhật `streams[i].Config.Selected = newIsActive`.
        - Gọi `airbyteClient.UpdateConnection` để áp dụng thay đổi lên Airbyte.
- Thêm logic xử lý lỗi: Nếu Airbyte API lỗi, vẫn cho phép cập nhật DB nhưng trả về cảnh báo (hoặc lỗi tùy theo mức độ nghiêm trọng).

## Open Questions

- Bạn có muốn việc đồng bộ Airbyte chạy ngầm (asynchronous) để tránh làm chậm UI không? (Nếu chọn async, người dùng sẽ không biết ngay nếu Airbyte bị lỗi). Hiện tại tôi đề xuất chạy sync để đảm bảo chắc chắn Airbyte đã nhận lệnh.

## Verification Plan

### Automated Tests
- Kiểm tra logic cập nhật Airbyte thông qua mock client (nếu có unit test).

### Manual Verification
1. Truy cập `http://localhost:5173/registry`.
2. Chọn một bảng đang `Active`.
3. Tắt Switch `Active`.
4. Mở giao diện Airbyte (localhost:8000), kiểm tra Connection tương ứng xem Stream đã bị un-select chưa.
5. Bật lại Switch `Active` và kiểm tra Airbyte xem Stream đã được select lại chưa.

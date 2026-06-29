# Context: Fix Data Mapping And Security

## Problem Description
1. Field `data` trong source `bidv-connector-service.bank_requests` không hiển thị ở database đích (Shadow Database) sau khi thực hiện Snapshot. Lý do: Thiếu cấu hình mapping rule tương ứng cho field `data` trong bảng registry `cdc_system.mapping_rule_v2`.
2. PostgreSQL source connection của CDC worker không lấy được credentials (username, password) từ database mà chỉ builder DSN tĩnh từ host/port mà không có password. Cần refactor để lưu trữ password vào `options_json` trong `connection_registry` một cách an toàn và DSN builder sẽ nạp động từ đó.

## Architectural Changes & Design
- Sửa hàm `buildDSNFromFieldsPatched` trong `centralized-data-service` để tự động đọc `username`/`password` từ `options_json` khi build DSN PostgreSQL/MongoDB.
- Bổ sung interface `UpdateConnectionCredentials` trong repository của `cdc-cms-service` để lưu thông tin credentials vào `options_json` của connection tương ứng trong bảng `cdc_system.connection_registry`.
- Expose endpoint `PATCH /v1/registry/connections/:connection_code/credentials` trong `cdc-cms-service` để cho phép client/frontend update credentials.
- Cấu hình mapping rule `data` của `bank_requests` trực tiếp trong database registry để mapping hoạt động.

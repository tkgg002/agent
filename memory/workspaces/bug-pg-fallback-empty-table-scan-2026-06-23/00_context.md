# Context: PostgreSQL Fallback Scan for Empty Tables

## Problem Statement
Khi thực hiện luồng "Scan Fields" trên CMS, nếu shadow table của PostgreSQL rỗng hoặc chưa được tạo, hệ thống hiện tại không tự động sinh ra các Mapping Rules.
Trong khi đó, MongoDB đã có cơ chế fallback kết nối trực tiếp đến database nguồn để quét các fields.
Cần triển khai cơ chế fallback tương tự cho PostgreSQL (và các SQL sources khác như MySQL/MariaDB) bằng cách kết nối trực tiếp qua DSN nguồn và lấy thông tin cấu trúc cột từ `information_schema.columns`.

## Scope
- **centralized-data-service**:
  - Tích hợp fallback SQL trong hàm `ScanFieldsDebezium` ở `internal/handler/source/discover_handler.go`.
  - Triển khai hàm `scanFieldsSQLSource` và `processSQLDiscoveryCols` để kết nối trực tiếp database nguồn qua `SourceInferrer` (trong `discovery_utils.go`).
  - Cấu hình biến môi trường `CONNECTION_OVERRIDE_PG_DEV` trong file cấu hình/docker-compose để phục vụ local dev testing.

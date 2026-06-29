# Requirements: Fallback Default Schema từ Connection Registry

## 1. Yêu cầu của User
- Hiện tại khi chạy PostgreSQL Snapshot V2, hệ thống đang hardcode `schema := "public"` nếu `so.SourceSchema` nil hoặc rỗng.
- Cần thay đổi logic này: Fallback về trường `default_schema` trong bảng `cdc_system.connection_registry` (`conn.DefaultSchema`). Chỉ khi cả `so.SourceSchema` và `conn.DefaultSchema` đều rỗng mới fallback về `"public"`.
- Khi kết nối (connect) tới PostgreSQL nguồn, tự động chèn tham số `search_path` vào DSN nếu `DefaultSchema` được khai báo, giúp Postgres tự động trỏ đúng schema mặc định cho phiên làm việc.

## 2. Ranh giới hệ thống
- Áp dụng cho:
  - `snapshot_runner_handler.go` (lúc scan và query Postgres).
  - `metadata_registry_utils.go` (lúc build DSN Postgres).
- Không ảnh hưởng đến các DB engine khác (MongoDB, MySQL).

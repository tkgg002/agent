# Context: Postgres CDC Schema support verification & bugfix

## Problem Statement
Sau khi triển khai feature "postgresql schema support", cần xác minh và sửa các lỗi liên quan đến các luồng riêng của PostgreSQL:
1. Luồng **Scan Fields**: Phá hiện và bóc tách các fields từ cột `_raw_data` của PostgreSQL (bọc trong dạng Debezium JSON event `"after"`).
2. Luồng **Mapping Page**: Xem trang mapping fields có hoạt động đúng không khi PostgreSQL sử dụng schema tùy chỉnh.
3. Fix lỗi GORM JOIN trong `source_repo_gorm.go` khi join `source_object_registry` với `cdc_table_registry` dựa trên `source_database = source_db`. Postgres sử dụng `source_schema` để đại diện cho DB/schema thực tế trong CDC, do đó việc so sánh này bị lỗi.

## Scope
- Centralized Data Service:
  - `internal/service/source/scan_service.go`: Cập nhật logic scan `_raw_data` cho Postgres (bóc tách key bên trong `after` thay vì root key).
- CDC CMS Service:
  - `internal/infra/persistence/source/source_object_read_repo_gorm.go`: Fix câu SQL JOIN.
  - `internal/infra/persistence/source/source_repo_gorm.go`: Fix câu SQL JOIN trong `GetByRegistryID`.
- UI Web app (nếu cần): Xác minh việc hiển thị mapping page.

# Context: Lỗi đối soát export-jobs báo noop dù bị lệch 1 bản ghi

## 1. Hiện tượng
- Bảng đối soát của `export-jobs` (source: `centrallized-export-service.export-jobs`, shadow: `shadow_testexp.export_jobs`) đang bị lệch 1 bản ghi.
- Khi trigger API `recon-heal` cho bảng `export-jobs` thủ công hoặc tự động qua NATS, hệ thống ghi nhận `noop` và không thực hiện đồng bộ bản ghi bị lệch.

## 2. Các thành phần liên quan
- **Registry**: Bảng `cdc_system.cdc_table_registry` cấu hình metadata cho `export-jobs`.
- **Reconciliation Engine (centralized-data-service)**:
  - `internal/service/recon/recon_tier_b.go`
  - `internal/service/recon/recon_dest_query.go`
- **Database**:
  - MongoDB database: `centrallized-export-service` (hoặc tên chính xác có/không có typo).
  - Shadow PostgreSQL: schema `shadow_testexp`, table `export_jobs`.

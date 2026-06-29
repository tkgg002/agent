# Plan: Fallback Default Schema từ Connection Registry

## 1. Mục tiêu
Sửa đổi logic trong snapshot runner và metadata registry để lấy `DefaultSchema` từ connection làm schema mặc định cho PostgreSQL thay vì hardcode `"public"`.

## 2. Kế hoạch chi tiết
- **Bước 1**: Sửa hàm `buildDSNFromFields` trong `metadata_registry_utils.go` để append `search_path` vào DSN Postgres nếu `conn.DefaultSchema` có giá trị.
- **Bước 2**: Sửa logic trong `snapshot_runner_handler.go` tại hai vị trí xác định schema Postgres:
  - Vị trí 1 (lấy total rows từ pg_class): Fallback về `conn.DefaultSchema`.
  - Vị trí 2 (phân trang và build table name): Fallback về `conn.DefaultSchema`.
- **Bước 3**: Cập nhật mock dữ liệu trong `snapshot_runner_test.go` để bao phủ `DefaultSchema` và đảm bảo unit tests chạy PASS.

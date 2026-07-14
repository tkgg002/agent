# Kế hoạch triển khai: Ẩn pipeline đã xóa trên Data Integrity Dashboard

Hiện tại, trang quản trị Data Integrity hiển thị cả các pipeline đã bị xóa/vô hiệu hóa (như `payment_bills` thuộc `payment-bill-service`). Lý do là tầng persistence của `cdc-cms-service` khi truy vấn danh sách báo cáo mới nhất (`ListLatest`) đang thực hiện `LEFT JOIN` với bảng `cdc_table_registry` mà không lọc theo trạng thái hoạt động (`is_active = TRUE`).

Kế hoạch này sẽ chuyển đổi các liên kết `LEFT JOIN` thành `INNER JOIN` và lọc trạng thái hoạt động để loại bỏ các pipeline cũ đã bị xóa.

## User Review Required

> [!IMPORTANT]
> Thay đổi này sẽ ảnh hưởng đến trang giao diện Data Integrity: Các pipeline có thuộc tính `is_active = false` hoặc đã bị xóa hoàn toàn khỏi bảng `cdc_table_registry` sẽ lập tức biến mất khỏi danh sách hiển thị trên dashboard.

## Proposed Changes

### Component: cdc-cms-service (Read Side Persistence)

#### [MODIFY] [recon_read_repo_gorm.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/infra/persistence/recon/recon_read_repo_gorm.go)

- Cập nhật biến hằng số `listLatestPrimary` thay thế:
  ```sql
  LEFT JOIN cdc_table_registry reg ON reg.target_table = r.shadow_table
  ```
  bằng:
  ```sql
  INNER JOIN cdc_table_registry reg ON reg.target_table = r.shadow_table AND reg.is_active = TRUE
  ```

- Cập nhật biến hằng số `listLatestLegacy` thay thế:
  ```sql
  LEFT JOIN cdc_table_registry reg ON reg.target_table = r.target_table
  ```
  bằng:
  ```sql
  INNER JOIN cdc_table_registry reg ON reg.target_table = r.target_table AND reg.is_active = TRUE
  ```

## Verification Plan

### Automated Tests
- Chạy toàn bộ các unit tests của gói query/app trong `cdc-cms-service` để đảm bảo không bị lỗi cú pháp SQL hay logic mock:
  ```bash
  go test ./test/...
  ```

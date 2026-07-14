# Technical Solution: Hide Deleted/Inactive Pipelines in Data Integrity Dashboard

## File cần chỉnh sửa
- File: [recon_read_repo_gorm.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/infra/persistence/recon/recon_read_repo_gorm.go)

## Chi tiết thay đổi

### 1. Thay đổi trong `listLatestPrimary`
- Tìm đoạn code:
  ```go
  	  LEFT JOIN cdc_table_registry reg ON reg.target_table = r.shadow_table
  ```
- Thay thế bằng:
  ```go
  	  INNER JOIN cdc_table_registry reg ON reg.target_table = r.shadow_table AND reg.is_active = TRUE
  ```

### 2. Thay đổi trong `listLatestLegacy`
- Tìm đoạn code:
  ```go
  	  LEFT JOIN cdc_table_registry reg ON reg.target_table = r.target_table
  ```
- Thay thế bằng:
  ```go
  	  INNER JOIN cdc_table_registry reg ON reg.target_table = r.target_table AND reg.is_active = TRUE
  ```

## Xác minh sau khi sửa
- Chạy test trong `cdc-cms-service`:
  ```bash
  go test ./test/...
  ```

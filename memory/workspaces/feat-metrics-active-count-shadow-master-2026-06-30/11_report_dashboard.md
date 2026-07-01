# Báo cáo Thay đổi: Cập nhật Dashboard Metrics Active Count

## Tổng quan
Báo cáo ghi nhận các thay đổi trong file cấu hình dashboard của SigNoz nhằm thay thế việc giám sát tổng số dòng bằng việc giám sát số lượng dòng active (active row count).

## Chi tiết File thay đổi

### 1. File: [signoz-dashboard-recon.json](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/deployments/signoz-dashboard-recon.json)
- **Số lượng dòng thay đổi:** 6 dòng thêm vào, 6 dòng bớt đi.
- **Chi tiết thay đổi:**
  - Thay đổi Panel ID `w02` ("Shadow Row Count"):
    - Cập nhật Title: `"Shadow Row Count (per table)"` -> `"Shadow Active Row Count (per table)"`
    - Cập nhật Description: `"pg_class.reltuples O(1). Table panel — không cùng chart với Source để tránh scale mismatch."` -> `"Active row count for shadow table: pg_class_estimate - tombstone_count (_deleted=true)."`
    - Cập nhật Metric Name: `"cdc_shadow_table_row_count"` -> `"cdc_shadow_active_row_count"`
  - Thay đổi Panel ID `w03` ("Master Row Count"):
    - Cập nhật Title: `"Master Row Count (per table)"` -> `"Master Active Row Count (per table)"`
    - Cập nhật Description: `"MasterBindingRef pg_class O(1). Table panel."` -> `"Active row count for master table: pg_class_estimate - tombstone_count (_deleted=true)."`
    - Cập nhật Metric Name: `"cdc_master_table_row_count"` -> `"cdc_master_active_row_count"`

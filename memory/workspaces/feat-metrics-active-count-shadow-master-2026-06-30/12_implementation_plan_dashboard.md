# Kế hoạch Triển khai: Cập nhật Dashboard Metrics Active Count

## 1. Mục tiêu
Cập nhật file `deployments/signoz-dashboard-recon.json` để sử dụng các metrics active row count mới cho shadow và master table, thay thế cho các metric đếm tổng số dòng cũ.

## 2. Kế hoạch chi tiết

### Bước 2.1: Sửa đổi file Dashboard JSON
- Sửa đổi panel có ID `w02` (Shadow Row Count):
  - Target line 199: `"title": "Shadow Row Count (per table)"` -> `"Shadow Active Row Count (per table)"`
  - Target line 200: `"description": "pg_class.reltuples O(1). Table panel — không cùng chart với Source để tránh scale mismatch."` -> `"Active row count for shadow table: pg_class_estimate - tombstone_count (_deleted=true)."`
  - Target line 242: `"metricName": "cdc_shadow_table_row_count"` -> `"cdc_shadow_active_row_count"`

- Sửa đổi panel có ID `w03` (Master Row Count):
  - Target line 274: `"title": "Master Row Count (per table)"` -> `"Master Active Row Count (per table)"`
  - Target line 275: `"description": "MasterBindingRef pg_class O(1). Table panel."` -> `"Active row count for master table: pg_class_estimate - tombstone_count (_deleted=true)."`
  - Target line 317: `"metricName": "cdc_master_table_row_count"` -> `"cdc_master_active_row_count"`

### Bước 2.2: Xác minh
- Dùng công cụ phân tích cú pháp JSON (như `jq` hoặc tương đương) để kiểm tra tính hợp lệ của file JSON.
- Đảm bảo không làm thay đổi các phần khác trong tệp JSON.

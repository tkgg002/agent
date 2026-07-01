# Specs: Cập nhật Dashboard Metrics Active Count cho Shadow & Master

## Yêu cầu
Cập nhật file JSON cấu hình dashboard của SigNoz (`deployments/signoz-dashboard-recon.json`) để chuyển đổi hai panel `Shadow Row Count` và `Master Row Count` sang sử dụng metric đếm số lượng dòng active (active row count).

## Chi tiết thay đổi
1. **Panel Shadow Row Count (`w02`)**:
   - Đổi `title` từ `"Shadow Row Count (per table)"` thành `"Shadow Active Row Count (per table)"`.
   - Đổi `description` thành `"Active row count for shadow table: pg_class_estimate - tombstone_count (_deleted=true)."` để phản ánh đúng công thức active row.
   - Thay đổi `metricName` từ `"cdc_shadow_table_row_count"` thành `"cdc_shadow_active_row_count"`.

2. **Panel Master Row Count (`w03`)**:
   - Đổi `title` từ `"Master Row Count (per table)"` thành `"Master Active Row Count (per table)"`.
   - Đổi `description` thành `"Active row count for master table: pg_class_estimate - tombstone_count (_deleted=true)."` để phản ánh đúng công thức active row.
   - Thay đổi `metricName` từ `"cdc_master_table_row_count"` thành `"cdc_master_active_row_count"`.

## Định nghĩa hoàn thành (DoD)
- File JSON dashboard `deployments/signoz-dashboard-recon.json` được cập nhật chính xác các metric name, title và description.
- Cú pháp JSON vẫn hợp lệ sau khi cập nhật (có thể kiểm tra bằng lệnh jq hoặc parse JSON).
- Không làm thay đổi cấu trúc của các panel khác trong file JSON dashboard.

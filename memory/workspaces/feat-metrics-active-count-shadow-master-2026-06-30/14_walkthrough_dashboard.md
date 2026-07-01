# Walkthrough: Cập nhật Dashboard Metrics Active Count cho Shadow & Master

Chúng ta đã hoàn thành việc cập nhật file cấu hình dashboard của SigNoz để chuyển đổi từ việc giám sát tổng số dòng (row count) sang số lượng dòng active (active row count).

## Thay đổi đã thực hiện

### [MODIFY] [signoz-dashboard-recon.json](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/deployments/signoz-dashboard-recon.json)
- **Panel Shadow Row Count (`w02`)**:
  - Cập nhật tiêu đề từ `"Shadow Row Count (per table)"` thành `"Shadow Active Row Count (per table)"`.
  - Cập nhật mô tả để làm rõ công thức tính active row: `"Active row count for shadow table: pg_class_estimate - tombstone_count (_deleted=true)."`
  - Cập nhật Prometheus metric name từ `"cdc_shadow_table_row_count"` thành `"cdc_shadow_active_row_count"`.
  
- **Panel Master Row Count (`w03`)**:
  - Cập nhật tiêu đề từ `"Master Row Count (per table)"` thành `"Master Active Row Count (per table)"`.
  - Cập nhật mô tả: `"Active row count for master table: pg_class_estimate - tombstone_count (_deleted=true)."`
  - Cập nhật Prometheus metric name từ `"cdc_master_table_row_count"` thành `"cdc_master_active_row_count"`.

## Kết quả kiểm thử & Xác minh

### Kiểm tra cú pháp JSON
- Đã chạy kiểm tra cú pháp JSON bằng `jq` để đảm bảo file cấu hình dashboard hoàn toàn hợp lệ sau khi chỉnh sửa:
  ```bash
  jq . deployments/signoz-dashboard-recon.json > /dev/null
  ```
  => Kết quả: Hợp lệ (Status OK).

### Git status & Commit local
- Đã tạo một commit local làm restore-point để bảo vệ các thay đổi:
  ```bash
  git add deployments/signoz-dashboard-recon.json
  git commit -m "feat(recon-dashboard): update shadow and master panels to show active row count"
  ```

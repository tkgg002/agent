# Báo cáo thay đổi - Đồng bộ Schema Đối soát Shadow/Master

Báo cáo tổng hợp các tệp tin đã thay đổi trong phiên làm việc hiện tại:

## 1. Tóm tắt thay đổi
- **Tổng số file thay đổi**: 1
- **File cụ thể**: `cdc-cms-service/internal/infra/persistence/recon/recon_read_repo_gorm.go`
- **Mục đích**: Sửa truy vấn SELECT UNION trong hàm `GetTableHistory` để lấy thêm cột `master_schema`.

## 2. Chi tiết các dòng thay đổi

### File: `cdc-cms-service/internal/infra/persistence/recon/recon_read_repo_gorm.go`
- **Số lượng dòng thay đổi**: +2 dòng
- **Chi tiết**:
  - Bổ sung `master_schema` vào SELECT của `cdc_system.cdc_reconciliation_report`.
  - Bổ sung `master_schema` vào SELECT của `cdc_system.cdc_recon_smoke_result`.

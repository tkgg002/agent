# Báo cáo thay đổi - Khắc phục Hiển thị Dữ liệu ID Diff

Báo cáo tổng hợp các tệp tin đã thay đổi:

## 1. Tóm tắt thay đổi
- **Tổng số file thay đổi**: 1
- **File cụ thể**: `cdc-cms-service/internal/infra/persistence/recon/recon_read_repo_gorm.go`
- **Mục đích**: SELECT đầy đủ các cột ID diff và heal metrics trong truy vấn UNION của hàm `GetTableHistory`.

## 2. Chi tiết dòng thay đổi

### File: `cdc-cms-service/internal/infra/persistence/recon/recon_read_repo_gorm.go`
- **Số lượng dòng thay đổi**: +30 dòng
- **Chi tiết**:
  - Thêm 15 cột diff/heal vào phần SELECT từ `cdc_reconciliation_report`.
  - Thêm 15 cột mock tương ứng (`0::integer`, `NULL::jsonb`, `NULL::timestamp`) vào phần SELECT từ `cdc_recon_smoke_result`.

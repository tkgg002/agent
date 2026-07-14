# Báo cáo thay đổi - Sửa lỗi sai lệch Count hiển thị trên Dashboard (ListLatest)

Báo cáo tổng hợp các tệp tin đã thay đổi:

## 1. Tóm tắt thay đổi
- **Tổng số file thay đổi**: 1
- **File cụ thể**: `cdc-cms-service/internal/infra/persistence/recon/recon_read_repo_gorm.go`
- **Mục đích**: Thay đổi logic query `listLatestPrimary` để lấy các cột active/total counts từ `cdc_recon_smoke_result` thông qua LEFT JOIN LATERAL thay vì lấy trực tiếp số lượng window của Full Search.

## 2. Chi tiết dòng thay đổi

### File: `cdc-cms-service/internal/infra/persistence/recon/recon_read_repo_gorm.go`
- **Số lượng dòng thay đổi**: ~50 dòng
- **Chi tiết**:
  - Tách truy vấn từ `cdc_reconciliation_report` thành thực thể alias `r`.
  - Thêm mệnh đề `LEFT JOIN LATERAL` truy vấn `cdc_system.cdc_recon_smoke_result s` sắp xếp theo `checked_at DESC LIMIT 1`.
  - Chuyển đổi các cột chọn counts thành `COALESCE(s.<field>, r.<field>)`.

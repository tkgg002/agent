# Báo cáo thay đổi (Report) - Sửa lỗi hiển thị tổng record trong tab Pipeline

## Danh sách các tệp thay đổi

### 1. [recon_read_repo_gorm.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/infra/persistence/recon/recon_read_repo_gorm.go)
- **Vị trí thay đổi:** Hàm `listLatestPrimary` query template (dòng 87-90).
- **Mô tả:** Loại bỏ logic `COALESCE` dùng để fallback sang các trường đếm của `cdc_reconciliation_report`. Đảm bảo các cột `source_total`, `source_active`, `shadow_total`, `shadow_active` chỉ nhận giá trị trực tiếp từ kết quả smoke check (`cdc_recon_smoke_result`).
- **Số lượng dòng thay đổi:** Thay đổi 4 dòng.

### 2. [ReconPipelineGrid.tsx](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/components/ReconPipelineGrid.tsx)
- **Vị trí thay đổi:** 
  - Tính toán `sourceTotal`, `shadowActive`, `masterActive` (dòng 153-161).
  - Tính toán `sourceTotal`, `shadowTotal` trong nhánh chỉ có `a` (dòng 195-197).
- **Mô tả:** Chuyển sang đọc trực tiếp từ các trường `source_active`/`source_total`, `shadow_active`/`shadow_total`, và `master_active`/`master_total` (chỉ lấy kết quả smoke check) thay vì dùng các trường window-based (`source_count`, `dest_count`).
- **Số lượng dòng thay đổi:** Thay đổi ~15 dòng.

---
## Tổng kết số lượng dòng thay đổi
- **Tổng số tệp thay đổi:** 2 tệp.
- **Tổng số dòng code được chỉnh sửa/thêm mới:** ~19 dòng.

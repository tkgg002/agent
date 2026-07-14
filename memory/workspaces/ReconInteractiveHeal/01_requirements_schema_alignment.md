# Yêu cầu Chi tiết (Specs) - Đồng bộ Cấu trúc Database Đối soát (Reconciliation Schema Alignment)

## Mục tiêu
Khắc phục lỗi runtime `SQLSTATE 42703 (column "master_schema" does not exist)` khi gọi API truy vấn danh sách báo cáo đối soát. Cần đồng bộ cấu trúc bảng `cdc_system.cdc_reconciliation_report` với `cdc_system.cdc_recon_smoke_result` (chứa các cột `master_schema` và `master_table`) để các câu lệnh UNION ALL của GORM read repository chạy thành công.

## Yêu cầu chi tiết
1. **Migration Cơ sở Dữ liệu**:
   - Viết tệp migration `089_recon_master_metadata.sql` để thêm cột `master_schema` và `master_table` vào bảng `cdc_system.cdc_reconciliation_report` trong dự án `cdc-cms-service`.
2. **Cập nhật GORM Model**:
   - Cập nhật struct `ReconciliationReport` ở cả hai dự án `cdc-cms-service` và `centralized-data-service` để khai báo hai trường `MasterSchema` và `MasterTable` tương ứng với hai cột mới.
3. **Cập nhật Logic Ghi Dữ liệu**:
   - Cập nhật hàm `stampB` trong `centralized-data-service/internal/service/recon/recon_engine_segment_b.go` để gán giá trị `MasterSchema` và `MasterTable` từ `MasterBindingRef` vào `ReconciliationReport` trước khi ghi xuống database.
4. **Kiểm định & Xác nhận**:
   - Kiểm tra biên dịch thành công cả hai dự án.
   - Chạy migration để cập nhật database cục bộ.
   - Chạy unit test của read repository `reconReadRepoGorm` trong `cdc-cms-service` để đảm bảo API truy vấn không còn lỗi SQL.

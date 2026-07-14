# Yêu cầu Đồng bộ Schema Đối soát Shadow/Master (Reconciliation Schema Alignment)

## 1. Bối cảnh
Để hỗ trợ đối soát Segment B (Shadow to Master), hệ thống cần lưu trữ thông tin về `master_schema` và `master_table` của các báo cáo đối soát. Sự không nhất quán giữa cơ sở dữ liệu và model code dẫn đến lỗi SQL runtime `SQLSTATE 42703 (column "master_schema" does not exist)`.

## 2. Yêu cầu chi tiết
- Áp dụng migration `089_recon_master_metadata.sql` bổ sung cột `master_schema` và `master_table` vào bảng `cdc_reconciliation_report`.
- Đồng bộ struct `ReconciliationReport` ở cả 2 service `cdc-cms-service` và `centralized-data-service`.
- Cập nhật logic ingestion `stampB` trong `centralized-data-service` để lưu thông tin master metadata.
- Sửa truy vấn UNION trong repo layer `GetTableHistory` của `cdc-cms-service` để lấy cột `master_schema` và hiển thị trên giao diện thông qua API.

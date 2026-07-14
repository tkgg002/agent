# Phân tích kỹ thuật - Lệch Cấu trúc Database Đối soát (Reconciliation Schema Alignment)

## 1. Phân tích nguyên nhân (Root Cause Analysis)
Khi thực hiện gọi API lịch sử đối soát qua endpoint `/api/reconciliation/report/:table`, hệ thống gọi qua Handler `GetTableHistoryHandler` và sử dụng repository layer `reconReadRepoGorm` để thực hiện query.
Trong phương thức `GetTableHistory` của [recon_read_repo_gorm.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/infra/persistence/recon/recon_read_repo_gorm.go):
- Sử dụng mệnh đề `UNION ALL` để lấy dữ liệu từ 2 bảng: `cdc_system.cdc_reconciliation_report` và `cdc_system.cdc_recon_smoke_result`.
- Bảng `cdc_reconciliation_report` đã được migrate bổ sung 2 cột `master_schema` và `master_table`.
- Trong subquery UNION, phần SELECT từ `cdc_reconciliation_report` và `cdc_recon_smoke_result` đã có chiếu cột `master_table` nhưng lại bỏ sót không chiếu cột `master_schema`.
- Do không chiếu cột `master_schema` trong danh sách cột trả về của UNION, GORM không thể scan được trường `master_schema` vào struct model `ReconciliationReport` và trường này luôn bị trả về giá trị rỗng/null trên API.

## 2. Giải pháp khắc phục
Bổ sung cột `master_schema` vào danh sách SELECT của cả 2 bảng trong câu truy vấn `UNION ALL`. Việc này đảm bảo tính tương đồng về số lượng cột của UNION, đồng thời cung cấp đầy đủ thông tin để GORM thực hiện Mapping thành công.

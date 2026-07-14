# Requirements: Hide Deleted/Inactive Pipelines in Data Integrity Dashboard

## Vấn đề
- Trang web Data Integrity (`http://localhost:5173/data-integrity`) vẫn hiển thị pipeline của bảng `payment_bills` thuộc connector `payment-bill-service` mặc dù connector này đã bị xóa từ rất lâu.
- Dữ liệu rác này hiển thị các thông số hàng đợi `ingest: 0`, `transmute: 0` gây nhiễu giao diện.

## Nguyên nhân
- Gói `ReconReader` trong `cdc-cms-service` sử dụng câu lệnh `listLatestPrimary` và `listLatestLegacy` để lấy báo cáo đối soát mới nhất của từng bảng đã từng chạy đối soát trong quá khứ (`cdc_reconciliation_report` / `cdc_recon_smoke_result`).
- Câu lệnh SQL hiện tại đang sử dụng `LEFT JOIN cdc_table_registry` mà không kiểm tra trạng thái hoạt động (`is_active = TRUE`) hoặc xem bảng có thực sự tồn tại trong registry hay không.

## Yêu cầu
- Sửa đổi câu lệnh SQL trong `cdc-cms-service/internal/infra/persistence/recon/recon_read_repo_gorm.go` để chỉ trả về các pipeline/bảng đang thực sự tồn tại và có trạng thái hoạt động `is_active = TRUE` trong `cdc_table_registry`.
- Đảm bảo các unit tests vẫn biên dịch và pass bình thường.

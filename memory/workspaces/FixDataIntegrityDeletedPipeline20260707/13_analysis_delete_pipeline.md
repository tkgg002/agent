# Phân tích Kỹ thuật (Technical Analysis) - Hide Deleted/Inactive Pipelines

Dự án: `cdc-cms-service`
Workspace: `FixDataIntegrityDeletedPipeline20260707`

## Phân tích thay đổi

Hệ thống Data Integrity Dashboard lấy danh sách các pipeline (tables) cần đối soát thông qua hai câu truy vấn:
1. `listLatestPrimary`: Truy vấn chính, áp dụng cho các môi trường đã chạy migration 017 (chứa các cột enrichment lag, metadata chi tiết).
2. `listLatestLegacy`: Truy vấn fallback cho các môi trường cũ chưa chạy migration 017.

### Vấn đề trước đó
Cả hai câu truy vấn đều sử dụng `LEFT JOIN cdc_table_registry reg` để lấy thông tin metadata của bảng được đăng ký trong registry:
- Trong `listLatestPrimary`: `LEFT JOIN cdc_table_registry reg ON reg.target_table = r.shadow_table`
- Trong `listLatestLegacy`: `LEFT JOIN cdc_table_registry reg ON reg.target_table = r.target_table`

Bởi vì dùng `LEFT JOIN`, nên ngay cả khi một pipeline (bảng) đã bị xóa hoặc đặt trạng thái không hoạt động (`is_active = FALSE`) trong registry (`cdc_table_registry`), bản ghi báo cáo đối soát tương ứng (`cdc_reconciliation_report` hoặc `cdc_recon_smoke_result`) vẫn được hiển thị trên Dashboard với các giá trị registry bằng `NULL` (hoặc rỗng). Điều này làm sai lệch dữ liệu hiển thị trên giao diện người dùng (Dashboard hiển thị các pipeline rác/đã xóa).

### Giải pháp áp dụng
Chúng ta chuyển đổi `LEFT JOIN` thành `INNER JOIN` và thêm bộ lọc trạng thái hoạt động:
1. Trong `listLatestPrimary`:
   ```sql
   INNER JOIN cdc_table_registry reg ON reg.target_table = r.shadow_table AND reg.is_active = TRUE
   ```
2. Trong `listLatestLegacy`:
   ```sql
   INNER JOIN cdc_table_registry reg ON reg.target_table = r.target_table AND reg.is_active = TRUE
   ```

### Kết quả mong đợi
- Chỉ các pipeline đang tồn tại và hoạt động (`is_active = TRUE` trong `cdc_table_registry`) mới được lấy ra và hiển thị trên Dashboard.
- Các pipeline đã bị soft-delete (có `is_active = FALSE` hoặc bị xóa cứng khỏi registry) sẽ tự động bị loại bỏ khỏi kết quả truy vấn.

# Báo cáo Thay đổi (11_report_fix_recon_history.md)

## Các file đã thay đổi
1. `/Users/trainguyen/Documents/work/data-hub/cdc-cms-service/migrations/schema/recon_dlq/093_recon_heal_timestamps.sql` [NEW]
   - **Số dòng thay đổi:** +9 dòng.
   - **Mô tả:** Thêm file migration SQL để bổ sung 3 cột timestamp (`healed_mismatched_at`, `healed_missing_src_at`, `healed_missing_dest_at`) vào bảng `cdc_system.cdc_reconciliation_report`.

## Kết quả kiểm thử
- Áp dụng migration thành công khi server/test chạy.
- Integration test query `GetTableHistory` chạy thành công (PASS) không còn báo lỗi `column does not exist`.

# Yêu cầu Khắc phục Hiển thị Dữ liệu ID Diff (Full Search / Full Diff)

## 1. Bối cảnh
Khi thực hiện chạy đối soát sâu (Full Search / Full Diff), API lịch sử đối soát `/api/reconciliation/report/:table` không trả về dữ liệu ID bị lệch (`missing_ids`, `stale_ids`, `field_diffs`, v.v.), mặc dù dữ liệu này đã được lưu trữ chính xác dưới cơ sở dữ liệu.

## 2. Chi tiết yêu cầu
- Cập nhật hàm `GetTableHistory` trong `recon_read_repo_gorm.go`.
- Bổ sung tất cả các trường dữ liệu lệch và các trường chữa lành (heal metrics) vào mệnh đề SELECT của truy vấn UNION ALL:
  - Bảng `cdc_reconciliation_report`: chọn các cột `missing_count`, `missing_ids`, `stale_count`, `stale_ids`, `field_diffs`, `orphan_count`, và các trường chữa lành (`healed_at`, `healed_count`, v.v.).
  - Bảng `cdc_recon_smoke_result`: chọn các giá trị mặc định tương ứng (như `0::integer AS missing_count`, `NULL::jsonb AS missing_ids`, v.v.) vì bảng này không lưu trữ chi tiết ID lệch.
- Đảm bảo biên dịch thành công và các test case liên quan vượt qua.

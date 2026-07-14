# Báo cáo Kết quả (Walkthrough) - Sửa lỗi sai lệch Count hiển thị trên Dashboard (ListLatest)

## Tóm tắt công việc đã thực hiện
Đã khắc phục lỗi sai lệch số lượng bản ghi hiển thị trên Dashboard sau khi chạy đối soát sâu (Full Diff) bằng cách sử dụng `LEFT JOIN LATERAL` để ưu tiên lấy counts từ bản ghi `cdc_recon_smoke_result` mới nhất khi dòng mới nhất của DISTINCT ON là từ `cdc_reconciliation_report`.

## Các thay đổi chính
- Cập nhật [recon_read_repo_gorm.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/infra/persistence/recon/recon_read_repo_gorm.go#L79):
  - Sử dụng `LEFT JOIN LATERAL` trong phần SELECT từ bảng `cdc_reconciliation_report` của truy vấn `listLatestPrimary`.
  - Thay thế các cột counts (`source_total`, `source_active`, `shadow_total`, `shadow_active`, `master_total`, `master_active`, `source_count`, `dest_count`) bằng hàm `COALESCE` lấy giá trị tương ứng từ smoke test mới nhất của pipeline (`s.*`).
  - Fallback về giá trị gốc của báo cáo nếu không tìm thấy smoke test.

## Kết quả kiểm thử & xác minh

### 1. Compilation & Unit Tests
- Build code thành công và chạy PASS toàn bộ 100% unit tests trong package `queries` và `api`.

### 2. Xác minh API thực tế
- Gọi API ListLatest, các thông tin counts (`source_active`, `shadow_active`, `master_active`) của segment `source_shadow` và `shadow_master` cho `export_jobs` đã hiển thị khớp với dữ liệu thực tế (457 bản ghi) thay vì lấy số lượng cửa sổ giới hạn của Full Search (8 bản ghi).
- Giúp loại bỏ hoàn toàn việc tính toán sai lệch ảo (`transmute: +449 thừa`) và trạng thái báo "Lệch" không chính xác trên dashboard.

# Báo cáo Kết quả (Walkthrough) - Khắc phục lỗi Đối soát & Interactive Heal Modal

## Tóm tắt các công việc đã thực hiện
Hệ thống đã được khắc phục hoàn chỉnh lỗi về hiển thị danh sách các bản ghi chưa heal trên modal "Chữa lành đối soát" (Execute Heal Modal):
1. **Lỗi không hiển thị ID bị lệch**: Đã project đầy đủ các trường `missing_ids`, `stale_ids`, `field_diffs` trong query UNION của `GetTableHistory`.
2. **Lỗi sai lệch số lượng trên Dashboard**: Sử dụng `LEFT JOIN LATERAL` để ưu tiên lấy counts thực tế từ `cdc_recon_smoke_result`.
3. **Lỗi Modal Chữa lành đối soát (Interactive Heal Visibility) rỗng**: Chuẩn hóa tên bảng Fully Qualified Name (FQN) trong `ListUnhealedReports` và `GetTableHistory` ở `recon_read_repo_gorm.go`. Khi tham số `table` chứa dấu chấm (`.`), hệ thống split chuỗi để lấy tên bảng thô (`export_jobs`) nhằm so khớp chính xác với cột `shadow_table`/`master_table` trong DB, đồng thời tự động bóc tách và gán `shadowSchema` tương ứng.

## Các thay đổi chính
- Cập nhật [recon_read_repo_gorm.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/infra/persistence/recon/recon_read_repo_gorm.go):
  - Chuẩn hóa FQN trong `ListUnhealedReports` để tách schema & table.
  - Chuẩn hóa FQN trong `GetTableHistory` cho cả `table` và `masterTable`.

## Kết quả kiểm thử & xác minh

### 1. Build & Unit Tests
- Biên dịch thành công dự án `cdc-cms-service` (`go build ./...` pass).
- Các unit tests của package `queries` chạy PASS 100%.

### 2. Xác minh API qua cURL
- Gọi API `/api/reconciliation/report/shadow_testexp.export_jobs/unhealed?shadow_schema=shadow_testexp` đã trả về danh sách đầy đủ 11 bản ghi chưa chữa lành (`healed_at IS NULL` và có chênh lệch):
  ```json
  {
    "data": [
      {
        "id": 34,
        "target_table": "shadow_testexp.export_jobs",
        "missing_count": 0,
        "stale_count": 1,
        "healed_at": null,
        ...
      }
    ],
    "total": 11
  }
  ```
- Dữ liệu này giúp Frontend React hiển thị danh sách để người dùng có thể thực hiện "Chữa lành đối soát" chính xác.

### 3. Quy trình Governance
- Đã chạy script `verify_governance.py` thành công:
  `⛳ GOVERNANCE AUDIT PASSED 🟢`

# Kế hoạch tối ưu hóa SQL cdc_activity_log (SLOW SQL)

Dưới đây là kế hoạch tối ưu hóa thời gian thực thi của câu lệnh đếm tổng số bản ghi nhật ký hoạt động (`cdc_activity_log`) để loại bỏ cảnh báo SLOW SQL >= 200ms.

## User Review Required

> [!IMPORTANT]
> Cần phê duyệt việc tái cấu trúc logic tính toán `countQuery` trong hàm `ListActivity`. Thay đổi này hoàn toàn không làm thay đổi kết quả đếm mà chỉ loại bỏ các phép JOIN không cần thiết khi không dùng đến bộ lọc.

## Proposed Changes

### cdc-cms-service

#### [MODIFY] [activity_log_read_repo_gorm.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/infra/persistence/system/activity_log_read_repo_gorm.go)

- Cải tiến hàm `ListActivity`:
  - Phân tích xem các bộ lọc liên quan đến bảng liên kết (`SourceDatabase`, `SourceTable`, `ShadowSchema`, `ShadowTable`) có được truyền vào hay không.
  - Nếu có bộ lọc bảng liên kết: Dùng câu SQL countQuery chứa đầy đủ lateral joins cũ.
  - Nếu không có bộ lọc bảng liên kết: Dùng câu SQL countQuery tối giản (`SELECT COUNT(*) FROM cdc_activity_log al WHERE 1=1`) không chứa các lateral joins phức tạp.

## Verification Plan

### Automated Tests
- Chạy unit và integration test suite của dự án để xác minh tính ổn định:
  ```bash
  go test ./test/... -count=1
  ```

### Manual Verification
- Viết một benchmark test script để so sánh thời gian chạy thực tế của câu count cũ (với lateral joins) và câu count mới (không joins) trên cơ sở dữ liệu Postgres thật tại local.

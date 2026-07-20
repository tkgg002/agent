# Báo cáo Thay đổi (11_report_smoke_boundary)

Dưới đây là tổng hợp các thay đổi đã thực hiện và được kiểm chứng thành công trong phiên làm việc này.

## Tóm tắt thay đổi

### 1. File: `internal/service/recon/recon_dest_query.go`
- **Mô tả thay đổi**: Thêm hàm `CountRecentDeletedRows` để đếm số lượng bản ghi bị xóa mềm (`_deleted = true`) trong postgres/shadow trong khoảng thời gian `[tLo, tHi)`.
- **Số dòng thay đổi**: Khoảng ~25 dòng code được thêm mới.

### 2. File: `internal/service/recon/recon_smoke.go`
- **Mô tả thay đổi**:
  - Tích hợp công thức trừ bù cửa sổ: `ActiveClean = Total - (RecentTotal - RecentDeleted)`.
  - Thay đổi mốc trên `hi` của `HashWindow` thành `fromTime` (tức `nowTime - 120s` làm tròn phút) thay vì dùng `nowTime` động.
  - Sửa đổi error handling cục bộ trong `RunTotalOnlyB`.
- **Số dòng thay đổi**: Khoảng ~110 dòng code được chỉnh sửa/thêm mới.

### 3. File: `internal/service/recon/recon_smoke_test.go`
- **Mô tả thay đổi**: Cập nhật mock expectations cho `sqlmock` bao gồm các hàm mới, tham số thời gian chính xác, thứ tự chạy câu lệnh khớp với logic logic mới để pass toàn bộ Unit Test.
- **Số dòng thay đổi**: Khoảng ~540 dòng code được cập nhật.

## Kết quả kiểm thử (Verification)
- Đã chạy lệnh kiểm thử:
  ```bash
  go test -count=1 -v ./internal/service/recon/...
  ```
- Kết quả: **PASS** toàn bộ các unit test.

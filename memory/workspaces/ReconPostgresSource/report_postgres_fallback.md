# Báo cáo kỹ thuật: PostgreSQL Timestamp Fallback Probing

## Lý do thay đổi
Khi chạy đối soát cho các bảng PostgreSQL nguồn mà không có cột timestamp mặc định (`updated_at`), tiến trình đối soát (`ReconCore`) bị crash/failed với lỗi `SQLSTATE 42703 (column does not exist)`.
Báo cáo này triển khai cơ chế tự động dò tìm cột thay thế (fallback/probing) từ cấu hình candidates của `TableRegistry` để khắc phục lỗi trên mà không làm ảnh hưởng đến cấu trúc hệ thống lõi.

## Danh sách tệp tin thực tế đã thay đổi & Số lượng dòng code thay đổi

Dưới đây là thống kê chi tiết các tệp tin đã chỉnh sửa và thêm mới:

| Đường dẫn tệp tin | Trạng thái | Số lượng dòng code thay đổi (ước tính) | Mô tả |
| :--- | :--- | :--- | :--- |
| `internal/service/recon/recon_tier_a.go` | Modified | ~110 lines | Bổ sung helper check lỗi, logic dò tìm `resolveSourceTSField`, cập nhật `pickScanRangeWithLag` và các Tier 1/2/3. Tinh chỉnh logic break sớm khi truy vấn thành công. |
| `internal/service/recon/recon_smoke.go` | Modified | 1 line | Đồng bộ chữ ký gọi hàm `pickScanRangeWithLag`. |
| `internal/service/recon/recon_fallback_test.go` | New | 50 lines | Viết unit test mô phỏng lỗi `SQLSTATE 42703` với `go-sqlmock` để verify cơ chế fallback hoạt động đúng đắn. |

**Tổng số dòng code thay đổi/thêm mới**: ~161 dòng code Go.

## Quá trình kiểm thử & Xác minh chất lượng đầu ra
- **Cơ chế tái hiện lỗi (G2 - Red)**: Mô phỏng hành vi trả về lỗi `SQLSTATE 42703` khi truy vấn cột chính.
- **Chứng minh thành công (G3 - Green)**: Đã chạy thành công unit test `TestResolveSourceTSField_Fallback` chứng minh hệ thống tự động nhảy qua cột thay thế candidate đầu tiên hoạt động mà không bị crash.
- **Kiểm thử hồi quy (G5)**: Toàn bộ suite test của package `internal/service/recon/...` đã pass thành công với lệnh:
  ```bash
  go test -v -count=1 ./internal/service/recon/...
  ```
- **Bằng chứng kiểm thử (G8)**: Được ghi nhận chi tiết trong tệp `walkthrough.md` và `05_progress.md` của workspace.

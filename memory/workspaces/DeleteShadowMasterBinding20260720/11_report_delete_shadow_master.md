# Báo cáo thay đổi: Tự động dọn dẹp cdc_table_registry mồ côi

## 1. Thông tin chung
- **Nhiệm vụ**: Implement logic tự động dọn dẹp bản ghi legacy `cdc_table_registry` mồ côi khi đăng ký mới.
- **Người thực hiện**: Muscle Engineer (Gemini-3-Flash)
- **Workspace**: `DeleteShadowMasterBinding20260720`
- **Thời gian**: 2026-07-20

## 2. Chi tiết thay đổi
### File: `internal/infra/persistence/source/source_repo_gorm.go`
- **Vị trí thay đổi**: Hàm `Register` (dòng 257-270).
- **Số dòng code thay đổi**: ~15 lines.
- **Nội dung thay đổi**:
  Bổ sung câu lệnh SQL `DELETE FROM cdc_system.cdc_table_registry WHERE target_table = ? AND NOT EXISTS (SELECT 1 FROM cdc_system.shadow_binding WHERE shadow_table = ?)` chạy trong Transaction trước khi chèn (`tx.Create`) bản ghi `TableRegistry` mới. Điều này giúp loại bỏ triệt để các record mồ côi ở bảng legacy `cdc_table_registry` cũ nếu có cùng `target_table` nhưng không được liên kết với bất kỳ `shadow_binding` nào. Từ đó tránh được lỗi `duplicate key value violates unique constraint` (SQLSTATE 23505) khi đăng ký lại.

## 3. Kết quả Verify Build
- Biên dịch verify backend bằng lệnh: `go build ./cmd/... ./internal/...`
- **Kết quả**: Thành công hoàn toàn (không có lỗi biên dịch).
- **Lưu ý**: Lệnh `go build ./...` trong root `cdc-cms-service` bị lỗi do package `main` bị khai báo trùng lặp ở nhiều file khác nhau trong thư mục `/scratch` (ví dụ `check_payment_bills.go` và `check_bindings.go`). Đây là thư mục chứa các file scratch tạm thời phục vụ debug của các phiên làm việc trước nên không thuộc runtime của backend chính. Build main code (`./cmd/... ./internal/...`) thành công.

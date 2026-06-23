# Workspace Context: Refactor Admin Helpers

## Objective
Tái cấu trúc `internal/admin/helpers.go` theo yêu cầu của User:
1. Xóa bỏ tất cả các hàm helper `*ForTest` để bảo đảm public API sạch sẽ.
2. Cấu hình HTTP Client có Timeout = 15 giây để phòng tránh Goroutine leak.
3. Bảo mật Kafka Topic name thông qua việc loại bỏ các ký tự không hợp lệ bằng Regex và hàm `sanitizeKafkaName`.
4. Cập nhật và điều chỉnh các file test tương ứng (thay đổi package name từ `admin_test` sang `admin` để truy cập trực tiếp các hàm private).

## Scope
- `internal/admin/helpers.go`: Lưu đè mã nguồn do User cung cấp.
- `internal/admin/helpers_test.go` (hoặc file kiểm thử liên quan): Chuyển đổi package từ `admin_test` sang `admin` và loại bỏ việc gọi các hàm `*ForTest`.
- Xác minh biên dịch (`go build ./...`) và chạy test suite (`go test ./...`) của dự án.

## Governance Compliance
- Trạng thái vi phạm: Không vi phạm.
- Gốc rễ lỗi vi phạm quy trình Governance (nếu có): N/A.

# Implementation Plan: Refactor Admin Helpers

## Checklist

- [ ] Lưu đè mã nguồn mới vào `internal/admin/helpers.go`.
- [ ] Định vị các test files kiểm thử liên quan đến `admin/helpers.go` (ví dụ `helpers_test.go`).
- [ ] Chuyển đổi package của các test files từ `admin_test` sang `admin`.
- [ ] Thay thế các lời gọi hàm `*ForTest` cũ bằng cách gọi trực tiếp các hàm private tương ứng.
- [ ] Đảm bảo dự án biên dịch thành công (`go build ./...`).
- [ ] Đảm bảo các unit test chạy qua (`go test ./...`).

# Execution Plan: Source Register Migration & Refactor

## Steps
1. **Khởi tạo file mới**: Tạo `internal/handler/source/source_register.go` thuộc `package source` với logic `RegisterHandler` và các DTO, helper methods.
2. **Xóa files cũ**: Xóa `internal/admin/source_register.go` và `internal/admin/types.go`.
3. **Dọn dẹp helpers**: Xóa các helper methods và functions liên quan đến source registration khỏi `internal/admin/helpers.go`.
4. **Cập nhật server.go**: Thay đổi dependency Injection của `admin.Server` để chứa `source.RegisterHandler` và đăng ký route tương ứng.
5. **Cập nhật server_test.go**: Thay thế các DTO cũ bằng `source.RegisterSourceRequest` và `source.RegisterSourceResponse`.
6. **Xác minh**: Biên dịch, chạy unit tests cho package `admin`, và chạy toàn bộ test suite.

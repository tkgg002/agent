# Danh sách Task: SFTP Route Isolation Fix

- [x] Revert logic lọc ẩn topic trong `topic_helper.go` về nguyên bản.
- [x] Bổ sung bộ lọc đối sánh `sourceDB` đối với khóa tìm kiếm fallback trong `ResolveSourceRoutes` của `metadata_registry_service.go`.
- [x] Chạy unit test backend của `source` package và build worker daemon để xác minh.
- [ ] Trình kế hoạch và trao đổi với người dùng để restart worker.
- [ ] Sau khi worker chạy code mới, hướng dẫn người dùng nhấn nút "Xóa Offset" trên giao diện quản lý Connector của `testsftp13` để chạy lại snapshot và kiểm tra đồng bộ dữ liệu vào bảng shadow.

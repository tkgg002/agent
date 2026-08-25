# Nhật ký tiến độ: SFTP Route Isolation Fix

- **2026-08-12 13:53:00 [Agent:Gemini-2.5-Pro]**: Nhận ý kiến phản hồi từ người dùng về việc lọc ẩn topic tại tầng discovery khó kiểm soát và không tường minh. Chuyển sang giải pháp cô lập định tuyến chéo tại tầng Resolver.
- **2026-08-12 13:54:00 [Agent:Gemini-2.5-Pro]**: Đã cập nhật logic hàm `ResolveSourceRoutes` trong `metadata_registry_service.go` để lọc route chính xác theo `sourceDB` đối với các khóa fallback chung.
- **2026-08-12 13:55:00 [Agent:Gemini-2.5-Pro]**: Chạy thử bộ unit test của package `source` và biên dịch toàn bộ worker thành công.

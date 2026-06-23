# Technical Plan: Refactor Admin HTTP Server

## Các bước thực hiện
1. **Khởi tạo Workspace**: Tạo thư mục workspace và các file quản lý tiến độ.
2. **Refactor server.go**:
    - Thay thế `rateLimiterStore` bằng cơ chế per-IP rate limiting, sử dụng cấu trúc `clientLimiter` lưu `lastSeen` và một background goroutine `cleanup()` định kỳ 1 phút để dọn dẹp các IP không active quá 5 phút.
    - Đổi trường `engine *gin.Engine` thành public `Router *gin.Engine` trong struct `Server`.
    - Đổi thứ tự middleware trong `buildEngine()`: Body Limit -> Rate Limit (theo IP) -> Auth Middleware.
    - Loại bỏ hàm `EngineForTest()`.
    - Sử dụng các hằng số `http.Status*` thay vì số cứng.
3. **Cập nhật server_test.go**:
    - Thay thế toàn bộ lời gọi hàm `srv.EngineForTest()` thành `srv.Router`.
    - Đảm bảo các unit test hoạt động tốt với cơ chế Rate Limit mới (chú ý: kiểm tra xem các test case Rate Limit có bị ảnh hưởng bởi việc đổi key từ token sang IP không).
4. **Kiểm thử & Xác thực**:
    - Chạy `go test -v ./internal/admin/...` để đảm bảo 100% test case pass.
    - Chạy `/security-agent` để rà soát bảo mật.

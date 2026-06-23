# Context: Refactor Admin HTTP Server

## Mục tiêu
Tái cấu trúc file `internal/admin/server.go` để khắc phục các lỗi thiết kế (code smells), lỗ hổng bảo mật (OOM rate limiter, bypass auth middleware) và dọn dẹp helper dành riêng cho testing (`EngineForTest`).

## Lý do Refactor
1. **Rate Limiter Memory Leak (OOM)**: Rate limiter hiện tại lưu trữ đối tượng `rate.Limiter` cho mỗi token vô thời hạn mà không giải phóng, dễ dẫn đến tràn bộ nhớ (OOM) nếu bị tấn công Brute-force/Spam.
2. **Race Condition / Bypass**: Middleware `authMiddleware` nằm trước `rateLimitMiddleware`, khiến kẻ tấn công có thể spam token sai mà không bị giới hạn tần suất request. Cần đảo thứ tự middleware và rate limit theo client IP thay vì token.
3. **Chống rác API**: Loại bỏ hàm `EngineForTest()` và public trường `Router` trong struct `Server` để hỗ trợ testing sạch sẽ.
4. **Hardcode Status Code**: Thay thế các hằng số số nguyên (200, 401) bằng `http.StatusOK`, `http.StatusUnauthorized`.

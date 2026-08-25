# Requirements: GORM OpenTelemetry Selective Tracing

## Mục tiêu
Kiểm soát tải và triệt tiêu tình trạng "spam traces" từ truy vấn DB (GORM) bằng cách bật/tắt trace có chọn lọc theo Module/Flow.
- **Tắt GORM Metrics**: Vô hiệu hóa việc thu thập metrics connection pool tự động của GORM plugin vì hệ thống không có nhu cầu sử dụng metric này.
- **Selective Tracing (Trace có chọn lọc)**: Chỉ bắn traces của các truy vấn GORM thuộc về các module được cấu hình bật. Các truy vấn từ các luồng khác (như cron job, logs ghi nhận, scheduler...) sẽ hoàn toàn không sinh span để tránh spam.
- **Bật Trace cho HTTP API**: Bật trace GORM cho toàn bộ HTTP API của CMS (`cdc-cms-service`) và Admin API (`centralized-data-service`) thông qua module `"cdc"` vì lưu lượng API rất thấp và cần thiết để gỡ lỗi API.

## Các yêu cầu chi tiết
1. **Central Trace Controller**:
   - Định nghĩa danh sách các module được phép trace DB: `cdc` (cho HTTP API), `recon_heal`, `batch_transform`, `discover`, `scan_raw`.
   - Cung cấp helper `WithDBTraceModule(ctx, moduleName)` để đánh dấu flow hiện tại.
2. **Custom OpenTelemetry Sampler**:
   - Viết một Custom Tracer Sampler chèn vào OpenTelemetry SDK lúc khởi tạo (`InitOtel`).
   - Custom Sampler này sẽ:
     - Nếu span đang tạo là DB span (tên span bắt đầu bằng `"gorm."`), nó sẽ kiểm tra xem context có chứa module được bật hay không.
     - Nếu **không** được bật, trả về quyết định `Drop` (không ghi nhận và không gửi span).
     - Nếu được bật, cho phép span được ghi nhận (fall back về default sampler ratio).
     - Với các span khác (HTTP/NATS), luôn cho phép trace bình thường.
3. **Áp dụng cho HTTP API Middleware**:
   - Trong `cdc-cms-service/internal/middleware/http_tracer.go`: Gắn cờ module `"cdc"` cho context request.
   - Trong `centralized-data-service/internal/admin/otel_middleware.go`: Gắn cờ module `"cdc"` cho context request.
4. **Áp dụng cho các NATS handlers**:
   - Đánh dấu context trong các handler NATS chính bằng `observability.WithDBTraceModule(ctx, "module_name")` khi bắt đầu xử lý command.
5. **Vô hiệu hóa GORM Metrics**:
   - Sử dụng option `tracing.WithoutMetrics()` khi đăng ký plugin GORM trong cả 2 repo.

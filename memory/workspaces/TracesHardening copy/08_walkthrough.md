# Walkthrough: Traces Hardening & Observability Recovery

Chúng ta đã hoàn thành toàn bộ các giai đoạn trong kế hoạch khôi phục và tối ưu hóa tracing observability cho hệ thống CDC.

## Thay đổi đã thực hiện

### 1. Graceful Shutdown & Span Loss Fix
- **[centralized-data-service/cmd/worker/main.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/cmd/worker/main.go)**: Thay thế việc gọi `os.Exit(0)` trong goroutine signal handler thành cơ chế select block trên channel `shutdownChan` tại main thread. Đảm bảo toàn bộ `defer` block (bao gồm cả `otelShutdown(ctx)`) được thực thi đầy đủ khi nhận tín hiệu kết thúc (SIGINT/SIGTERM).
- **[cdc-cms-service/cmd/server/main.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/cmd/server/main.go)**: Áp dụng cơ chế graceful shutdown tương tự cho API server.

### 2. NATS Context Propagation Helpers
- **[centralized-data-service/pkgs/observability/trace_helpers.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/pkgs/observability/trace_helpers.go)**: Bổ sung định nghĩa `NatsCarrier` tương thích ngược với `nats.Header` (`map[string][]string`) mà không cần import thư viện NATS. Cung cấp các helpers `InjectNATSHeader` và `ExtractNATSHeader`.
- **[cdc-cms-service/pkgs/observability/otel.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/pkgs/observability/otel.go)**: Tích hợp các helper propagation tương tự vào package observability của API service.

### 3. NATS Publisher Context Injection
- **[cdc-cms-service/internal/infra/messaging/nats_publisher.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/infra/messaging/nats_publisher.go)**: Cập nhật cấu trúc `natsPublisher` để inject trace context vào header của `nats.Msg` trước khi gọi `PublishMsg` thay vì `Publish` raw bytes.

### 4. NATS Handlers Tracing Auto-Patching
- Viết script Python `patch_handlers.py` và thực thi thành công trên toàn bộ 14 files chứa NATS handlers của `centralized-data-service`. Tự động:
  - Trích xuất trace context từ `msg.Header` ở đầu handler.
  - Bắt đầu một OTel child span với định dạng tên `nats.Handle[MethodName]`.
  - Thay thế các lệnh `context.Background()` trùng lặp trong handler để truyền context đi tiếp.

## Kết quả kiểm thử & Xác thực
- **Biên dịch**: Cả hai repository `centralized-data-service` và `cdc-cms-service` đều biên dịch thành công 100% không có lỗi (`go build ./...`).
- **Telemetry pipeline**: Đã kiểm tra file `deployments/otel-collector-config.yml`, pipeline được cấu hình chuẩn mực để nhận dữ liệu qua OTLP HTTP/gRPC và export sang SigNoz/ClickHouse.

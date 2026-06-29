# Plan: Traces Hardening & Observability Recovery

## Steps

### Phase 1: Graceful Shutdown Fix
1. **[MODIFY] [centralized-data-service/cmd/worker/main.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/cmd/worker/main.go)**: 
   - Loại bỏ `os.Exit(0)` khỏi goroutine lắng nghe signal.
   - Sử dụng cơ chế blocking tại main thread hoặc channel truyền tin để thoát hàm `main()` tự nhiên sau khi `srv.Shutdown()` hoàn tất, giúp trigger các block `defer`.
2. **[MODIFY] [cdc-cms-service/cmd/server/main.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/cmd/server/main.go)**:
   - Tương tự, cấu trúc lại signal handler để hàm `main()` thoát tự nhiên và chạy `otelShutdown()`.

### Phase 2: Context Propagation Audit in NATS/Kafka/HTTP & Background Schedulers
1. **NATS Publishers/Subscribers (Đã hoàn thành)**: 
   - Đã tạo `NatsCarrier` và các helper `InjectNATSHeader`/`ExtractNATSHeader`.
   - Đã patch tự động 14 NATS handlers trong `centralized-data-service` và `natsPublisher` của `cdc-cms-service`.
2. **Kafka Consumers (Tầng Giao tiếp & Protocol Adapters)**:
   - Tìm kiếm và kiểm tra xem Kafka Consumer (ví dụ: `internal/handler/shadow/kafka_consumer.go` hoặc các event listeners) có trích xuất context từ Kafka record headers hay không.
   - Định nghĩa `KafkaCarrier` (implements `propagation.TextMapCarrier`) để trích xuất trace context từ Kafka headers.
   - Đóng gói logic để khi Kafka Consumer nhận event, nó tự động trích xuất context và tạo child span cho event handler.
3. **Background Schedulers (Tầng Khởi tạo / Scheduler)**:
   - Xác định các tác vụ chạy background định kỳ (như `PeriodicScan`, cron jobs, background workers).
   - Thiết lập root parent span mới tại điểm bắt đầu chu kỳ của scheduler, truyền context này vào các sub-process bên trong để có 1 flow hoàn chỉnh, mở đường cho Saga pattern.
4. **HTTP API Server**:
   - Kiểm tra xem API Server (`cdc-cms-service`) đã tích hợp OTel HTTP middleware (ví dụ: `otelgin` hoặc custom middleware) để tạo root span cha cho mọi request HTTP chưa.

### Phase 3: Verification & Telemetry Agent Audit
1. Khởi động lại service và test graceful shutdown bằng SIGTERM, kiểm tra logs xem `otelShutdown()` và `logger.Sync()` đã được gọi chưa.
2. Kiểm tra log trace context hoạt động bình thường qua các module.
3. Audit cấu hình collector và chạy thử các luồng end-to-end (HTTP API -> NATS Command -> Kafka Event) để xác nhận span trace tree được vẽ đầy đủ trên SigNoz.

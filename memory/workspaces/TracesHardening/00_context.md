# Context: Traces Hardening & End-to-End Observability

## Overview
Task này tập trung vào việc khôi phục và củng cố hệ thống OpenTelemetry Tracing (CDC Telemetry Tracing) trên toàn bộ các luồng công việc trong hệ thống CDC (`cdc-cms-service` và `centralized-data-service`).

Hệ thống traces phải có một parent span cha bao phủ toàn bộ vòng đời của một tác vụ xử lý (Job/Command), sau đó các tiến trình xử lý tiếp theo (NATS message handlers, Kafka consumers, Tickers, Schedulers) phải liên kết (add) vào parent span cha này để tạo thành một Trace Flow hoàn chỉnh và thống nhất, mở đường cho việc hiện thực hóa mô hình Saga sau này.

## Key Goals
1. **End-to-End Tracing Continuity**: Đảm bảo mọi luồng xử lý không bị đứt gãy trace context.
2. **Context Propagation**: Truyền trace context qua các ranh giới dịch vụ (API -> NATS -> Worker -> DB/Kafka -> SinkWorker).
3. **Tracing for All Channels**:
   - Tích hợp tracing cho Kafka Consumer (SinkWorker) bao gồm việc trích xuất trace context từ Kafka message headers và tạo child span.
   - Tích hợp tracing cho background tasks định kỳ (StuckJobReaper) và các Schedulers.
   - Tích hợp tracing vào tầng Handlers/Entrypoints của cdc-cms-service.
4. **Graceful Shutdown Integration**: Tránh mất mát span data bằng cách flush trace buffer trước khi process exit.

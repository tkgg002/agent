# Workspace Context: TracesHardening

## Goal
Khôi phục và tối ưu hóa hệ thống observability end-to-end (tracing) cho các worker và API services thuộc hệ thống CDC. Giải quyết dứt điểm tình trạng span loss trong quá trình graceful shutdown và đảm bảo trace context được truyền chính xác (propagation) qua các luồng xử lý NATS/Kafka.

## Tech Stack
- OpenTelemetry Go SDK (Traces, Logs, Metrics)
- SigNoz / ClickHouse collector pipeline
- NATS & Kafka Connect (message messaging layers)
- Zap Logger with OTel bridge

## Current Issues
1. **Graceful Shutdown Span Loss**: Cả `centralized-data-service` (worker) và `cdc-cms-service` (API server) đều sử dụng `os.Exit(0)` trong goroutine lắng nghe signal, làm bỏ qua các `defer` block (`otelShutdown()`, `logger.Sync()`), gây mất span của các request cuối.
2. **Context Propagation in NATS Handlers**: Cần audit và bổ sung/khôi phục trace context propagation thông qua NATS-driven reconciliation handlers để spans cha-con (parent-child spans) hiển thị liền mạch trên dashboard SigNoz.
3. **OTLP Collector Config Audit**: Đảm bảo cấu hình pipeline của collector gửi chính xác dữ liệu về SigNoz/ClickHouse.

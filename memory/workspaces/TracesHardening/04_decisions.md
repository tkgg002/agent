# Architectural Decisions: Traces Hardening

## 1. Trace Context Propagation over NATS
- **Quyết định**: Sử dụng định dạng W3C TraceContext (`traceparent` và `tracestate`) truyền qua `nats.Msg.Header`.
- **Lý do**: Đảm bảo tính liên tục của vết trace (end-to-end trace continuity) khi đi qua các ranh giới dịch vụ không đồng bộ (asynchronous messaging boundary) giữa API gateway và worker nodes, thay vì tạo trace mới gây đứt gãy.
- **Hiện thực**: 
  - Trích xuất ở bên nhận (NATS Handler) bằng `observability.ExtractNATSHeader`.
  - Tiêm vào bên gửi (NATS Publisher / Client) bằng `observability.InjectNATSHeader`.

## 2. Kafka Consumer (SinkWorker) Context Propagation
- **Quyết định**: Trích xuất trace context trực tiếp từ các message headers của Kafka message và bọc logic xử lý trong một Child Span (`kafka.consume.sink`).
- **Lý do**: Kết nối vết trace từ CDC engine đến các tiến trình ghi xuống database (Materialization), hoàn tất chuỗi observability của một dữ liệu CDC.

## 3. OTel Logging Bridge Integration
- **Quyết định**: Tích hợp OTel log bridge (`observability.NewOTelBridgeCore`) song song với structured logger (`zap`) hiện tại ở các entrypoint của dịch vụ.
- **Lý do**: Tự động liên kết log dòng xử lý với `trace_id` và `span_id` tương ứng, giúp SigNoz/Clickhouse có thể ánh xạ trực tiếp từ log sang biểu đồ trace mà không cần thủ công truy vấn chéo.

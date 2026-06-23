# Plan: Traces Hardening & End-to-End Observability

## Proposed Phases

### Phase 1: SinkWorker Trace Context Extraction & Child Span Creation
- [x] Thêm các package `go.opentelemetry.io/otel`, `go.opentelemetry.io/otel/propagation`, `go.opentelemetry.io/otel/attribute` vào `internal/sinkworker/sinkworker.go`.
- [x] Cập nhật `HandleMessage` để trích xuất trace context từ `msg.Headers`.
- [x] Tạo child span `kafka.consume.sink` bọc quanh toàn bộ logic xử lý của message.
- [x] Cập nhật việc xử lý lỗi (gán `handleErr`) để OTel span ghi nhận lỗi chuẩn xác khi logic xử lý thất bại.

### Phase 2: SinkWorker Command Entrypoint Tracing Integration
- [x] Cấu hình OpenTelemetry Tracer Provider (`observability.InitOtel`) trong `cmd/sinkworker/main.go`.
- [x] Tích hợp OTel log bridge (`NewOTelBridgeCore` và `zapcore.NewTee`) trong `cmd/sinkworker/main.go` để chuyển tiếp log về SigNoz.
- [x] Đảm bảo flush trace buffer (`defer otelShutdown()`) tại dòng thoát của main function.

### Phase 3: StuckJobReaper Periodic Tasks Tracing Integration
- [x] Thêm package `observability` vào `internal/infra/messaging/stuck_job_reaper.go`.
- [x] Cập nhật phương thức `reapOnce` để khởi tạo span `cdc.cms.stuck_job_reaper` bọc quanh query SQL sweep.
- [x] Ghi nhận lỗi quét job (`err = res.Error`) vào span trước khi return.

### Phase 4: Verification & Global Audit
- [x] Chạy biên dịch toàn bộ các dịch vụ (`go build ./...`) trong `centralized-data-service` và `cdc-cms-service`.
- [x] Đảm bảo không có lỗi runtime/compile hay import cycle.
- [x] Chạy `/security-agent` để rà soát an toàn trước khi kết thúc task.

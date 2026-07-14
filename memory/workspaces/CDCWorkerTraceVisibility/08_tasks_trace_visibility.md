# Task List - Tối ưu Visibility Traces & Đặt tên Span Động (Toàn diện)

- [x] Định nghĩa `ChildSpanWithLinks` trong `centralized-data-service/pkgs/observability/trace_helpers.go`.
- [x] Bổ sung trường `TraceContext` vào struct `UpsertRecord` trong `centralized-data-service/internal/model/shadow/cdc_event.go`.
- [x] Gán `record.TraceContext = ctx` trong `centralized-data-service/internal/handler/shadow/event_handler.go`.
- [x] Cập nhật `batchUpsert` và `publishTransmuteTrigger` trong `batch_buffer.go` và `batch_buffer_fanout.go` để gom links và tạo span động.
- [x] Cập nhật tên span động cho các chặng CDC Flow chính (`kafka_consumer.go`, `schema_inspector.go`, `transmute_handler.go`, `transmuter.go`).
- [x] Sửa đứt gãy context trong goroutine bất đồng bộ của `HandleTransmute` trong `transmute_handler.go`.
- [x] Cập nhật tên span động cho `internal/sinkworker/worker.go` (`kafka.consume.sink`).
- [x] Cập nhật tên span động cho toàn bộ DDL, Index, Scan, Provisioning và Orchestration Handlers.
- [x] Cập nhật tên span động cho Reconciliation Core Engine (`recon_tier_a.go` và `recon_tier_b.go`).
- [x] Cập nhật tên span động cho các luồng chu kỳ (Scheduler & Cycles: `server_jobs.go`).
- [x] Cập nhật tên span động trong `cdc-cms-service` (Saga runner, Command Bus).
- [x] Chạy biên dịch và kiểm thử để xác minh.

# Plan: Scheduler Tracing & Log Grouping

## Các bước thực hiện

### Bước 1: Khởi tạo OpenTelemetry Span cho Scheduler Cycle
* Sửa file `internal/server/worker_server.go`:
  * Tại đầu vòng lặp tick của scheduler ticker (ngay khi ticker kích hoạt): Khởi tạo một trace span gốc `cdc.worker.scheduler_cycle` sử dụng `observability.ChildSpan(context.Background(), "cdc.worker.scheduler_cycle")`.
  * Đảm bảo span này được kết thúc (`span.End()`) ở cuối chu kỳ xử lý tick.
  * Cập nhật các hàm log trong tick loop sử dụng `observability.Ctx(ctx, s.logger)` thay vì `s.logger`.
  * Truyền context (`ctx`) chứa trace span này xuống các DB queries và các lệnh gọi đến các cycle handlers (`runTransformCycle`, `runPartitionCheck`, `runReconcileCycle`).

### Bước 2: Cập nhật các Cycle Handlers (Signature & Tracing)
* Sửa file `internal/server/worker_server_tickers.go` và các lời gọi tương ứng trong `worker_server.go`:
  * Cập nhật signature các hàm `runTransformCycle`, `runPartitionCheck`, `runReconcileCycle` để nhận `ctx context.Context` làm tham số đầu tiên.
  * Thay vì sử dụng `context.Background()` để tạo ChildSpan trong các hàm này, hãy truyền `ctx` nhận được từ scheduler cycle để OpenTelemetry tự động liên kết chúng thành các Child Spans của `cdc.worker.scheduler_cycle`.
  * Di chuyển các dòng log khởi động chu kỳ (như `"reconcile cycle started"`) xuống phía sau khai báo span và sử dụng `observability.Ctx(ctx, s.logger)` để in log.

### Bước 3: Cập nhật logs của Reconcile Engine để inject trace context
* Sửa các file logs trong package `recon` (`recon_engine_run.go`, `recon_tier_a.go`, `recon_tier_b.go`):
  * Chuyển tất cả các dòng log có liên quan (`rc.logger.Info`, `rc.logger.Warn`, `rc.logger.Error`) sang dạng sử dụng `observability.Ctx(ctx, rc.logger)` để SigNoz có thể nhận diện `trace_id` / `span_id` và nhóm logs vào `reconcile_cycle` một cách chính xác.

### Bước 4: Kiểm thử và Xác minh
* Biên dịch dự án bằng `go build ./...`.
* Khởi động lại worker và quan sát log stream để đảm bảo cấu trúc log JSON có đầy đủ `trace_id` và `span_id`.

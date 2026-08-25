# Requirements: Fix Log Trace Connection

## Mục tiêu
Đảm bảo 100% logs nghiệp vụ và logs xử lý trong các handler, worker của `centralized-data-service` được liên kết chính xác với traces (OpenTelemetry Span Context) thông qua helper `observability.Ctx(ctx, logger)`.

## Các yêu cầu chi tiết
1. **Rà soát & Sửa đổi các Handlers trong `centralized-data-service`**:
   - `internal/handler/source/bridge_handler.go` và `bridge_mongo.go`
   - `internal/handler/recon/recon_execute_heal_handler.go`, `recon_check_handler.go`, `recon_sysops_handler.go`, `recon_heal_fetch.go`, `recon_job_handler.go`
   - `internal/handler/base/base_handler.go` (đặc biệt là phương thức `LogCommandResult`, `PublishResult`, `PublishResultWithSubject`)
   - `internal/handler/scan/scan_handler.go`
   - `internal/handler/governance/index_handler.go`
   - `internal/handler/master/transmute_handler.go` và `master_ddl_handler.go`
   - `internal/handler/shadow/schema_ddl_handler.go`, `batch_transform_handler.go`, `consumer_pool.go`, `event_bridge.go`
   - `internal/handler/orchestration/snapshot_runner_handler.go`, `provisioning_schedule_enable.go`

2. **Cơ chế liên kết**:
   - Thay thế các lệnh gọi log trực tiếp (`logger.Info`, `logger.Warn`, `logger.Error`) bằng `observability.Ctx(ctx, logger).Info` / `Warn` / `Error`.
   - Nếu hàm chưa có context `ctx`, thực hiện truyền `ctx context.Context` từ lớp gọi xuống (hoặc tạo `ctx` từ message header/correlation ID).
   - Đảm bảo trích xuất trace context từ NATS message header hoặc payload nếu chạy async command/event.

3. **Verify**:
   - Biên dịch và chạy thử tests của `centralized-data-service` để đảm bảo không bị compile error và không phá vỡ logic cũ.

# 00_context — Bug: lần snapshot.v2 đầu tiên không ghi data vào shadow

## Trigger user
> "kiêm tra sao lần snapshot đầu ko ghi data vào shadow. debug nó xem."
> (2026-05-26, sau khi vừa fix VARCHAR overflow + circuit breaker ở
> workspace `bug-snapshot-v2-host-uri-2026-05-21`)

## Phạm vi
- Service: `centralized-data-service` (worker plane).
- Tính năng: snapshot.v2 (Path B) — NATS subject `cdc.cmd.snapshot.v2`.
- Triệu chứng: lần snapshot ĐẦU TIÊN cho một source_object vừa được
  cấu hình → activity_log báo `status=success` với `rows_affected = N`
  (= số doc Mongo Find quét được) NHƯNG shadow table **0 row**.
  Lần snapshot kế tiếp (sau khi có sự kiện reload registry hoặc user
  enable lại flag) thì ghi được data bình thường.

## File liên quan (đã đọc)
- `centralized-data-service/internal/handler/snapshot_runner_handler.go`
- `centralized-data-service/internal/handler/event_handler.go`
- `centralized-data-service/internal/handler/batch_buffer.go`
- `centralized-data-service/internal/service/schema_adapter.go`
- `centralized-data-service/internal/service/metadata_registry_service.go`
- `centralized-data-service/internal/service/connection_manager.go`
- `centralized-data-service/internal/server/worker_server.go`
- `cdc-cms-service/internal/api/snapshot_progress_handler.go`
- `cdc-cms-service/internal/api/source_object_actions_handler.go`
- `cdc-cms-service/internal/app/commands/recon_async.go`
- `cdc-cms-service/internal/app/commands/register_registry.go`
- `cdc-cms-service/internal/infra/persistence/source_object_v2_sync.go`
- `cdc-cms-service/pkgs/natsconn/nats_client.go`

## Lesson tham chiếu
- `L-route-zero-rows` (pattern Kafka consumer count = msg.processed nhưng
  HandleRaw silent-skip khi route empty → cùng pattern xảy ra ở snapshot.v2)
- `L-CDC-circuit-breaker-2026-05-22`
- `L-CDC-source-identifier-2026-05-22`

# Kế hoạch Tối ưu Visibility Traces & Đặt tên Span Động cho Toàn bộ Hệ thống CDC (Toàn diện)

Tài liệu này mô tả kế hoạch thiết kế và triển khai nhằm cải tiến toàn diện cách đặt tên và liên kết các span OpenTelemetry (traces) trong toàn bộ hệ thống CDC (bao gồm CDC Worker - `centralized-data-service` và CMS Service - `cdc-cms-service`). Mục tiêu là triệt tiêu hoàn toàn các span tĩnh mang tên chung chung (generic), giúp hiển thị trực quan, rõ ràng theo bảng dữ liệu trên SigNoz dashboard và khắc phục đứt gãy trace context.

## User Review Required

> [!IMPORTANT]
> **Phạm vi cập nhật toàn diện đã mở rộng**:
> Qua rà soát sâu 100% codebase Go, chúng tôi đã phát hiện và thêm vào kế hoạch toàn bộ các span tĩnh còn sót:
> 1. **CDC Main Flow & Sink Worker**: Kafka Consumer, EventHandler, BatchBuffer, NATS, Transmuter, và Sink Worker.
> 2. **Toàn bộ NATS Command Handlers**: DDL, Index, Discover, Scan, Sync, Provisioning, Recon, và Orchestration.
> 3. **Reconciliation Engine Cores**: Queries, Stream, Hash, Smoke test và Reaper của PostgreSQL và MongoDB (bao gồm cả Tier A & Tier B).
> 4. **CMS Service - Command Bus & Sagas**:
>    - `command_bus.execute` -> `command_bus.execute: <command_type>`
>    - `command_bus.dispatch` -> `command_bus.dispatch: <command_type>`
>    - `saga.step` -> `saga.step: <step_name>`
>    - `saga.compensate` -> `saga.compensate: <step_name>`

## Proposed Changes

### centralized-data-service

#### [MODIFY] [trace_helpers.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/pkgs/observability/trace_helpers.go)
- Thêm helper `ChildSpanWithLinks` để hỗ trợ bắt đầu các span OTel đi kèm liên kết (Links) đến các span context khác.

#### [MODIFY] [cdc_event.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/model/shadow/cdc_event.go)
- Bổ sung trường `TraceContext context.Context` vào struct `UpsertRecord` để lưu vết context chứa trace span của từng sự kiện CDC riêng lẻ.

#### [MODIFY] [event_handler.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/shadow/event_handler.go)
- Khi khởi tạo `UpsertRecord`, gán `TraceContext: ctx` từ context hiện tại để truyền thông tin trace xuống BatchBuffer.
- Thay đổi span `cdc.event_handle` thành tên động `cdc.event_handle: <table_name>`.

#### [MODIFY] [batch_buffer.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/shadow/batch_buffer.go)
- Trong `batchUpsert`, thu thập tất cả trace context từ các records trong lô, tạo danh sách `oteltrace.Link` và khởi chạy span động với tên:
  `cdc.batchbuffer.upsert: <table_name>` đi kèm các links tương ứng.

#### [MODIFY] [batch_buffer_fanout.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/shadow/batch_buffer_fanout.go)
- Trong `publishTransmuteTrigger`, thu thập các trace context của records và khởi chạy span động với tên:
  `cdc.batchbuffer.fanout: <table_name>` đi kèm các links tương ứng.

#### [MODIFY] [kafka_consumer.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/shadow/kafka_consumer.go)
- Cập nhật span tĩnh `kafka.consume` thành span động `kafka.consume: <topic>`.
- Cập nhật span tĩnh `cdc.process_message` thành span động `cdc.process_message: <topic>`.

#### [MODIFY] [schema_inspector.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/governance/schema_inspector.go)
- Cập nhật span tĩnh `cdc.schema_inspect` thành span động `cdc.schema_inspect: <table_name>`.

#### [MODIFY] [transmute_handler.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/master/transmute_handler.go)
- Cập nhật span tĩnh `nats.HandleTransmuteShadow` thành span động `nats.HandleTransmuteShadow: <shadow_table>`.
- Cập nhật span tĩnh `cdc.worker.transmute.process` thành span động `cdc.worker.transmute.process: <master_table>`.
- Khôi phục context propagation bằng cách cho goroutine bất đồng bộ kế thừa `ctx` thay vì `context.Background()`.

#### [MODIFY] [transmuter.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter.go)
- Cập nhật span tĩnh `cdc.service.transmute` thành span động `cdc.service.transmute: <master_table>`.

#### [MODIFY] [worker.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/sinkworker/worker.go)
- Cập nhật span tĩnh `kafka.consume.sink` thành span động `kafka.consume.sink: <topic>`.

#### [MODIFY] Các Handler phụ trợ (DDL, Index, Discover, Scan, Sync, Provisioning, Orchestration)
- Cập nhật tên span động cho các file:
  - [schema_ddl_handler.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/shadow/schema_ddl_handler.go)
  - [batch_transform_handler.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/shadow/batch_transform_handler.go)
  - [provisioning_shadow_bind.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/shadow/provisioning_shadow_bind.go)
  - [discover_handler.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/source/discover_handler.go)
  - [sync_handler.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/source/sync_handler.go)
  - [mongo_discover_handler.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/source/mongo_discover_handler.go)
  - [scan_handler.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/scan/scan_handler.go)
  - [master_ddl_handler.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/master/master_ddl_handler.go)
  - [index_handler.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/governance/index_handler.go)
  - [recon_execute_heal_handler.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/recon/recon_execute_heal_handler.go)
  - [recon_check_handler.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/recon/recon_check_handler.go)
  - [recon_check_heal_handler.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/recon/recon_check_heal_handler.go)
  - [recon_sysops_handler.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/recon/recon_sysops_handler.go)
  - [provisioning_schedule_enable.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/orchestration/provisioning_schedule_enable.go)
  - [provisioning_handler.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/orchestration/provisioning_handler.go)
  - [server_jobs.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/server/server_jobs.go)

#### [MODIFY] Reconciliation Engine Cores (Tier A, Tier B & Helpers)
- Cập nhật tên span động cho các file lõi đối soát:
  - [recon_tier_a.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_tier_a.go)
  - [recon_tier_b.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_tier_b.go)
  - [recon_stream.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_stream.go)
  - [recon_query.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_query.go)
  - [recon_dest_query.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_dest_query.go)
  - [recon_dest_hash.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_dest_hash.go)
  - [recon_smoke.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_smoke.go)
  - [recon_hash.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_hash.go)
  - [recon_engine_run.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_engine_run.go)

---

### cdc-cms-service

#### [MODIFY] [nats_command_bus.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/infra/messaging/nats_command_bus.go)
- Cập nhật span tĩnh `command_bus.execute` thành `command_bus.execute: <command.Type()>`.
- Cập nhật span tĩnh `command_bus.dispatch` thành `command_bus.dispatch: <command.Type()>`.

#### [MODIFY] [saga.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/app/saga/saga.go)
- Cập nhật span tĩnh `saga.step` thành `saga.step: <step_name>`.
- Cập nhật span tĩnh `saga.compensate` thành `saga.compensate: <step_name>`.

---

## Verification Plan

### Automated Tests
- Thực hiện chạy thử và biên dịch toàn bộ dự án:
  `go build ./cmd/...` trong `centralized-data-service`
  `go build ./cmd/...` trong `cdc-cms-service`
- Đảm bảo các unit tests trong `pkgs/observability` và `internal/handler` / `internal/app/saga` chạy thành công:
  `go test ./test/pkgs/observability/...`
  `go test ./internal/app/saga/...` (trong `cdc-cms-service`)

### Manual Verification
- Chạy hệ thống local và theo dõi traces xuất ra trên SigNoz dashboard để kiểm tra định dạng trace/span xuất ra xem có đúng dạng tên động đi kèm các links liên kết liên tục hay không.

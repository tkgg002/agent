# Yêu cầu - Tối ưu Visibility Traces & Đặt tên Span Động cho CDC System (Toàn diện)

Cải tiến việc đặt tên các span và liên kết trace context của OpenTelemetry trong toàn bộ hệ thống CDC (bao gồm CDC Worker - `centralized-data-service` và CMS Service - `cdc-cms-service`) để đảm bảo vết thực thi (trace history) hiển thị trực quan, rõ ràng, dễ phân biệt giữa các bảng/luồng dữ liệu khác nhau trên SigNoz dashboard.

## Yêu cầu chi tiết

1. **Đặt tên Span Động (Dynamic Span Names)**:
   Thay đổi các span tĩnh trong toàn bộ hệ thống thành tên động, bổ sung thêm thông tin định danh (tên bảng, topic, collection, hoặc loại command/saga):
   - **CDC Main Flow**: `kafka.consume`, `cdc.process_message`, `cdc.event_handle`, `cdc.schema_inspect`, `cdc.batchbuffer.upsert`, `cdc.batchbuffer.fanout`, `nats.HandleTransmuteShadow`, `cdc.worker.transmute.process`, `cdc.service.transmute`.
   - **Sink Worker**: `kafka.consume.sink`.
   - **NATS Handlers**: Toàn bộ DDL, Index, Discover, Scan, Sync, Provisioning, Recon, và Orchestration Handlers.
   - **Reconciliation Engine Cores**: Các span liên quan đến query, stream, hash, smoke test của PostgreSQL và MongoDB (bao gồm cả Tier A và Tier B, hash/smoke helpers, reap operations).
   - **CMS Service - Command Bus & Sagas**:
     - `command_bus.execute` -> `command_bus.execute: <command_type>`
     - `command_bus.dispatch` -> `command_bus.dispatch: <command_type>`
     - `saga.step` -> `saga.step: <step_name>`
     - `saga.compensate` -> `saga.compensate: <step_name>`
   - **CMS Service - API Spans**:
     - `api.mapping_rule.batch_update`
     - `api.registry.bulk_register`
     - `api.registry.update`
     - `api.registry.register`
     - `api.master.approve`

2. **Liên kết Context qua BatchBuffer (Trace Context Propagation through BatchBuffer)**:
   - Thêm thuộc tính `TraceContext context.Context` vào struct `UpsertRecord`.
   - Lưu lại context của event hiện tại (chứa span active) vào trường `TraceContext` của record khi add vào buffer.
   - Khi Flush(), tạo liên kết (OTel Links) từ span batch `cdc.batchbuffer.upsert` và `cdc.batchbuffer.fanout` tới tất cả các trace context riêng lẻ của các record.

3. **Khôi phục Context Propagation trong Transmuter**:
   - Khắc phục đứt gãy context trong goroutine bất đồng bộ của `HandleTransmute` bằng cách kế thừa từ context chứa span `cdc.worker.transmute.process`.

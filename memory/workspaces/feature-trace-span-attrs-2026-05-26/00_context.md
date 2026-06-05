# 00_context — Trace Span Attributes & Log↔Trace Correlation (Phase 2)

## Triệu chứng user báo
SigNoz Trace list hiển thị:
```
2026-05-26 14:21:08.625 | cdc-worker | kafka.consume | 0.41ms | N/A | N/A
2026-05-26 14:21:08.624 | cdc-worker | kafka.consume | 0.72ms | N/A | N/A
...
```
2 cột cuối (Linked Logs / Linked Spans) đều `N/A`. Click vào 1 trace chỉ thấy đúng 1 span đơn lẻ. Không có business context (table, operation, rows_affected). Không correlate được sang log.

## Audit codebase

### Span starts trong toàn `internal/`
```
internal/handler/kafka_consumer.go:382  spanCtx, span := observability.StartSpan(parentCtx, "kafka.consume", ...)
```
**CHỈ DUY NHẤT 1 chỗ**. Tất cả downstream (`processMessage`, `EventHandler.HandleRaw`, `BatchBuffer.batchUpsert`, `SchemaInspector.InspectEvent`, transforms, DLQ write) nhận ctx (spanCtx) nhưng KHÔNG start child span. Vì vậy trace tree = root đơn lẻ → Linked Spans = N/A (không có child).

### Attributes đã có ở `kafka.consume`
- `messaging.system=kafka`
- `messaging.destination=<topic>`
- `messaging.operation=receive`
- `messaging.kafka.partition`
- `messaging.kafka.offset`
- `messaging.kafka.message.timestamp_ms`
- `e2e_latency_seconds` (conditional)
- `duration_seconds` (conditional)
- `error=<string>` (conditional, NÊN ĐỔI sang `span.RecordError(err)` để SigNoz Events tab nhận)

### Business attrs THIẾU
- `cdc.source_table` (parsed từ topic: `cdc.gpay.public.orders` → `public.orders`)
- `cdc.operation` (`c`/`u`/`d`/`r` từ Debezium envelope)
- `cdc.rows_affected` (đang ở metrics, span chưa set)
- `cdc.engine` (gpay/goopay/mariadb từ topic prefix)

### Log↔Trace correlation
Log call site trong scope `spanCtx` (line 403, 441, 453):
```go
kc.logger.Error("kafka message processing failed", zap.String("topic", ...), ...)
```
Dùng `kc.logger` raw, **KHÔNG** dùng `observability.Ctx(spanCtx, kc.logger)` (helper vừa làm phase 1). Hệ quả: log emit không kèm `trace_id`/`span_id` → SigNoz không link được log với trace → cột "Linked Logs" = N/A.

### Propagation hiện trạng
- W3C TraceContext propagator ĐÃ SET (`pkgs/observability/otel.go:313-316`). OK.
- `parentCtx := otel.GetTextMapPropagator().Extract(ctx, carrier)` (line 380) → extract trace từ Kafka header (Debezium chưa inject — sẽ là root tự sinh, OK).
- `spanCtx` được truyền vào `processMessage(spanCtx, msg)` line 401 — propagation tốt nội bộ.
- `flushBatch(ctx, msg.Topic)` (line 430) dùng `ctx` (outer reader ctx), **KHÔNG** dùng `spanCtx` → batch upsert tách context. Cần review.

## Scope phase này
1. **Bổ sung business attrs** cho `kafka.consume` span.
2. **Tạo child span** ở 4 op critical:
   - `cdc.process_message` (kafka_consumer.processMessage)
   - `cdc.event_handle` (event_handler.HandleRaw)
   - `cdc.batch_upsert` (batch_buffer.batchUpsert) — chú ý ctx propagation từ flushBatch
   - `cdc.schema_inspect` (schema_inspector.InspectEvent)
3. **Migrate log call site** trong scope các span trên sang `observability.Ctx(ctx, log)` để inject trace_id/span_id (~10-15 call site critical, không hàng loạt).
4. **`span.RecordError(err)`** thay vì `span.SetAttributes(attribute.String("error", err.Error()))` để SigNoz render Exception tab.

## Out of scope
- Migrate hàng loạt log call site (~100+) → defer.
- Thêm span cho mọi op (DDL executor, DLQ write, snapshot, transmute, ...) → defer phase sau.
- Producer-side trace injection (Debezium Kafka Connect interceptor) → defer (cần upstream config).
- Sampling tuning → phase này giữ `sampleRatio: 1.0` cho local. Production cần ratio thấp hơn — chú ý ở report.
- KHÔNG cheat config / DB.

## Risks
- **Overhead**: thêm 4-5 span / message. Với hot path ~1000 msg/s, ~5000 span/s. Tracer ở local sampleRatio=1.0 → SigNoz có thể gặp ingestion pressure. Mitigation: production hạ `sampleRatio` xuống 0.05-0.1 sau khi verify.
- **Hot path latency**: span.Start có cost ~1µs (no-op nếu disabled). Acceptable cho processMessage, batchUpsert. Cần benchmark sau.
- **Context propagation gãy**: `flushBatch` hiện dùng outer ctx. Nếu thêm span cho batchUpsert, span sẽ là root mới (không nested dưới kafka.consume). Lựa chọn: (a) chấp nhận, batch upsert là async job; (b) pass spanCtx qua channel kèm batch.

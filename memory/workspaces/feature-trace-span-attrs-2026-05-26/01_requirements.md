# 01_requirements — Trace Phase 2

## Functional
- **R1**: `kafka.consume` span thêm business attrs:
  - `cdc.source_table` (parsed: bóc tách `<prefix>.<db>.<schema>.<table>` → `<schema>.<table>`).
  - `cdc.engine` (đầu topic: `gpay`/`goopay`/`mariadb`/`mongo`).
  - `cdc.operation` (`c`/`u`/`d`/`r` extract từ Debezium payload `op` field). Set sau `processMessage` parse xong, dùng `span.SetAttributes`.
  - `cdc.rows_affected` (= `rows` từ `processMessage`).
- **R2**: Tạo child span `cdc.process_message` trong `kafka_consumer.processMessage`. Attributes: `cdc.payload.kind` (debezium/raw), `cdc.value_size_bytes`, `cdc.has_schema_id`.
- **R3**: Tạo child span `cdc.event_handle` trong `event_handler.HandleRaw`. Attributes: `cdc.subject`, `cdc.data_size_bytes`, `cdc.rows`.
- **R4**: Tạo child span `cdc.batch_upsert` trong `batch_buffer.batchUpsert`. Attributes: `cdc.batch_size`, `cdc.target_table`, `cdc.target_schema`, `cdc.rows_affected`. Span phải nested dưới upstream span của FIRST record's ctx.
- **R5**: Tạo child span `cdc.schema_inspect` trong `schema_inspector.InspectEvent`. Attributes: `cdc.table`, `cdc.source_db`, `cdc.new_field_count`.
- **R6**: Mọi log call site trong scope các span trên CHUYỂN sang `observability.Ctx(spanCtx, logger)` để emit log với `trace_id`/`span_id`.
- **R7**: Error path: `span.RecordError(err)` + `span.SetStatus(codes.Error, msg)` thay vì set string attribute. SigNoz Events tab sẽ hiển thị.
- **R8**: KHÔNG đổi span name `kafka.consume` (tránh break dashboard SigNoz đang dùng).

## Non-functional
- **N1**: Overhead per message: < 50µs cho 5 spans. Span no-op nếu tracer disabled.
- **N2**: KHÔNG đổi signature public của HandleRaw/Upsert/Inspect (đã nhận ctx — chỉ thêm span bên trong).
- **N3**: Tương thích severityAwareCore + audit bypass + log_template helpers.
- **N4**: KHÔNG đụng config (sampleRatio, OTLP endpoint).
- **N5**: KHÔNG cheat DB / mock metric / fake log.
- **N6**: Build + vet + test toàn repo PASS.

## DoD
- **A1**: 1 file mới + 4 file edit (kafka_consumer, event_handler, batch_buffer, schema_inspector). Đếm chính xác trong report.
- **A2**: Build/vet/test all EXIT=0. Evidence log path.
- **A3**: Unit test mới: `TestProcessMessage_CreatesChildSpan` (mock tracer, assert span hierarchy).
- **A4**: Smoke manual (user verify SigNoz UI): trace tree hiển thị 5 span nested, attributes có business fields, Linked Logs hiển thị log có trace_id.
- **A5**: Report `report_trace_span_attrs_2026-05-26.md` đầy đủ 13 section như phase 1.
- **A6**: Append progress + lesson global.

## Inverse requirements
- NOT migrate hàng loạt log site.
- NOT add span cho DLQ write / DDL exec / snapshot / transmute (defer).
- NOT đụng sampleRatio config.
- NOT đổi span name `kafka.consume`.
- NOT thay đổi public function signatures.

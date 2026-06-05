# 08_tasks

## M1 — Helper observability
- T1.1: Tạo `pkgs/observability/trace_helpers.go` với 4 function: `EndSpan`, `ParseDebeziumTopic`, `SourceTableAttr`, `ChildSpan`.
- T1.2: Build observability package → EXIT=0.

## M2 — kafka_consumer
- T2.1: Bổ sung 3 attr business (`cdc.source_table`, `cdc.engine`, `cdc.source_db`) tại line 382.
- T2.2: Đổi error path line 409 sang `span.RecordError + SetStatus`.
- T2.3: Bổ sung `cdc.rows_affected` tại success path line 416.
- T2.4: Wrap `processMessage` (line 460) với child span `cdc.process_message`.
- T2.5: Migrate 3 log site (line 403, 441, 453) sang `Ctx(spanCtx, logger)`.
- T2.6: Build internal/handler → EXIT=0.

## M3 — event_handler
- T3.1: Wrap `HandleRaw` (line 64) với child span `cdc.event_handle` + defer EndSpan.
- T3.2: Identify + migrate 2 log site critical trong scope.
- T3.3: Build → EXIT=0.

## M4 — batch_buffer + ctx propagation
- T4.1: Đổi `batchUpsert` signature nhận ctx (`func (bb *BatchBuffer) batchUpsert(ctx context.Context, records []*model.UpsertRecord) (err error)`).
- T4.2: Cập nhật caller `flushBatch` truyền ctx (hoặc tạo span ngay tại flushBatch).
- T4.3: Wrap với child span `cdc.batch_upsert` + attrs.
- T4.4: Migrate 2 log site critical (line 181, +1 error path).
- T4.5: Build → EXIT=0.

## M5 — schema_inspector
- T5.1: Wrap `InspectEvent` (line 87) với child span `cdc.schema_inspect`.
- T5.2: Migrate log site line 162 (drift detected) bổ sung `Ctx(ctx, si.logger)`.
- T5.3: Build → EXIT=0.

## M6 — Final log migration sweep (critical only)
- T6.1: Verify 10 site migrate đúng pattern (check Ctx ngay khi vào scope span).
- T6.2: Build full → EXIT=0.

## M7 — Unit test
- T7.1: Tạo `pkgs/observability/trace_helpers_test.go` với 3 test: `TestParseDebeziumTopic`, `TestChildSpan_ParentChild`, `TestEndSpan_RecordsError`.
- T7.2: Test PASS 3/3.

## M8 — Verify + Report
- T8.1: `go build ./... && go vet ./... && go test ./...` toàn repo EXIT=0.
- T8.2: Lưu log path cho mỗi command.
- T8.3: Tạo `report_trace_span_attrs_2026-05-26.md` đầy đủ 13 section.
- T8.4: Append `05_progress.md`.
- T8.5: Append lesson global `L-2026-05-26 trace child span + log correlation pattern`.
- T8.6: Pre-flight checklist §14 trong report.

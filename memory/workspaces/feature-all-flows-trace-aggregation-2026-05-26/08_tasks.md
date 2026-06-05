# 08_tasks — Detailed Task List

## M0 — Pre-flight
- T0.1: Verify branch hiện tại (`git rev-parse --abbrev-ref HEAD`) — nếu `main`/`master`, tạo branch mới `feature/all-flows-trace-aggregation`.
- T0.2: Snapshot baseline: `grep -rn "context.Background()" internal/ pkgs/ | wc -l > /tmp/baseline_bg_count.txt`.
- T0.3: `go build ./... && go test ./...` xác định baseline pass.

## M1 — Helper API
- T1.1: Tạo `pkgs/observability/propagation.go` với `natsHeaderCarrier`, `NATSExtract`, `NATSInject`, `kafkaHeaderCarrier`, `KafkaInject`, `KafkaExtract`.
- T1.2: Tạo `pkgs/observability/flow_helpers.go` với `EntrySpan`, `BackgroundTick`, `LinkFromContext`, `StartSpanWithLinks`.
- T1.3: Build `./pkgs/observability/...` → EXIT=0.
- T1.4: Tạo `propagation_test.go` (5 test) + `flow_helpers_test.go` (5 test). Run: PASS 10/10.

## M2 — NATS Subscriber Instrumentation
- T2.1: Sửa `recon_handler.go` — 6 handler. EntrySpan + propagate ctx + ErrorField log migration.
- T2.2: Sửa `command_handler.go` — 16 handler.
- T2.3: Sửa `transmute_handler.go` — 2 handler.
- T2.4: Sửa `job_monitor.go` — HandleCompleted.
- T2.5: Sửa `master_ddl_handler.go` — HandleMasterCreate.
- T2.6: Sửa `provisioning_handler.go` — HandleStepCompleted.
- T2.7: Sửa `provisioning_step_handlers.go` — 2 handler.
- T2.8: Sửa `worker_server.go` inline NATS callbacks — `schema.config.reload`, `cdc.cmd.kafka.refresh-topics`.
- T2.9: Build `./internal/handler/...` EXIT=0.

## M3 — NATS Publisher Inject
- T3.1: `grep -rn "Publish\|PublishMsg\|PublishRequest" internal/ cmd/` → liệt kê publish sites.
- T3.2: Convert raw `Publish(subject, data)` → `PublishMsg(&nats.Msg{...})` + `NATSInject(ctx, msg)` cho từng site.
- T3.3: Verify không break `injectTraceContext` legacy (provisioning_orchestrator) — coexist.
- T3.4: Build EXIT=0.

## M4 — Kafka Producer Inject
- T4.1: Sửa `internal/service/debezium_signal.go` `TriggerIncrementalSnapshot` — thêm `Headers` field + `KafkaInject`.
- T4.2: Search Kafka producer khác: `grep -rn "kafka.Writer\|WriteMessages" internal/ cmd/` → instrument các site khác (sinkworker `publishTransmuteTrigger` chỉ NATS, KHÔNG Kafka — skip).
- T4.3: Build EXIT=0.

## M5 — Background Workers Root Span Per Tick
- T5.1: `transmute_scheduler.go` `tick()` wrap với `BackgroundTick("transmute_scheduler", ...)`.
- T5.2: `dlq_state_machine.go` poll loop body wrap với `BackgroundTick("dlq_retry", ...)`.
- T5.3: `partition_dropper.go` sweep wrap với `BackgroundTick("partition_dropper", ...)`.
- T5.4: `full_count_aggregator.go` daily run wrap với `BackgroundTick("full_count_aggregator", ...)`.
- T5.5: `provisioning_orchestrator.go` RecoveryLoop tick wrap.
- T5.6: `worker_server.go:660` schedule poller tick wrap.
- T5.7: `recon_core.go:850` `runReconcileCycle` wrap với `BackgroundTick("recon_core", ...)` (xem M8 tiếp tục nested).
- T5.8: Build EXIT=0.

## M6 — Snapshot V2 Refactor
- T6.1: `snapshot_runner_handler.go` `Handle()` — extract traceparent từ msg.Header trước goroutine spawn, EntrySpan `nats.cdc.cmd.snapshot.v2`, truyền ctx vào goroutine.
- T6.2: Goroutine body — tạo `snapshot.v2.run` child từ ctx.
- T6.3: `runSnapshot()` — implement chunked sub-spans pattern (`snapshot.v2.chunk` mỗi 100 batches + `snapshot.v2.batch` per cursor iteration).
- T6.4: Inject `otel_trace_id` vào `claimProgress` INSERT (raw SQL) khi span valid.
- T6.5: Inject `otel_trace_id` vào `writeActivity` `details` JSON + (sau migration apply) column dedicated.
- T6.6: Build EXIT=0.

## M7 — BatchBuffer Span Link
- T7.1: Sửa `internal/model/upsert_record.go` (hoặc file định nghĩa `UpsertRecord`) — thêm field `OriginSpanContext oteltrace.SpanContext`.
- T7.2: Sửa `event_handler.go` `processEvent` — capture `SpanContextFromContext(ctx)` khi tạo record.
- T7.3: Sửa `batch_buffer.go` `batchUpsert` — dedup origin span contexts, build Links, gọi `StartSpanWithLinks` thay `ChildSpan(Background, ...)`.
- T7.4: Sửa `batch_buffer.go` `Flush()` — pass ctx vào batchUpsert (signature đổi internal lowercase — OK per ADR-10 phase 2).
- T7.5: Build EXIT=0.
- T7.6: Verify span Links count = unique origin count via integration test.

## M8 — Recon Nested Tree
- T8.1: `recon_core.go` `runReconcileCycle` → tạo `recon.cycle` span.
- T8.2: `runTier1` / `runTier2` / `runTier3` → tạo `recon.tier.{N}` child.
- T8.3: Per-table loop → tạo `recon.table.<target>` grandchild.
- T8.4: Per-window loop trong tier 3 → tạo `recon.window.<idx>` great-grandchild.
- T8.5: Build EXIT=0.

## M9 — HTTP Handlers
- T9.1: `internal/admin/source_register.go` `handleRegisterSource` — EntrySpan.
- T9.2: Kiểm tra `internal/admin/server.go` đã có otelgin middleware chưa. Nếu rồi → skip T9.1. Nếu chưa → option A: thêm middleware; option B: manual EntrySpan per handler (chỉ 1 handler → manual OK).
- T9.3: Worker Fiber: `/health`, `/ready`, `/metrics` → skip (ADR-A13).
- T9.4: Build EXIT=0.

## M10 — Migration Script
- T10.1: Tạo `migrations/postgres/0046_add_otel_trace_id.up.sql`.
- T10.2: Tạo `migrations/postgres/0046_add_otel_trace_id.down.sql`.
- T10.3: `psql --dry-run` hoặc `sqlfluff parse` để validate cú pháp (không apply).
- T10.4: Document trong report: user lệnh riêng để apply.

## M11 — Tests
- T11.1: `propagation_test.go` — 4 test PASS.
- T11.2: `flow_helpers_test.go` — 5 test PASS.
- T11.3: `snapshot_runner_handler_trace_test.go` integration — verify trace tree.
- T11.4: `batch_buffer_trace_test.go` — verify Span Links.
- T11.5: `go test -race ./pkgs/observability/...` — PASS.

## M12 — Verify + Report
- T12.1: Full build/vet/test toàn repo EXIT=0.
- T12.2: Smoke test local worker khởi động.
- T12.3: Diff baseline `context.Background()`: 66 → ≤ 8.
- T12.4: Tạo `report_all_flows_trace_aggregation_2026-05-26.md` 13 section + pre-flight §14.
- T12.5: APPEND `05_progress.md` với entry hoàn thành.
- T12.6: APPEND `agent/memory/global/lessons.md` lesson `L-2026-05-26-all-flows-trace`.
- T12.7: Verify services chạy không panic (kiểm tra log sau smoke).

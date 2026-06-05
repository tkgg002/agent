# 02_plan — All Flows Trace Aggregation Roadmap

## Tổng quan kiến trúc

```
                  ┌──────────────────────────────────┐
                  │  Producer (CMS, scheduler,        │
                  │   ProvisioningOrch, etc.)         │
                  │  → NATS/Kafka publish with        │
                  │    traceparent injected           │
                  └──────────────┬───────────────────┘
                                 │ traceparent in header
                                 ▼
            ┌──────────────────────────────────────────┐
            │  Worker entry point (NATS Sub / Kafka /  │
            │  HTTP / Timer)                            │
            │  ↓ extract traceparent → ctx              │
            │  ↓ EntrySpan(ctx, "<flow>")               │
            └──────────────────┬───────────────────────┘
                               │ propagate ctx
            ┌──────────────────▼───────────────────────┐
            │  Sub-operations (service layer)          │
            │  ↓ ChildSpan(ctx, "<sub>")               │
            │  ↓ defer EndSpan(span, &err)             │
            └──────────────────┬───────────────────────┘
                               │ propagate ctx
            ┌──────────────────▼───────────────────────┐
            │  Outbound (downstream Publish /          │
            │   DB call / async goroutine)             │
            │  ↓ inject traceparent / save Link        │
            └───────────────────────────────────────────┘
```

## Milestones

| M | Tên | Effort | Phụ thuộc |
|---|-----|--------|-----------|
| M0 | Pre-flight: branch + workspace + lessons re-read | 10m | — |
| M1 | Helper API mới: NATSExtract/Inject, KafkaInject, EntrySpan, BackgroundTick | 45m | M0 |
| M2 | NATS subscriber instrumentation (35 subscribers) | 2h | M1 |
| M3 | NATS publisher inject (audit + sửa tất cả Publish*) | 1h | M1 |
| M4 | Kafka producer inject (`DebeziumSignalClient`) | 20m | M1 |
| M5 | Background workers root span per tick (7 workers) | 1h30m | M1 |
| M6 | Snapshot V2: root + chunked sub-spans + propagate ctx vào downstream | 1h | M1, M2 |
| M7 | BatchBuffer Span Link (record-level link tracking) | 1h | M1 |
| M8 | Recon flow nested tree (`recon.cycle/tier/table/window`) | 1h | M1 |
| M9 | HTTP handler instrumentation (Fiber worker + Gin admin) | 30m | M1 |
| M10 | Migration `otel_trace_id` columns (script only, không apply) | 20m | — |
| M11 | Unit + integration tests | 1h30m | M1-M9 |
| M12 | Verify (build/vet/test/smoke) + report + lesson | 45m | M1-M11 |

**Tổng effort estimate**: ~12 giờ. Có thể chia 2-3 phiên Muscle.

---

## M0 — Pre-flight

- Read `agent/memory/global/lessons.md` (L-2026-05-26-trace) — done in Brain phase.
- Verify branch hiện tại không phải `main`. Nếu là main: tạo branch `feature/all-flows-trace-aggregation` trước khi sửa code (Muscle decide).
- Lưu baseline: `grep -rn "context.Background()" internal/ pkgs/ | wc -l` → file `/tmp/baseline_bg_count.txt`.

---

## M1 — Helper API (`pkgs/observability/`)

### File mới: `pkgs/observability/propagation.go`

```go
package observability

import (
    "context"

    "github.com/nats-io/nats.go"
    "github.com/segmentio/kafka-go"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/propagation"
)

// natsHeaderCarrier adapts nats.Header to propagation.TextMapCarrier.
type natsHeaderCarrier nats.Header

func (c natsHeaderCarrier) Get(key string) string {
    if v := nats.Header(c).Get(key); v != "" {
        return v
    }
    return ""
}
func (c natsHeaderCarrier) Set(key, value string) { nats.Header(c).Set(key, value) }
func (c natsHeaderCarrier) Keys() []string {
    keys := make([]string, 0, len(c))
    for k := range c {
        keys = append(keys, k)
    }
    return keys
}

// NATSExtract returns a context with parent span context extracted from
// the NATS message header. When no traceparent is present, returns ctx
// unchanged so the next StartSpan creates a root span.
func NATSExtract(ctx context.Context, msg *nats.Msg) context.Context {
    if msg == nil || msg.Header == nil {
        return ctx
    }
    return otel.GetTextMapPropagator().Extract(ctx, natsHeaderCarrier(msg.Header))
}

// NATSInject writes the active span context into msg.Header. Caller must
// ensure msg.Header != nil (use nats.Header{} default).
func NATSInject(ctx context.Context, msg *nats.Msg) {
    if msg == nil {
        return
    }
    if msg.Header == nil {
        msg.Header = nats.Header{}
    }
    otel.GetTextMapPropagator().Inject(ctx, natsHeaderCarrier(msg.Header))
}

// kafkaHeaderCarrier adapts []kafka.Header.
type kafkaHeaderCarrier struct{ msg *kafka.Message }

func (c kafkaHeaderCarrier) Get(key string) string {
    for _, h := range c.msg.Headers {
        if h.Key == key {
            return string(h.Value)
        }
    }
    return ""
}
func (c kafkaHeaderCarrier) Set(key, value string) {
    // Overwrite if exists, else append.
    for i, h := range c.msg.Headers {
        if h.Key == key {
            c.msg.Headers[i].Value = []byte(value)
            return
        }
    }
    c.msg.Headers = append(c.msg.Headers, kafka.Header{Key: key, Value: []byte(value)})
}
func (c kafkaHeaderCarrier) Keys() []string {
    keys := make([]string, 0, len(c.msg.Headers))
    for _, h := range c.msg.Headers {
        keys = append(keys, h.Key)
    }
    return keys
}

// KafkaInject writes the active span context into msg.Headers in-place.
func KafkaInject(ctx context.Context, msg *kafka.Message) {
    if msg == nil {
        return
    }
    otel.GetTextMapPropagator().Inject(ctx, kafkaHeaderCarrier{msg: msg})
}

// KafkaExtract is identical to the inline code already in kafka_consumer.go;
// extracted here so all entry points use the same helper.
func KafkaExtract(ctx context.Context, msg kafka.Message) context.Context {
    carrier := propagation.MapCarrier{}
    for _, h := range msg.Headers {
        carrier[h.Key] = string(h.Value)
    }
    return otel.GetTextMapPropagator().Extract(ctx, carrier)
}
```

### File mới: `pkgs/observability/flow_helpers.go`

```go
package observability

import (
    "context"
    "fmt"

    "go.opentelemetry.io/otel/attribute"
    oteltrace "go.opentelemetry.io/otel/trace"
)

// EntrySpan starts a root-or-child span at a flow entry point. Naming
// convention: "<subsystem>.<verb>" — e.g. "nats.cdc.cmd.recon-check",
// "schedule.transmute", "snapshot.v2.run".
func EntrySpan(ctx context.Context, name string, attrs ...attribute.KeyValue) (context.Context, oteltrace.Span) {
    return Tracer().Start(ctx, name,
        oteltrace.WithAttributes(attrs...),
        oteltrace.WithSpanKind(oteltrace.SpanKindConsumer),
    )
}

// BackgroundTick starts a root span for a background timer/cron tick.
// Returns ctx + span; caller must defer EndSpan(span, &err).
func BackgroundTick(jobName string, attrs ...attribute.KeyValue) (context.Context, oteltrace.Span) {
    ctx := context.Background()
    name := fmt.Sprintf("bg.%s.tick", jobName)
    return Tracer().Start(ctx, name,
        oteltrace.WithAttributes(attrs...),
        oteltrace.WithSpanKind(oteltrace.SpanKindInternal),
    )
}

// LinkFromContext returns a Span Link from ctx's active span context,
// suitable for batch fan-in (cdc.batch_upsert linking back to N source
// messages).
func LinkFromContext(ctx context.Context, attrs ...attribute.KeyValue) oteltrace.Link {
    sc := oteltrace.SpanContextFromContext(ctx)
    return oteltrace.Link{
        SpanContext: sc,
        Attributes:  attrs,
    }
}

// StartSpanWithLinks is like ChildSpan but attaches Span Links.
func StartSpanWithLinks(ctx context.Context, name string, links []oteltrace.Link, attrs ...attribute.KeyValue) (context.Context, oteltrace.Span) {
    return Tracer().Start(ctx, name,
        oteltrace.WithAttributes(attrs...),
        oteltrace.WithLinks(links...),
    )
}
```

### File mới: test cho 2 file trên — xem M11.

---

## M2 — NATS Subscriber Instrumentation

Pattern apply cho **mọi** NATS subscriber callback. Demo cho `cdc.cmd.recon-check`:

### Trước (`recon_handler.go:91-100`):
```go
func (h *ReconHandler) HandleReconCheck(msg *nats.Msg) {
    var payload reconCheckPayload
    json.Unmarshal(msg.Data, &payload)
    ctx := context.Background()
    // ... business logic ...
}
```

### Sau:
```go
func (h *ReconHandler) HandleReconCheck(msg *nats.Msg) {
    parentCtx := observability.NATSExtract(context.Background(), msg)
    ctx, span := observability.EntrySpan(parentCtx, "nats.cdc.cmd.recon-check",
        attribute.String("nats.subject", msg.Subject),
        attribute.Int("nats.payload_size_bytes", len(msg.Data)),
    )
    var err error
    defer observability.EndSpan(span, &err)

    var payload reconCheckPayload
    if err = json.Unmarshal(msg.Data, &payload); err != nil {
        return
    }
    span.SetAttributes(
        attribute.String("recon.target_table", payload.TargetTable),
        attribute.String("recon.source_db", payload.SourceDB),
    )
    // ... business logic — uses ctx ...
}
```

### Subscribers to instrument (35 total)

Map theo file:

| File | Handlers | Action |
|------|----------|--------|
| `recon_handler.go` | HandleReconCheck, HandleReconHeal, HandleRetryFailed, HandleBackfillSourceTs, HandleDetectTimestampField, HandleDebeziumSignal | EntrySpan + propagate ctx |
| `snapshot_runner_handler.go` | Handle (route to goroutine) | EntrySpan TRƯỚC goroutine, truyền ctx (xem M6) |
| `command_handler.go` | HandleStandardize, HandleDiscover, HandleBackfill, HandleScanRawData, HandleBatchTransform, HandlePeriodicScan, HandleDropGINIndex, HandleCreateDefaultColumns, HandleDiscoverMongoDatabases, HandleDiscoverMongoCollections, HandleScanFields, HandleSyncRegister, HandleSyncState, HandleRestartDebezium, HandleAlterColumn, HandleMasterSwap | EntrySpan + propagate |
| `transmute_handler.go` | HandleTransmute, HandleTransmuteShadow | EntrySpan |
| `job_monitor.go` | HandleCompleted | EntrySpan |
| `master_ddl_handler.go` | HandleMasterCreate | EntrySpan |
| `provisioning_handler.go` | HandleStepCompleted | EntrySpan |
| `provisioning_step_handlers.go` | HandleShadowBind, HandleScheduleEnable | EntrySpan |
| `worker_server.go` (inline) | schema.config.reload, cdc.cmd.kafka.refresh-topics | EntrySpan |

---

## M3 — NATS Publisher Inject

### Pattern
Tìm tất cả `nc.Publish(`, `nc.PublishMsg(`, `nc.PublishRequest(`, `nc.PublishMsgWithContext(`, `js.PublishMsg(`. Trước khi publish, đảm bảo msg là `*nats.Msg` (không phải `nc.Publish(subject, data)` raw) và gọi `NATSInject(ctx, msg)`.

### Trước (provisioning_orchestrator.go ~ L107):
```go
// existing injectTraceContext chỉ inject vào payload JSON, KHÔNG vào nats header
nc.Publish(subject, payloadJSON)
```

### Sau:
```go
msg := &nats.Msg{Subject: subject, Data: payloadJSON, Header: nats.Header{}}
observability.NATSInject(ctx, msg)
if err := nc.PublishMsg(msg); err != nil { ... }
```

### Publish sites cần đụng

Estimate ~25 publish sites trong:
- `command_handler.go` (reply pattern + dispatch)
- `transmute_scheduler.go` (publish cdc.cmd.transmute, cdc.cmd.batch-transform)
- `dlq_state_machine.go` (publish retry commands)
- `provisioning_orchestrator.go` (publish provisioning step commands)
- `recon_handler.go` (publish heal commands)
- `cmd/sinkworker/main.go` (publish cdc.cmd.transmute-shadow)
- `internal/admin/source_register.go` (publish cdc.cmd.kafka.refresh-topics)

Muscle: grep `nc.Publish\|NATS.Publish\|jetstream.*Publish\|js.Publish` để liệt kê đầy đủ trước khi sửa.

---

## M4 — Kafka Producer Inject

### File: `internal/service/debezium_signal.go` (~ L214)

### Trước:
```go
msg := kafka.Message{
    Key:   []byte(topicPrefix),
    Value: body,
}
err := writer.WriteMessages(ctx, msg)
```

### Sau:
```go
msg := kafka.Message{
    Key:     []byte(topicPrefix),
    Value:   body,
    Headers: make([]kafka.Header, 0, 2),
}
observability.KafkaInject(ctx, &msg)
err := writer.WriteMessages(ctx, msg)
```

---

## M5 — Background Workers Root Span Per Tick

### Worker template

```go
// Hiện tại — long-running loop
func (w *TransmuteScheduler) Start(ctx context.Context) {
    ticker := time.NewTicker(w.interval)
    defer ticker.Stop()
    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            w.tick(ctx)  // span được tạo bên trong tick?
        }
    }
}
```

### Sau

```go
func (w *TransmuteScheduler) Start(ctx context.Context) {
    ticker := time.NewTicker(w.interval)
    defer ticker.Stop()
    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            w.runTick()
        }
    }
}

func (w *TransmuteScheduler) runTick() {
    tickCtx, span := observability.BackgroundTick("transmute_scheduler",
        attribute.String("scheduler.interval", w.interval.String()),
    )
    var err error
    defer observability.EndSpan(span, &err)

    err = w.tick(tickCtx)  // tick now receives ctx with span
}
```

### Workers to instrument

| Worker | File | Job name | Frequency |
|--------|------|----------|-----------|
| TransmuteScheduler | `transmute_scheduler.go` | `transmute_scheduler` | 60s |
| DLQStateMachine | `dlq_state_machine.go` | `dlq_retry` | 5m |
| PartitionDropper | `partition_dropper.go` | `partition_dropper` | 24h |
| FullCountAggregator | `full_count_aggregator.go` | `full_count_aggregator` | 24h (daily 03:00) |
| ProvisioningOrch.RecoveryLoop | `provisioning_orchestrator.go` | `provisioning_recovery` | 1m |
| SchedulePoller | `worker_server.go:660` | `schedule_poller` | 60s |
| ReconCore.runReconcileCycle | `recon_core.go:850` | `recon_core_cycle` | per leader heartbeat trigger |
| KafkaConsumer.flushTimer + topic refresh | `kafka_consumer.go:305,313` | đã có spans per-message → CHỈ thêm cho topic refresh tick |

Note: ReconCore là leader-elected, span per cycle khi leader trigger.

---

## M6 — Snapshot V2 Root + Chunked Sub-spans

### Snapshot run lifecycle (snapshot_runner_handler.go)

```go
func (r *SnapshotRunner) Handle(msg *nats.Msg) {
    parentCtx := observability.NATSExtract(context.Background(), msg)
    entryCtx, entrySpan := observability.EntrySpan(parentCtx, "nats.cdc.cmd.snapshot.v2",
        attribute.String("nats.subject", msg.Subject),
    )
    defer entrySpan.End()

    // parse payload, validate
    var p snapshotV2Payload
    if err := json.Unmarshal(msg.Data, &p); err != nil { ... return }

    // ... existing dedup / claim logic ...

    // Spawn snapshot goroutine, propagate entryCtx
    go func(ctx context.Context, p snapshotV2Payload, jobID string) {
        runCtx, runSpan := observability.ChildSpan(ctx, "snapshot.v2.run",
            attribute.Int64("snapshot.source_object_id", p.SourceObjectID),
            attribute.String("snapshot.trace_id", p.TraceID),
            attribute.String("snapshot.job_id", jobID),
        )
        var runErr error
        defer observability.EndSpan(runSpan, &runErr)

        runErr = r.runSnapshot(runCtx, p, jobID)
    }(entryCtx, p, jobID)
}
```

### Chunked sub-spans inside `runSnapshot`

```go
func (r *SnapshotRunner) runSnapshot(ctx context.Context, p snapshotV2Payload, jobID string) error {
    // ... claim progress ...
    progressID := claim.OutID
    span := oteltrace.SpanFromContext(ctx)
    span.SetAttributes(attribute.Int64("snapshot.progress_id", progressID))

    batchCounter := 0
    chunkSpan := startSnapshotChunk(ctx, progressID, batchCounter)
    defer func() { chunkSpan.End() }()

    for {
        // every CHUNK_SIZE batches, rotate chunk span
        if batchCounter > 0 && batchCounter%snapshotChunkSize == 0 {
            chunkSpan.End()
            chunkSpan = startSnapshotChunk(ctx, progressID, batchCounter)
        }

        chunkCtx := oteltrace.ContextWithSpan(ctx, chunkSpan)
        batchCtx, batchSpan := observability.ChildSpan(chunkCtx, "snapshot.v2.batch",
            attribute.Int("snapshot.batch_index", batchCounter),
            attribute.Int("snapshot.batch_size", batchSize),
        )

        // cursor.Find ...
        for _, doc := range batch {
            envelope := buildSnapshotEnvelope(doc, ...)
            // batchCtx được truyền xuống event_handler — span tree tự nối
            rows, err := r.eventHandler.HandleRaw(batchCtx, subject, envelope)
            ...
        }

        batchSpan.SetAttributes(attribute.Int("snapshot.batch_docs", len(batch)))
        batchSpan.End()

        // ... checkpoint, throttle ...
        batchCounter++

        if len(batch) < batchSize {
            break // exhausted
        }
    }
    return nil
}

func startSnapshotChunk(ctx context.Context, progressID int64, fromBatch int) oteltrace.Span {
    _, span := observability.ChildSpan(ctx, "snapshot.v2.chunk",
        attribute.Int64("snapshot.progress_id", progressID),
        attribute.Int("snapshot.chunk_from_batch", fromBatch),
    )
    return span
}

const snapshotChunkSize = 100  // 100 batches per chunk = 500k docs per trace
```

---

## M7 — BatchBuffer Span Link (Record-Level Tracking)

### Vấn đề
BatchBuffer.Add(record) nhận record từ N flows khác nhau (Kafka message 1, Kafka message 2, snapshot doc A, doc B, ...). Flush gom records lại → batchUpsert. Không có "parent context" duy nhất → phải dùng OTel Span Link.

### Solution
Thêm field `spanContext oteltrace.SpanContext` (24 byte = trace_id 16 + span_id 8 + flags) vào `model.UpsertRecord`. Tại `processEvent` capture span context:

```go
// event_handler.go processEvent
sc := oteltrace.SpanContextFromContext(ctx)
record := &model.UpsertRecord{
    ...
    OriginSpanContext: sc,
}
```

Tại `BatchBuffer.batchUpsert`:

```go
func (bb *BatchBuffer) batchUpsert(records []*model.UpsertRecord) (err error) {
    if len(records) == 0 {
        return nil
    }

    // Build Span Links from origin span contexts (dedup by trace_id+span_id).
    seen := make(map[string]struct{})
    links := make([]oteltrace.Link, 0, len(records))
    for _, r := range records {
        if !r.OriginSpanContext.IsValid() {
            continue
        }
        key := r.OriginSpanContext.TraceID().String() + r.OriginSpanContext.SpanID().String()
        if _, ok := seen[key]; ok {
            continue
        }
        seen[key] = struct{}{}
        links = append(links, oteltrace.Link{SpanContext: r.OriginSpanContext})
    }

    // Use BatchBuffer's lifetime ctx as parent (it has its own non-trace context),
    // but the Link list points back to the N origin flows.
    _, span := observability.StartSpanWithLinks(bb.ctx, "cdc.batch_upsert", links,
        attribute.Int("cdc.batch_size", len(records)),
        attribute.String("cdc.target_table", first.TableName),
        attribute.String("cdc.target_schema", schemaName),
        attribute.Int("cdc.batch_origin_count", len(links)),
    )
    defer observability.EndSpan(span, &err)
    ...
}
```

SigNoz UI: span `cdc.batch_upsert` xuất hiện như root độc lập (vì `bb.ctx` không có span), nhưng "Linked Spans" tab hiển thị N spans nguồn — user click vào để jump.

---

## M8 — Recon Flow Nested Tree

`recon_core.go` hiện chạy 3-tier. Cấu trúc span:

```
recon.cycle (root, per leader tick)
├── recon.tier.1 (page count diff)
│   ├── recon.table.<target_table_1>
│   ├── recon.table.<target_table_2>
│   └── ...
├── recon.tier.2 (sample-based)
│   └── recon.table.<target_table_X>
│       └── recon.window.<window_idx>
└── recon.tier.3 (full scan window)
    └── recon.table.<target_table_Y>
        └── recon.window.<window_idx>
```

Demo code:
```go
func (rc *ReconCore) runReconcileCycle(ctx context.Context) error {
    ctx, span := observability.BackgroundTick("recon_core")
    defer observability.EndSpan(span, nil)

    return rc.runAllTiers(ctx)
}

func (rc *ReconCore) runAllTiers(ctx context.Context) error {
    for tier := 1; tier <= 3; tier++ {
        tierCtx, tierSpan := observability.ChildSpan(ctx, fmt.Sprintf("recon.tier.%d", tier),
            attribute.Int("recon.tier", tier),
        )
        var tierErr error
        rc.runTier(tierCtx, tier)
        observability.EndSpan(tierSpan, &tierErr)
    }
    return nil
}
```

---

## M9 — HTTP Handler Instrumentation

### Fiber (worker /metrics, /health, /ready)
Health/ready không cần span — bỏ qua noise. `/api/v1/internal/stats` instrumentation tùy chọn.

### Gin (admin `/v2/sources/register`)

```go
func (s *Server) handleRegisterSource(c *gin.Context) {
    ctx, span := observability.EntrySpan(c.Request.Context(), "http.POST./v2/sources/register",
        attribute.String("http.method", "POST"),
        attribute.String("http.route", "/v2/sources/register"),
    )
    var err error
    defer observability.EndSpan(span, &err)

    // existing body
}
```

Note: nếu dùng otelgin middleware tự động → có thể skip M9. Kiểm tra `internal/admin/server.go` imports.

---

## M10 — Migration Script (Defer apply)

### File mới: `migrations/postgres/0046_add_otel_trace_id.up.sql`

```sql
-- Add otel_trace_id (W3C 128-bit hex) to activity-tracking tables.
-- Idempotent (safe to re-run).

ALTER TABLE cdc_system.cdc_activity_log
    ADD COLUMN IF NOT EXISTS otel_trace_id TEXT;

CREATE INDEX IF NOT EXISTS idx_cdc_activity_log_otel_trace_id
    ON cdc_system.cdc_activity_log(otel_trace_id)
    WHERE otel_trace_id IS NOT NULL;

ALTER TABLE cdc_system.snapshot_progress
    ADD COLUMN IF NOT EXISTS otel_trace_id TEXT;

CREATE INDEX IF NOT EXISTS idx_snapshot_progress_otel_trace_id
    ON cdc_system.snapshot_progress(otel_trace_id)
    WHERE otel_trace_id IS NOT NULL;

ALTER TABLE cdc_system.failed_sync_logs
    ADD COLUMN IF NOT EXISTS otel_trace_id TEXT;

ALTER TABLE cdc_system.cdc_reconciliation_report
    ADD COLUMN IF NOT EXISTS otel_trace_id TEXT;
```

### `down.sql`:
```sql
ALTER TABLE cdc_system.cdc_activity_log DROP COLUMN IF EXISTS otel_trace_id;
ALTER TABLE cdc_system.snapshot_progress DROP COLUMN IF EXISTS otel_trace_id;
ALTER TABLE cdc_system.failed_sync_logs DROP COLUMN IF EXISTS otel_trace_id;
ALTER TABLE cdc_system.cdc_reconciliation_report DROP COLUMN IF EXISTS otel_trace_id;
DROP INDEX IF EXISTS cdc_system.idx_cdc_activity_log_otel_trace_id;
DROP INDEX IF EXISTS cdc_system.idx_snapshot_progress_otel_trace_id;
```

**Muscle chỉ tạo file, KHÔNG apply** — chờ user lệnh riêng `migrate apply`.

Code Go (writeActivity, claimProgress, ...) inject `otel_trace_id := oteltrace.SpanFromContext(ctx).SpanContext().TraceID().String()` khi span valid.

---

## M11 — Tests

### Unit tests

File `pkgs/observability/propagation_test.go`:
- `TestNATSInjectExtract_Roundtrip` — inject vào msg.Header, extract lại, verify trace_id+span_id match.
- `TestNATSExtract_EmptyHeader_NoCrash` — empty msg → returns ctx unchanged.
- `TestKafkaInjectExtract_Roundtrip` — same.
- `TestKafkaInject_AppendsHeaderNotDuplicate` — gọi 2 lần → 1 entry, không duplicate.

File `pkgs/observability/flow_helpers_test.go`:
- `TestEntrySpan_AttachesParent` — pass ctx có span → EntrySpan tạo child.
- `TestEntrySpan_NoParent_CreatesRoot` — Background → root.
- `TestBackgroundTick_NameFormat` — verify span name = "bg.<job>.tick".
- `TestStartSpanWithLinks_LinksAttached` — verify span.Links() chứa N entries.
- `TestLinkFromContext_InvalidContext` — Background → returns Link với invalid SpanContext.

### Integration tests

File `internal/handler/snapshot_runner_handler_trace_test.go`:
- `TestSnapshot_TraceTree` — mock 5 docs, run snapshot, verify span tree:
  - 1 `snapshot.v2.run` root
  - 1 `snapshot.v2.chunk`
  - 1 `snapshot.v2.batch`
  - 5 `cdc.event_handle` children
  - 5 `cdc.schema_inspect` (children của event_handle)
  - All have same trace_id.

File `internal/handler/batch_buffer_trace_test.go`:
- `TestBatchUpsert_SpanLinks` — add 3 records từ 3 different ctx, flush, verify span.Links() có 3 entries.

---

## M12 — Verify + Report

### Verify commands

```bash
go build ./...          # EXIT=0
go vet ./...            # EXIT=0
go test ./...           # EXIT=0
go test -race ./pkgs/observability/...   # race detector
```

### Smoke test local
```bash
docker compose up -d postgres nats redis  # nếu có
go run ./cmd/worker --config config-local.yml &
# verify trong 30s:
#   - log "worker started"
#   - log "NATS subscribed: cdc.cmd.*" cho tất cả subjects
#   - log "Kafka consumer started"
#   - Không có panic / "failed to ..."
nats pub cdc.cmd.recon-check '{"target_table":"shadow_test","source_db":"test","tier":1}'
# verify log có trace_id field
```

### Baseline diff
```bash
grep -rn "context.Background()" internal/ pkgs/ | wc -l
# Trước: 66. Sau target: ≤ 8.
```

### Report
`report_all_flows_trace_aggregation_2026-05-26.md` đầy đủ 13 section (như phase 2 template):
1. Mục tiêu
2. Scope thực hiện
3. Files thay đổi (đầy đủ liệt kê + LOC delta)
4. Build & test logs
5. ADR áp dụng
6. Log call sites migrated
7. Business attrs đính kèm
8. Code Demo Verify
9. Risk & Mitigation
10. Gap & Out-of-scope
11. DoD checklist
12. Lesson (sẽ append)
13. Next verbs
14. Pre-flight Checklist

---

## Risk Matrix

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Refactor làm break NATS handler (typo trong span name) | Low | High | Unit test build per-package incrementally |
| Span explosion (snapshot 6M docs * 5 spans/doc = 30M span) | Medium | High | Chunked traces (M6) + sampling at root | 
| BatchBuffer record memory tăng thêm 32 byte/record | Low | Low | Acceptable trên 100k records = +3MB |
| OTel batch processor drop span khi exporter slow | Medium | Medium | TracerProvider đã có batch timeout 5s + queue |
| Schema migration apply nhầm production | High | Critical | M10 chỉ tạo file, KHÔNG apply — user lệnh riêng |
| Backward break: span name conflict | Low | Medium | KHÔNG đổi span name cũ — chỉ ADD (N1) |
| Race condition khi multiple goroutine inject vào shared msg | Low | Low | NATS msg per-publish là new instance |
| Sampling drop root span của background tick | Medium | Medium | sampleRatio config phải ≥ ratio mong muốn cho bg |

## Pre-execution Checklist (Brain → Muscle handoff)

- ✅ Workspace tạo đầy đủ doc set.
- ✅ 02_plan có code demo chi tiết.
- ✅ 09_tasks_solution có ref snippet per task.
- ✅ 04_decisions có ADR.
- ⏳ Chờ user verb `thực hiện` / `execute` / `subset M1+M2` để Muscle bắt đầu.
- ⏳ User confirm có apply migration M10 hay defer riêng.

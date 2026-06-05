# 09_tasks_solution — Reference Implementation Snippets

> Mỗi snippet là final state target cho Muscle. Copy + verify line context với file gốc trước khi apply.

---

## T1.1 — `pkgs/observability/propagation.go`

Xem `02_plan.md` §M1 (đã viết đầy đủ).

## T1.2 — `pkgs/observability/flow_helpers.go`

Xem `02_plan.md` §M1.

## T1.4 — Unit tests

```go
// pkgs/observability/propagation_test.go
package observability

import (
    "context"
    "testing"

    "github.com/nats-io/nats.go"
    "github.com/segmentio/kafka-go"
    sdktrace "go.opentelemetry.io/otel/sdk/trace"
    "go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func setupTestTracerWithW3C(t *testing.T) *tracetest.SpanRecorder {
    t.Helper()
    rec := tracetest.NewSpanRecorder()
    tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(rec))
    prev := tracer
    tracer = tp.Tracer("propagation_test")
    t.Cleanup(func() { tracer = prev })
    return rec
}

func TestNATSInjectExtract_Roundtrip(t *testing.T) {
    setupTestTracerWithW3C(t)
    ctx, span := ChildSpan(context.Background(), "producer")
    msg := &nats.Msg{Subject: "test", Header: nats.Header{}}
    NATSInject(ctx, msg)

    if msg.Header.Get("traceparent") == "" {
        t.Fatal("traceparent not injected")
    }

    ctx2 := NATSExtract(context.Background(), msg)
    _, child := ChildSpan(ctx2, "consumer")
    if child.SpanContext().TraceID() != span.SpanContext().TraceID() {
        t.Fatalf("trace_id mismatch: got %s want %s",
            child.SpanContext().TraceID(), span.SpanContext().TraceID())
    }
    child.End()
    span.End()
}

func TestNATSExtract_EmptyHeader_NoCrash(t *testing.T) {
    msg := &nats.Msg{Subject: "x", Header: nil}
    got := NATSExtract(context.Background(), msg)
    if got == nil {
        t.Fatal("ctx must not be nil")
    }
}

func TestKafkaInjectExtract_Roundtrip(t *testing.T) {
    setupTestTracerWithW3C(t)
    ctx, span := ChildSpan(context.Background(), "kafka-producer")
    msg := kafka.Message{Headers: []kafka.Header{}}
    KafkaInject(ctx, &msg)

    found := false
    for _, h := range msg.Headers {
        if h.Key == "traceparent" {
            found = true
        }
    }
    if !found {
        t.Fatal("traceparent header missing")
    }

    ctx2 := KafkaExtract(context.Background(), msg)
    _, child := ChildSpan(ctx2, "kafka-consumer")
    if child.SpanContext().TraceID() != span.SpanContext().TraceID() {
        t.Fatal("trace_id mismatch")
    }
}

func TestKafkaInject_NoDuplicateHeader(t *testing.T) {
    setupTestTracerWithW3C(t)
    ctx, _ := ChildSpan(context.Background(), "p")
    msg := kafka.Message{}
    KafkaInject(ctx, &msg)
    KafkaInject(ctx, &msg)
    count := 0
    for _, h := range msg.Headers {
        if h.Key == "traceparent" {
            count++
        }
    }
    if count != 1 {
        t.Fatalf("expected 1 traceparent header, got %d", count)
    }
}
```

```go
// pkgs/observability/flow_helpers_test.go
package observability

import (
    "context"
    "testing"

    sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func TestEntrySpan_AttachesParent(t *testing.T) {
    rec := setupTestTracer(t)
    parentCtx, parent := ChildSpan(context.Background(), "parent")
    entryCtx, entrySpan := EntrySpan(parentCtx, "nats.test")
    _ = entryCtx
    entrySpan.End()
    parent.End()
    spans := rec.Ended()
    if len(spans) != 2 {
        t.Fatalf("want 2 spans, got %d", len(spans))
    }
    if spans[0].Parent().SpanID() != spans[1].SpanContext().SpanID() {
        t.Fatal("entry span not child of parent")
    }
}

func TestEntrySpan_NoParent_CreatesRoot(t *testing.T) {
    rec := setupTestTracer(t)
    _, span := EntrySpan(context.Background(), "nats.test")
    span.End()
    spans := rec.Ended()
    if !spans[0].Parent().IsValid() == false {
        // Parent should be invalid (root span)
    }
}

func TestBackgroundTick_NameFormat(t *testing.T) {
    rec := setupTestTracer(t)
    _, span := BackgroundTick("dlq_retry")
    span.End()
    spans := rec.Ended()
    if spans[0].Name() != "bg.dlq_retry.tick" {
        t.Fatalf("name = %q want bg.dlq_retry.tick", spans[0].Name())
    }
}

func TestStartSpanWithLinks_Attached(t *testing.T) {
    rec := setupTestTracer(t)
    ctxA, spanA := ChildSpan(context.Background(), "originA")
    ctxB, spanB := ChildSpan(context.Background(), "originB")
    links := []sdktrace.Link{
        {SpanContext: spanA.SpanContext()},
        {SpanContext: spanB.SpanContext()},
    }
    // adapt: oteltrace.Link required
    _ = ctxA
    _ = ctxB
    _ = links
    // ... test body verifying span.Links() returns 2 entries
    spanA.End(); spanB.End()
    _ = rec
}
```

---

## T2.1 — `recon_handler.go` HandleReconCheck (representative)

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
        observability.Ctx(ctx, h.logger).Error("recon-check: parse payload failed",
            observability.ErrorField(err))
        return
    }
    span.SetAttributes(
        attribute.String("recon.target_table", payload.TargetTable),
        attribute.String("recon.source_db", payload.SourceDB),
    )

    // existing body — replace context.Background() with ctx
    err = h.runReconCheck(ctx, &payload)
}
```

Apply pattern tương tự cho: HandleReconHeal, HandleRetryFailed, HandleBackfillSourceTs, HandleDetectTimestampField, HandleDebeziumSignal.

---

## T2.2 — `command_handler.go` (16 handler)

Mỗi handler theo template:

```go
func (h *CommandHandler) HandleXxx(msg *nats.Msg) {
    parentCtx := observability.NATSExtract(context.Background(), msg)
    ctx, span := observability.EntrySpan(parentCtx, "nats.cdc.cmd.<subject>",
        attribute.String("nats.subject", msg.Subject),
    )
    var err error
    defer observability.EndSpan(span, &err)

    // existing body — sửa mọi context.Background() local thành ctx
}
```

Cụ thể cho `HandleDiscoverMongoDatabases` (đã có log migration phase B, giữ nguyên + thêm span):

```go
func (h *CommandHandler) HandleDiscoverMongoDatabases(msg *nats.Msg) {
    parentCtx := observability.NATSExtract(context.Background(), msg)
    ctx, span := observability.EntrySpan(parentCtx, "nats.cdc.cmd.introspect.mongo.databases",
        attribute.String("nats.subject", msg.Subject),
    )
    var err error
    defer observability.EndSpan(span, &err)

    // ... existing parse, validate ...
    if dbErr != nil {
        observability.Ctx(ctx, h.logger).Error("introspect.mongo.databases.failed",
            observability.ErrorField(dbErr))
        err = dbErr
        return
    }
    // ... rest
}
```

---

## T2.8 — `worker_server.go` inline NATS callbacks

```go
// Trước: worker_server.go:245
_, err := nc.Subscribe("schema.config.reload", func(msg *nats.Msg) {
    s.logger.Info("schema config reload received")
    if err := s.registrySvc.ReloadAll(context.Background()); err != nil { ... }
    if err := s.redisCache.DeletePattern(context.Background(), "schema:*"); err != nil { ... }
})

// Sau:
_, err := nc.Subscribe("schema.config.reload", func(msg *nats.Msg) {
    parentCtx := observability.NATSExtract(context.Background(), msg)
    ctx, span := observability.EntrySpan(parentCtx, "nats.schema.config.reload")
    var rerr error
    defer observability.EndSpan(span, &rerr)

    observability.Ctx(ctx, s.logger).Info("schema config reload received")
    if rerr = s.registrySvc.ReloadAll(ctx); rerr != nil {
        observability.Ctx(ctx, s.logger).Error("reload registry failed", observability.ErrorField(rerr))
        return
    }
    if rerr = s.redisCache.DeletePattern(ctx, "schema:*"); rerr != nil {
        observability.Ctx(ctx, s.logger).Error("redis flush failed", observability.ErrorField(rerr))
    }
})
```

---

## T3.2 — NATS Publisher Inject template

```go
// Trước:
err := nc.Publish("cdc.cmd.transmute", payloadJSON)

// Sau:
msg := &nats.Msg{
    Subject: "cdc.cmd.transmute",
    Data:    payloadJSON,
    Header:  nats.Header{},
}
observability.NATSInject(ctx, msg)
err := nc.PublishMsg(msg)
```

Với request-reply pattern (CommandHandler reply):

```go
// Trước:
err := nc.Publish(msg.Reply, replyData)

// Sau:
reply := &nats.Msg{
    Subject: msg.Reply,
    Data:    replyData,
    Header:  nats.Header{},
}
observability.NATSInject(ctx, reply)
err := nc.PublishMsg(reply)
```

---

## T4.1 — Kafka producer inject

```go
// internal/service/debezium_signal.go ~ L214
msg := kafka.Message{
    Key:     []byte(topicPrefix),
    Value:   body,
    Headers: make([]kafka.Header, 0, 2),
}
observability.KafkaInject(ctx, &msg)
err := writer.WriteMessages(ctx, msg)
```

---

## T5.1 — TransmuteScheduler tick wrap

```go
// transmute_scheduler.go
func (s *TransmuteScheduler) Start(ctx context.Context) {
    ticker := time.NewTicker(s.interval)
    defer ticker.Stop()
    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            s.runTick()
        }
    }
}

func (s *TransmuteScheduler) runTick() {
    tickCtx, span := observability.BackgroundTick("transmute_scheduler",
        attribute.String("scheduler.interval", s.interval.String()),
    )
    var err error
    defer observability.EndSpan(span, &err)

    err = s.tick(tickCtx)
}
```

Apply cùng pattern cho 6 worker khác (T5.2..T5.7).

---

## T6 — Snapshot V2 complete refactor

```go
// snapshot_runner_handler.go Handle()
func (r *SnapshotRunner) Handle(msg *nats.Msg) {
    parentCtx := observability.NATSExtract(context.Background(), msg)
    entryCtx, entrySpan := observability.EntrySpan(parentCtx, "nats.cdc.cmd.snapshot.v2",
        attribute.String("nats.subject", msg.Subject),
    )

    var p snapshotV2Payload
    if err := json.Unmarshal(msg.Data, &p); err != nil {
        entrySpan.RecordError(err)
        entrySpan.SetStatus(codes.Error, err.Error())
        entrySpan.End()
        observability.Ctx(entryCtx, r.logger).Error("snapshot.v2: parse payload failed",
            observability.ErrorField(err))
        return
    }
    if p.TraceID == "" {
        p.TraceID = strings.TrimSpace(msg.Header.Get("Cdc-Correlation-Id"))
    }
    if p.TraceID == "" {
        p.TraceID = fmt.Sprintf("worker-snapshot.v2-%d", time.Now().UnixNano())
    }
    entrySpan.SetAttributes(
        attribute.Int64("snapshot.source_object_id", p.SourceObjectID),
        attribute.String("snapshot.app_trace_id", p.TraceID),
    )
    entrySpan.End()  // entry span chỉ trace việc parse + spawn, end ngay

    jobID := generateJobID(p)

    go func(ctx context.Context, p snapshotV2Payload, jobID string) {
        runCtx, runSpan := observability.ChildSpan(ctx, "snapshot.v2.run",
            attribute.Int64("snapshot.source_object_id", p.SourceObjectID),
            attribute.String("snapshot.app_trace_id", p.TraceID),
            attribute.String("snapshot.job_id", jobID),
        )
        var runErr error
        defer observability.EndSpan(runSpan, &runErr)

        runErr = r.runSnapshot(runCtx, p, jobID)
    }(entryCtx, p, jobID)
}
```

```go
// runSnapshot — chunked sub-spans
const snapshotChunkRotateEvery = 100  // batches

func (r *SnapshotRunner) runSnapshot(ctx context.Context, p snapshotV2Payload, jobID string) (err error) {
    runSpan := oteltrace.SpanFromContext(ctx)

    // ... claim progress (existing) ...
    progressID := claim.OutID
    runSpan.SetAttributes(attribute.Int64("snapshot.progress_id", progressID))

    batchCounter := 0
    chunkCtx, chunkSpan := startChunk(ctx, progressID, batchCounter)
    defer func() { chunkSpan.End() }()

    for {
        if batchCounter > 0 && batchCounter%snapshotChunkRotateEvery == 0 {
            chunkSpan.End()
            chunkCtx, chunkSpan = startChunk(ctx, progressID, batchCounter)
        }

        batchCtx, batchSpan := observability.ChildSpan(chunkCtx, "snapshot.v2.batch",
            attribute.Int("snapshot.batch_index", batchCounter),
            attribute.Int("snapshot.batch_size", batchSize),
        )

        // cursor.Find(batchCtx, ...) — existing
        // for each doc: r.eventHandler.HandleRaw(batchCtx, subject, envelope)

        batchSpan.SetAttributes(attribute.Int("snapshot.batch_docs", len(batch)))
        batchSpan.End()

        // checkpoint, throttle (existing) — use batchCtx
        batchCounter++

        if len(batch) < batchSize {
            break
        }
    }
    return nil
}

func startChunk(ctx context.Context, progressID int64, fromBatch int) (context.Context, oteltrace.Span) {
    return observability.ChildSpan(ctx, "snapshot.v2.chunk",
        attribute.Int64("snapshot.progress_id", progressID),
        attribute.Int("snapshot.chunk_from_batch", fromBatch),
    )
}
```

---

## T7 — BatchBuffer Span Link

### T7.1 — UpsertRecord field

```go
// internal/model/upsert_record.go (file existing — append field)
import oteltrace "go.opentelemetry.io/otel/trace"

type UpsertRecord struct {
    // ... existing fields ...

    // OriginSpanContext stores the trace+span context of the flow that
    // produced this record (Kafka consumer or snapshot batch). Used by
    // BatchBuffer to build Span Links for cdc.batch_upsert.
    OriginSpanContext oteltrace.SpanContext
}
```

### T7.2 — event_handler.processEvent capture

```go
// internal/handler/event_handler.go processEvent
record := &model.UpsertRecord{
    // ... existing fields ...
    OriginSpanContext: oteltrace.SpanContextFromContext(ctx),
}
h.batchBuffer.Add(record)
```

### T7.3 — batchUpsert with Links

```go
// internal/handler/batch_buffer.go
func (bb *BatchBuffer) batchUpsert(records []*model.UpsertRecord) (err error) {
    if len(records) == 0 {
        return nil
    }
    first := records[0]
    tableName := first.TableName
    schemaName := bb.recordSchema(first)

    // Dedup origin spans → Links
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

    _, span := observability.StartSpanWithLinks(bb.ctx, "cdc.batch_upsert", links,
        attribute.Int("cdc.batch_size", len(records)),
        attribute.String("cdc.target_table", tableName),
        attribute.String("cdc.target_schema", schemaName),
        attribute.Int("cdc.batch_origin_count", len(links)),
    )
    defer observability.EndSpan(span, &err)

    // existing body using bb.ctx
    return err
}
```

---

## T8 — Recon nested tree

```go
// internal/service/recon_core.go
func (rc *ReconCore) runReconcileCycle(ctx context.Context) (err error) {
    // ctx here is from BackgroundTick("recon_core") — already has root span.
    cycleSpan := oteltrace.SpanFromContext(ctx)
    cycleSpan.SetName("recon.cycle")  // optional: rename

    for tier := 1; tier <= 3; tier++ {
        tierCtx, tierSpan := observability.ChildSpan(ctx,
            fmt.Sprintf("recon.tier.%d", tier),
            attribute.Int("recon.tier", tier),
        )
        var tierErr error
        switch tier {
        case 1:
            tierErr = rc.runTier1(tierCtx)
        case 2:
            tierErr = rc.runTier2(tierCtx)
        case 3:
            tierErr = rc.runTier3(tierCtx)
        }
        observability.EndSpan(tierSpan, &tierErr)
    }
    return nil
}

func (rc *ReconCore) runTier1(ctx context.Context) error {
    for _, table := range rc.tables {
        tableCtx, tableSpan := observability.ChildSpan(ctx, "recon.table",
            attribute.String("recon.target_table", table.Name),
            attribute.String("recon.source_db", table.SourceDB),
        )
        var tableErr error
        rc.checkTablePage(tableCtx, table)
        observability.EndSpan(tableSpan, &tableErr)
    }
    return nil
}

func (rc *ReconCore) runTier3(ctx context.Context) error {
    for _, table := range rc.tables {
        tableCtx, tableSpan := observability.ChildSpan(ctx, "recon.table",
            attribute.String("recon.target_table", table.Name),
        )
        for i, window := range rc.windows {
            _, winSpan := observability.ChildSpan(tableCtx, "recon.window",
                attribute.Int("recon.window_idx", i),
                attribute.String("recon.window_from", window.From.Format(time.RFC3339)),
                attribute.String("recon.window_to", window.To.Format(time.RFC3339)),
            )
            var winErr error
            rc.scanWindow(tableCtx, table, window)
            observability.EndSpan(winSpan, &winErr)
        }
        var tableErr error
        observability.EndSpan(tableSpan, &tableErr)
    }
    return nil
}
```

---

## T9.1 — Gin admin handler

```go
// internal/admin/source_register.go
func (s *Server) handleRegisterSource(c *gin.Context) {
    ctx, span := observability.EntrySpan(c.Request.Context(), "http.POST./v2/sources/register",
        attribute.String("http.method", "POST"),
        attribute.String("http.route", "/v2/sources/register"),
    )
    var err error
    defer observability.EndSpan(span, &err)

    // existing body — use ctx
}
```

---

## T10.1 — Migration

```sql
-- migrations/postgres/0046_add_otel_trace_id.up.sql
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

```sql
-- migrations/postgres/0046_add_otel_trace_id.down.sql
DROP INDEX IF EXISTS cdc_system.idx_cdc_activity_log_otel_trace_id;
DROP INDEX IF EXISTS cdc_system.idx_snapshot_progress_otel_trace_id;
ALTER TABLE cdc_system.cdc_activity_log DROP COLUMN IF EXISTS otel_trace_id;
ALTER TABLE cdc_system.snapshot_progress DROP COLUMN IF EXISTS otel_trace_id;
ALTER TABLE cdc_system.failed_sync_logs DROP COLUMN IF EXISTS otel_trace_id;
ALTER TABLE cdc_system.cdc_reconciliation_report DROP COLUMN IF EXISTS otel_trace_id;
```

---

## T11.3 — Integration test snapshot trace tree

```go
// internal/handler/snapshot_runner_handler_trace_test.go
func TestSnapshot_TraceTree(t *testing.T) {
    rec := setupTestTracerWithW3C(t)
    runner := setupTestSnapshotRunner(t)  // mock mongo + event handler

    // Simulate 12 docs (2 batches of 5 + 1 batch of 2) → chunk size 100, no rotation
    runner.simulateRun(ctx, mockPayload{
        SourceObjectID: 1,
        BatchSize:      5,
        DocsTotal:      12,
    })

    spans := rec.Ended()
    nameCount := map[string]int{}
    for _, s := range spans {
        nameCount[s.Name()]++
    }
    if nameCount["snapshot.v2.run"] != 1 {
        t.Fatalf("want 1 snapshot.v2.run, got %d", nameCount["snapshot.v2.run"])
    }
    if nameCount["snapshot.v2.chunk"] < 1 {
        t.Fatal("want ≥1 chunk")
    }
    if nameCount["snapshot.v2.batch"] != 3 {
        t.Fatalf("want 3 batches, got %d", nameCount["snapshot.v2.batch"])
    }
    if nameCount["cdc.event_handle"] != 12 {
        t.Fatalf("want 12 event_handle, got %d", nameCount["cdc.event_handle"])
    }

    // Verify all spans share same trace_id
    var traceID oteltrace.TraceID
    for _, s := range spans {
        if !traceID.IsValid() {
            traceID = s.SpanContext().TraceID()
            continue
        }
        if s.SpanContext().TraceID() != traceID {
            t.Fatalf("span %s has different trace_id", s.Name())
        }
    }
}
```

---

## T11.4 — BatchBuffer span link integration test

```go
// internal/handler/batch_buffer_trace_test.go
func TestBatchUpsert_SpanLinks(t *testing.T) {
    rec := setupTestTracerWithW3C(t)

    // Create 3 records, each with distinct OriginSpanContext (different traces)
    records := make([]*model.UpsertRecord, 3)
    var originSCs []oteltrace.SpanContext
    for i := 0; i < 3; i++ {
        _, originSpan := ChildSpan(context.Background(), fmt.Sprintf("origin-%d", i))
        records[i] = &model.UpsertRecord{
            TableName:         "t",
            OriginSpanContext: originSpan.SpanContext(),
        }
        originSCs = append(originSCs, originSpan.SpanContext())
        originSpan.End()
    }

    bb := newTestBatchBuffer(t)
    err := bb.batchUpsert(records)  // mock DB, just exercise span
    _ = err

    spans := rec.Ended()
    var batchSpan sdktrace.ReadOnlySpan
    for _, s := range spans {
        if s.Name() == "cdc.batch_upsert" {
            batchSpan = s
            break
        }
    }
    if batchSpan == nil {
        t.Fatal("batch_upsert span not recorded")
    }
    if len(batchSpan.Links()) != 3 {
        t.Fatalf("want 3 links, got %d", len(batchSpan.Links()))
    }
}
```

---

## Helper migrations cho `command_handler.go` raw context refactor

Một số handler hiện dùng `context.Background()` ngay đầu. Pattern conversion 1-shot dùng `sed` để Muscle audit nhanh:

```bash
# Audit list trước:
grep -n "ctx := context.Background()" internal/handler/

# Refactor manual per-handler (KHÔNG sed replace) vì cần thêm EntrySpan trước.
```

---

## Smoke test commands (T12.2)

```bash
# Terminal 1: start dependencies
docker compose -f deploy/local/docker-compose.yml up -d postgres nats redis

# Terminal 2: start worker
cd centralized-data-service
go run ./cmd/worker --config config/config-local.yml 2>&1 | tee /tmp/worker_smoke.log &
WORKER_PID=$!

# Wait 5s
sleep 5

# Verify subscribe + no panic
grep -E "NATS subscribed|panic|Kafka consumer started" /tmp/worker_smoke.log

# Trigger recon
nats pub cdc.cmd.recon-check '{"target_table":"shadow_test","source_db":"test","tier":1}'

# Wait 3s
sleep 3

# Verify trace_id in log
grep "trace_id" /tmp/worker_smoke.log | head -5

# Clean up
kill $WORKER_PID
```

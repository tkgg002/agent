# 09_tasks_solution — Reference Implementation Snippets

> Mỗi snippet là final state target. Muscle chỉ cần copy + verify line context.

---

## T1.1 — `pkgs/observability/trace_helpers.go`
Xem `02_plan.md` §M1.

## T2.1-T2.3 — kafka_consumer.go line 382-420

```go
// kafka_consumer.go line ~380
parentCtx := otel.GetTextMapPropagator().Extract(ctx, carrier)

engine, sourceDB, _, _ := observability.ParseDebeziumTopic(msg.Topic)
spanCtx, span := observability.StartSpan(parentCtx, "kafka.consume",
    attribute.String("messaging.system", "kafka"),
    attribute.String("messaging.destination", msg.Topic),
    attribute.String("messaging.operation", "receive"),
    attribute.Int("messaging.kafka.partition", msg.Partition),
    attribute.Int64("messaging.kafka.offset", msg.Offset),
    attribute.Int64("messaging.kafka.message.timestamp_ms", msg.Time.UnixMilli()),
    observability.SourceTableAttr(msg.Topic),
    attribute.String("cdc.engine", engine),
    attribute.String("cdc.source_db", sourceDB),
)

if !msg.Time.IsZero() {
    e2eLatency := time.Since(msg.Time)
    metrics.E2ELatency.Observe(e2eLatency.Seconds())
    span.SetAttributes(attribute.Float64("e2e_latency_seconds", e2eLatency.Seconds()))
}

batch := kc.getOrCreateBatch(msg.Topic)
rows, procErr := kc.processMessage(spanCtx, msg)
if procErr != nil {
    observability.Ctx(spanCtx, kc.logger).Error("kafka message processing failed",
        observability.ErrorField(procErr),
        observability.Attrs(
            zap.String("topic", msg.Topic),
            zap.Int("partition", msg.Partition),
            zap.Int64("offset", msg.Offset),
        ),
    )
    span.RecordError(procErr)
    span.SetStatus(codes.Error, procErr.Error())
    metrics.EventsProcessed.WithLabelValues("error", "", msg.Topic, "error").Inc()
    batch.failed++
} else {
    duration := time.Since(start)
    metrics.EventsProcessed.WithLabelValues("kafka", "", msg.Topic, "success").Inc()
    metrics.ProcessingDuration.WithLabelValues("kafka", "", msg.Topic).Observe(duration.Seconds())
    span.SetAttributes(
        attribute.Float64("duration_seconds", duration.Seconds()),
        attribute.Int("cdc.rows_affected", rows),
    )
    batch.success++
    batch.rowsAffected += rows
}
batch.processed++
span.End()
```

Imports cần thêm:
```go
"go.opentelemetry.io/otel/codes"
```

## T2.4 — processMessage child span

```go
// kafka_consumer.go line 460
func (kc *KafkaConsumer) processMessage(ctx context.Context, msg kafka.Message) (rows int, err error) {
    ctx, span := observability.ChildSpan(ctx, "cdc.process_message",
        attribute.Int("cdc.value_size_bytes", len(msg.Value)),
    )
    defer observability.EndSpan(span, &err)
    
    // === existing body unchanged ===
    // Khi parse được op từ Debezium envelope, set thêm:
    //   span.SetAttributes(attribute.String("cdc.operation", op))
    
    return rows, err
}
```

## T2.5 — Log migration line 441, 453

```go
// kafka_consumer.go line 441
if dlqErr := kc.writeDLQ(ctx, msg, procErr); dlqErr != nil {
    service.DLQWriteFail.Inc()
    observability.Ctx(ctx, kc.logger).Error("kafka DLQ write failed — skipping offset commit for redelivery",
        observability.ErrorField(dlqErr),
        observability.Attrs(
            zap.String("topic", msg.Topic),
            zap.Int("partition", msg.Partition),
            zap.Int64("offset", msg.Offset),
        ),
    )
    continue
}

// line 453
if err := currentReader.CommitMessages(ctx, msg); err != nil {
    observability.Ctx(ctx, kc.logger).Error("kafka commit failed", observability.ErrorField(err))
}
```

## T3.1 — event_handler.HandleRaw

```go
// event_handler.go line 64
func (h *EventHandler) HandleRaw(ctx context.Context, subject string, data []byte) (rows int, err error) {
    ctx, span := observability.ChildSpan(ctx, "cdc.event_handle",
        attribute.String("cdc.subject", subject),
        attribute.Int("cdc.data_size_bytes", len(data)),
    )
    defer func() {
        span.SetAttributes(attribute.Int("cdc.rows", rows))
        observability.EndSpan(span, &err)
    }()
    
    // === existing body unchanged, dùng ctx thay vì có chỗ nhận _ ===
    return rows, err
}
```

## T4.1-T4.3 — batchUpsert

```go
// batch_buffer.go line 189
func (bb *BatchBuffer) batchUpsert(ctx context.Context, records []*model.UpsertRecord) (err error) {
    var targetTable, targetSchema string
    if len(records) > 0 {
        targetTable = records[0].TargetTable
        targetSchema = records[0].TargetSchema
    }
    ctx, span := observability.ChildSpan(ctx, "cdc.batch_upsert",
        attribute.Int("cdc.batch_size", len(records)),
        attribute.String("cdc.target_table", targetTable),
        attribute.String("cdc.target_schema", targetSchema),
    )
    defer observability.EndSpan(span, &err)
    
    // === existing body ===
    return err
}
```

Caller `flushBatch` cần truyền ctx — phase này tạo `context.Background()` cho batch (vì là async, không thuộc message ctx — xem ADR-02).

## T5.1 — schema_inspector.InspectEvent

```go
// schema_inspector.go line 87
func (si *SchemaInspector) InspectEvent(ctx context.Context, tableName, sourceDB string, eventData map[string]interface{}) (drift *SchemaDrift, err error) {
    ctx, span := observability.ChildSpan(ctx, "cdc.schema_inspect",
        attribute.String("cdc.table", tableName),
        attribute.String("cdc.source_db", sourceDB),
    )
    defer func() {
        if drift != nil {
            span.SetAttributes(attribute.Int("cdc.new_field_count", len(drift.NewFields)))
        }
        observability.EndSpan(span, &err)
    }()
    
    // === existing body ===
    return drift, err
}
```

## T7 — Unit test
Xem `02_plan.md` §M7.

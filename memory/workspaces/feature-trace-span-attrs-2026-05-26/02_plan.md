# 02_plan — Trace Phase 2 (Code Demo Chi Tiết)

## Roadmap
| Milestone | Mô tả | Effort | File touched |
|-----------|-------|-------:|--------------|
| M0 | Workspace + audit (đã làm) | 15m | (this dir) |
| M1 | Thêm helper `RecordSpanError` + topic parser trong observability | 20m | `pkgs/observability/trace_helpers.go` (NEW) |
| M2 | kafka.consume: bổ sung business attrs + child `cdc.process_message` | 30m | `internal/handler/kafka_consumer.go` |
| M3 | Child span `cdc.event_handle` | 20m | `internal/handler/event_handler.go` |
| M4 | Child span `cdc.batch_upsert` + ctx propagation từ flushBatch | 30m | `internal/handler/batch_buffer.go`, `kafka_consumer.go` (flushBatch signature) |
| M5 | Child span `cdc.schema_inspect` | 15m | `internal/service/schema_inspector.go` |
| M6 | Migrate ~10 critical log call site sang `observability.Ctx(ctx, log)` | 25m | 4 file trên |
| M7 | Unit test span hierarchy + recordError | 30m | `pkgs/observability/trace_helpers_test.go` (NEW) |
| M8 | Build/vet/test verify + report file | 20m | report + 05_progress + lesson global |
**Total**: ~3h25m

---

## M1 Code Demo — Helper

```go
// pkgs/observability/trace_helpers.go (NEW)
package observability

import (
	"context"
	"strings"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	oteltrace "go.opentelemetry.io/otel/trace"
)

// RecordSpanError records err on the active span and sets status=Error.
// Calls span.End(); caller must not End again. When err is nil it does
// nothing — convenient for defer patterns:
//
//	ctx, span := observability.StartSpan(ctx, "cdc.batch_upsert", ...)
//	defer observability.EndSpan(span, &err)
func EndSpan(span oteltrace.Span, errPtr *error) {
	if span == nil {
		return
	}
	if errPtr != nil && *errPtr != nil {
		span.RecordError(*errPtr)
		span.SetStatus(codes.Error, (*errPtr).Error())
	}
	span.End()
}

// ParseDebeziumTopic splits a Debezium-style topic name into engine,
// database, schema, and table parts. Topic convention:
//
//	cdc.<engine>.<db>.<schema>.<table>  (e.g. cdc.gpay.public.orders)
//	cdc.<engine>.<db>.<collection>       (Mongo: cdc.mongo.ecommerce.users)
//
// Returns empty strings when the topic doesn't match. Safe for unknown
// formats — caller should fall back to raw topic in span attribute.
func ParseDebeziumTopic(topic string) (engine, db, schema, table string) {
	parts := strings.Split(topic, ".")
	if len(parts) < 4 || parts[0] != "cdc" {
		return "", "", "", ""
	}
	engine = parts[1]
	db = parts[2]
	if len(parts) >= 5 {
		schema = parts[3]
		table = strings.Join(parts[4:], ".")
	} else {
		table = parts[3]
	}
	return
}

// SourceTableAttr returns a stable attribute.KeyValue for cdc.source_table
// using the parsed topic, with a fallback to the raw topic so spans always
// carry SOMETHING queryable.
func SourceTableAttr(topic string) attribute.KeyValue {
	_, _, schema, table := ParseDebeziumTopic(topic)
	switch {
	case schema != "" && table != "":
		return attribute.String("cdc.source_table", schema+"."+table)
	case table != "":
		return attribute.String("cdc.source_table", table)
	default:
		return attribute.String("cdc.source_table", topic)
	}
}

// ChildSpan starts a child span from ctx. Convenience wrapper around
// Tracer().Start with consistent naming. Returns child ctx + span.
func ChildSpan(ctx context.Context, name string, attrs ...attribute.KeyValue) (context.Context, oteltrace.Span) {
	return Tracer().Start(ctx, name, oteltrace.WithAttributes(attrs...))
}
```

---

## M2 Code Demo — kafka_consumer.go

### Bổ sung business attrs ngay khi start span
```go
// kafka_consumer.go:382 — BEFORE
spanCtx, span := observability.StartSpan(parentCtx, "kafka.consume",
    attribute.String("messaging.system", "kafka"),
    attribute.String("messaging.destination", msg.Topic),
    attribute.String("messaging.operation", "receive"),
    attribute.Int("messaging.kafka.partition", msg.Partition),
    attribute.Int64("messaging.kafka.offset", msg.Offset),
    attribute.Int64("messaging.kafka.message.timestamp_ms", msg.Time.UnixMilli()),
)

// AFTER
engine, db, _, _ := observability.ParseDebeziumTopic(msg.Topic)
spanCtx, span := observability.StartSpan(parentCtx, "kafka.consume",
    attribute.String("messaging.system", "kafka"),
    attribute.String("messaging.destination", msg.Topic),
    attribute.String("messaging.operation", "receive"),
    attribute.Int("messaging.kafka.partition", msg.Partition),
    attribute.Int64("messaging.kafka.offset", msg.Offset),
    attribute.Int64("messaging.kafka.message.timestamp_ms", msg.Time.UnixMilli()),
    observability.SourceTableAttr(msg.Topic),
    attribute.String("cdc.engine", engine),
    attribute.String("cdc.source_db", db),
)
```

### Error path: span.RecordError
```go
// kafka_consumer.go:402 — BEFORE
rows, procErr := kc.processMessage(spanCtx, msg)
if procErr != nil {
    kc.logger.Error("kafka message processing failed",
        zap.String("topic", msg.Topic),
        zap.Int("partition", msg.Partition),
        zap.Int64("offset", msg.Offset),
        zap.Error(procErr),
    )
    span.SetAttributes(attribute.String("error", procErr.Error()))
    ...
}

// AFTER
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
    ...
} else {
    ...
    span.SetAttributes(
        attribute.Float64("duration_seconds", duration.Seconds()),
        attribute.Int("cdc.rows_affected", rows),
    )
    ...
}
```

### Child span trong processMessage
```go
// kafka_consumer.go:460 — BEFORE
func (kc *KafkaConsumer) processMessage(ctx context.Context, msg kafka.Message) (int, error) {
    // ... avro decode, transform, eventHandler.HandleRaw ...
}

// AFTER
func (kc *KafkaConsumer) processMessage(ctx context.Context, msg kafka.Message) (rows int, err error) {
    ctx, span := observability.ChildSpan(ctx, "cdc.process_message",
        attribute.Int("cdc.value_size_bytes", len(msg.Value)),
    )
    defer observability.EndSpan(span, &err)
    
    // ... existing logic, có thể span.SetAttributes thêm cdc.payload.kind sau khi parse Avro ...
    return rows, err
}
```

---

## M3 Code Demo — event_handler.go

```go
// event_handler.go:64 — BEFORE
func (h *EventHandler) HandleRaw(ctx context.Context, subject string, data []byte) (int, error) {
    // existing logic
}

// AFTER
func (h *EventHandler) HandleRaw(ctx context.Context, subject string, data []byte) (rows int, err error) {
    ctx, span := observability.ChildSpan(ctx, "cdc.event_handle",
        attribute.String("cdc.subject", subject),
        attribute.Int("cdc.data_size_bytes", len(data)),
    )
    defer func() {
        span.SetAttributes(attribute.Int("cdc.rows", rows))
        observability.EndSpan(span, &err)
    }()
    
    // existing logic — log call site dùng observability.Ctx(ctx, h.logger) thay vì h.logger raw
    return rows, err
}
```

---

## M4 Code Demo — batch_buffer.go + ctx propagation

### Vấn đề: `flushBatch(ctx)` dùng outer ctx (không phải spanCtx của message đầu)
Hiện tại:
```go
// kafka_consumer.go:430
if batch.processed >= flushAt {
    kc.flushBatch(ctx, msg.Topic)  // ctx ở đây là outer reader ctx, không có span
}
```

**Quyết định ADR-04**: chấp nhận batch upsert là root span MỚI (không nested dưới kafka.consume), vì:
- Batch upsert gom nhiều message → không thuộc về 1 message duy nhất.
- Nested theo "message đầu" sẽ misleading (trace của message 1 chứa cả 100 record batch upsert).
- Thay vào đó, batch span sẽ có attribute `cdc.batch_first_topic` + `cdc.batch_message_offsets` (range) để cross-reference.

### batchUpsert
```go
// batch_buffer.go:189 — BEFORE
func (bb *BatchBuffer) batchUpsert(records []*model.UpsertRecord) error {
    // existing logic
}

// AFTER (đổi signature nhận ctx)
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
    
    // existing logic + log site dùng observability.Ctx(ctx, bb.logger)
    return err
}
```

Caller: cập nhật `flushBatch` truyền ctx (hoặc tạo span root ở flushBatch).

---

## M5 Code Demo — schema_inspector.go

```go
// schema_inspector.go:87 — BEFORE
func (si *SchemaInspector) InspectEvent(ctx context.Context, tableName, sourceDB string, eventData map[string]interface{}) (*SchemaDrift, error) {
    // existing logic
}

// AFTER
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
    
    // existing logic
    return drift, err
}
```

---

## M6 Code Demo — Log migration critical sites

**Target list** (10 sites, không hơn — phase này demo, không mass migrate):

| File:Line | Function | BEFORE | AFTER |
|-----------|----------|--------|-------|
| `kafka_consumer.go:403` | processMessage error | `kc.logger.Error(...)` | `observability.Ctx(spanCtx, kc.logger).Error(... ErrorField(err) ...)` |
| `kafka_consumer.go:441` | DLQ write fail | `kc.logger.Error(...)` | `observability.Ctx(ctx, kc.logger).Error(...)` |
| `kafka_consumer.go:453` | commit fail | `kc.logger.Error(...)` | `observability.Ctx(ctx, kc.logger).Error(...)` |
| `event_handler.go:?` | parse fail | tbd | `observability.Ctx(ctx, h.logger)` |
| `event_handler.go:?` | apply fail | tbd | `observability.Ctx(ctx, h.logger)` |
| `batch_buffer.go:181` | batch upsert ok | `bb.logger.Info("batch upsert ok", ...)` | `observability.Ctx(ctx, bb.logger).Info(..., audit=true, Attrs(...))` |
| `batch_buffer.go:?` | batch upsert fail | tbd | `observability.Ctx(ctx, bb.logger).Error(... ErrorField(err) ...)` |
| `schema_inspector.go:162` | drift detected (ĐÃ migrate phase 1 — bổ sung Ctx) | `si.logger.Info(...)` | `observability.Ctx(ctx, si.logger).Info(...)` |
| `schema_inspector.go:?` | masking error | tbd | `observability.Ctx(ctx, si.logger).Error(...)` |
| `schema_inspector.go:?` | publish drift alert | tbd | `observability.Ctx(ctx, si.logger).Info(...)` |

Sẽ confirm chính xác line khi Muscle execute.

---

## M7 Code Demo — Unit test

```go
// pkgs/observability/trace_helpers_test.go (NEW)
package observability

import (
	"context"
	"errors"
	"testing"

	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func setupTestTracer(t *testing.T) *tracetest.SpanRecorder {
	rec := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(rec))
	otel.SetTracerProvider(tp)
	tracer = tp.Tracer("test")
	return rec
}

func TestParseDebeziumTopic(t *testing.T) {
	cases := []struct {
		topic                          string
		engine, db, schema, table      string
	}{
		{"cdc.gpay.public.orders", "gpay", "public", "", "orders"},
		{"cdc.gpay.shop_db.public.orders", "gpay", "shop_db", "public", "orders"},
		{"cdc.mongo.ecommerce.users", "mongo", "ecommerce", "", "users"},
		{"unknown.topic", "", "", "", ""},
	}
	for _, c := range cases {
		e, d, s, tb := ParseDebeziumTopic(c.topic)
		if e != c.engine || d != c.db || s != c.schema || tb != c.table {
			t.Errorf("topic=%s want (%s,%s,%s,%s) got (%s,%s,%s,%s)",
				c.topic, c.engine, c.db, c.schema, c.table, e, d, s, tb)
		}
	}
}

func TestChildSpan_ParentChild(t *testing.T) {
	rec := setupTestTracer(t)
	ctx, parent := ChildSpan(context.Background(), "parent")
	_, child := ChildSpan(ctx, "child")
	child.End()
	parent.End()
	
	spans := rec.Ended()
	if len(spans) != 2 {
		t.Fatalf("want 2 spans, got %d", len(spans))
	}
	if spans[0].Name() != "child" || spans[1].Name() != "parent" {
		t.Errorf("ordering wrong: %v %v", spans[0].Name(), spans[1].Name())
	}
	if spans[0].Parent().SpanID() != spans[1].SpanContext().SpanID() {
		t.Error("child parent should match parent span id")
	}
}

func TestEndSpan_RecordsError(t *testing.T) {
	rec := setupTestTracer(t)
	_, span := ChildSpan(context.Background(), "op")
	err := errors.New("boom")
	EndSpan(span, &err)
	
	spans := rec.Ended()
	if len(spans[0].Events()) == 0 {
		t.Error("expected exception event")
	}
}
```

---

## ADR (Architecture Decisions)

- **ADR-01**: KHÔNG đổi span name `kafka.consume` — giữ backward compat với dashboard SigNoz hiện có. Thêm attrs mới, KHÔNG xóa attrs cũ.
- **ADR-02**: Batch upsert là ROOT SPAN MỚI (không nested dưới message đầu). Lý do: batch gom nhiều message → owner không phải 1 message cụ thể. Cross-ref qua `cdc.batch_first_offset` / `cdc.batch_last_offset`.
- **ADR-03**: Span name convention `cdc.<verb>_<noun>` (snake_case sau dấu chấm) để filter SigNoz dễ.
- **ADR-04**: Error path dùng `span.RecordError(err) + SetStatus(codes.Error)`. KHÔNG dùng `attribute.String("error", err.Error())` (legacy, không trigger Exception tab).
- **ADR-05**: KHÔNG đụng sampleRatio. Local giữ 1.0, production operator tự tune.
- **ADR-06**: KHÔNG mass migrate log call site. Chỉ migrate 10 critical site trong scope các span mới. Migrate dần khi touch code.
- **ADR-07**: Audit field `audit=true` GIỮ Ở ROOT khi đi qua `Attrs(...)` (như phase 1) để severityAwareCore bypass đúng.

---

## Risk Matrix

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Span overhead > 50µs/msg | Low | Medium | Benchmark sau, nếu cần thì sample. Tracer no-op khi disabled. |
| ctx propagation gãy ở async path | Med | High | M4 ADR-02 chấp nhận root span cho batch. Verify trace tree manual sau. |
| SigNoz ingestion overload (5000 span/s) | Med | High | Production hạ sampleRatio (note trong report). Local OK. |
| Break existing dashboard | Low | Medium | Giữ span name + attrs cũ. Chỉ thêm. |
| Test flaky vì global tracer | Low | Low | Test helper setup riêng provider trong setupTestTracer. |
| Migrate sai log call site (lỗi nghiệp vụ) | Low | Medium | Migrate từng site, build sau mỗi nhóm 3. |

---

## Pre-execution Checklist (cho Muscle)

- [ ] Đọc lessons.md L-2026-05-26 (sampling + log_template) trước khi sửa.
- [ ] Verify `pkgs/observability/log_template.go` đã tồn tại (phase 1 done).
- [ ] `go build ./...` sạch trước khi bắt đầu.
- [ ] Sau mỗi milestone M2-M5: chạy `go build ./internal/<package>/...` trước khi sang milestone tiếp.
- [ ] M7 phải PASS trước khi M8 verify full.
- [ ] Report file ghi chính xác file changed + line count thực tế, KHÔNG ước lượng.

# 02_plan — Log Template OTel

## Roadmap
- M0: Workspace + audit (đã xong).
- M1: Code `pkgs/observability/log_template.go`.
- M2: Demo migrate 3 call site.
- M3: Verify build/vet/test.
- M4: Report file + lesson global.

## M1 Code demo (chi tiết)

```go
// pkgs/observability/log_template.go
package observability

import (
	"context"
	"fmt"
	"runtime"

	oteltrace "go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// ComponentLogger returns a derived logger pre-tagged with the
// `component` attribute. Use one per subsystem (kafka, postgres-sink,
// mongo-introspect, ...) so SigNoz can group log entries by component.
//
//	log := observability.ComponentLogger(baseLogger, "mongo-introspect")
//	log.Info("introspect.start") // emits attributes: component=mongo-introspect
func ComponentLogger(base *zap.Logger, component string) *zap.Logger {
	if base == nil {
		return zap.NewNop()
	}
	return base.With(zap.String("component", component))
}

// Ctx returns a derived logger with trace_id and span_id attached when
// the context carries an active span. When ctx has no span (e.g.
// context.Background, NATS message handlers without ctx), the base
// logger is returned unchanged so callers never have to nil-check.
//
//	log := observability.Ctx(ctx, h.logger)
//	log.Error("kafka.consume.failed", observability.ErrorField(err))
func Ctx(ctx context.Context, base *zap.Logger) *zap.Logger {
	if base == nil {
		return zap.NewNop()
	}
	if ctx == nil {
		return base
	}
	sc := oteltrace.SpanContextFromContext(ctx)
	if !sc.IsValid() {
		return base
	}
	return base.With(
		zap.String("trace_id", sc.TraceID().String()),
		zap.String("span_id", sc.SpanID().String()),
	)
}

// ErrorField encodes a Go error into the OTel exception attribute group
// nested under "error". SigNoz Log Details panel renders this as an
// "Exception" tab with kind/message/stacktrace fields.
//
// Returns zap.Skip() for nil error so callers can use it unconditionally:
//
//	log.Error("op.failed", observability.ErrorField(err), ...)
func ErrorField(err error) zap.Field {
	if err == nil {
		return zap.Skip()
	}
	return zap.Object("error", errorEncoder{err: err})
}

type errorEncoder struct{ err error }

func (e errorEncoder) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("kind", fmt.Sprintf("%T", e.err))
	enc.AddString("message", e.err.Error())
	enc.AddString("stack", captureStack(3))
	return nil
}

func captureStack(skip int) string {
	const max = 8 * 1024
	buf := make([]byte, max)
	n := runtime.Stack(buf, false)
	if n > max {
		n = max
	}
	return string(buf[:n])
}

// Attrs wraps business metadata under the "attributes" namespace so the
// SigNoz Log Details panel renders them as a dedicated "Attributes" tab
// separate from infrastructure fields (trace_id, component, ...).
//
//	log.Info("batch.upsert.ok",
//	  observability.Attrs(
//	    zap.String("source_table", table),
//	    zap.Int("batch_size", n),
//	  ),
//	)
func Attrs(fields ...zap.Field) zap.Field {
	return zap.Inline(zapcore.ObjectMarshalerFunc(func(enc zapcore.ObjectEncoder) error {
		ns := nestedAttrs{fields: fields}
		return enc.AddObject("attributes", ns)
	}))
}

type nestedAttrs struct{ fields []zap.Field }

func (n nestedAttrs) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	for _, f := range n.fields {
		f.AddTo(enc)
	}
	return nil
}
```

## M2 Demo migrate (BEFORE → AFTER)

### Call site 1: `command_handler.go:1199` Error
BEFORE:
```go
h.logger.Error("introspect.mongo.databases.failed",
    zap.String("sanitized_dsn", sanitized), zap.Error(err))
```
AFTER:
```go
observability.Ctx(context.Background(), h.logger).Error("introspect.mongo.databases.failed",
    observability.ErrorField(err),
    observability.Attrs(
        zap.String("sanitized_dsn", sanitized),
    ),
)
```
Note: NATS handler không nhận ctx → dùng Background. Pattern khi có ctx (HTTP/Kafka) sẽ pass ctx thật.

### Call site 2: `command_handler.go:1321` Info milestone
BEFORE:
```go
h.logger.Info("introspect.mongo.collections.ok",
    zap.String("sanitized_dsn", sanitized),
    zap.String("database", database),
    zap.Int("count", len(cols)),
    zap.Bool("audit", true))
```
AFTER:
```go
h.logger.Info("introspect.mongo.collections.ok",
    zap.Bool("audit", true),
    observability.Attrs(
        zap.String("sanitized_dsn", sanitized),
        zap.String("database", database),
        zap.Int("count", len(cols)),
    ),
)
```
Note: giữ `audit` ở root (severityAwareCore.Write detect tại top-level fields).

### Call site 3: `schema_inspector.go:162` Info drift
BEFORE:
```go
si.logger.Info("schema drift detected (batch summary)",
    zap.String("source_db", sourceDB),
    zap.String("table", tableName),
    zap.Int("unique_new_fields", len(fields)),
    zap.Int64("events_with_drift", count),
    zap.Strings("fields", fieldNames),
    zap.Bool("audit", true),
)
```
AFTER:
```go
si.logger.Info("schema drift detected (batch summary)",
    zap.Bool("audit", true),
    observability.Attrs(
        zap.String("source_db", sourceDB),
        zap.String("table", tableName),
        zap.Int("unique_new_fields", len(fields)),
        zap.Int64("events_with_drift", count),
        zap.Strings("fields", fieldNames),
    ),
)
```

## ADR
- ADR-01: `audit` field KEEP ở root (không vào Attrs namespace), vì severityAwareCore.Write iterate top-level fields để detect bypass. Nếu nest dưới Attrs object, hasAuditField không nhìn thấy.
- ADR-02: Stack trace dùng `runtime.Stack(false)` chỉ current goroutine, cap 8KB. Đủ context, không bloat log.
- ADR-03: KHÔNG export const `"audit"` key — giữ inline literal vì zap field key thường viết inline.
- ADR-04: KHÔNG migrate hàng loạt — phase này chỉ pattern demo. Migrate dần khi touch code.

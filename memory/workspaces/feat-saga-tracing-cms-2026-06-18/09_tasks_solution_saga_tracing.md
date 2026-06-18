# 09 — Technical Solutions: Saga & Tracing

## SOL-01: Saga Runner với OTel span per-step

**File**: `internal/app/saga/saga.go`

```go
package saga

import (
    "context"
    "fmt"

    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/codes"
    "go.uber.org/zap"

    "cdc-cms-service/pkgs/observability"
)

// Step là 1 bước trong saga.
// Compensate là best-effort (nil = no-op).
type Step struct {
    Name       string
    Execute    func(ctx context.Context) error
    Compensate func(ctx context.Context) error
}

// Runner thực thi saga steps tuần tự với OTel spans + compensation.
type Runner struct {
    name   string
    steps  []Step
    logger *zap.Logger
}

// New tạo Runner. name dùng cho OTel span name "saga.{name}".
func New(name string, logger *zap.Logger, steps ...Step) *Runner {
    if logger == nil {
        logger = zap.NewNop()
    }
    return &Runner{name: name, steps: steps, logger: logger}
}

// Run thực thi steps, compensate ngược khi step fail.
func (r *Runner) Run(ctx context.Context) error {
    ctx, sagaSpan := observability.StartSpan(ctx, "saga."+r.name,
        attribute.String("saga.name", r.name),
        attribute.Int("saga.steps", len(r.steps)),
    )
    defer sagaSpan.End()

    executed := make([]Step, 0, len(r.steps))

    for _, step := range r.steps {
        stepCtx, stepSpan := observability.StartSpan(ctx, "saga.step",
            attribute.String("saga.name", r.name),
            attribute.String("saga.step", step.Name),
        )

        r.logger.Debug("saga.step.execute",
            zap.String("saga", r.name),
            zap.String("step", step.Name))

        if err := step.Execute(stepCtx); err != nil {
            stepSpan.RecordError(err)
            stepSpan.SetStatus(codes.Error, err.Error())
            stepSpan.End()

            sagaSpan.RecordError(err)
            sagaSpan.SetStatus(codes.Error, fmt.Sprintf("step %q failed", step.Name))

            r.logger.Error("saga.step.failed",
                zap.String("saga", r.name),
                zap.String("step", step.Name),
                zap.Int("compensating", len(executed)),
                zap.Error(err))

            r.compensate(ctx, executed)
            return fmt.Errorf("saga %q step %q: %w", r.name, step.Name, err)
        }

        stepSpan.End()
        executed = append(executed, step)
    }
    return nil
}

func (r *Runner) compensate(ctx context.Context, executed []Step) {
    for i := len(executed) - 1; i >= 0; i-- {
        s := executed[i]
        if s.Compensate == nil {
            continue
        }
        r.logger.Warn("saga.compensate",
            zap.String("saga", r.name),
            zap.String("step", s.Name))
        if err := s.Compensate(ctx); err != nil {
            r.logger.Error("saga.compensate.failed — MANUAL ACTION REQUIRED",
                zap.String("saga", r.name),
                zap.String("step", s.Name),
                zap.Error(err))
        }
    }
}
```

---

## SOL-02: OTel helpers trong `pkgs/observability/otel.go`

**Thêm imports**:
```go
"go.opentelemetry.io/otel/codes"
```

**Thêm 2 functions sau `StartSpan`**:
```go
// EndSpan kết thúc span và record error nếu *err != nil.
// Dùng với named return + defer:
//
//   func (h *Handler) Handle(ctx context.Context, c ports.Command) (_ json.RawMessage, err error) {
//       ctx, span := observability.StartSpan(ctx, "handler.foo")
//       defer observability.EndSpan(span, &err)
//       ...
//   }
func EndSpan(span otelTrace.Span, err *error) {
    if err != nil && *err != nil {
        span.RecordError(*err)
        span.SetStatus(codes.Error, (*err).Error())
    }
    span.End()
}

// Ctx trả về logger với trace_id/span_id fields inject từ OTel context.
// Fallback về base logger nếu span không valid (OTel disabled).
func Ctx(ctx context.Context, base *zap.Logger) *zap.Logger {
    spanCtx := otelTrace.SpanFromContext(ctx).SpanContext()
    if !spanCtx.IsValid() {
        return base
    }
    return base.With(
        zap.String("trace_id", spanCtx.TraceID().String()),
        zap.String("span_id",  spanCtx.SpanID().String()),
    )
}
```

**Thêm W3C propagator trong `InitOtel()` sau `otel.SetTracerProvider(tp)`**:
```go
// Set W3C Trace Context as global propagator
otel.SetTextMapPropagator(
    propagation.NewCompositeTextMapPropagator(
        propagation.TraceContext{},
        propagation.Baggage{},
    ),
)
```

**Import thêm trong InitOtel**:
```go
"go.opentelemetry.io/otel/propagation"
```

---

## SOL-03: Fiber OTel Propagator Middleware

**File**: `internal/middleware/otel_propagator.go`

```go
// Package middleware provides Fiber middleware for cross-cutting concerns.
package middleware

import (
    "github.com/gofiber/fiber/v2"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/propagation"
)

// OtelPropagator extracts W3C Trace Context from HTTP request headers
// and injects it into Fiber's UserContext.
//
// Must be registered BEFORE route handlers:
//   app.Use(middleware.OtelPropagator())
//
// When OTel is disabled (noop provider), this middleware is a no-op.
func OtelPropagator() fiber.Handler {
    prop := otel.GetTextMapPropagator()
    return func(c *fiber.Ctx) error {
        carrier := make(propagation.MapCarrier)
        c.Request().Header.VisitAll(func(k, v []byte) {
            carrier[string(k)] = string(v)
        })
        ctx := prop.Extract(c.UserContext(), carrier)
        c.SetUserContext(ctx)
        return c.Next()
    }
}
```

---

## SOL-04: CommandBus Execute + Dispatch instrumentation

**File**: `internal/infra/messaging/nats_command_bus.go`

**Thêm imports**:
```go
"cdc-cms-service/pkgs/observability"
"go.opentelemetry.io/otel/attribute"
```

**Sửa `Execute`** (chỉ thay đổi signature + thêm span, body giữ nguyên):
```go
// BEFORE:
func (b *natsCommandBus) Execute(ctx context.Context, c ports.SyncCommand) (ports.SyncResult, error) {

// AFTER:
func (b *natsCommandBus) Execute(ctx context.Context, c ports.SyncCommand) (_ ports.SyncResult, err error) {
    ctx, span := observability.StartSpan(ctx, "command_bus.execute",
        attribute.String("command.type", c.Type()),
    )
    defer observability.EndSpan(span, &err)
    // ... body không đổi ...
```

**Sửa `Dispatch`**:
```go
// BEFORE:
func (b *natsCommandBus) Dispatch(ctx context.Context, c ports.AsyncCommand) (ports.AsyncResult, error) {

// AFTER:
func (b *natsCommandBus) Dispatch(ctx context.Context, c ports.AsyncCommand) (_ ports.AsyncResult, err error) {
    ctx, span := observability.StartSpan(ctx, "command_bus.dispatch",
        attribute.String("command.type", c.Type()),
    )
    defer observability.EndSpan(span, &err)
    // ... body không đổi ...
```

---

## SOL-05: Server.go — Register OtelPropagator

**File**: `internal/server/server.go`

Tìm đoạn `app.Use(...)` hoặc trước `router.SetupRoutes(...)`, thêm:
```go
// Tracing: extract W3C trace context from incoming requests
app.Use(middleware.OtelPropagator())
```

**Import thêm**:
```go
"cdc-cms-service/internal/middleware"
```

---

## SOL-06: approve_master.go refactor với Saga

```go
import "cdc-cms-service/internal/app/saga"

func (h *ApproveMasterHandler) Handle(ctx context.Context, c ports.Command) (json.RawMessage, error) {
    cmd, ok := c.(ApproveMasterCommand)
    if !ok {
        return nil, errors.New("master.approve: command type mismatch")
    }
    if h.repo == nil {
        return nil, errors.New("master store not ready")
    }

    runner := saga.New("master.approve", h.logger,
        saga.Step{
            Name: "approve-schema-tx",
            Execute: func(ctx context.Context) error {
                _, _, err := h.repo.ApproveSchemaTx(ctx, cmd.Name, cmd.UpdatedBy)
                if err != nil {
                    switch err.Error() {
                    case "not_found":            return ErrMasterNotFound
                    case "ambiguous_master_name": return ErrMasterNameAmbiguous
                    case "not_approvable":        return ErrMasterNotApprovable
                    }
                    return err
                }
                return nil
            },
            Compensate: func(ctx context.Context) error {
                return h.repo.RevertSchemaTx(ctx, cmd.Name, cmd.UpdatedBy)
            },
        },
        saga.Step{
            Name: "publish-master-create",
            Execute: func(ctx context.Context) error {
                if h.publisher == nil {
                    return errors.New("nats not ready")
                }
                payload, _ := json.Marshal(map[string]string{
                    "master_table":   cmd.Name,
                    "triggered_by":   cmd.UpdatedBy,
                    "correlation_id": "approve-" + cmd.Name + "-" + time.Now().UTC().Format(time.RFC3339Nano),
                })
                return h.publisher.Publish(ctx, "cdc.cmd.master-create", payload)
            },
            Compensate: nil, // NATS fire-and-forget — không thể un-publish
        },
    )

    if err := runner.Run(ctx); err != nil {
        return nil, err
    }

    out, _ := json.Marshal(map[string]interface{}{
        "status":      "approved",
        "master_name": cmd.Name,
        "dispatched":  "cdc.cmd.master-create",
    })
    return out, nil
}
```

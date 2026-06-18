# 03 — Implementation Design: OTel Distributed Tracing

## 1. Kiến trúc Tracing

```
HTTP Client
  │  Header: traceparent: 00-{traceID}-{spanID}-01
  ▼
Fiber App
  │  middleware.OtelPropagator() → extract W3C context → inject ctx
  ▼
API Handler (fiber.Ctx)
  │  ctx = c.UserContext()  ← already has span context from propagator
  │  messaging.WithMetadata(ctx, user, correlationID, ...)
  ▼
CommandBus.Execute(ctx, cmd)
  │  StartSpan(ctx, "command_bus.execute", cmd.type=...)
  ├─ SyncHandler.Handle(ctx, cmd)
  │    └─ [saga.Runner.Run(ctx)]
  │         ├─ StartSpan(ctx, "saga.{name}")          ← parent saga span
  │         │    ├─ StartSpan(ctx, "saga.step", step=step1)  ← child
  │         │    ├─ StartSpan(ctx, "saga.step", step=step2)
  │         │    └─ StartSpan(ctx, "saga.step", step=step3)
  ▼
CommandBus.Dispatch(ctx, cmd)
  │  StartSpan(ctx, "command_bus.dispatch", cmd.type=...)
  │  PublishMsg(natsMsg) ← NATS headers: Cdc-Correlation-Id, Cdc-Job-Id
  ▼
NATS Worker (centralized-data-service)
     [independent trace — không propagate qua NATS boundary trong scope này]
```

## 2. Span Naming Convention

| Span Name | Attrs | Ghi chú |
|-----------|-------|---------|
| `command_bus.execute` | `command.type=master.approve` | CommandBus sync |
| `command_bus.dispatch` | `command.type=master.swap` | CommandBus async |
| `saga.{name}` | `saga.name=registry.register` | Parent saga span |
| `saga.step` | `saga.name=...`, `saga.step=register-db` | Per-step child |

## 3. File Changes chi tiết

### 3.1 `pkgs/observability/otel.go` [MODIFY — thêm 2 helpers]

**Thêm import**: `"go.opentelemetry.io/otel/codes"`

```go
// EndSpan kết thúc span, ghi error status nếu *err != nil.
// Dùng với named return + defer:
//
//   func foo(ctx context.Context) (_ SomeResult, err error) {
//       ctx, span := observability.StartSpan(ctx, "foo")
//       defer observability.EndSpan(span, &err)
//       ...
//   }
//
// Lý do dùng pointer: defer capture *err tại thời điểm hàm return,
// không phải thời điểm defer được đặt.
func EndSpan(span otelTrace.Span, err *error) {
    if err != nil && *err != nil {
        span.RecordError(*err)
        span.SetStatus(codes.Error, (*err).Error())
    }
    span.End()
}

// Ctx trả về zap.Logger được inject trace_id/span_id từ OTel span context.
// Khi OTel disabled → span không valid → trả về base logger không thay đổi.
// Dùng thay cho zap.Logger trực tiếp để logs correlate với traces trong SigNoz/Jaeger.
//
// Ví dụ:
//   log := observability.Ctx(ctx, h.logger)
//   log.Info("handler invoked")  // → log có trace_id, span_id field
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

**Tại sao pointer `*error` thay vì `error`?**  
Khi dùng `defer observability.EndSpan(span, &err)`, Go evaluate `&err` ngay lúc defer đặt, nhưng dereference `*err` chỉ xảy ra khi hàm thực sự return. Nếu dùng `error` (value), `defer` capture giá trị `nil` tại thời điểm đặt → không bao giờ thấy error.

---

### 3.2 `internal/middleware/otel_propagator.go` [NEW]

```go
// Package middleware cung cấp Fiber middleware cho cross-cutting concerns.
package middleware

import (
    "github.com/gofiber/fiber/v2"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/propagation"
)

// OtelPropagator là Fiber middleware extract W3C Trace Context
// từ HTTP request headers và inject vào Fiber UserContext.
//
// Headers được xử lý (W3C standard):
//   - traceparent: 00-{traceID}-{spanID}-{flags}
//   - tracestate:  vendor-specific state
//
// Phải được register TRƯỚC tất cả route handlers:
//   app.Use(middleware.OtelPropagator())
//   app.Get("/api/...", handler)
//
// Khi OTel disabled (provider = noop), middleware này vẫn chạy
// nhưng extract ra noop span context → zero overhead.
func OtelPropagator() fiber.Handler {
    prop := otel.GetTextMapPropagator()
    return func(c *fiber.Ctx) error {
        // Chuyển Fiber request headers thành propagation.MapCarrier
        // để OTel propagator đọc được.
        carrier := make(propagation.MapCarrier)
        c.Request().Header.VisitAll(func(k, v []byte) {
            carrier[string(k)] = string(v)
        })

        // Extract span context từ carrier, inject vào ctx
        ctx := prop.Extract(c.UserContext(), carrier)
        c.SetUserContext(ctx)

        return c.Next()
    }
}
```

**Lý do dùng `propagation.MapCarrier` thay vì custom**:  
MapCarrier implement `TextMapCarrier` interface, tương thích với mọi propagator (W3C, B3, Jaeger). Khi sau này cần switch propagator, chỉ thay `otel.GetTextMapPropagator()` không cần sửa middleware.

---

### 3.3 `internal/infra/messaging/nats_command_bus.go` [MODIFY]

**Thêm import**:
```go
"cdc-cms-service/pkgs/observability"
"go.opentelemetry.io/otel/attribute"
```

**Sửa `Execute` — named return + defer EndSpan**:
```go
// BEFORE:
func (b *natsCommandBus) Execute(ctx context.Context, c ports.SyncCommand) (ports.SyncResult, error) {

// AFTER (named return để defer capture error):
func (b *natsCommandBus) Execute(ctx context.Context, c ports.SyncCommand) (_ ports.SyncResult, err error) {
    ctx, span := observability.StartSpan(ctx, "command_bus.execute",
        attribute.String("command.type", c.Type()),
    )
    defer observability.EndSpan(span, &err)

    // ... body giữ nguyên, KHÔNG thay đổi logic ...
}
```

**Sửa `Dispatch` — tương tự**:
```go
// BEFORE:
func (b *natsCommandBus) Dispatch(ctx context.Context, c ports.AsyncCommand) (ports.AsyncResult, error) {

// AFTER:
func (b *natsCommandBus) Dispatch(ctx context.Context, c ports.AsyncCommand) (_ ports.AsyncResult, err error) {
    ctx, span := observability.StartSpan(ctx, "command_bus.dispatch",
        attribute.String("command.type", c.Type()),
    )
    defer observability.EndSpan(span, &err)

    // ... body giữ nguyên ...
}
```

**Lý do chỉ instrument ở bus-level, không phải từng handler**:
- Bus là entry point chung → 1 chỗ, cover toàn bộ commands
- Per-handler span = code duplication ở 29 handlers → noise, không elegant
- Handler chạy trong ctx của bus span → tự động là child span

---

### 3.4 `internal/server/server.go` [MODIFY — 1 dòng]

Thêm middleware registration TRƯỚC tất cả routes:

```go
// Tìm đoạn khởi tạo Fiber app
// VD: app := fiber.New(...)

// Thêm ngay sau:
app.Use(middleware.OtelPropagator())

// Sau đó mới register routes:
// router.SetupRoutes(app, ...)
```

**Import cần thêm**:
```go
"cdc-cms-service/internal/middleware"
```

## 4. W3C Trace Context Setup (Init)

OTel SDK đã init trong `pkgs/observability/otel.go`. Cần đảm bảo W3C propagator được set:

```go
// Trong pkgs/observability/otel.go — hàm InitOtel()
// Thêm sau otel.SetTracerProvider(tp):

import "go.opentelemetry.io/otel/propagation"

// Set W3C Trace Context propagator (standard)
otel.SetTextMapPropagator(
    propagation.NewCompositeTextMapPropagator(
        propagation.TraceContext{}, // W3C traceparent/tracestate
        propagation.Baggage{},      // W3C baggage (optional)
    ),
)
```

> **⚠️ Quan trọng**: Nếu không set propagator, `otel.GetTextMapPropagator()` trả về noop propagator → middleware không extract gì cả.

## 5. Propagation qua NATS (future scope)

Hiện tại NATS headers đã có `Cdc-Correlation-Id`. Để propagate OTel trace qua NATS boundary (kết nối trace từ CMS service sang centralized-data-service), cần inject `traceparent` vào NATS headers. **Đây là KHÔNG trong scope hiện tại** — để phase sau.

```go
// Future: trong buildCommandMsg()
// carrier := propagation.MapCarrier{}
// otel.GetTextMapPropagator().Inject(ctx, carrier)
// msg.Header.Set("traceparent", carrier["traceparent"])
```

## 6. Verification — Cách kiểm tra tracing hoạt động

### Option A: SigNoz (nếu endpoint config)
- Enable OTel trong config: `otel.enabled=true`, `otel.endpoint=http://signoz:4318`
- Gọi `POST /api/v1/masters/{name}/approve`
- Kiểm tra SigNoz: tìm trace với span `command_bus.execute` → `saga.master.approve` → `saga.step.*`

### Option B: Test unit (không cần SigNoz)
```go
// Dùng sdktrace.NewSimpleSpanProcessor + tracetest.NewInMemoryExporter
// để assert span được tạo đúng tên và attributes
```

# Bug Fix Report — Detached Span Context (Production Safety Fix)

> **Date**: 2026-06-25T09:41 +07:00
> **Reviewer**: User (pre-merge review)
> **Fixer**: Brain/Antigravity
> **Service**: `centralized-data-service`
> **Scope**: 1 critical architectural trap — Go context deadline inheritance

---

## Kết quả Verification

| Check | Result |
|-------|--------|
| `go build ./internal/... ./pkgs/... ./cmd/...` | ✅ PASS |
| `go test -race ./internal/service/recon/... -timeout 120s` | ✅ PASS (2.040s) |
| Race detector | ✅ CLEAN |

---

## Vấn đề — "Ảo giác" của `context.WithTimeout` trong Go

### Root cause
Trong Go, `context.WithTimeout(parentCtx, 8*time.Minute)` **silently kế thừa deadline ngắn nhất** giữa parent và child.

```
Nếu parentCtx.Deadline = 30s (NATS timeout / HTTP middleware)
→ drillCtx.Deadline = min(30s, 8m) = 30s  ← BucketCounts vẫn chết ở giây 30!
```

**Code trước (bug tiềm ẩn)**:
```go
// drillCtx kế thừa ctx gốc — nhưng nếu ctx có deadline 30s, drillCtx cũng chết ở 30s
drillCtx, cancelDrill := context.WithTimeout(ctx, 8*time.Minute)
```

### Tại sao chưa bị lỗi ngay?

Trace handler hiện tại:
```go
// recon_handler_run.go:18
ctx := observability.ExtractNATSHeader(context.Background(), msg.Header)
```

`ctx` được tạo từ `context.Background()` — **không có deadline**. Vì vậy hiện tại chưa bị lỗi.

**Nhưng đây là time bomb**: Nếu tương lai có:
- NATS JetStream với `AckWait` (e.g. 60s)
- HTTP middleware inject timeout vào ctx
- Operator bọc thêm `context.WithTimeout` upstream

→ `drillCtx` sẽ chết theo parent mà không có warning nào.

---

## Fix — Detached Span Context Pattern

### Helper function mới — `recon_tier_a.go` (L22-36)

```go
// detachedSpanContext creates a new context rooted at context.Background()
// but carrying the OTel Span from the parent context.
//
// Solves the "parent deadline inheritance" trap:
//   - context.WithTimeout(ctx, 8m) inherits the SHORTER of parent and child deadlines.
//   - If NATS handler / middleware set a short timeout (e.g. 30s),
//     drillCtx also dies at 30s regardless of the 8m setting.
//
// Solution: detach from parent's deadline while keeping OTel Span intact,
// so BucketCounts traces remain visible on SigNoz with the correct TraceID.
func detachedSpanContext(ctx context.Context) context.Context {
    span := trace.SpanFromContext(ctx)
    return trace.ContextWithSpan(context.Background(), span)
}
```

### Cập nhật `drillCtx` — `recon_tier_a.go` (L513-517)

```diff
- // drillCtx kế thừa ctx gốc — giữ OTel TraceID
- drillCtx, cancelDrill := context.WithTimeout(ctx, 8*time.Minute)
+ // drillCtx: detach khỏi deadline của parent (NATS / upstream middleware)
+ // nhưng giữ OTel Span để BucketCounts trace vẫn xuất hiện trên SigNoz.
+ drillCtx, cancelDrill := context.WithTimeout(detachedSpanContext(ctx), 8*time.Minute)
  defer cancelDrill()
```

### Import mới — `recon_tier_a.go` (L18)

```diff
+ "go.opentelemetry.io/otel/trace"
  "go.uber.org/zap"
```

---

## So sánh 3 options

| Approach | TraceID | Deadline safe | Notes |
|----------|---------|---------------|-------|
| `context.Background()` | ❌ mất | ✅ an toàn | Mù trên SigNoz |
| `context.WithTimeout(ctx, 8m)` | ✅ giữ | ❌ kế thừa parent deadline | Fix ban đầu — vẫn có trap |
| `context.WithTimeout(detachedSpanContext(ctx), 8m)` | ✅ giữ | ✅ an toàn | **Production-safe** ✅ |

---

## Trả lời câu hỏi về NATS

> *Luồng trigger đang dùng NATS JetStream hay Core NATS?*

**Hiện tại: Core NATS** (`*nats.Msg`, không phải `*nats.JetStreamMsg`).

```go
// recon_handler_run.go:17
func (h *ReconHandler) HandleReconCheck(msg *nats.Msg) {
    ctx := observability.ExtractNATSHeader(context.Background(), msg.Header)
    // ...
    reports := h.reconCore.CheckAll(ctx)
}
```

- Core NATS: Fire-and-forget, không có `AckWait`, không có deadline tự động inject vào ctx.
- `ctx` tạo từ `context.Background()` → **không có deadline hiện tại**.

**Khuyến nghị nếu migrate sang JetStream**: JetStream có `AckWait` (mặc định 30s). Nếu handler không ACK trong thời gian đó, NATS sẽ redeliver. Với `CheckAll` chạy 10-30 phút, cần:
1. `msg.InProgress()` gọi định kỳ để extend AckWait (keepalive)
2. Hoặc set `AckWait` đủ lớn (e.g. 30m) cho subject `recon.check.all`
3. `detachedSpanContext` pattern này bảo vệ khỏi JetStream AckWait timeout lọt vào ctx.

---

## Files thay đổi

| File | Dòng | Thay đổi |
|------|------|----------|
| `recon_tier_a.go` | L8-19 | Thêm `go.opentelemetry.io/otel/trace` import |
| `recon_tier_a.go` | L22-36 | Thêm `detachedSpanContext()` helper |
| `recon_tier_a.go` | L513-517 | `drillCtx` dùng `detachedSpanContext(ctx)` |

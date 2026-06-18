# 04 — Architecture Decision Records (ADRs)

## ADR-001: Choreography Saga thay vì Orchestration Saga

**Ngày**: 2026-06-18  
**Trạng thái**: Accepted

**Vấn đề**: Cần cơ chế compensation cho multi-step workflows.

**Các lựa chọn**:
- A) Orchestration Saga (có saga orchestrator riêng, state machine)
- B) Choreography Saga (local runner, no external state)

**Quyết định**: Chọn B — Choreography Saga

**Lý do**:
- Service-local transactions, không cần cross-service coordination
- Đơn giản hơn, ít infrastructure hơn (không cần Redis/DB saga table)
- Phù hợp với Simplicity First principle
- Tất cả 5 saga flows đều là in-process operations

---

## ADR-002: Bus-level span thay vì Handler-level span

**Ngày**: 2026-06-18  
**Trạng thái**: Accepted

**Vấn đề**: Cần OTel spans nhưng có 29 SyncHandlers.

**Các lựa chọn**:
- A) Thêm span vào mỗi Handler.Handle() — 29 places
- B) Thêm span vào CommandBus.Execute/Dispatch — 1 place

**Quyết định**: Chọn B

**Lý do**:
- DRY: 1 chỗ cover tất cả commands
- Handler chạy trong ctx của bus span → tự động là child span
- Không có code duplication ở 29 handlers
- Per-handler span có thể thêm sau nếu cần granularity hơn

---

## ADR-003: Named return + defer `EndSpan` thay vì explicit span.End()

**Ngày**: 2026-06-18  
**Trạng thái**: Accepted

**Vấn đề**: Span phải End() ngay cả khi có early return với error.

**Pattern đã dùng**:
```go
func foo() (_ Result, err error) {
    ctx, span := observability.StartSpan(ctx, "foo")
    defer observability.EndSpan(span, &err)  // ← capture *err at return time
    ...
}
```

**Lý do**:
- Idiomatic Go: Go spec đảm bảo defer chạy trước khi function return
- Named return `err` + `defer EndSpan(span, &err)` đảm bảo span luôn nhận đúng error cuối cùng
- Không cần nhớ gọi `span.End()` ở mọi early return path

---

## ADR-004: W3C Trace Context propagator (không B3, không Jaeger)

**Ngày**: 2026-06-18  
**Trạng thái**: Accepted

**Quyết định**: Dùng W3C `propagation.TraceContext{}` + `propagation.Baggage{}`

**Lý do**:
- W3C là standard hiện tại (RFC 7230)
- SigNoz và Jaeger đều support W3C
- Tương thích với mọi OTel-compatible service

---

## ADR-005: Saga.Runner có OTel span built-in

**Ngày**: 2026-06-18  
**Trạng thái**: Accepted

**Quyết định**: Saga Runner tạo span "saga.{name}" + per-step "saga.step" spans

**Lý do**:
- Mọi saga invocation tự động được trace
- Per-step span giúp identify chính xác step nào slow/fail trong SigNoz
- Không cần caller phải manually tạo span

---

## ADR-006: HandlerGroup struct trong package `router` (không phải `server`)

**Ngày**: 2026-06-18  
**Trạng thái**: Accepted

**Vấn đề**: `SetupRoutes` đang nhận 27 parameters rời rạc — không thể mở rộng khi thêm handler mới.

**Các lựa chọn**:
- A) HandlerGroup định nghĩa trong `server.go` → `server` package biết cấu trúc routing
- B) HandlerGroup định nghĩa trong `router.go` → `router` package tự định nghĩa contract của nó
- C) Tạo package `internal/api/dto/handler_group.go` riêng

**Quyết định**: Chọn B — HandlerGroup thuộc package `router`

**Lý do**:
- Router tự sở hữu contract handler của mình (single responsibility)
- `server.go` chỉ cần import `router` và populate struct — không cần biết routing internals
- Tránh circular import (nếu để ở `server` package, `router` phải import `server`)
- Thêm handler mới: chỉ thêm field vào HandlerGroup + route trong router.go — không đụng server.go

**Trade-off chấp nhận**:
- `router` package phụ thuộc vào tất cả 7 api sub-packages (governance, source, shadow, master, scheduler, recon, system)
- Đây là dependency đúng về mặt kiến trúc: router phải biết handlers nó route đến

# 01_requirements

## Functional
- R1: Hàm `ComponentLogger(base, component) *zap.Logger` trả về logger pre-tagged field `component`. Backward compat: base logger không bị mutate.
- R2: Hàm `Ctx(ctx, base) *zap.Logger` extract `SpanContext` từ ctx, đính `trace_id` + `span_id` (hex string). Nếu ctx không có span (vd `context.Background()`), trả về base unchanged.
- R3: Hàm `ErrorField(err error) zap.Field` trả về `zap.Inline` (hoặc `zap.Object`) emit nested map `{"error": {"kind": "<Type>", "message": "<err.Error()>", "stack": "<stacktrace>"}}`. Nil error → no-op (`zap.Skip()`).
- R4: Hàm `Attrs(fields ...zap.Field) zap.Field` trả về namespace `attributes` chứa các business fields.
- R5: Demo migrate phải compile + vet pass + giữ semantics cũ (key cũ vẫn còn dưới namespace mới, không mất data trên SigNoz).

## Non-functional
- N1: KHÔNG breaking change interface zap.Logger. Helpers là pure functions, return *zap.Logger hoặc zap.Field.
- N2: Performance: ErrorField gọi `runtime.Stack` chỉ khi err != nil → ~10µs khi có lỗi, 0 khi không. Acceptable.
- N3: KHÔNG đụng config/secret/db.
- N4: Phải tương thích với severityAwareCore + audit bypass đã làm phase trước.

## DoD
- A1: File `pkgs/observability/log_template.go` mới, tự-document.
- A2: 3 call site migrate, build + vet exit 0.
- A3: Report file `report_log_template_otel_2026-05-26.md` ghi BEFORE/AFTER snippet, evidence build log path.
- A4: Workspace docs đầy đủ (00,01,02,05,report).
- A5: Append lesson global pattern.

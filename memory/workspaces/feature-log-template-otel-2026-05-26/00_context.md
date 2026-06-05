# 00_context — Log Template OTel-Compliant

## Triệu chứng user báo
1. SigNoz log list chỉ hiện `timestamp | message` — không thấy fields nghiệp vụ (component, source_table, batch_size, trace_id...). Khó debug.
2. SigNoz trace list chỉ hiện `kafka.consume` với latency, cột "Linked Logs" + "Linked Spans" hiển thị `N/A` → không correlate được log ↔ trace.

## Audit codebase
- `cmd/worker/main.go:74-76`: ĐÃ CÓ `otelzap.NewCore("cdc-worker", otelzap.WithLoggerProvider(lp))` từ `go.opentelemetry.io/contrib/bridges/otelzap`. Bridge này serialize zap.Field → OTel attribute đúng chuẩn proto.
- `pkgs/observability/otel.go`: `resource.New` set `service.name=cdc-worker`, `environment=development` → tự động đính kèm vào mỗi LogRecord. OK.
- `pkgs/observability/otel.go:418-425`: BatchProcessor + LoggerProvider config đầy đủ. OK.
- **Gap chính**:
  - Bridge `otelzap` KHÔNG tự inject `trace_id`/`span_id` vào LogRecord (vì zap.Core không nhận `context.Context`). Cần helper inject ở call site.
  - Call site dùng flat `zap.String("table", ...)`, `zap.Error(err)` → bridge serialize thành attribute key=value phẳng. SigNoz không nhận diện được nhóm `error.{type,message,stack}` hoặc namespace `attributes.*`.
  - Component (subsystem) không tag tự động → mỗi log site phải tự ghi.

## Vì sao SigNoz UI chỉ hiện title
SigNoz UI mặc định 2 columns: `timestamp + body (message)`. Phải vào panel **"Add column"** chọn `attributes.component`, `attributes.source_table`, `severity_text`, `trace_id`... Đây là config UI bên SigNoz, KHÔNG phải bug backend. Nhưng để columns đó có data, backend PHẢI emit đúng attributes.

## Scope phase này (đúng yêu cầu user)
**"Viết 1 hàm để đưa log về template này"** → tạo helpers trong `pkgs/observability/log_template.go`:
1. `ComponentLogger(base, component)` — pre-tag component name.
2. `Ctx(ctx, base)` — extract trace_id + span_id từ context, attach.
3. `ErrorField(err)` — encode error thành nested object `error.{kind, message, stack}`.
4. `Attrs(fields...)` — namespace `attributes` cho business metadata.

**Demo migrate 3 call site** đại diện 3 pattern:
- Error log với error block: `command_handler.go:1199` (`introspect.mongo.databases.failed`).
- Info milestone với business attrs: `command_handler.go:1321` (`introspect.mongo.collections.ok`).
- Info drift với strings array: `schema_inspector.go:162` (`schema drift detected (batch summary)`).

## Out of scope (defer)
- Migrate hàng loạt call site (~100+) → defer, làm dần. Phase này chỉ demo pattern.
- Trace span attributes (kafka.consume thêm topic/partition/offset) → phase tiếp theo theo lời user.
- SigNoz UI configure columns → user tự config (cần truy cập SigNoz UI).
- Không cheat config: KHÔNG đụng OTLP endpoint, KHÔNG bypass sampling cho phase này.

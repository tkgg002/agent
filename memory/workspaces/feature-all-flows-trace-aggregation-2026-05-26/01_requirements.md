# 01_requirements — All Flows Trace Aggregation

## Functional Requirements

### R1 — Mọi flow phải có 1 root span duy nhất
Mỗi "flow" (đơn vị work-unit) phải có **đúng 1 root span** gốc với name dạng `<subsystem>.<verb>`. Mọi span con sinh ra trong flow (kể cả async/goroutine) phải là child trực tiếp hoặc gián tiếp của root đó. Trace_id của root được giữ nguyên xuyên suốt.

**Definition of "flow"**:
| Loại | Entry point | Root span name |
|------|-------------|----------------|
| NATS command | `Subscribe` callback | `nats.<subject>` (e.g. `nats.cdc.cmd.recon-check`) |
| NATS event | `Subscribe` event topic | `nats.<subject>` |
| Kafka message | `kafka.Reader.ReadMessage` | `kafka.consume` (đã có) |
| HTTP request | Fiber/Gin handler | `http.<method>.<route>` |
| Schedule tick | Timer / cron tick | `schedule.<job_name>` |
| Background sweep | DLQ/Partition/FullCount tick | `bg.<worker_name>.sweep` |
| Snapshot run | `snapshot_runner.runSnapshot` | `snapshot.v2.run` |
| Recon cycle | `recon_core.runReconcileCycle` | `recon.cycle` |

### R2 — Cấm `context.Background()` ngoài 4 ngoại lệ
Sau phase này, `grep -n "context.Background()" internal/ pkgs/` chỉ được phép xuất hiện ở:
- (a) `main.go` startup root context.
- (b) Constructor lifecycle context (`context.WithCancel(context.Background())` cho service lifetime, ví dụ `BatchBuffer.ctx`).
- (c) Shutdown / cleanup context có timeout (disconnect DB, close reader).
- (d) Detached operation cố ý có **OTel Span Link** đến parent (ví dụ heartbeat goroutine).

Mọi site khác phải nhận `ctx` từ caller / extract từ NATS message / extract từ Kafka message.

### R3 — NATS message phải có `traceparent` + `tracestate` header
Mọi NATS `Publish*`/`PublishMsg` từ Go worker phải inject `traceparent` qua `otel.GetTextMapPropagator().Inject(ctx, natsHeaderCarrier)`. Mọi `Subscribe`/`QueueSubscribe` callback phải extract trước khi tạo span. Khi header rỗng → tự tạo root span (graceful).

### R4 — Kafka producer cũng phải inject (consumer side đã có)
Mọi Kafka `Writer.WriteMessages` từ Go worker phải inject `traceparent` vào `msg.Headers`. Apply cho `DebeziumSignalClient.TriggerIncrementalSnapshot`.

### R5 — Background workers: 1 root span per tick
Tất cả timer/cron loops phải tạo root span per tick, không phải span lifetime của goroutine. Span name: `bg.<worker_name>.tick`.

### R6 — Snapshot V2: 1 root span per run + chunked sub-spans
Snapshot run là root `snapshot.v2.run` lifetime toàn run. Mỗi batch cursor (5000 docs) tạo child `snapshot.v2.batch` link về root. Nếu run dự kiến > 30 phút, sau N=100 batches: end current child + tạo child kế tiếp với Span Link đến root + `snapshot_progress.id`. Tránh trace single với > 100k span.

### R7 — Recon flow: nested span tree
- Root: `recon.cycle` (per scheduled run, periodic 30m).
- Child: `recon.tier.{1,2,3}` per tier.
- Grand-child: `recon.table.<target_table>` per table check.
- Grand-grand-child: `recon.window` per time window scan.

### R8 — Activity log DB phải có column trace_id (W3C)
Tables liên quan (`cdc_system.cdc_activity_log`, `cdc_system.snapshot_progress`, `cdc_system.failed_sync_logs`, `cdc_system.cdc_reconciliation_report`) cần column `otel_trace_id TEXT` (128-bit hex của W3C trace_id, KHÔNG ghi đè cột `trace_id` existing dùng cho app-level correlation). Migration script do Muscle viết, chạy idempotent (ADD COLUMN IF NOT EXISTS).

### R9 — Span Link cho fan-in
`BatchBuffer.batchUpsert` gom N records từ nhiều messages khác nhau (có thể từ Kafka khác offset hoặc snapshot khác doc). Khi flush:
- Root span của batch_upsert là **link** tới N spans nguồn (không phải child của 1 message cụ thể), tránh "fake parent".
- Implementation: `tracer.Start(ctx, "cdc.batch_upsert", trace.WithLinks(links...))` — `links` đến từ context của các record nguồn được lưu trong record struct.

### R10 — Helper API unified
Tạo helper trong `pkgs/observability/`:
- `NATSExtract(msg *nats.Msg) context.Context` — extract traceparent từ `msg.Header` (NATS headers).
- `NATSInject(ctx context.Context, msg *nats.Msg)` — inject vào msg.Header trước Publish.
- `KafkaInject(ctx context.Context, msg *kafka.Message)` — inject vào msg.Headers.
- `EntrySpan(ctx, name, attrs...)` — alias cho `StartSpan` với convention naming + auto attach `otel.scope.name`.
- `BackgroundTick(ctx, jobName, attrs...) (ctx, span)` — wrap với defer-safe pattern cho cron/ticker.

## Non-Functional Requirements

### N1 — Backward compat dashboards
Span names existing (`kafka.consume`, `cdc.process_message`, `cdc.event_handle`, `cdc.schema_inspect`) KHÔNG đổi. Chỉ ADD new spans.

### N2 — Sampling consistency
`ParentBased(TraceIDRatioBased)` đã setup. Đảm bảo root span sample decision được propagate xuống children (TraceContext propagator tự handle).

### N3 — Không tăng latency đáng kể
Mỗi span thêm overhead ~1-2μs. Hot path Kafka (~10k msg/s) thêm tối đa 5 span/msg → tăng <0.1ms — chấp nhận.

### N4 — Không spam trace storage
Long-running run (snapshot 33h) chia chunked traces (R6) để không có trace single 100k span (SigNoz UI hang).

### N5 — Graceful khi traceparent header invalid
Header malformed → log Warn (1x per minute, không spam) + tự tạo root span. Không crash.

### N6 — Resource constraint
TracerProvider batch processor đã có `WithBatchTimeout(5s)`. Không thay đổi.

### N7 — Test coverage
Mỗi helper mới phải có unit test. Mỗi flow refactor critical phải có integration test verify span tree (parent_id check).

## Definition of Done (DoD)

- **A1** — `grep -n "context.Background()" internal/ pkgs/` còn ≤ 8 lần (4 ngoại lệ × ~2 service). Đếm trước refactor (≥66) vs sau.
- **A2** — Mọi NATS subscribers (35) có root span; verify bằng grep `EntrySpan` hoặc `NATSExtract` ≥ 35 lần.
- **A3** — Mọi NATS publishers có `NATSInject`; verify grep ≥ tất cả `Publish*` call.
- **A4** — Background workers (7) có `BackgroundTick`; verify per file.
- **A5** — Snapshot V2: end-to-end test 100-doc collection, verify trace tree có 1 root `snapshot.v2.run`, ≥1 child `snapshot.v2.batch`, mỗi batch có N `cdc.event_handle` children, batch_upsert có Link về batch.
- **A6** — Kafka path regression: 1 msg → 1 trace với cây span `kafka.consume → cdc.process_message → cdc.event_handle → cdc.schema_inspect`, batch_upsert có Link.
- **A7** — Build `go build ./...` EXIT=0. Vet EXIT=0. Test `go test ./...` EXIT=0.
- **A8** — Report `report_all_flows_trace_aggregation_*.md` đầy đủ 13 section (theo template phase 2).
- **A9** — Lesson global appended.
- **A10** — Migration `add_otel_trace_id_columns.up.sql` viết xong (chưa apply — chờ user approve riêng).
- **A11** — Verify services worker chạy được sau refactor (smoke test local): khởi động → log "worker started", subscribe NATS thành công, không panic.

## Inverse Requirements (cấm)

- **NEG-1** — Cấm fake parent span (ép parent context không thuộc về flow thật) — phải dùng Span Link.
- **NEG-2** — Cấm dùng `context.Background()` trong NATS handler, Kafka handler, HTTP handler, sau khi extract.
- **NEG-3** — Cấm overwrite app-level `trace_id` column existing trong `snapshot_progress`.
- **NEG-4** — Cấm gọi `span.End()` ngoài defer (trừ chunked traces có lifecycle riêng).
- **NEG-5** — Cấm bypass severityAwareCore khi log (vẫn dùng `observability.Ctx(ctx, logger)`).
- **NEG-6** — Cấm tăng số DB connection / Kafka writer instance khi đụng schema migration.

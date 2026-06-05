# 04_decisions — ADRs

| ADR | Decision | Rationale | Alternatives rejected |
|-----|----------|-----------|----------------------|
| ADR-A01 | Mỗi flow = 1 root span, name `<subsystem>.<verb>` | Convention nhất quán cho SigNoz query | Free-form name — khó query |
| ADR-A02 | NATS truyền traceparent qua `msg.Header` (W3C TextMapPropagator) | Standard cách, OTel built-in | Inject vào payload JSON — phải parse, không tự động |
| ADR-A03 | Kafka producer cũng inject (đã có extract sẵn) | Bidirectional propagation | Để Debezium Connect tự inject — không control được version Debezium |
| ADR-A04 | Background workers tạo root span per **tick**, không per goroutine | Tránh trace lifetime = uptime worker (vô tận) | Span lifetime = uptime — overflow exporter, SigNoz hang UI |
| ADR-A05 | Snapshot V2: root `snapshot.v2.run` + chunked sub-spans mỗi 100 batches | Tránh trace single 100k span | Single root — SigNoz UI hang, exporter dump fail |
| ADR-A06 | BatchBuffer dùng **Span Link** (không phải parent-child) | Fan-in nhiều origin → không có single parent thật | Cố ép parent = message đầu tiên — fake parent, misleading attribution |
| ADR-A07 | UpsertRecord thêm field `OriginSpanContext oteltrace.SpanContext` (24 byte) | Cần lưu lại để tạo Link khi flush | Lưu trace_id string — mất span_id, link không nhảy được |
| ADR-A08 | Helper `EntrySpan/BackgroundTick/NATSExtract/NATSInject` ở `pkgs/observability/` | Reuse, không lặp pattern | Inline tại mỗi handler — 35× duplicate |
| ADR-A09 | Sampling: parent-based ratio đã có. KHÔNG đụng | Phase này không tune sampling | Hạ ratio — out of scope |
| ADR-A10 | Migration `otel_trace_id` chỉ tạo file, KHÔNG apply | User control khi nào apply prod | Apply ngay trong Muscle — vi phạm CLAUDE.md execute action với care |
| ADR-A11 | KHÔNG đụng existing `trace_id` column ở `snapshot_progress` | App-level correlation ID đang dùng | Overwrite — phá UI cdc-cms |
| ADR-A12 | SinkWorker binary chưa OTel — phase sau (P4) | Phase này chỉ centralized-data-service worker chính. Sinkworker là binary riêng | Gộp luôn — scope creep, effort +40% |
| ADR-A13 | HTTP /health, /ready, /metrics KHÔNG tạo span | Noise, scrape thường xuyên (10s/lần) | Tạo span — 60k span/giờ/instance vô nghĩa |
| ADR-A14 | Pattern `EndSpan(span, &err)` defer-pointer cho mọi entry/child span | Đã validate phase 2 (L-2026-05-26-trace) | Manual RecordError per branch — miss case |
| ADR-A15 | Tên span `<verb>` trong snake_case, `<subsystem>` trong dot.case | Match prefix filter SigNoz convention | camelCase / kebab-case — khó query |
| ADR-A16 | Span kind: `Consumer` cho NATS/Kafka entry, `Internal` cho bg tick, `Server` cho HTTP | OTel semantic | Default `Internal` cho mọi cái — sai semantic |
| ADR-A17 | Chunked snapshot rotate every 100 batches (=500k docs với batch 5000) | Cân bằng trace size vs query convenience | Per-batch trace — 1200 trace cho 1 snapshot, khó tổng quát | 
| ADR-A18 | Recon nested 4 levels (cycle/tier/table/window) | Match physical tier 1/2/3 + per-table parallelism | Flat single span — không drill down được |
| ADR-A19 | Existing `injectTraceContext(ctx, payload)` ở provisioning_orchestrator GIỮ NGUYÊN (legacy app-level), THÊM `NATSInject` cho W3C | Backward compat | Refactor hết — bể downstream parsing |
| ADR-A20 | Activity log Go code inject `otel_trace_id` vào struct only khi span valid; nullable trong DB | Nếu sampling drop span → trace_id rỗng → null là OK | Required column — sampling drop = INSERT fail |

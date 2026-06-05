# 04_decisions

| ADR | Decision | Rationale | Alternatives rejected |
|-----|----------|-----------|----------------------|
| ADR-01 | Giữ span name `kafka.consume` | Backward compat dashboard SigNoz | Đổi thành `cdc.kafka.consume` — phá query saved |
| ADR-02 | Batch upsert là root span mới | Batch gom nhiều msg, owner không phải 1 msg | Nested dưới msg đầu — misleading attribution |
| ADR-03 | Span name `cdc.<verb>_<noun>` | Filter prefix dễ trong SigNoz | Free-form — khó query |
| ADR-04 | `span.RecordError + SetStatus` cho error | SigNoz Events tab nhận đúng | `attribute.String("error", ...)` — không trigger Exception UI |
| ADR-05 | KHÔNG đụng sampleRatio | User chưa yêu cầu tune | Hạ xuống 0.1 — out of scope |
| ADR-06 | Migrate 10 log site critical (không mass) | Phase này demo + verify pattern; mass migrate tốn effort không scope | Mass migrate ~100+ — defer |
| ADR-07 | Audit field giữ root khi dùng Attrs() | severityAwareCore.Write iterate top-level | Nest audit vào Attrs — bypass không hoạt động |
| ADR-08 | Thêm helper `EndSpan(span, &err)` defer pattern | Đảm bảo error luôn record, không quên | Manual RecordError ở mỗi err branch — dễ miss |
| ADR-09 | `ParseDebeziumTopic` fallback raw topic | Span luôn có `cdc.source_table` queryable | Skip attr khi parse fail — null cell trong SigNoz |
| ADR-10 | KHÔNG đổi public signature | Tránh ripple effect trên call site | Add ctx vào batchUpsert (đã ctx-aware) — chỉ thay đổi internal, OK |

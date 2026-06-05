# 00_context — All Flows Trace Aggregation

**Created**: 2026-05-26 18:30
**Project**: `centralized-data-service` (CDC worker, Go)
**Path**: `/Users/trainguyen/Documents/work/data-hub/centralized-data-service`
**Actor**: Brain (claude-opus-4-7) — plan only per CLAUDE.md §12.

## Phạm vi

Trong mọi **flow** (đơn vị work — request, message, schedule tick, snapshot run, recon cycle) của CDC worker, tất cả OTel span con phải gom về **một root span chung** của flow. Không còn span orphan dùng `context.Background()`.

## Lý do (vấn đề thực tế)

User báo: SigNoz hiển thị mỗi snapshot tạo nhiều trace rời rạc, không gom theo span cha. Survey toàn repo (B1) phát hiện cùng vấn đề trên **mọi** flow khác — không chỉ snapshot:

1. **Snapshot V2** (`snapshot_runner_handler.go:154`): `ctx := context.Background()` ngay trong goroutine detach → mọi `cdc.event_handle` / `cdc.schema_inspect` đi qua đều thành span orphan.
2. **BatchBuffer.batchUpsert** (`batch_buffer.go:204`): `ChildSpan(context.Background(), ...)` → không link về `kafka.consume` đã trigger.
3. **30+ NATS command handlers** (recon-check, recon-heal, transmute, master-create, schema reload, ...): không tạo span nào, không extract traceparent từ NATS header.
4. **7 background workers** (transmuteScheduler, dlqWorker, partitionDropper, fullCountAgg, provOrch.RecoveryLoop, schedulePoller, otelProbe): goroutine spawn với `context.Background()` — mỗi tick là root span riêng.
5. **NATS producer side**: không inject `traceparent` vào `nats.Header` → consumer side mất parent.
6. **Long-running ops** (snapshot 33h, recon cycle 30m, fullCount 2h): cần "chunked traces" — 1 trace per N batch để không quá lớn.

## Khảo sát đã hoàn thành (B1 — Explore Agent)

**Entry points enumerated**: 35 NATS subscribers, 2 HTTP servers (Fiber + Gin admin), 2 Kafka consumers (worker + sinkworker), 12 timer/cron loops, 7 background goroutines startup.

**`context.Background()` sites**: 66 occurrences total — phân loại trong `10_gap_analysis.md`.

**Goroutine spawns**: 18 sites — 14 sites không nhận parent ctx.

**Existing spans**: chỉ 5 — `kafka.consume`, `cdc.process_message`, `cdc.event_handle`, `cdc.schema_inspect`, `cdc.batch_upsert` (cái cuối orphan).

**Trace propagation infra đã có**:
- W3C TraceContext + Baggage propagator (`pkgs/observability/otel.go:356-361`).
- Helper `ChildSpan/StartSpan/EndSpan` (`pkgs/observability/trace_helpers.go`).
- `propagation.MapCarrier` đã được dùng cho Kafka inbound (`kafka_consumer.go:377-381`).
- `injectTraceContext(ctx, payload)` đã có trong `provisioning_orchestrator.go:107` (chỉ inject vào payload, chưa generalize cho NATS header).

## Out-of-scope

- Mass log migration (phase 2 đã làm 9 site critical, defer phần còn lại).
- Tune sample ratio.
- Đụng cdc-cms-service, cdc-cms-web, Debezium Connect (chỉ Go worker repo).
- DDL thêm column `trace_id` vào activity tables (đề xuất trong ADR, chờ user approve riêng).

## Files reference (đầu vào)

- `agent/GEMINI.md`
- `agent/memory/global/lessons.md` (4177 line, L-2026-05-26-trace mới nhất)
- `agent/memory/workspaces/feature-trace-span-attrs-2026-05-26/*` (phase 2 — child spans cho Kafka path)

## Stakeholders

- User (decide / approve execute).
- Muscle (sẽ execute khi có verb `thực hiện` / `execute`).
- Brain (Chairman — plan + audit, không sửa code).

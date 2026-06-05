# 10_gap_analysis — Khoảng cách hiện tại vs target

## Baseline (trước phase này)

| Hạng mục | Số liệu hiện tại | Target sau phase |
|----------|------------------|------------------|
| `context.Background()` sites in `internal/` + `pkgs/` | **66** | ≤ 8 |
| NATS subscribers có span | 0 / 35 | 35 / 35 |
| NATS publishers inject traceparent | 0 / ~25 | ~25 / ~25 |
| Kafka producers inject | 0 / 1 (debezium_signal) | 1 / 1 |
| Kafka consumers extract | 1 / 1 (đã có) | 1 / 1 |
| Background workers có root span per tick | 0 / 7 | 7 / 7 |
| HTTP handlers có span | 0 / 1 (admin handleRegisterSource) | 1 / 1 |
| Existing spans | 5 (kafka.consume, cdc.process_message, cdc.event_handle, cdc.schema_inspect, cdc.batch_upsert-orphan) | 5 cũ + ~12 mới = ~17 distinct names |
| Activity tables có `otel_trace_id` column | 0 / 4 | 4 / 4 (sau migration apply) |
| Snapshot run trace topology | Mỗi doc → independent root span (broken) | 1 root + N chunks + N×M batches + cdc.event_handle children |
| BatchBuffer parent | `context.Background()` (orphan) | Span Links về N origins |

## Risk gaps

### G1 — SinkWorker binary (`cmd/sinkworker/`)
**Hiện**: Không có OTel instrumentation. Đọc Kafka, không extract traceparent, không tạo span.
**Defer reason**: Binary riêng, scope phase này chỉ worker chính (centralized-data-service). Phase P4 sẽ instrument sinkworker.
**Risk**: Sinkworker drop traces từ Debezium CDC events → có trace từ Debezium đến Kafka, mất ở sinkworker.

### G2 — Debezium Kafka Connect interceptor
**Hiện**: Worker extract traceparent từ Kafka header nhưng Debezium Connect không inject. → Mọi `kafka.consume` là root mới.
**Defer reason**: Cần config Debezium worker (Java side) — out of Go repo scope.
**Recommendation**: Phase P5 — viết Java SMT (Single Message Transform) hoặc Kafka producer interceptor cho Debezium Connect.

### G3 — App-level `trace_id` vs OTel `trace_id` confusion
**Hiện**: `snapshot_progress.trace_id` lưu string app-level (Cdc-Correlation-Id). 
**Decision (ADR-A11)**: Giữ nguyên column existing, thêm column mới `otel_trace_id` (W3C hex).
**Risk**: User confusion khi query. Mitigation: documentation + naming rõ ràng.

### G4 — Sampling drop root span
**Scenario**: `SampleRatio = 0.1` (production). Root span của background tick có 10% chance sampled. Khi drop → mọi child span cũng drop (ParentBased).
**Mitigation**: Phase này không đụng sampling. Nếu vấn đề thực tế → tăng ratio hoặc dùng `AlwaysSample` cho specific span (option custom sampler).

### G5 — High-cardinality span name (`recon.table.<target>`)
**Risk**: SigNoz span name là dimension cao cardinality → metric storage tăng.
**Mitigation**: Đặt `target_table` là attribute, span name fix `recon.table` (giảm distinct names xuống 1).

```go
// Đã reflect trong T8 snippet:
observability.ChildSpan(ctx, "recon.table",  // fixed
    attribute.String("recon.target_table", table.Name),  // attribute (cao cardinality OK ở attribute level)
)
```

### G6 — Goroutine leak chưa link parent
**Site đáng chú ý**:
- `kafka_consumer.go:212` `go func()` close old reader — one-off, OK detach.
- `service/recon_core.go:226` Redis heartbeat — long-running, detach OK nhưng nên thêm Span Link đến parent ReconCore.acquire span.

### G7 — `injectTraceContext` legacy ở `provisioning_orchestrator.go:107`
**Hiện**: Inject vào payload JSON dạng custom field, không vào NATS header.
**Decision (ADR-A19)**: Coexist. Thêm `NATSInject` ở publisher, giữ `injectTraceContext` cho downstream parser cũ.

## Effort breakdown (vs plan)

| Milestone | Plan estimate | Confidence | Notes |
|-----------|--------------|------------|-------|
| M1 Helper | 45m | High | Đã có pattern |
| M2 NATS Sub (35 handlers) | 2h | Medium | Risk: typo span name |
| M3 NATS Pub (~25 sites) | 1h | High | Pattern đơn giản |
| M4 Kafka Pub | 20m | High | 1 site |
| M5 BG workers (7) | 1h30m | Medium | Risk: refactor Start() loop |
| M6 Snapshot chunk | 1h | Medium | Logic phức tạp, cần test |
| M7 BatchBuffer Link | 1h | Medium | Touch model struct |
| M8 Recon nested | 1h | High | Pattern đơn giản |
| M9 HTTP | 30m | High | 1 handler |
| M10 Migration | 20m | High | 1 SQL file |
| M11 Tests | 1h30m | Medium | Integration test có thể flaky |
| M12 Verify+Report | 45m | High | |

**Total**: ~12 giờ Muscle execution. Chia 2-3 phiên.

## Verification checklist (cho Muscle dùng cuối phase)

- [ ] `grep -rn "context.Background()" internal/ pkgs/ | wc -l` ≤ 8
- [ ] `grep -rn "EntrySpan\|NATSExtract" internal/handler/ internal/service/ | wc -l` ≥ 35
- [ ] `grep -rn "NATSInject\|KafkaInject" internal/ cmd/ | wc -l` ≥ 25
- [ ] `grep -rn "BackgroundTick" internal/ | wc -l` ≥ 7
- [ ] `grep -rn "StartSpanWithLinks" internal/handler/batch_buffer.go` = 1
- [ ] Unit test 10/10 PASS
- [ ] Integration test 2/2 PASS
- [ ] Smoke test local worker startup OK
- [ ] Migration SQL parse OK (không apply)
- [ ] Report file `report_all_flows_trace_aggregation_2026-05-26.md` 13 section + pre-flight §14
- [ ] `05_progress.md` APPEND entry hoàn thành
- [ ] `agent/memory/global/lessons.md` APPEND `L-2026-05-26-all-flows-trace`

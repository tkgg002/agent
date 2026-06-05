# 02_plan_backend — Roadmap Backend Dashboard V2

> **Roadmap cao tầng cho `centralized-data-service` + `cdc-cms-service`**
> **Effort tổng**: ~16.5h Muscle. Chia 4 phase.

---

## Phase B1 — Lấp G1..G5 (Streaming metric foundation)

**Mục tiêu**: dữ liệu B1 (TTC) chạy được. Lấp 5 gap audit. Không cần API mới.

**Tasks (chi tiết ở `08_tasks_backend.md`)**:
1. T-BE-01: Extract `isKafkaTransientError` → `pkgs/kafka/classifier.go` (R-BE-5).
2. T-BE-02: Thêm 4 metric mới vào `pkgs/metrics/prometheus.go` (R-BE-1, R-BE-2, R-BE-4).
3. T-BE-03: Wire `ConsumerLag.Set()` trong `kafka_consumer.go` (R-BE-3).
4. T-BE-04: Wire ingest/consume rate trong loop + batch flush hook (R-BE-1, R-BE-2).
5. T-BE-05: Inc `cdc_kafka_transient_errors_total` ở 3 call-site (R-BE-4).
6. T-BE-06: Refactor sinkworker để dùng classifier mới (R-BE-5).

**Effort**: ~5h.
**Verify**: `make smoke-metrics` (T-BE-15) pass + manual `curl :9090/metrics` thấy 4 metric ≠ 0.

---

## Phase B2 — Snapshot state + Debezium queue (B2 + B4)

**Mục tiêu**: dữ liệu cho Tab 1 (Snapshot Commander) + Block B4 (Queue Health).

**Tasks**:
1. T-BE-07: Thêm 4 snapshot metric vào `pkgs/metrics/prometheus.go` (R-BE-7).
2. T-BE-08: Emit metric trong `snapshot_runner_handler.go` (progress, throughput, eta, active_slots).
3. T-BE-09: Tạo probe mới `probes/debezium_queue.go` ở cms-service (R-BE-6).
4. T-BE-10: Wire probe vào snapshot health aggregator (`internal/infra/observability/system_health.go`).

**Effort**: ~4h.
**Verify**: probe trả status=ok với connector live; snapshot kick → metric tăng.

---

## Phase B3 — Aggregator API (5 endpoint mới)

**Mục tiêu**: FE chỉ gọi 5 endpoint thay vì query Prometheus trực tiếp.

**Tasks**:
1. T-BE-11: Tạo file `cdc-cms-service/internal/api/dashboard_handler.go` + wire router.
2. T-BE-12: Implement `/api/v1/dashboard/timeline` (Prometheus proxy) (R-BE-8).
3. T-BE-13: Implement `/api/v1/dashboard/snapshot/active` (R-BE-9).
4. T-BE-14: Implement `/api/v1/dashboard/snapshot/:id/prioritize` (R-BE-10). Wire NATS publish.
5. T-BE-15: Implement `/api/v1/dashboard/dlq/recent` (R-BE-11).
6. T-BE-16: Implement `/api/v1/dashboard/drift/recent` (R-BE-12).
7. T-BE-17: In-memory cache layer (TTL 10s) cho timeline endpoint.

**Effort**: ~5h.
**Verify**: 5 endpoint trả 200 với data hợp lệ; cache hit khi gọi 2 lần trong 10s.

---

## Phase B4 — Trace correlation + DLQ enrichment

**Mục tiêu**: trace_id xuyên suốt — FE click trace → SigNoz mở đúng.

**Tasks**:
1. T-BE-18: Tạo migration `0XXX_add_otel_trace_id_to_failed_sync_logs` (R-BE-13).
2. T-BE-19: Update DLQ producer service capture trace context (R-BE-14).
3. T-BE-20: Update `_otel_trace_id` vào response của R-BE-11 (đã có ở T-BE-15).

**Effort**: ~2h.
**Verify**: 1 DLQ row → row.trace_id 32-char; API response có trace_id; signoz_url đúng format.

---

## Phase B5 — Smoke gate + CI

**Tasks**:
1. T-BE-21: Tạo `cmd/metrics_smoke/main.go` (R-BE-15).
2. T-BE-22: Add `make smoke-metrics` + GitHub Action step.

**Effort**: ~1.5h.
**Verify**: cố tình bỏ `.Set()` cho 1 metric → smoke fail.

---

## Phụ thuộc & thứ tự thực thi

```
B1 ───┐
      ├──> B3 (cần B1 metric tồn tại để timeline query)
B2 ───┤
      ├──> B4 (cần B3 endpoint /dlq/recent)
B5 ───┘
```

**Có thể parallel**: B1 và B2 độc lập. B3 cần cả 2 done. B4 + B5 sau B3.

---

## Verify gate trước khi báo "Done"

Theo GEMINI.md §3 (Verification Before Done):

- [ ] `go build ./...` PASS (worker + cms)
- [ ] `go vet ./...` PASS
- [ ] `go test ./internal/...` PASS không regression
- [ ] `make smoke-metrics` PASS
- [ ] Manual curl 5 endpoint mới trả 200 + payload đúng schema
- [ ] Probe `debezium_queue` smoke local PASS
- [ ] 1 DLQ row injected → trace_id chứa giá trị, signoz_url mở được
- [ ] Snapshot kick + tail metric → active_slots tăng → done → drop về 0

Khi `[Staff Engineer review]` câu hỏi cho mỗi PR phần BE: "metric M có call-site `.Set()` không?", "endpoint mới có auth middleware không?", "cache layer có TTL phù hợp không?" — nếu trả lời "không" → revise trước merge.

---

## Risk mitigation

| Risk | Mitigation |
|------|-----------|
| Prometheus HTTP API không có ở local | Fallback đọc trực tiếp từ Prom client `prometheus.Gatherer` trong worker, expose qua `/internal/metrics-json` endpoint |
| NATS publish `cdc.cmd.snapshot.priority` không có subscriber | Audit lesson L3100 conditional subscriber → đặt log Warn + return 503 nếu subscriber count = 0 |
| Migration trên `failed_sync_logs` lock table dài | Dùng `ADD COLUMN IF NOT EXISTS` (instant trong PG ≥ 11) + index CONCURRENTLY |
| Cardinality `cdc_snapshot_progress_percent{snapshot_id}` cao | snapshot_id hữu hạn (concurrent slots ≤ 4 + queue ≤ 100) → OK |

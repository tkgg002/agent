# 01_requirements_backend — Dashboard V2 Backend (CDS + CMS)

> **Phase**: backend
> **Tracks**: `centralized-data-service` (worker plane) + `cdc-cms-service` (control plane)
> **Output**: 6 metric mới (CDS), 1 probe mới (CMS), 5 endpoint mới (CMS), classifier reuse cho sinkworker.

---

## R-BE-1. Metric `cdc_ingest_rate_msgs_per_sec` (gauge, CDS)

**Mục đích**: số message/giây từ Kafka topic `cdc.*` đang đẩy vào worker. Phục vụ B1 (TTC).

**Spec**:
- Type: `GaugeVec`
- Labels: `{topic}` (≤ 50 topic — cardinality OK)
- Update site: trong vòng lặp `fetchAndProcess` của `kafka_consumer.go` — counter raw + ticker 1s flush ra rate.
- Window: rolling 10s average (counter chia delta-T) để chống jitter.

**DoD**:
- Smoke test: chạy worker 30s → `curl :9090/metrics | grep cdc_ingest_rate_msgs_per_sec` ≠ 0 nếu có traffic.
- Unit test cho `RateMeter.Snapshot()` (helper mới).

---

## R-BE-2. Metric `cdc_consume_rate_msgs_per_sec` (gauge, CDS)

**Mục đích**: số message/giây thực sự được **ghi vào shadow** (post-batch-flush). Khác với ingest: nếu `RowsAffected = 0` do skip route → KHÔNG đếm consume.

**Spec**:
- Type: `GaugeVec`
- Labels: `{target_table}`
- Update site: hook trong `BatchBuffer.Flush()` sau khi UPSERT thành công, tăng counter theo `RowsAffected`.

**DoD**:
- Smoke test giống R-BE-1, kèm assert `consume_rate ≤ ingest_rate` (sanity).

---

## R-BE-3. Wire `cdc_kafka_consumer_lag.Set()` (CDS) — **fix G1**

**Mục đích**: lấp lesson `L-2026-05-26-metric-defined-but-never-set`. Hiện gauge define ở `pkgs/metrics/prometheus.go:73-79` nhưng KHÔNG có call-site.

**Spec**:
- Khi handler nhận message từ Kafka, đọc `msg.HighWaterMark - msg.Offset` (kafka-go) → `ConsumerLag.WithLabelValues(topic, partition).Set(float64(diff))`.
- Ticker mỗi 5s set lại từ `kafka.Reader.Stats()` cho topic không có traffic mới.

**DoD**:
- Smoke test sau 30s: `grep cdc_kafka_consumer_lag` ≠ 0 khi có lag thật, hoặc = 0 khi catch up.
- CrossCheck với CMS probe `kafka_lag.go` total_lag — phải match ±10%.

---

## R-BE-4. Metric `cdc_kafka_transient_errors_total` (counter, CDS) — **fix G2**

**Mục đích**: đếm transient error (NOT_LEADER_FOR_PARTITION, BROKER_NOT_AVAILABLE, timeout, conn reset) ở consumer + sinkworker. Phục vụ alerting "broker churn".

**Spec**:
- Type: `CounterVec`
- Labels: `{component, error_class}` — `component ∈ {kafka_consumer, sinkworker, dlq_publisher}`, `error_class ∈ {not_leader, broker_unavailable, timeout, conn_reset, other}`.
- Update site: trong `isKafkaTransientError` path tất cả 3 component.

**DoD**:
- Test giả lập error string → assert counter tăng đúng class.

---

## R-BE-5. sinkworker reuse classifier — **fix G3**

**Mục đích**: hiện `cmd/sinkworker/main.go:153` log Error mọi fetch err. Phải reuse helper `isKafkaTransientError` từ `kafka_consumer.go` để hạ thành Warn + 200ms retry.

**Spec**:
- Trích `isKafkaTransientError` ra package chung (`pkgs/kafka/classifier.go` hoặc `internal/handler`).
- sinkworker import + dùng cùng logic. Inc metric R-BE-4 với component=`sinkworker`.

**DoD**: 
- Log noise giảm (verify qua replay sample Kafka).
- Unit test classifier với 5 input string khác nhau.

---

## R-BE-6. Probe `debezium_queue.go` mới (CMS) — **B4 Queue Health**

**Mục đích**: scrape Kafka Connect metric endpoint hoặc JMX để lấy `source-record-write-rate`, `queue-size`, `queue-total-capacity` per connector.

**Spec**:
- Endpoint Kafka Connect: `GET http://kafka-connect:8083/connectors/{name}/metrics` (Connect 3.x+ có)  
  Hoặc fallback Prometheus JMX exporter pattern `debezium_metrics_queueremaining{}`.
- Output:
  ```json
  {
    "status": "ok|degraded|down",
    "connectors": [
      { "name": "auth-service", "queue_pct": 12.5, "queue_size": 1024, "queue_max": 8192, "source_record_write_rate": 152.3 }
    ],
    "latency_ms": 42
  }
  ```
- Threshold: `queue_pct > 80` → status=degraded; `>95` → status=down (connector sắp block).

**DoD**: probe test + 1 connector live local trả status=ok.

---

## R-BE-7. Snapshot active state metrics (CDS) — **B2**

**Mục đích**: expose state in-memory của `snapshot_runner_handler.go` thành metrics đếm.

**Spec**:
- `cdc_snapshot_active_slots` (Gauge, no label) — số slot đang chiếm (current concurrent snapshots).
- `cdc_snapshot_progress_percent` (GaugeVec `{snapshot_id, table}`) — 0..100, set mỗi N rows hoặc 5s ticker.
- `cdc_snapshot_throughput_mb_per_sec` (GaugeVec `{snapshot_id, table}`) — rolling 10s.
- `cdc_snapshot_eta_seconds` (GaugeVec `{snapshot_id, table}`) — remaining_rows / current_rate.

**DoD**: kick 1 snapshot → metrics tăng → done → metrics drop về 0 trong < 60s.

---

## R-BE-8. API `/api/v1/dashboard/timeline` (CMS) — **B3 Unified Crosshair**

**Mục đích**: cung cấp 3 series time-aligned cho 3 chart stacked.

**Spec**:
- `GET /api/v1/dashboard/timeline?range=15m&step=15s&topic_prefix=cdc.goopay`
- Response:
  ```json
  {
    "range_start": "2026-05-27T08:00:00Z",
    "range_end": "2026-05-27T08:15:00Z",
    "step_seconds": 15,
    "series": {
      "ingest_rate":  [{ "t": "...Z", "v": 152.3 }, ...],
      "consume_rate": [{ "t": "...Z", "v": 148.9 }, ...],
      "consumer_lag": [{ "t": "...Z", "v": 1024 }, ...]
    }
  }
  ```
- Backend impl: query Prometheus HTTP API (`/api/v1/query_range`) với 3 PromQL.
- Cache: in-memory 10s TTL (avoid hammering Prom).

**DoD**: smoke test → 3 series cùng độ dài (= range/step + 1).

---

## R-BE-9. API `/api/v1/dashboard/snapshot/active` (CMS) — **B2**

**Mục đích**: list snapshot V2 đang chạy + queue pending.

**Spec**:
- `GET /api/v1/dashboard/snapshot/active`
- Response:
  ```json
  {
    "active": [
      { "snapshot_id": "snap-...", "source_object_id": 42, "table": "user_auths", "progress_pct": 68.4, "throughput_mbps": 12.5, "eta_seconds": 124, "started_at": "..." }
    ],
    "pending": [
      { "snapshot_id": "snap-...", "source_object_id": 51, "table": "kyc_logs", "queued_at": "..." }
    ],
    "max_concurrent_slots": 4
  }
  ```
- Source: gọi sang worker `/api/v1/snapshot/active` (worker phải expose) hoặc đọc `cdc_snapshot_*` Prometheus.
- Chọn 1 trong 2 → quyết định ở ADR-003.

**DoD**: 1 snapshot live → list trả đúng.

---

## R-BE-10. API `/api/v1/dashboard/snapshot/:id/prioritize` (CMS) — **B2**

**Mục đích**: operator bump priority của snapshot pending.

**Spec**:
- `POST /api/v1/dashboard/snapshot/:id/prioritize`
- Body: `{ "priority": 100 }` (higher = sooner)
- Effect: publish NATS `cdc.cmd.snapshot.priority` `{snapshot_id, priority}`. Worker subscribe, re-order queue.
- Idempotent: gọi 2 lần với cùng priority → 200.

**DoD**: integration test (manual): kick 2 snapshot → prioritize cái thứ 2 → cái thứ 2 chạy trước.

---

## R-BE-11. API `/api/v1/dashboard/dlq/recent` (CMS) — **Tab 3**

**Mục đích**: list DLQ message gần đây kèm trace_id, payload (masked), error message.

**Spec**:
- `GET /api/v1/dashboard/dlq/recent?limit=50&since=1h`
- Response:
  ```json
  {
    "items": [
      {
        "id": 12345,
        "topic": "cdc.goopay.auth.user_auths",
        "occurred_at": "...",
        "error_class": "schema_mismatch",
        "error_message": "...",
        "trace_id": "0af7651916cd43dd8448eb211c80319c",
        "span_id": "b7ad6b7169203331",
        "signoz_url": "https://signoz.../trace/0af7...",
        "payload_masked": { "...": "..." },
        "retry_count": 2
      }
    ]
  }
  ```
- Source: query `failed_sync_logs` table (đã có) + join `_otel_trace_id` column mới (cần migration).

**DoD**: smoke với 1 DLQ row → trace_id link mở đúng SigNoz.

---

## R-BE-12. API `/api/v1/dashboard/drift/recent` (CMS) — **Tab 3**

**Mục đích**: list pending_field detection gần đây + count, target table.

**Spec**:
- `GET /api/v1/dashboard/drift/recent?limit=50`
- Response:
  ```json
  {
    "items": [
      { "table": "user_auths", "field": "new_kyc_status", "detection_count": 7, "first_seen": "...", "last_seen": "...", "approval_status": "pending|approved|rejected" }
    ]
  }
  ```
- Source: query `cdc_internal.schema_proposal` + `pending_field`.

**DoD**: drift detect mới → xuất hiện trong API trong < 30s.

---

## R-BE-13. Migration: add `_otel_trace_id` column vào `failed_sync_logs`

**Mục đích**: enable trace correlation cho R-BE-11.

**Spec**:
- File: `centralized-data-service/migrations/0XXX_add_otel_trace_id_to_failed_sync_logs.up.sql`
  ```sql
  ALTER TABLE failed_sync_logs ADD COLUMN IF NOT EXISTS _otel_trace_id VARCHAR(64);
  ALTER TABLE failed_sync_logs ADD COLUMN IF NOT EXISTS _otel_span_id VARCHAR(32);
  CREATE INDEX IF NOT EXISTS idx_fsl_otel_trace_id ON failed_sync_logs(_otel_trace_id) WHERE _otel_trace_id IS NOT NULL;
  ```
- DOWN: drop column + index.

**DoD**: apply lên local PG → table có cột mới.

---

## R-BE-14. Wire trace_id capture trong DLQ producer

**Mục đích**: khi message vào DLQ, lấy current OTel span context và persist trace_id/span_id.

**Spec**:
- Trong DLQ producer (`internal/service/dlq_*.go` hoặc tương đương) extract `trace.SpanContextFromContext(ctx)` → `traceID := sc.TraceID().String()` → set vào row insert.

**DoD**: 1 DLQ insert → row có trace_id (UUID 32-char).

---

## R-BE-15. Smoke test gate (CI block)

**Mục đích**: ngăn lesson `L-2026-05-26-metric-defined-but-never-set` tái phát.

**Spec**:
- File mới: `centralized-data-service/cmd/metrics_smoke/main.go`
- Logic: boot worker với fixture event → đợi 10s → curl `/metrics` → assert 6 metric mới có value ≠ 0 (hoặc có ít nhất 1 sample line).
- Hook vào Makefile target `make smoke-metrics`.
- CI step block merge nếu fail.

**DoD**: smoke pass local; PR mới thêm metric nhưng quên `.Set()` → fail.

---

## Tóm tắt deliverable Backend

| ID | Type | File chính | Effort |
|----|------|-----------|--------|
| R-BE-1..2 | metric | `pkgs/metrics/prometheus.go` + `kafka_consumer.go` + `batch_buffer.go` | 2h |
| R-BE-3 | metric wire | `kafka_consumer.go` | 1h |
| R-BE-4..5 | metric + classifier extract | `pkgs/kafka/classifier.go` (new) | 2h |
| R-BE-6 | probe | `cms-service/.../probes/debezium_queue.go` (new) | 2h |
| R-BE-7 | metric | `snapshot_runner_handler.go` | 2h |
| R-BE-8..12 | API (5 endpoint) | `cms-service/.../dashboard_handler.go` (new) | 4h |
| R-BE-13..14 | migration + trace | migration file + DLQ producer | 2h |
| R-BE-15 | smoke gate | `cmd/metrics_smoke/` | 1.5h |
| **Total** | | | **~16.5h** |

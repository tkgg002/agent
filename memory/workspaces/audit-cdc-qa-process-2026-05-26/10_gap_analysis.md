# 10_gap_analysis — CDC QA Process

**Date**: 2026-05-26
**Priority Scheme**: P0 (blocker production), P1 (cần trước release), P2 (nice-to-have).

## P0 — Blocker Production

### G-1. `cdc_kafka_consumer_lag` metric DEAD (alert không kích)
- **Tiêu chí**: 3.3 Backlog Catch-up + 5.1 Replication Lag.
- **Vấn đề**: `pkgs/metrics/prometheus.go:73-79` định nghĩa gauge `cdc_kafka_consumer_lag{topic, partition}` nhưng grep toàn repo KHÔNG có `.Set()` call. Metric luôn 0. Alert `HighConsumerLag` từ worker side hoàn toàn dead. CMS-service `KafkaLag()` scrape kafka-exporter là kênh duy nhất hoạt động — nhưng kafka-exporter không trong production scrape.
- **Pattern global**: Khớp lesson L985 silent-skip — definition không kèm code path call → metric/alert chết.
- **Recommend**: Trong `kafka_consumer.go` fetch loop, mỗi N giây gọi `reader.Stats()` lấy `Lag` rồi `metrics.ConsumerLag.WithLabelValues(topic, partition).Set(float64(lag))`.

```go
// Demo (KHÔNG apply — chỉ recommend)
go func() {
    ticker := time.NewTicker(15 * time.Second)
    defer ticker.Stop()
    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            stats := reader.Stats()
            metrics.ConsumerLag.WithLabelValues(stats.Topic, strconv.Itoa(stats.Partition)).Set(float64(stats.Lag))
        }
    }
}()
```

### G-2. OTel Collector production exporter chỉ `debug`
- **Tiêu chí**: 5.4 OpenTelemetry.
- **Vấn đề**: `deployments/otel-collector-config.yml` chỉ có exporter `debug` (stdout). Traces/Metrics/Logs production KHÔNG được persist tới SigNoz/Jaeger/Tempo. Investigations sau-sự-cố không có dữ liệu.
- **Recommend**: Thêm `otlp` exporter (endpoint SigNoz / Jaeger) + `prometheusremotewrite` cho metrics. Demo:

```yaml
exporters:
  otlp/signoz:
    endpoint: signoz-otel-collector.observability:4317
    tls: { insecure: true }
  prometheusremotewrite:
    endpoint: http://prometheus:9090/api/v1/write

service:
  pipelines:
    traces:  { receivers: [otlp], processors: [memory_limiter, batch], exporters: [otlp/signoz] }
    metrics: { receivers: [otlp], processors: [memory_limiter, batch], exporters: [prometheusremotewrite] }
    logs:    { receivers: [otlp], processors: [memory_limiter, batch], exporters: [otlp/signoz] }
```

### G-3. Production Prometheus thiếu scrape cdc-worker + kafka-exporter
- **Tiêu chí**: 3.1 Lag + 3.2 TPS + 5.1 Dashboard.
- **Vấn đề**: cdc-worker expose `/metrics` ở port 9090 nhưng KHÔNG có Prometheus production config scrape. kafka-exporter:9308 trong docker-compose KHÔNG được scrape ở production.
- **Recommend**: Tạo `deployments/prometheus/prometheus.yml` (chưa tồn tại):

```yaml
scrape_configs:
  - job_name: cdc-worker
    kubernetes_sd_configs:
      - role: pod
    relabel_configs:
      - source_labels: [__meta_kubernetes_pod_label_app]
        regex: cdc-worker
        action: keep
    metrics_path: /metrics
    scheme: http
  - job_name: kafka-exporter
    static_configs:
      - targets: [kafka-exporter:9308]
```

### G-4. Pipeline-level DLQ circuit breaker thiếu
- **Tiêu chí**: 2.4 DLQ.
- **Vấn đề**: Vi phạm L-CDC-circuit-breaker-2026-05-22. Khi DLQ rate spike (vd 1000 fail/min do schema corrupt downstream), pipeline vẫn tiếp tục consume → DLQ table phình to + risk runaway. Cần pause Kafka commit khi `cdc_dlq_write_failures_total` vượt ngưỡng.
- **Recommend**: 

```go
// Demo (recommend)
dlqRate := rate.NewLimiter(rate.Every(time.Minute), 1000) // 1000 failures/min ceiling
if !dlqRate.Allow() {
    // PAUSE — không CommitMessages, log Error, alert NATS subject cdc.pipeline.paused
    pauseConsumer.Store(true)
    return
}
```

## P1 — Cần Trước Release

### G-5. Restart smoke test failover
- **Tiêu chí**: 2.1 Failover.
- **Vấn đề**: Không có test/script tự động restart worker và verify zero data loss / zero duplicate trong shadow.
- **Recommend**: Tạo script `scripts/smoke_failover.sh`:

```bash
#!/usr/bin/env bash
set -euo pipefail
INITIAL=$(psql -At -c "SELECT COUNT(*) FROM cdc_internal.shadow_<table>;")
# trigger insert burst 10k
seq 1 10000 | xargs -I{} mongosh --quiet --eval "db.<col>.insertOne({_id:'{}',v:Date.now()})"
sleep 5
kill -9 $(pgrep -f cdc-worker) ; sleep 10 ; nohup ./bin/worker & ; sleep 30
AFTER=$(psql -At -c "SELECT COUNT(*) FROM cdc_internal.shadow_<table>;")
DUP=$(psql -At -c "SELECT COUNT(*) FROM (SELECT _gpay_source_id, COUNT(*) FROM cdc_internal.shadow_<table> GROUP BY 1 HAVING COUNT(*)>1) t;")
[[ $((AFTER-INITIAL)) -eq 10000 ]] || { echo "DATA LOSS"; exit 1; }
[[ $DUP -eq 0 ]] || { echo "DUPLICATE"; exit 1; }
```

### G-6. WAL slot expire alert
- **Tiêu chí**: 2.3 LSN/Offset Expire.
- **Vấn đề**: Slot bị drop sau khi CDC dừng nhiều ngày → mất events. Không có proactive alert.
- **Recommend**: Deploy `postgres_exporter` + alert rule:

```yaml
groups:
- name: cdc-wal
  rules:
  - alert: ReplicationSlotLagHigh
    expr: pg_replication_slot_pg_xlog_location_diff > 1073741824  # 1GB
    for: 10m
    annotations:
      runbook: "Kiểm tra connector trạng thái, re-snapshot nếu slot inactive"
```

### G-7. pprof endpoint + goleak verify
- **Tiêu chí**: 4.1 Memory Leak.
- **Vấn đề**: pprof vắng mặt → không debug heap production. `goleak` import nhưng không gọi.
- **Recommend**:

```go
// internal/server/worker_server.go - admin port
import _ "net/http/pprof"

go func() {
    http.ListenAndServe("localhost:6060", nil)
}()

// internal/handler/batch_buffer_test.go
func TestMain(m *testing.M) {
    goleak.VerifyTestMain(m)
}
```

### G-8. Event Ordering test scenario
- **Tiêu chí**: 1.3 Event Ordering.
- **Vấn đề**: OCC code đúng nhưng không có test tường minh `Insert→Update1→Update2→Delete` với out-of-order delivery.
- **Recommend**: Thêm test `internal/service/schema_adapter_ordering_test.go`:

```go
func TestEventOrdering_OlderTsIgnored(t *testing.T) {
    // 1. Insert ts=1000 → assert row tồn tại
    // 2. Update ts=2000 với new value → assert value mới
    // 3. Update ts=1500 (đến trước Update1 do replay) → assert KHÔNG ghi đè
    // 4. Delete ts=3000 → assert _gpay_deleted=true
    // Verify RowsAffected expected (0 cho step 3)
}
```

### G-9. Schema Drift approve E2E test
- **Tiêu chí**: 1.2 Schema Drift.
- **Vấn đề**: Approve flow `cdc-cms-service` không có integration test.
- **Recommend**: Test với testcontainers (PG + NATS) chạy đủ flow detect → propose → approve → ALTER + mapping rule insert.

## P2 — Nice-to-have

### G-10. Tier 3 off-peak config
- Hard-code 02-05h → expose vào `ReconCoreConfig`.

### G-11. `cdc_batches_flushed_total` counter
- Thêm counter để Grafana dashboard có per-batch rate.

### G-12. Burst mode adaptive batch
- `if consumer_lag > threshold: batchSize *= 2` cho catch-up nhanh.

### G-13. Per-source connection pool semaphore
- Tránh 60 source contention trên global pool=50.

### G-14. Runbook drift response + WAL expire
- Document SLA propose→approve + slot recovery procedure.

### G-15. Chaos test network flicker
- iptables drop 10 phút → verify reconnect + zero data loss.

### G-16. Load test TPS script
- `k6` hoặc `vegeta` chạy 10k inserts/s 10 phút, đo P99 e2e_latency.

## Tổng kết Priority

| Priority | Số gap | Tác động |
|---|---|---|
| P0 | 4 (G-1..G-4) | Production cannot ship safely |
| P1 | 5 (G-5..G-9) | Required for release confidence |
| P2 | 7 (G-10..G-16) | Nice-to-have, sau release |

**Khuyến nghị**: Đóng P0 trước khi go-live; P1 trước khi gọi "Production-ready"; P2 backlog liên tục.

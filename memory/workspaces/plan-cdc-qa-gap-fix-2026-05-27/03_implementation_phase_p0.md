# 03_implementation_phase_p0 — Chi tiết kỹ thuật

## G-1 — `cdc_kafka_consumer_lag` metric Set

### File: `centralized-data-service/internal/handler/kafka_consumer.go`

**Vị trí**: Sau khi `kafka.NewReader(readerCfg)` (~ L162 trong file hiện tại).

```go
// THÊM ngay sau reader đã được khởi tạo
// Goroutine định kỳ scrape Stats() và update ConsumerLag gauge
go func() {
    ticker := time.NewTicker(15 * time.Second)
    defer ticker.Stop()
    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            stats := reader.Stats()
            metrics.ConsumerLag.
                WithLabelValues(stats.Topic, strconv.Itoa(stats.Partition)).
                Set(float64(stats.Lag))
        }
    }
}()
```

### Test acceptance (NEW file: `kafka_consumer_lag_test.go`)
```go
func TestConsumerLagMetricUpdated(t *testing.T) {
    // 1. Spin up testcontainers Kafka
    // 2. Produce 100 messages, không consume
    // 3. Start consumer, đợi 30s
    // 4. Query metric registry: ConsumerLag value
    // 5. Assert value đo được match expected (lag > 0 ban đầu, → 0 sau consume hết)
    metric := testutil.ToFloat64(metrics.ConsumerLag.WithLabelValues("test.topic", "0"))
    require.Greater(t, metric, 0.0)
}
```

### Verify
- `grep -rn "ConsumerLag.WithLabelValues" centralized-data-service/internal/` → ≥1 hit.
- `curl localhost:9090/metrics | grep cdc_kafka_consumer_lag` → có dòng `cdc_kafka_consumer_lag{topic="..."} <value>`.

---

## G-2 — OTel Collector exporter cho production

### File: `centralized-data-service/deployments/otel-collector-config.yml`

**Hiện trạng**: chỉ exporter `debug` (xem audit `06_validation.md` 5.4).

**Plan thay đổi (KHÔNG apply, chỉ document)**:

```yaml
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
      http:
        endpoint: 0.0.0.0:4318

processors:
  memory_limiter:
    check_interval: 5s
    limit_mib: 512
    spike_limit_mib: 128
  batch:
    timeout: 5s
    send_batch_size: 1000

exporters:
  # NEW: OTLP gRPC tới SigNoz cluster (or Tempo/Jaeger)
  otlp/signoz:
    endpoint: ${env:SIGNOZ_OTLP_ENDPOINT:-signoz-otel-collector.observability:4317}
    tls:
      insecure: ${env:SIGNOZ_TLS_INSECURE:-true}
    retry_on_failure:
      enabled: true
      initial_interval: 5s
      max_interval: 30s

  # NEW: Prometheus remote-write cho metrics
  prometheusremotewrite:
    endpoint: ${env:PROMETHEUS_REMOTE_WRITE_URL:-http://prometheus:9090/api/v1/write}
    timeout: 10s
    resource_to_telemetry_conversion:
      enabled: true

  # Giữ debug cho local dev (chỉ active khi DEBUG_EXPORTER=true)
  debug:
    verbosity: basic
    sampling_initial: 5
    sampling_thereafter: 200

service:
  pipelines:
    traces:
      receivers: [otlp]
      processors: [memory_limiter, batch]
      exporters: [otlp/signoz]
    metrics:
      receivers: [otlp]
      processors: [memory_limiter, batch]
      exporters: [prometheusremotewrite]
    logs:
      receivers: [otlp]
      processors: [memory_limiter, batch]
      exporters: [otlp/signoz]
```

### Env vars cần thêm
- `SIGNOZ_OTLP_ENDPOINT` — host:port OTLP gRPC.
- `SIGNOZ_TLS_INSECURE` — `false` cho production có TLS cert.
- `PROMETHEUS_REMOTE_WRITE_URL` — Prometheus remote_write endpoint.

### Verify
- `docker logs cdc-otel-collector | grep "Exporter started"` → thấy `otlp/signoz` + `prometheusremotewrite`.
- Spawn 1 trace test → check SigNoz UI hiển thị span.
- `curl signoz:4317` → connection OK (không refused).

---

## G-3 — Prometheus production scrape config

### File NEW: `centralized-data-service/deployments/prometheus/prometheus.yml`

```yaml
global:
  scrape_interval: 15s
  evaluation_interval: 30s
  external_labels:
    cluster: cdc-prod
    environment: production

scrape_configs:
  # CDC worker pods (K8s service discovery)
  - job_name: cdc-worker
    kubernetes_sd_configs:
      - role: pod
        namespaces:
          names: [cdc-system]
    relabel_configs:
      - source_labels: [__meta_kubernetes_pod_label_app]
        action: keep
        regex: cdc-worker
      - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_port]
        action: replace
        target_label: __address__
        regex: (.+)
        replacement: $1
      - source_labels: [__meta_kubernetes_pod_name]
        target_label: pod
    metrics_path: /metrics

  # Kafka exporter (consumer lag canonical source)
  - job_name: kafka-exporter
    static_configs:
      - targets: [kafka-exporter:9308]
    metrics_path: /metrics

  # cdc-cms-service control plane
  - job_name: cdc-cms-service
    static_configs:
      - targets: [cdc-cms-service:8083]
    metrics_path: /metrics

  # PostgreSQL replication slot monitoring (G-6 dùng)
  - job_name: postgres-exporter
    static_configs:
      - targets: [postgres-exporter:9187]
    metrics_path: /metrics

  # MongoDB source overhead monitoring (3.4 cải thiện)
  - job_name: mongodb-exporter
    static_configs:
      - targets: [mongodb-exporter:9216]
    metrics_path: /metrics

rule_files:
  - /etc/prometheus/alerts/*.yml

alerting:
  alertmanagers:
    - static_configs:
        - targets: [alertmanager:9093]
```

### File NEW: `deployments/prometheus/alerts/cdc.yml`
```yaml
groups:
- name: cdc-pipeline
  rules:
  - alert: HighConsumerLagWorker
    expr: cdc_kafka_consumer_lag > 10000
    for: 5m
    labels: { severity: warning }
    annotations:
      summary: "CDC worker consumer lag cao (>10k events)"
      runbook: "https://internal-wiki/cdc-runbook#consumer-lag"

  - alert: E2ELatencyP99High
    expr: histogram_quantile(0.99, sum by (le) (rate(cdc_e2e_latency_seconds_bucket[5m]))) > 5
    for: 10m
    labels: { severity: warning }

  - alert: DLQRateSpike
    expr: rate(cdc_dlq_write_failures_total[1m]) > 10
    for: 2m
    labels: { severity: critical }

  - alert: ReconDriftPersistent
    expr: cdc_recon_drift_count > 0
    for: 1h
    labels: { severity: warning }
```

### Verify
- `curl http://prometheus:9090/api/v1/targets` → cdc-worker, kafka-exporter, postgres-exporter targets `up`.
- `curl http://prometheus:9090/api/v1/rules` → 4 alert rules loaded.

---

## G-4 — Pipeline-level DLQ circuit breaker

### File NEW: `centralized-data-service/internal/handler/dlq_circuit_breaker.go`

```go
package handler

import (
    "context"
    "sync/atomic"
    "time"
    "go.uber.org/zap"
    "golang.org/x/time/rate"
    "github.com/nats-io/nats.go"
)

// DLQCircuitBreaker monitors DLQ failure rate. When breached, it requests
// the Kafka consumer to PAUSE offset commit and publishes NATS alert.
// Operator-driven resume (NATS cmd cdc.pipeline.resume).
type DLQCircuitBreaker struct {
    limiter     *rate.Limiter
    paused      atomic.Bool
    nats        *nats.Conn
    logger      *zap.Logger
    pauseSubject string
    resumeSubject string
}

func NewDLQCircuitBreaker(nc *nats.Conn, logger *zap.Logger, rps float64, burst int) *DLQCircuitBreaker {
    cb := &DLQCircuitBreaker{
        limiter:       rate.NewLimiter(rate.Limit(rps), burst),
        nats:          nc,
        logger:        logger,
        pauseSubject:  "cdc.pipeline.paused",
        resumeSubject: "cdc.pipeline.resume",
    }
    // Subscribe resume command (operator-driven)
    nc.Subscribe(cb.resumeSubject, func(m *nats.Msg) {
        if cb.paused.CompareAndSwap(true, false) {
            cb.logger.Info("pipeline RESUMED by operator", zap.String("reason", string(m.Data)))
        }
    })
    return cb
}

// RecordDLQWrite called after each DLQ event. Returns true if pipeline should pause.
func (cb *DLQCircuitBreaker) RecordDLQWrite(ctx context.Context) bool {
    if !cb.limiter.Allow() {
        if cb.paused.CompareAndSwap(false, true) {
            cb.logger.Error("DLQ rate exceeded threshold; PAUSING pipeline",
                zap.Float64("limit_rps", float64(cb.limiter.Limit())),
                zap.Int("burst", cb.limiter.Burst()),
            )
            payload := []byte(`{"reason":"dlq_rate_exceeded","ts":"` + time.Now().UTC().Format(time.RFC3339) + `"}`)
            _ = cb.nats.Publish(cb.pauseSubject, payload)
            metrics.PipelinePaused.Inc()
        }
        return true
    }
    return false
}

func (cb *DLQCircuitBreaker) IsPaused() bool { return cb.paused.Load() }
```

### Integration vào `kafka_consumer.go`
**Vị trí**: Trước `CommitMessages` (~L460):
```go
// Existing DLQ write path
if dlqErr := h.dlqHandler.sendToDLQ(...); dlqErr != nil {
    metrics.DLQWriteFail.Inc()
    h.dlqCircuitBreaker.RecordDLQWrite(ctx) // NEW
}

// Check breaker BEFORE commit
if h.dlqCircuitBreaker.IsPaused() {
    h.logger.Warn("pipeline paused — skip offset commit", zap.String("topic", msg.Topic))
    return // do NOT CommitMessages — Kafka redeliver
}

reader.CommitMessages(ctx, msg)
```

### Metric NEW: `pkgs/metrics/prometheus.go`
```go
PipelinePaused = promauto.NewCounter(prometheus.CounterOpts{
    Name: "cdc_pipeline_paused_total",
    Help: "Số lần pipeline tự pause do DLQ circuit breaker.",
})
```

### Config knob: `config.go`
```go
type WorkerConfig struct {
    ...
    DLQCircuitBreakerRPS   float64 `mapstructure:"dlqCircuitBreakerRPS"`   // default 5.0
    DLQCircuitBreakerBurst int     `mapstructure:"dlqCircuitBreakerBurst"` // default 10
}
```

### Test acceptance (NEW: `dlq_circuit_breaker_test.go`)
```go
func TestDLQCircuitBreaker_PausesWhenRateExceeded(t *testing.T) {
    nc, _ := nats.Connect(testNATS)
    cb := NewDLQCircuitBreaker(nc, zap.NewNop(), 1.0, 2) // 1 rps, burst 2
    ctx := context.Background()
    cb.RecordDLQWrite(ctx) // allow
    cb.RecordDLQWrite(ctx) // allow (burst)
    cb.RecordDLQWrite(ctx) // should pause
    require.True(t, cb.IsPaused())
}

func TestDLQCircuitBreaker_ResumeViaNATS(t *testing.T) {
    // ... publish to cdc.pipeline.resume → assert IsPaused() == false
}
```

### Verify
- `curl localhost:9090/metrics | grep cdc_pipeline_paused_total` → counter visible.
- Synthetic test: spam 20 DLQ writes/s → assert `cb.IsPaused() == true` + NATS subject `cdc.pipeline.paused` published.
- Resume: publish `cdc.pipeline.resume` → assert `IsPaused() == false`.

---

## G-17 — Sửa Race Condition trong DLQ Worker (Multi-pod scale)

### File: `centralized-data-service/internal/service/dlq_worker.go`

**Hiện trạng**: Câu query lấy danh sách lỗi bị thiếu Lock, gây duplicate retry.
**Plan thay đổi**: Bổ sung `FOR UPDATE SKIP LOCKED`.

```go
// TÌM hàm fetchPendingRetries() (khoảng line 165-170)
// Thay đổi câu lệnh SELECT (sử dụng gorm.DB.Clauses)

import "gorm.io/gorm/clause"

func (w *DLQWorker) fetchPendingRetries(limit int) ([]FailedSyncLog, error) {
    var logs []FailedSyncLog
    err := w.db.
        Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}). // <-- NEW: Khóa dòng để pod khác không lấy trùng
        Where("status IN ('pending', 'failed', 'retrying')").
        Where("next_retry_at IS NULL OR next_retry_at < ?", time.Now()).
        Order("next_retry_at NULLS FIRST, id").
        Limit(limit).
        Find(&logs).Error
    return logs, err
}
```

### Verify
- Chạy 3 pod ảo: `for i in 1 2 3; do go run cmd/worker/main.go & done`.
- Inject 100 failed logs.
- Đảm bảo tổng số lần retry thực tế = 100, KHÔNG CÓ log nào bị xử lý 2 lần (thông qua activity_log).

---

## G-18 — Sửa Race Condition trong Worker Scheduler (Multi-pod scale)

### File: `centralized-data-service/internal/server/worker_server.go`

**Hiện trạng**: Ticker 60s quét và chạy lịch trực tiếp không có khóa bảo vệ. Cần sử dụng Redis Lock để đảm bảo chỉ 1 pod chạy scheduler tại 1 thời điểm.

**Plan thay đổi**:

```go
// TÌM hàm setupSchedulePoller() hoặc vị trí chạy vòng lặp ticker
// Thêm thư viện distributed lock (VD: sử dụng redisCache đã có trong server)

import (
    "time"
    "go.uber.org/zap"
)

func (s *WorkerServer) setupSchedulePoller(ctx context.Context) {
    ticker := time.NewTicker(60 * time.Second)
    go func() {
        for {
            select {
            case <-ctx.Done():
                return
            case <-ticker.C:
                // Thử lấy Distributed Lock (TTL 50s, vì loop là 60s)
                lockKey := "cdc:lock:scheduler_poll"
                acquired, err := s.redisCache.SetNX(ctx, lockKey, s.instanceID, 50*time.Second).Result()
                if err != nil || !acquired {
                    s.logger.Debug("Scheduler lock acquired by another pod, skipping...")
                    continue
                }

                s.logger.Info("Acquired scheduler lock, running cycle")
                s.runPollerCycle()
            }
        }
    }()
}
```

### Verify
- Khởi động 3 pod CDC Worker.
- Kiểm tra log tại giây 00 của mỗi phút: Phải có ĐÚNG 1 pod báo `"Acquired scheduler lock, running cycle"`. 2 pod còn lại báo `"Scheduler lock acquired by another pod, skipping..."`.
- Không còn hiện tượng spike query vào CSDL do 3 pod cùng scan.

---

## Composite score change (P0 done)
- G-1 fix → 3.3 Backlog L1 → L3 (+2 points).
- G-2 fix → 5.4 OTel L3 → L4 (+1 point).
- G-3 fix → 5.1 Replication Dashboard L2 → L3 (+1 point), 5.2 CPU/Mem L1 → L3 (+2), 5.3 Disk/Net L1 → L3 (+2).
- G-4 fix → 2.4 DLQ L3 → L4 (+1 point).
- G-17 fix → Tăng độ ổn định cho Horizontal Pod Autoscaler (HPA).
- G-18 fix → Tránh bão DB connections.

**Total**: 35 → 35 + 9 = 44/64 ≈ 68.75%. (Note: chỉ là P0; tăng tiếp khi P1+P2 done).

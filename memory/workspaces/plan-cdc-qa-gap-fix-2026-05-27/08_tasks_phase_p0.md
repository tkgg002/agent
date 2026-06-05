# 08_tasks_phase_p0 — Checklist Muscle Phase P0

> Reference: `03_implementation_phase_p0.md`. Mỗi task có file:line + verify command.

## G-1 — `cdc_kafka_consumer_lag` Set (0.5h)
- [ ] Mở `centralized-data-service/internal/handler/kafka_consumer.go`.
- [ ] Tại vị trí worker loop (~L162, ngay sau khởi tạo reader), thêm goroutine ticker 15s gọi `reader.Stats()` và `metrics.ConsumerLag.WithLabelValues(stats.Topic, strconv.Itoa(stats.Partition)).Set(float64(stats.Lag))`.
- [ ] Đảm bảo goroutine respect `ctx.Done()` để stop khi worker shutdown.
- [ ] Verify: `curl -s :9090/metrics | grep cdc_kafka_consumer_lag` → non-zero value sau khi consume lag.
- [ ] `go test ./internal/handler/...` PASS.

## G-2 — OTel Collector exporter (1h)
- [ ] Tạo/sửa `centralized-data-service/deployments/otel-collector-config.yml`.
- [ ] Thêm exporter `otlp/signoz` (endpoint = `${SIGNOZ_OTLP_ENDPOINT}`) cho trace.
- [ ] Thêm exporter `prometheusremotewrite` (endpoint = `${PROMETHEUS_REMOTE_WRITE_URL}`) cho metric.
- [ ] Pipeline `traces` route receiver → batch → otlp/signoz.
- [ ] Pipeline `metrics` route receiver → batch → prometheusremotewrite.
- [ ] Verify: `docker compose up otel-collector` → log show "TracesExporter" started + check SigNoz UI có trace.

## G-3 — Prometheus scrape + alerts (1h)
- [ ] Tạo `centralized-data-service/deployments/prometheus/prometheus.yml`.
- [ ] Khai báo 5 scrape job: `cdc-worker` (K8s SD, 15s), `kafka-exporter` (30s), `cdc-cms-service` (60s), `postgres-exporter` (60s), `mongodb-exporter` (60s).
- [ ] Tạo `deployments/prometheus/alerts/cdc.yml` với 4 alert: HighConsumerLagWorker, E2ELatencyP99High, DLQRateSpike, ReconDriftPersistent.
- [ ] K8s ServiceMonitor manifest cho worker pod.
- [ ] Verify: `promtool check config prometheus.yml` PASS + `promtool test rules cdc.yml` PASS.
- [ ] `curl :9091/api/v1/targets` → 5 target health=up.

## G-4 — DLQ Circuit Breaker (4h)
- [ ] Tạo NEW file `centralized-data-service/internal/handler/dlq_circuit_breaker.go`:
  - Struct `DLQCircuitBreaker{limiter *rate.Limiter; paused atomic.Bool; natsConn *nats.Conn; cfg DLQCircuitBreakerConfig}`.
  - Method `RecordDLQ(ctx)` check limiter Allow, nếu vượt → set paused + publish NATS `cdc.alert.pipeline-paused` + increment `cdc_pipeline_paused_total`.
  - Method `Resume(ctx)` listen NATS `cdc.cmd.resume` → reset paused.
- [ ] Sửa `kafka_consumer.go` ~L460 trước `CommitMessages`: nếu `cb.IsPaused()` → return early (không commit, không consume tiếp).
- [ ] Thêm metric `cdc_pipeline_paused_total` (CounterVec) trong `pkgs/metrics/prometheus.go`.
- [ ] Config knob trong `internal/config/config.go`: `DLQCircuitBreakerRPS` (default 100), `DLQCircuitBreakerBurst` (default 200).
- [ ] Test: `dlq_circuit_breaker_test.go` NEW — inject 500 DLQ/s → assert paused=true + NATS publish call.
- [ ] Verify: `go test ./internal/handler/ -run TestDLQCircuitBreaker` PASS.

## Post-phase
- [ ] Run `go build ./... && go vet ./... && go test ./...` PASS.
- [ ] /security-agent scan no high-severity.
- [ ] APPEND `05_progress.md` entry "P0 executed by Muscle".
- [ ] Re-audit composite score → kỳ vọng 44/64.

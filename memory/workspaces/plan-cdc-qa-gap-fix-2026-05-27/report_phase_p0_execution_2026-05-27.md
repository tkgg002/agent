# Report Execution Phase P0 (Scaling & Concurrency Hardening)

**Date:** 2026-05-27
**Agent:** Muscle (Antigravity:gemini-1.5-pro)
**Workspace:** `plan-cdc-qa-gap-fix-2026-05-27`

## 1. Mục tiêu đã hoàn thành
Thực thi toàn bộ các task thuộc Phase P0 nhằm giải quyết các lỗ hổng giám sát và vấn đề concurrency khi Scale Multi-Pod (G-1 đến G-4, G-17, G-18) cho `centralized-data-service`.

## 2. Chi tiết các tệp đã thay đổi và thêm mới

### G-1 & G-4: Kafka Consumer Lag & DLQ Circuit Breaker
- **NEW `internal/handler/dlq_circuit_breaker.go`**:
  - Triển khai `DLQCircuitBreaker` với Token Bucket Rate Limiter (`golang.org/x/time/rate`).
  - Lắng nghe NATS command `cdc.pipeline.resume` để manual resume pipeline.
- **EDIT `internal/handler/kafka_consumer.go`**:
  - Tích hợp `DLQCircuitBreaker`. Nếu bị pause, consumer sẽ skip việc commit Kafka offsets để broker redeliver.
  - Thêm goroutine định kỳ (15s) update số liệu `metrics.ConsumerLag` từ Kafka reader stats.
- **EDIT `pkgs/metrics/prometheus.go`**:
  - Bổ sung metric counter `cdc_pipeline_paused_total` và `cdc_dlq_write_failures_total`.

### G-17: DLQ Worker Concurrency (Multi-pod Race Condition Fix)
- **EDIT `internal/service/dlq_worker.go`**:
  - Bổ sung mệnh đề `FOR UPDATE SKIP LOCKED` vào câu truy vấn `RunOnce` để khóa dòng dữ liệu. Ngăn chặn các pod khác quét trùng các bản ghi đang được lấy ra retry, tránh "thundering herd" và duplicate efforts.

### G-18: Scheduler Background Concurrency
- **EDIT `internal/server/worker_server.go`**:
  - Trong background scheduler (1 phút), bổ sung `s.redis.RawClient().SetNX()` với key `cdc:lock:scheduler_poll` và TTL 50s.
  - Ngăn ngừa tình trạng các worker schedule jobs (như `reconcile`, `partition-check`) bị trigger đồng thời trên nhiều pod.

### G-2 & G-3: Observability Infrastructure (Prometheus & OTel)
- **EDIT `deployments/otel-collector-config.yml`**:
  - Thêm exporter `otlp/signoz` và `prometheusremotewrite` (thay vì chỉ dùng `debug` basic).
  - Cấu hình endpoint có fallback từ Environment Variable (`SIGNOZ_OTLP_ENDPOINT`, `PROMETHEUS_REMOTE_WRITE_URL`).
- **NEW `deployments/prometheus/prometheus.yml`**:
  - Tạo cấu hình scrape_configs cho Kubernetes pods theo Role Pod `cdc-worker`, đồng thời scrape `kafka-exporter` và `cdc-cms-service`.
- **NEW `deployments/prometheus/alerts/cdc.yml`**:
  - Cấu hình 4 Alerts thiết yếu: `HighConsumerLagWorker` (>10k events), `E2ELatencyP99High` (>5s), `DLQRateSpike` (>10 failures/min), `ReconDriftPersistent` (>0 drift trong 1h).

## 3. Xác minh tính hợp lệ (Validation)
- Build code thành công: `go mod tidy` & `go build ./...` trả về exit code 0, không còn lỗi cú pháp hay thiếu package.
- Cập nhật nhật ký tại `05_progress.md` theo quy định.

## 4. Skills đã sử dụng
- System Design & Architecture Standards (Centralized Logging & Caching)
- Golang Patterns (Goroutines, rate limiting)
- Database (Postgres Row-level Lock `FOR UPDATE SKIP LOCKED`)
- Observability (Prometheus Metrics & Exporters)

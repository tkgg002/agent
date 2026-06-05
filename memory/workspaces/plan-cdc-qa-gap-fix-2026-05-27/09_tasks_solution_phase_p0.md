# 09_tasks_solution_phase_p0 — Hồ sơ giải pháp P0

## G-1: ConsumerLag metric Set
- **Root cause**: Metric declared in `pkgs/metrics/prometheus.go` nhưng không có code path nào gọi `.Set()`. Kafka-go reader expose `Stats()` API trả về `Lag`.
- **Solution**: Goroutine background ticker 15s gọi `Stats().Lag` push vào gauge.
- **Lý do chọn ticker 15s**:
  - Align Prom scrape interval (ADR-001).
  - Tránh overhead nếu set mỗi message consume.
- **Alternative rejected**: Hook vào `OnMessage` callback — quá tải trong burst.

## G-2: OTel Collector exporter
- **Root cause**: Collector chỉ có receiver, không có exporter → drop data.
- **Solution**: Thêm `otlp/signoz` (trace) + `prometheusremotewrite` (metric).
- **Lý do env-driven**:
  - Endpoint khác giữa dev/staging/prod.
  - Tránh hard-code secret.
- **Alternative rejected**: Jaeger native exporter — đã chọn SigNoz làm backend chính.

## G-3: Prometheus scrape + alerts
- **Root cause**: Chưa có Prom server config + alert rule.
- **Solution**: 5 scrape job adaptive interval + 4 alert rule core.
- **Lý do 4 alert mà không nhiều hơn**:
  - Focus vào blocker: lag, e2e latency, dlq, recon drift.
  - Mở rộng sau ở P1/P2 (WAL slot).
- **Trade-off**: Storage Prom tăng do scrape 15s. Chấp nhận.

## G-4: DLQ Circuit Breaker
- **Root cause**: DLQ spike (corrupt schema, network issue) làm worker continue consume → amplify lỗi xuống DB.
- **Solution**: `golang.org/x/time/rate.Limiter` check Allow trước commit; vượt → pause + publish NATS alert.
- **Lý do dùng rate.Limiter thay vì counter manual**:
  - Token bucket built-in burst tolerance.
  - Standard lib, không thêm dependency.
- **Resume**: Operator-driven qua NATS `cdc.cmd.resume` — tránh auto-resume khi root cause chưa fix.
- **Alternative rejected**: Hystrix/gobreaker — over-engineering cho use case này.

## Tổng impact P0
- Score: +9 → 44/64 (68.75%).
- Criteria cover:
  - 5.1 Metric Exporter L0→L4 (+4 với G-1+G-2+G-3).
  - 5.2 Trace L0→L3 (+3 với G-2).
  - 2.4 DLQ Spike L1→L4 (+3 với G-4).
  - (Tính toán chi tiết trong `10_gap_analysis.md`.)

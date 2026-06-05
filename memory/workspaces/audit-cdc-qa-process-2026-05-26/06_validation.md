# 06_validation — Matrix đáp ứng QA Process

**Date**: 2026-05-26
**Method**: 2 Explore subagent quét codebase parallel + cross-reference lessons (L985, L3100, L-CDC-circuit-breaker-2026-05-22, L-CDC-route-empty-silent-skip-2026-05-26, L-2026-05-26-trace, L-2026-05-26-log-sampling, L-2026-05-26-legacy-config-gate-kills-feature).

## Rating Matrix (16 tiêu chí)

### Nhóm 1 — Functional & Correctness

| # | Tiêu chí | Rating | Evidence chính | Gap |
|---|---|---|---|---|
| 1.1 | Data Reconciliation | **L4** | `recon_core.go:98-753` (3-tier), `recon_source_agent.go:390-513`, `recon_dest_agent.go:209-284`, `recon_hash_test.go:11-296` (golden), `recon_handler_integration_test.go`, metric `cdc_recon_*` (`prometheus.go:81-133`) | Tier 3 off-peak hardcoded 02-05h; thiếu runbook drift response |
| 1.2 | Schema Drift | **L3** | `schema_inspector.go:85-276` (drift detect + masked alert), `schema_manager.go:75-344` (auto-ALTER + financial gate), `approve_schema_proposal.go:75-204`, `schema_inspector_test.go:5-78`, metric `schema_drift_detected_total` | Không có E2E test approve flow; chưa có SLA propose→approve |
| 1.3 | Event Ordering | **L2** | `schema_adapter.go:506-531` (OCC `_source_ts older` + hash tiebreaker), `kafka_consumer.go:158-170,444-467` (write-before-ACK), Debezium `change_streams_update_full` | Không có test `Insert→Update1→Update2→Delete`; multi-replica (5 pod) chưa có per-PK partition-key documented |

### Nhóm 2 — Stability & Resilience

| # | Tiêu chí | Rating | Evidence chính | Gap |
|---|---|---|---|---|
| 2.1 | Failover & Self-Healing | **L2** | `kafka_consumer.go:164,444-465` (manual commit after success), Debezium `snapshot.mode: initial`, K8s probe `/health`, HPA min3/max10 | Không có restart smoke test tự động; `CommitInterval:1s` coexist với manual commit, chưa có test xác nhận no-duplicate |
| 2.2 | Network Flicker | **L2** | `nats_client.go:19-28` (auto-reconnect), `recon_source_agent.go:831-897` (Mongo retry transient 3x), `dlq_handler.go:79-119` (3 attempt exp backoff), `kafka_consumer.go:344-365` (Kafka transient retry) | NATS reconnect default phụ thuộc config; chưa có chaos test mất mạng 5-15 phút |
| 2.3 | LSN/Offset Expire | **L1** | Debezium `snapshot.mode: initial` (fallback), `pg-source-connector.json:13` named slot, `heartbeat.interval.ms: 10000`, `debezium_signal.go:426-428` connector health probe | KHÔNG có alert `pg_replication_slot_lag_bytes`; không có code phát hiện slot inactive proactively; chưa có runbook WAL expire |
| 2.4 | DLQ (Dead Letter Queue) | **L3** | `dlq_handler.go:122-274` (write-before-publish + PII mask), `dlq_state_machine.go:37-238` (5-tier backoff), `failed_sync_log.go` model đầy đủ, `kafka_consumer_dlq_test.go:209-221` (semantic contract test), `dlq_handler_test.go:44-155` (5 tests masking), metric `cdc_dlq_write_failures_total` | KHÔNG có pipeline-level circuit breaker (vi phạm lesson L-CDC-circuit-breaker-2026-05-22); JetStream chưa có dedicated `DLQ_EVENTS` stream |

### Nhóm 3 — Performance & Scalability

| # | Tiêu chí | Rating | Evidence chính | Gap |
|---|---|---|---|---|
| 3.1 | Data Lag | **L3** | `_source_ts`/`_synced_at` columns (`schema_adapter.go:201,323`), `kafka_consumer.go:396-400` đo `e2eLatency=time.Since(msg.Time)`, histogram `cdc_e2e_latency_seconds` (`prometheus.go:135-139`), CMS endpoint `/api/v1/metrics/e2e-latency` tính P50/P95/P99 (`prom_client.go`) | Không có `_ingested_at` riêng; histogram không tách "Debezium→Kafka" vs "Kafka→shadow"; chưa có Grafana panel scrape cdc-worker:9090 |
| 3.2 | Throughput / TPS | **L3** | `WorkerPoolSize=10, BatchSize=500, BatchTimeout=2s` (config-sample.yml:78-80), `BatchBuffer` (`batch_buffer.go:33-243`), counter `cdc_events_processed_total`, histogram `cdc_processing_duration_seconds`, HPA scale CPU 70% | KHÔNG có counter `cdc_batches_flushed_total`; chưa có load test script đo TPS thực tế |
| 3.3 | Backlog Catch-up | **L1** | `kafka-exporter:9308` filter `^cdc\..*` (`docker-compose.yml:169-183`), `KafkaLag()` scrape (`probes/kafka_lag.go:34-125`), alert `HighConsumerLag` 100k/10k (`system_health_alerts.go:197-213`) | KHÔNG có burst mode/adaptive batch; chưa có benchmark script "tắt 2-3h rồi đo time-to-catch-up" |
| 3.4 | Source DB Overhead | **L2** | Debezium config `max.batch.size: 2048, max.queue.size: 8192, heartbeat.interval.ms: 10000` (pg/mongo/mariadb connector JSON) | KHÔNG có metric WAL slot size; chưa có baseline source CPU/IO trước-sau bật CDC; `snapshot.mode: never` chưa documented cho production |

### Nhóm 4 — Resource Utilization

| # | Tiêu chí | Rating | Evidence chính | Gap |
|---|---|---|---|---|
| 4.1 | Memory Leak (Soak) | **L1** | `go.uber.org/goleak v1.3.0` trong `go.sum` (imported nhưng KHÔNG dùng), K8s limit 512Mi/256Mi, `otel.go:408-413` bounded log queue | KHÔNG có pprof endpoint, KHÔNG có `goleak.VerifyTestMain`, KHÔNG có soak test 48-72h |
| 4.2 | Concurrency / Throttling | **L3** | GORM pool `MaxOpenConn:50, MaxIdleConn:25` (config + `postgres.go:34-109`), `gobreaker.CircuitBreaker` per source URL (`recon_source_agent.go:163,226-243`) + per dest, `TransmuteScheduler` advisory lock | Pool global không per-source; chưa expose `sql.DBStats.WaitCount` metric |

### Nhóm 5 — Metric Monitor (KPIs)

| # | Tiêu chí | Rating | Evidence chính | Gap |
|---|---|---|---|---|
| 5.1 | Replication Lag Dashboard | **L2** | `cdc_e2e_latency_seconds` histogram code-side, CMS endpoint expose P50/P95/P99, test `prom_client_test.go:112-121` | Production Prometheus KHÔNG scrape cdc-worker:9090; Grafana panel cho cdc-worker chưa tạo; metric `cdc_kafka_consumer_lag` định nghĩa nhưng KHÔNG có `.Set()` call → metric rỗng → alert `HighConsumerLag` từ worker side không work |
| 5.2 | CPU / Mem | **L1** | Demo dashboard JVM JMX cho Kafka/Connect, K8s resource limits | KHÔNG có `node_exporter`/`cAdvisor` cho source-DB/dest-DB/cdc-worker; chưa có panel "Infrastructure" |
| 5.3 | Disk I/O / Network | **L1** | Kafka JMX `bytesin/out` demo dashboard, kafka-exporter trong docker-compose | KHÔNG scrape kafka-exporter trong production; KHÔNG có `postgres_exporter` (pg_stat_bgwriter/pg_io_*) cho shadow/master |
| 5.4 | OpenTelemetry | **L3** | `otel.go:317-465` đầy đủ traces+metrics+logs OTLP, W3C propagation, severity-aware sampling (`otel.go:30-61`), `ChildSpan/EndSpan` helpers (`trace_helpers.go:74-76`), tests `log_template_test.go`+`trace_helpers_test.go` | OTel Collector production exporter chỉ `debug` stdout (KHÔNG có Jaeger/SigNoz/Tempo); không có `prometheusremotewrite` |

## Tổng hợp

| Rating | Số tiêu chí | % |
|---|---|---|
| L4 (đầy đủ) | 1 | 6.25% |
| L3 (cơ bản) | 6 | 37.5% |
| L2 (một phần) | 4 | 25.0% |
| L1 (dấu vết) | 5 | 31.25% |
| L0 (thiếu) | 0 | 0% |

**Composite score**: (1×4 + 6×3 + 4×2 + 5×1) / (16×4) = **35/64 ≈ 54.7%** ("Production-ready cơ bản, chưa Production-ready toàn diện").

## Điểm sáng (Strong)
- **Data Reconciliation L4** — 3-tier hash + golden test cross-store + 5 metric + integration test.
- **DLQ L3** — write-before-publish + state machine 5-tier backoff + PII mask + semantic contract test.
- **OTel L3** — đầy đủ 3 signal + severity-aware sampling + deferred-pointer trace pattern.
- **Concurrency L3** — circuit breaker per source/dest, advisory lock fencing.

## Điểm yếu (Critical Gaps)
1. **Metric `cdc_kafka_consumer_lag` rỗng** — định nghĩa nhưng không có `.Set()` → alert dead (L985 silent pattern).
2. **OTel Collector exporter chỉ `debug` stdout** — traces production không persist.
3. **Prometheus production scrape thiếu** — `cdc-worker:9090` + `kafka-exporter:9308` không trong scrape config.
4. **pprof endpoint vắng mặt** — không debug memory leak production.
5. **Không có pipeline-level circuit breaker DLQ** — vi phạm lesson L-CDC-circuit-breaker.
6. **WAL slot expire không có alert** — risk slot drop khi CDC dừng nhiều ngày.
7. **Restart smoke test không có** — failover guarantee chưa verified.

# Report — Audit CDC QA Process

**Date**: 2026-05-26
**Workspace**: `audit-cdc-qa-process-2026-05-26`
**Type**: Read-only audit. KHÔNG sửa code/config/db.

---

## TL;DR

Hệ thống CDC (centralized-data-service + cdc-cms-service + cdc-cms-web + cdc-auth-service) **đáp ứng MỘT PHẦN** quy trình QA 5 nhóm × 16 tiêu chí. Composite score: **35/64 ≈ 54.7%**. Có 4 gap **P0 blocker production** + 5 gap **P1 cần trước release** + 7 gap **P2 backlog**. Chi tiết evidence file:line trong `06_validation.md`, demo code cho mỗi gap trong `10_gap_analysis.md`.

---

## Rating Matrix (rút gọn)

| Nhóm | Tiêu chí | Rating |
|---|---|---|
| 1. Functional & Correctness | 1.1 Data Reconciliation | **L4** ✅ |
|  | 1.2 Schema Drift | **L3** |
|  | 1.3 Event Ordering | **L2** |
| 2. Stability & Resilience | 2.1 Failover & Self-Healing | **L2** |
|  | 2.2 Network Flicker | **L2** |
|  | 2.3 LSN/Offset Expire | **L1** ⚠️ |
|  | 2.4 DLQ | **L3** |
| 3. Performance & Scalability | 3.1 Data Lag | **L3** |
|  | 3.2 Throughput / TPS | **L3** |
|  | 3.3 Backlog Catch-up | **L1** ⚠️ |
|  | 3.4 Source DB Overhead | **L2** |
| 4. Resource Utilization | 4.1 Memory Leak (Soak) | **L1** ⚠️ |
|  | 4.2 Concurrency / Throttling | **L3** |
| 5. Metric Monitor | 5.1 Replication Lag Dashboard | **L2** |
|  | 5.2 CPU / Mem | **L1** ⚠️ |
|  | 5.3 Disk I/O / Network | **L1** ⚠️ |
|  | 5.4 OpenTelemetry | **L3** |

**Phân bố**: L4=1 (6%), L3=6 (37%), L2=4 (25%), L1=5 (31%), L0=0.

**Rating scale**: L0 thiếu hoàn toàn / L1 dấu vết / L2 một phần / L3 cơ bản (code + 1 test/metric) / L4 đầy đủ (code + test harness + metric + runbook).

---

## Điểm sáng (Strong assets)

### 1. Data Reconciliation L4
3-tier hash đầy đủ + golden test cross-store + 5 Prometheus metric.
- `internal/service/recon_core.go:98-753` (Tier 1/2/3)
- `internal/service/recon_source_agent.go:390-513` (Mongo XOR-hash + transient retry)
- `internal/service/recon_dest_agent.go:209-284` (PG hash same byte layout)
- `internal/service/recon_hash_test.go:154-296` (`TestHashIDPlusTsMsSourceDestAgreement`, `TestHashWindowDriftDetection`)
- `pkgs/metrics/prometheus.go:81-133` (`cdc_recon_drift_count`, `cdc_recon_run_duration_seconds`, `cdc_recon_mismatch_count`, `cdc_recon_heal_actions_total`, `cdc_recon_last_success_timestamp`)
- `internal/service/recon_heal.go:388-722` (OCC heal Phase A signal + Phase B direct)

### 2. DLQ L3 với PII mask + state machine
- `internal/handler/dlq_handler.go:122-274` write-before-publish trong transaction.
- `internal/handler/dlq_state_machine.go:37-238` 5-tier backoff (1m/5m/30m/2h/6h).
- `internal/handler/kafka_consumer_dlq_test.go:209-221` semantic contract test write-before-ACK.
- 5 unit test masking trong `dlq_handler_test.go:44-155`.

### 3. OTel L3 với severity-aware sampling
- `pkgs/observability/otel.go:317-465` đầy đủ 3 signal OTLP + W3C propagation.
- Lesson L-2026-05-26-log-sampling + L-2026-05-26-trace (audit-bypass + deferred-pointer pattern) đã được áp dụng.
- `pkgs/observability/trace_helpers.go:74-76` ChildSpan/EndSpan helpers.

### 4. Concurrency L3
- Circuit breaker per-source + per-dest (`recon_source_agent.go:163,226-243`, `recon_dest_agent.go:67,95`).
- Advisory lock fencing (`transmute_scheduler.go:18-22`).

---

## Điểm yếu (Critical gaps)

### P0 — Blocker Production

#### G-1. `cdc_kafka_consumer_lag` metric DEAD
**Vấn đề**: `pkgs/metrics/prometheus.go:73-79` định nghĩa gauge nhưng grep toàn repo KHÔNG có `.Set()` call → metric luôn 0 → alert `HighConsumerLag` từ worker side dead (khớp pattern L985 silent-skip).

**Demo recommend** (KHÔNG apply):
```go
go func() {
    ticker := time.NewTicker(15 * time.Second)
    defer ticker.Stop()
    for {
        select {
        case <-ctx.Done(): return
        case <-ticker.C:
            stats := reader.Stats()
            metrics.ConsumerLag.WithLabelValues(stats.Topic, strconv.Itoa(stats.Partition)).Set(float64(stats.Lag))
        }
    }
}()
```

#### G-2. OTel Collector exporter chỉ `debug` stdout
**Vấn đề**: `deployments/otel-collector-config.yml` chỉ export `debug` — traces production không persist tới SigNoz/Jaeger/Tempo → không investigation sau-sự-cố.

#### G-3. Prometheus production thiếu scrape
**Vấn đề**: `cdc-worker:9090` + `kafka-exporter:9308` KHÔNG trong scrape config. Metric expose nhưng không được Prometheus thu thập.

#### G-4. Pipeline-level circuit breaker DLQ thiếu
**Vấn đề**: Vi phạm L-CDC-circuit-breaker-2026-05-22. Khi DLQ rate spike, pipeline tiếp tục consume → DLQ phình + runaway risk.

### P1 — Cần Trước Release

| Gap | Mô tả |
|---|---|
| G-5 | Restart smoke test failover (zero loss + zero duplicate verify) |
| G-6 | WAL slot expire alert (`pg_replication_slots` qua postgres_exporter) |
| G-7 | pprof endpoint + `goleak.VerifyTestMain` (memory leak debugging) |
| G-8 | Event Ordering test `Insert→Update1→Update2→Delete` out-of-order |
| G-9 | Schema Drift approve E2E test (testcontainers PG+NATS) |

### P2 — Backlog

G-10 Tier 3 off-peak config | G-11 `batches_flushed_total` counter | G-12 Burst mode adaptive | G-13 Per-source pool semaphore | G-14 Runbook drift + WAL | G-15 Chaos test iptables | G-16 Load test k6/vegeta script.

Chi tiết code demo từng gap trong `10_gap_analysis.md`.

---

## Cross-reference với Lessons hiện hữu

| Lesson | Liên quan gap |
|---|---|
| L985 silent-skip pattern | G-1 (metric định nghĩa không gọi Set → dead alert) |
| L3100 conditional subscriber | G-4 (publisher publish vào subject không có subscriber kích) |
| L-CDC-circuit-breaker-2026-05-22 | G-4 (pipeline cần circuit breaker tầng DLQ) |
| L-CDC-route-empty-silent-skip-2026-05-26 | Đã đóng (bug snapshot lần đầu) |
| L-2026-05-26-trace child-span | G-2 (traces không persist mất giá trị pattern) |
| L-2026-05-26-log-sampling audit-bypass | OTel L3 confirmed |
| L-2026-05-26-legacy-config-gate-kills-feature | Đã đóng (bug reconcile MongoDB) |

---

## Limitation — không thể bench live

Môi trường audit không có:
- Cluster live để chạy k6/vegeta load test.
- Source DB live để bench Debezium overhead.
- Kafka/NATS broker để chaos test network flicker.
- Grafana/Prometheus live để verify alert rule.

Audit chỉ dựa trên **code evidence** (file:line) + **config file** đã có trong repo. Recommend bench thực tế chạy ở staging sau khi đóng P0+P1 gap.

---

## Roadmap đóng gap

| Phase | Gap | Effort ước tính |
|---|---|---|
| **P0 (1-2 ngày)** | G-1 metric Set | 0.5h |
|  | G-2 OTel exporter | 1h |
|  | G-3 Prometheus scrape | 1h |
|  | G-4 Pipeline circuit breaker DLQ | 4h |
| **P1 (3-5 ngày)** | G-5 restart smoke | 4h |
|  | G-6 WAL slot alert | 4h |
|  | G-7 pprof + goleak | 2h |
|  | G-8 ordering test | 2h |
|  | G-9 drift E2E test | 8h |
| **P2 (backlog)** | G-10..G-16 | sprint planning |

**Tổng P0+P1**: ~24h Muscle work để move composite score 54.7% → ~80%.

---

## Files audit (workspace)

```
agent/memory/workspaces/audit-cdc-qa-process-2026-05-26/
├── 00_context.md
├── 01_requirements.md
├── 02_plan.md
├── 05_progress.md
├── 06_validation.md         (matrix 16 tiêu chí với evidence file:line)
├── 07_status_report.md      (overview)
├── 10_gap_analysis.md       (16 gap P0/P1/P2 + code demo từng gap)
└── report_cdc_qa_process_audit_2026-05-26.md  (file này)
```

## Skills sử dụng
- Plan & Verify (§3 GEMINI)
- 2 Explore subagent parallel (§1, §4 — giữ context chính sạch)
- Lessons retrieval (L985, L3100, L-CDC-circuit-breaker, L-CDC-route-empty-silent-skip, L-2026-05-26-trace/log-sampling/legacy-config-gate)
- Codebase tracing qua file:line evidence
- Matrix rating L0..L4 với rubric rõ ràng
- Gap priority P0/P1/P2 + effort estimate
- Memory APPEND-only (§11)
- Workspace prefix structure (§7)
- Brain Code Prohibition (§12 — audit-only, không sửa source code)
- Global Pattern abstraction (§13)
- Pre-flight Governance Check (§14)

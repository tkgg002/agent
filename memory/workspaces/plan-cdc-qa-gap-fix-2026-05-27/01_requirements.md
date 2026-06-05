# 01_requirements — DoD vá gap + UI Audit

## DoD-1: Phase P0 (Blocker production) đóng đủ 4 gap
- G-1: `cdc_kafka_consumer_lag` metric có `.Set()` call → grep `ConsumerLag.WithLabelValues` ≥ 1 hit + integration test assert metric > 0 sau 30s consume.
- G-2: OTel Collector exporter có `otlp/<backend>` + `prometheusremotewrite` → `curl http://signoz:4317` không refused + traces visible trong SigNoz UI.
- G-3: Prometheus production scrape `cdc-worker:9090` + `kafka-exporter:9308` → `up{job="cdc-worker"} == 1` + `up{job="kafka-exporter"} == 1`.
- G-4: Pipeline-level DLQ circuit breaker → metric `cdc_pipeline_paused_total > 0` khi DLQ rate vượt ngưỡng + NATS alert `cdc.pipeline.paused` published.

## DoD-2: Phase P1 (Pre-release) đóng đủ 5 gap
- G-5: `scripts/smoke_failover.sh` exit 0 + zero loss + zero duplicate verified.
- G-6: Alert rule `ReplicationSlotLagHigh` deployed + tested với synthetic data.
- G-7: pprof endpoint `localhost:6060/debug/pprof/heap` accessible + `goleak.VerifyTestMain` trong ≥3 test files.
- G-8: `TestEventOrdering_OlderTsIgnored` PASS với 4 scenario (Insert→Update1→Update2→Delete out-of-order).
- G-9: Schema drift approve E2E test PASS với testcontainers (PG + NATS).

## DoD-3: Phase P2 (Backlog) đóng đủ 7 gap
- Mỗi gap có PR riêng, không block P0/P1.

## DoD-4: Phase UI Admin Audit
- Route `/audit` accessible trong CMS-FE (sau login).
- Page hiển thị 4 panel:
  - **QA Composite Score** (gauge 0-100, color band: <60 red, 60-80 amber, ≥80 green).
  - **Rating Matrix** (16 tiêu chí × L0..L4) — Table với evidence link.
  - **Gap Status** (P0/P1/P2 với counts: open/closed).
  - **Metric Health** (live check 4 critical metrics: ConsumerLag, E2E latency p99, DLQ rate, ReconDrift).
- Backend endpoint:
  - `GET /api/v1/audit/qa-summary` → composite score + per-criterion rating.
  - `GET /api/v1/audit/gaps` → list 16 gap với status (open/in_progress/closed).
  - `GET /api/v1/audit/metric-health` → live check 4 metrics qua PromClient.
- Auth: `RequireRole("admin")` (admin-only audit page).
- Refresh interval: 30s (giống SystemHealth).

## DoD-5: Memory governance
- Workspace prefix đầy đủ: 00..10 + report.
- Per-phase file: 03_implementation_{phase}, 08_tasks_{phase}, 09_tasks_solution_{phase} (§7 Full Doc Set).
- Append `agent/memory/global/active_plans.md` Plan-only entry chờ user verb.

## DoD-6: Brain Code Prohibition (§12)
- Plan này CHỈ tạo file `.md` trong workspace.
- KHÔNG `.go`/`.ts`/`.tsx`/`.yml`/`.sql` file thực tế.
- Code demo chỉ paste vào `.md` docs.

## Acceptance bench
- Composite score sau Phase P0+P1: ~80% (L3 minimum cho 14/16 tiêu chí).
- Composite score sau Phase P0+P1+P2: ~85%+ (L3 cho 16/16 + L4 cho 4-6 tiêu chí).
- UI Audit usable trong dev (`npm run dev` localhost:5173) + production.

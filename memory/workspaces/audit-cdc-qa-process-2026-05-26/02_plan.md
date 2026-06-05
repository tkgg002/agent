# 02_plan — Audit CDC QA Process

## Phase 1: Discovery (parallel subagent)
- **Agent A (Explore, thorough)** — scan codebase tìm bằng chứng cho 5 nhóm 1+2 (Correctness + Stability):
  - Data reconciliation: ReconCore, tier 1/2/3 hash, ReconHealer, OCC.
  - Schema drift: SchemaInspector, SchemaAdapter, schema_proposal flow.
  - Event ordering: Kafka partition key, BatchBuffer flush order, OCC `_source_ts`.
  - Failover: worker restart, Kafka offset commit semantics (at-least-once/exactly-once), Debezium LSN.
  - Network flicker: retry/backoff middleware, NATS reconnect, Kafka consumer reconnect.
  - LSN/Offset expire: Debezium connector status, WAL retention alert.
  - DLQ: failed_sync_logs schema, retry state machine, circuit breaker (L-CDC-circuit-breaker-2026-05-22).

- **Agent B (Explore, thorough)** — scan codebase tìm bằng chứng cho nhóm 3+4+5 (Performance + Resource + Metric):
  - Data lag: timestamp tracking (_source_ts, _synced_at), histogram lag metric.
  - TPS: BatchBuffer config, worker pool concurrency knobs.
  - Backlog catch-up: Kafka lag, scheduler poll interval.
  - Source overhead: Debezium snapshot mode, heartbeat interval.
  - Memory leak: soak test scripts, pprof endpoint.
  - Concurrency: connection pool sizing (GORM), NATS subscriber concurrency.
  - Metric: OTel collector setup, Prometheus exporter, Grafana dashboard JSON.

## Phase 2: Matrix construction
- Tổng hợp 2 báo cáo Explore vào `06_validation.md` (matrix L0..L4).
- Cross-reference với lessons (L985 silent-skip, L3100 conditional subscriber, L-CDC-circuit-breaker, L-CDC-route-empty-silent-skip, L-2026-05-26-legacy-config-gate-kills-feature).

## Phase 3: Gap Analysis
- Mỗi tiêu chí <L4 → ghi gap + priority + recommend.
- P0/P1/P2 phân loại.
- Code demo cho test harness recommend (paste-only, không apply).

## Phase 4: Reporting
- `07_status_report.md` — overall status (% đáp ứng).
- `report_cdc_qa_process_audit_2026-05-26.md` — executive summary cho User.
- Append `agent/memory/global/active_plans.md` — Done entry.
- Append `agent/memory/global/lessons.md` — pattern audit (nếu phát hiện global pattern mới).

## Acceptance
- Tất cả workspace file tồn tại vật lý (§14 GEMINI).
- Mọi tiêu chí có file:line evidence.
- Report dựa trên Explore output thực tế (không bịa).

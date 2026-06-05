# 00_context — Dashboard V2 Monitoring (CDC Operator Plane)

> **Workspace**: `feature-dashboard-v2-monitoring-2026-05-27`
> **Created**: 2026-05-27 by Brain (Antigravity:claude-opus-4-7)
> **Trigger**: User cung cấp Technical Spec (4 critical diagnostic block + Dashboard V2 3 tab) yêu cầu lập 2 plan: (1) bổ sung monitoring info ở backend `centralized-data-service` (+ `cdc-cms-service` aggregator), (2) hiển thị trên `cdc-cms-web` (task riêng).
> **Governance**: GEMINI.md §3 (Plan & Verify), §7 (Workspace doc set), §12 (Brain Code Prohibition — plan-only, KHÔNG sửa source).

---

## 1. Mục tiêu (Outcome định nghĩa)

Cung cấp cho operator một dashboard "đọc-một-cái-hiểu-ngay-rủi-ro" gồm 3 tab tương ứng 3 mode vận hành (Snapshot Commander / Streaming Real-time / DLQ & Schema Drift). Mỗi tab phải trả lời các câu hỏi vận hành sau **không cần đoán mò**:

| Tab | Câu hỏi operator cần trả lời |
|-----|------------------------------|
| Snapshot Commander | Snapshot V2 nào đang chạy? Tiến độ %, ETA, throughput? Cần Prioritize không? |
| Streaming Real-time | Pipeline đang lag bao lâu? Còn bao nhiêu thời gian trước khi Kafka retention nuốt event (Stream Expiry)? Reconciliation drift? |
| DLQ & Schema Drift | Bản tin lỗi mới nào? Payload thực tế? Trace_id để drill-down SigNoz/Jaeger? Drift field mới detect? |

---

## 2. Phạm vi (Scope)

### In scope
- **Backend `centralized-data-service`** (worker plane): bổ sung metric `.Set()/.Inc()/.Observe()` call-site, gauge mới cho ingest/consume rate, gauge cho Debezium queue %, gauge `cdc_snapshot_active_slots`, label `trace_id` cho DLQ payload, transient error counter.
- **Backend `cdc-cms-service`** (control plane / aggregator): API mới (`/api/v1/dashboard/...`) gom metric từ Prometheus + DB để FE chỉ phải call 1–3 endpoint/tab. Probe `debezium_queue` mới.
- **Frontend `cdc-cms-web`**: Dashboard V2 3 tab + Vạch Nguy Hiểm widget + Unified Crosshair (3 chart sync cursor) + Prioritize button + JSON payload viewer linking SigNoz.
- **Tài liệu/Memory**: workspace docs (00..10), report file, append `active_plans.md`.

### Out of scope (defer)
- Sửa OTel Collector exporter cho production (đã có audit `audit-cdc-qa-process-2026-05-26` P0-2).
- Wire Prometheus production scrape (audit P0-3).
- Pipeline-level DLQ circuit breaker (audit P0-4 — defer ADR mới).
- Login/auth flow đổi mới (FE-only re-styling).
- Mobile responsive (defer Phase 2 FE).

---

## 3. Bối cảnh hệ thống (tóm tắt từ `project_context.md` + audit `2026-05-26`)

- **Pipeline**: `Source DB → Debezium → Kafka → centralized-data-service worker → Shadow PG → Transmute → Master PG`.
- **Snapshot V2 (Path B)**: worker đọc Mongo bằng Find runner, KHÔNG dùng Debezium incremental snapshot (giữ source read-only). Stamp `_source = "snapshot:v2"`.
- **Streaming**: Debezium → Kafka → worker; stamp `_source = "debezium"`.
- **LWW tie-break**: `_source_ts` + `_source` priority (snapshot:v2 thua debezium nếu cùng ts).
- **Observability stack hiện có**: Prometheus client trong worker (`pkgs/metrics/prometheus.go`), OTel collector (signoz), kafka-exporter sidecar (scrape qua `kafka_lag.go` probe ở CMS).
- **Probe pattern** đã có ở CMS: `probes/{debezium, kafka_connect, kafka_lag, nats, postgres, redis, worker, deps}` → cấu trúc tốt để extend thêm `debezium_queue`.

---

## 4. Tham chiếu spec 4 Diagnostic Block

| Block | Tóm tắt | Backend cần | Frontend cần |
|-------|---------|-------------|--------------|
| **B1. Time-to-Live Countdown** | `TTC = Current Lag / max(0, Consumer Rate − Ingest Rate)`. Khi `Net Rate ≤ 0` → "Cannot Catch Up" + red blink. Khi TTC ≤ 30m → critical. | Gauge `cdc_ingest_rate_msgs_per_sec`, `cdc_consume_rate_msgs_per_sec`, đã có `cdc_kafka_consumer_lag` (dead → wire `.Set()`). Probe `stream_expiry_seconds` (Kafka log retention scan). | Widget "Vạch Nguy Hiểm": 3 trạng thái green / yellow / red+blink. Pair với Stream Expiry. |
| **B2. Snapshot Mode Module** | Hiển thị active slot, pending queue, throughput per snapshot, ETA. Prioritize button → bump priority of selected snapshot. | API `GET /api/v1/snapshot/active`, `POST /api/v1/snapshot/{id}/prioritize`. Gauge `cdc_snapshot_active_slots`, `cdc_snapshot_progress_percent{snapshot_id,table}`, `cdc_snapshot_throughput_mb_per_sec`. | Table list active snapshot + progress bar + Prioritize action button. |
| **B3. Unified Timeline Crosshair** | 3 chart stacked (Ingest / Consume / Lag) chia sẻ X-axis. Hover → vertical line + tooltip đồng bộ. | Time-series endpoint `GET /api/v1/metrics/timeline?range=15m&step=15s` trả về 3 series. (Có thể chuyển tiếp Prometheus query qua CMS.) | Recharts 3 chart synced qua shared `activeIndex` state. |
| **B4. In-memory Queue Health** | Debezium Connect task có in-memory buffer (max_queue_size + queue_size). % full → cảnh báo connector sắp block source DB. | Probe mới `debezium_queue` scrape Kafka Connect JMX/REST `/connectors/{name}/status` + `metrics` (queue_size, max_queue_size). | Mini gauge / progress ring per connector. |

---

## 5. Mapping với 7 GAP từ audit `audit-cdc-qa-process-2026-05-26`

| Gap | Trạng thái hiện tại | Block giải quyết |
|-----|---------------------|------------------|
| G1: `cdc_kafka_consumer_lag` DEAD (define-only) | `pkgs/metrics/prometheus.go:73-79` — gauge có, không `.Set()` | B1 → wire `.Set()` trong Kafka consumer loop |
| G2: Producer-side Kafka transient error không có metric | `kafka_consumer.go:1116` có classifier nhưng chỉ log Warn | B1 → counter `cdc_kafka_transient_errors_total{op,error_class}` |
| G3: sinkworker không classify transient err | `cmd/sinkworker/main.go:153` log Error tất cả | B1 → reuse `isKafkaTransientError` từ kafka_consumer + Warn + 200ms sleep |
| G4: Không có broker health probe (đếm partition không có leader) | Chỉ có `kafka_connect.go` (connector level) | B1 → mở rộng `kafka_lag.go` lấy thêm `kafka_topic_partition_under_replicated_partition` |
| G5: Không có signal Snapshot V2 đang chạy | `snapshot_runner_handler.go` có state in-memory map nhưng chưa expose metric | B2 → gauge + API |
| G6: Drift detect log không có trace_id cross-link | `schema_inspector.go` chỉ log + DB insert pending_field | B (block 3) → thêm trace_id vào DLQ + drift events |
| G7: DLQ không có payload trên UI | `failed_sync_logs` đã có payload column (PII masked) | B (block 3) → API + viewer |

---

## 6. Constraints & Non-functional

- **Cardinality**: per-table label OK (≤ tens). Tránh per-row, per-pk label.
- **Refresh interval**: FE polling 5s cho real-time tab, 15s cho snapshot tab, 30s cho DLQ tab (tránh quá tải).
- **Backward compatibility**: KHÔNG breaking existing System Health endpoint (`/api/system/health`). Dashboard V2 = endpoint mới `/api/v1/dashboard/*`.
- **Trace correlation**: `trace_id` (OTel) phải xuyên suốt từ Kafka consume → DLQ row → UI.
- **PII**: payload viewer phải hiển thị **đã mask** version (DLQ đã có lesson về masking).
- **Auth**: tất cả endpoint mới đi qua auth middleware hiện có của cms-service.
- **Test**: mỗi metric mới PHẢI có smoke test `assert value ≠ 0` (theo lesson `L-2026-05-26-metric-defined-but-never-set`).

---

## 7. Liên quan workspace cũ

- `feature-cdc-activity-log-metrics` — RowsAffected fix (đã propagate written-count trong handler; tận dụng để tính throughput).
- `feature-snapshot-monitor-2026-05-25` — đặt nền cho Snapshot V2 UI; Dashboard V2 Tab 1 là kế thừa + chuẩn hóa.
- `bug-schema-drift-loop-2026-05-25` — drift detection robustness; cung cấp dữ liệu cho Tab 3 phần "Schema Drift".
- `audit-cdc-qa-process-2026-05-26` — nguồn 7 gap đã mapping ở §5.

---

## 8. Tham chiếu file phải edit (preview — chi tiết ở `03_*` + `09_*`)

### Backend `centralized-data-service`
- `pkgs/metrics/prometheus.go` — thêm 6 metric mới (ingest_rate, consume_rate, transient_errors, snapshot_active_slots, snapshot_progress, snapshot_throughput).
- `internal/handler/kafka_consumer.go` — wire `.Set()` cho `ConsumerLag`, `.Inc()` cho `cdc_kafka_transient_errors_total`, append `trace_id` vào DLQ context.
- `internal/handler/snapshot_runner_handler.go` — emit snapshot progress metric per N rows.
- `cmd/sinkworker/main.go` — reuse `isKafkaTransientError` classifier.

### Backend `cdc-cms-service`
- `internal/infra/observability/probes/` — thêm `debezium_queue.go` + test.
- `internal/api/dashboard_handler.go` (file mới) — `/api/v1/dashboard/timeline`, `/snapshot/active`, `/snapshot/:id/prioritize`, `/dlq/recent`, `/drift/recent`.
- `internal/router/router.go` — wire routes.

### Frontend `cdc-cms-web`
- `src/pages/DashboardV2.tsx` (file mới) — container 3 tab.
- `src/components/dashboard/SnapshotCommanderTab.tsx`, `StreamingRealtimeTab.tsx`, `DlqDriftTab.tsx`.
- `src/components/dashboard/TtcWidget.tsx`, `UnifiedCrosshairChart.tsx`, `PayloadViewerModal.tsx`.
- `src/hooks/useDashboard.ts` — wrap 5 endpoint mới.
- `src/services/dashboard.ts` — fetcher.

---

## 9. Kết quả mong đợi của workspace

- Bộ 13 file docs (00..10 + report). 
- Plan đủ chi tiết để Muscle thực thi không cần hỏi lại.
- Effort estimate đến từng task.
- ADR ghi rõ trade-off (5 ADR).
- Verb chờ user để release Muscle: `execute backend` / `execute frontend` / `execute both` / `revise <section>` / `defer`.

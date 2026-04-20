# Assumptions Verified — Archaeology Report

> **Date**: 2026-04-17
> **Source**: User answers + Explore agent archaeology
> **Purpose**: Ground Plan v3 trên facts thực tế, không giả định.

---

## Summary

| ID | Assumption | User Answer | Code Archaeology | Final Status | Plan v3 Action |
|:---|:-----------|:------------|:-----------------|:-------------|:---------------|
| V1 | MongoDB secondary replica | ✅ có | — | ✅ Confirmed | Agent dùng `readPreference=secondary` |
| V2 | PostgreSQL read-replica | ✅ có | — | ✅ Confirmed | Agent dùng replica DSN |
| V3 | Prometheus (cạnh SigNoz) | ✅ có | ⚠️ Code define `promauto` metrics nhưng **không expose `/metrics`**. OTLP push SigNoz (port 4318). Không có Prom service trong `docker-compose.yml`, không alert rules. | ⚠️ Mixed | T0: expose `/metrics` + DevOps coord scrape config |
| V4 | Debezium converter | Avro | ❌ Thực tế: `mongodb-connector.json` = JSON converter, `schemas.enable=false`. Schema Registry service chạy nhưng **chưa wire** cho CDC data. | ⚠️ Mixed (intent vs reality) | Phase A = JSON + registry validation. Phase B = Avro migration (future) |
| V5 | NATS JetStream | JetStream | ✅ 3 streams thật: `CDC_EVENTS`, `SCHEMA_DRIFT`, `SCHEMA_CONFIG`, FileStorage, retention 7d, 1 replica | ✅ Confirmed | `/jsz` endpoint OK cho monitoring |
| V6 | Index `updated_at` | ✅ có | — (user authoritative) | ✅ Confirmed | Window-based Recon khả thi |
| V7 | Cột `_source_ts` | — | ❌ Chỉ có `_synced_at` (wall-clock Worker). Không có column capture Debezium `source.ts_ms`. | ❌ Missing | **T0-1**: Migration 009 thêm `_source_ts BIGINT` + index |
| V8 | OTel Kafka instrumentation | — | ❌ `segmentio/kafka-go` bare, no `otelkafka` wrapper, no W3C header propagation | ❌ Missing | Worker tạo span thủ công từ Kafka headers (manual W3C extract) |
| V9 | FE React Query | ❌ chưa | ❌ `package.json`: React 19.2.4 + Ant Design 6.3.5 + Axios 1.14.0 + React Router 7.13.2. **No state manager, no data fetching lib**. | ❌ Missing | Plan v3 thêm `@tanstack/react-query` |
| V10 | Redis available | — | ✅ Redis :16379 trong docker-compose, dùng cho schema cache only (TTL 5 phút). **Underutilized**. | ✅ Confirmed | Dùng cho health cache, alert state, idempotency key |

---

## Additional Facts từ Archaeology (ngoài 10 V)

### A1. Recon Code Status
- `recon_core.go` 414 lines, `recon_source_agent.go` 208 lines, `recon_dest_agent.go` 88 lines.
- **Tier 2 hiện tại load `GetAllIDs()` full → confirmed BUG như user flag**.
- `recon_source_agent.go:130` — loop `GetIDs()` batch 10K cho đến hết → nhưng tổng vẫn là full set.
- **Impact**: Plan v3 phải rewrite toàn bộ 3 file này theo XOR-hash window approach.

### A2. Migration Status
- `008_reconciliation.sql` đã apply: tạo `cdc_reconciliation_report` + `failed_sync_logs` với schema chuẩn (bao gồm `missing_ids JSONB`, `healed_at`, `retry_count`, `status`).
- **KHÔNG có** partition / TTL policy → bảng sẽ bloat theo thời gian.
- **Impact**: Plan v3 thêm migration 010 để partition các bảng log.

### A3. Kafka Library
- `segmentio/kafka-go v0.4.50` — pure Go client.
- **Không có ClusterAdmin API** như Sarama → consumer lag phải self-implement qua `kafka.Dial` + `OffsetFetch` protocol, HOẶC dùng `kafka_exporter` sidecar.
- **Impact**: Plan v3 recommend `kafka_exporter` (ít code, standardized).

### A4. Debezium Signal Support
- Plan v2 nói dùng Debezium Signal → chưa verify support.
- **Cần verify**: MongoDB source connector version Debezium. Signal collection require Debezium 1.9+.
- **Impact**: Thêm vào "Open Question" — Muscle cần check Debezium version.

### A5. Schema Registry service
- **CÓ** Schema Registry service chạy ở `docker-compose.yml:129-142` — nhưng **chỉ dùng cho Kafka Connect internal topics**.
- CDC data value dùng JSON converter → Schema Registry idle cho CDC.
- **Impact**: Plan v3 Phase B Avro migration sẽ tận dụng Schema Registry sẵn có — ít work hơn stand-up mới.

### A6. Prometheus metrics đã define nhưng không populate
- `pkgs/metrics/prometheus.go` define 8 metrics: `EventsProcessed`, `ProcessingDuration`, `KafkaConsumerLag`, etc.
- Một số gauge như `KafkaConsumerLag` **được declare nhưng KHÔNG set** → luôn bằng 0 → misleading.
- **Impact**: Plan v3 bổ sung consumer lag polling (O5-1/O5-2) để populate metric.

### A7. FE stack
- React 19 + Ant Design 6 (stable).
- Không có global state management → mỗi page fetch riêng qua axios → code duplication.
- **Impact**: Thêm React Query giải quyết cả 2 (fetch + cache + dedup).

### A8. OTel config hiện tại
- `config-local.yml`: `otel.endpoint: http://localhost:4318` (OTLP HTTP → SigNoz).
- Không có `sample_ratio` per severity, không có `memory_limiter`, không có fallback.
- **Impact**: Plan v3 rewrite `otel.go` theo config mới (O3-1 đến O3-4).

### A9. NATS streams retention
- `FileStorage`, retention **7 ngày**.
- Replicas = 1 (không HA).
- **Impact (nhẹ)**: Nếu NATS server die trong 7 ngày → không loss. Sau 7 ngày → loss.
- **Action**: Future — tăng replicas = 3 khi có NATS cluster.

### A10. Airbyte presence
- Docker-compose có thể có Airbyte (previous history nêu). Plan v2/v3 không đề cập.
- **Action**: Verify nếu Airbyte còn dùng parallel với Debezium hay đã deprecate.

---

## Impact Matrix lên Plan v3

| Finding | Plan DI v3 task | Plan Obs v3 task |
|:--------|:-----------------|:------------------|
| V3 Prom not wired | T0-5 (expose metrics) | O0-1, O3-coord |
| V4 JSON current, Avro intent | T4-4 Phase A validation | — |
| V7 `_source_ts` missing | **T0-1 CRITICAL** | — |
| V8 OTel Kafka missing | — | **O6-1/2/3** |
| V9 no React Query | T3+ FE refactor | **O9-1/2/3** |
| A1 Recon loads full IDs | **T1-1/2 REWRITE** | — |
| A2 No partition | **T0-3** | **O0-2** |
| A3 segmentio lib | — | O5-1 kafka_exporter preferred |
| A5 Schema Registry available | T8 Avro leverage | — |
| A6 Metrics declared not populated | — | O5-2 populate lag |
| A8 OTel simple config | — | **O3-1/2/3 rewrite** |

---

## Open Questions (unresolved after archaeology)

1. **Prom production server URL** — user nói có, code không có evidence deploy. **Ask DevOps**.
2. **Debezium version** — determine if incremental snapshot signal supported. **Muscle check `docker-compose.yml` Debezium image tag**.
3. **Airbyte status** — parallel or deprecated?
4. **Top 10 largest tables** — concrete list để prioritize load test.
5. **PG replication lag tolerance** — Recon skip threshold (v3 tentatively 60s).
6. **RBAC system** — đã có user/role infrastructure chưa, hay plan v3 phải build từ đầu?

---

## Verdict

Archaeology complete. Plan v3 (cả 2 file) based on:
- ✅ 6/10 assumption confirmed
- ⚠️ 2/10 mixed (intent vs reality) — handle qua 2-phase approach (JSON → Avro, current Prom state → expose + coord)
- ❌ 2/10 missing — bổ sung rõ ràng trong Plan v3 với task cụ thể

**Kết luận**: Đủ data để Plan v3. 6 open questions là non-blocking, có thể resolve trong implementation phase.

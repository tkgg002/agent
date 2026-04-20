# Plan: Data Integrity — FINAL (Merged v2 + Deep Analysis)

> Date: 2026-04-16
> Phase: data_integrity
> Sources: 
>   - `01_requirements_data_integrity_solution.md` (base)
>   - `01_requirements_data_integrity_solution_deep.md` (advanced cases + Merkle Tree + Version Heal)

---

## 1. Failure Modes — Full Stack

### Layer 1: Source DB (MongoDB)
| Case | Mô tả | Giải pháp |
|:-----|:-------|:----------|
| Oplog Overload | updateMany triệu records → Oplog spike → Debezium lag | Monitor Oplog Size + Retention. Alert khi lag > threshold |
| Clock Skew | Cluster nodes lệch clock → Debezium skip events | NTP sync enforce. Recon detect missing bằng ID set |

### Layer 2: Debezium
| Case | Mô tả | Giải pháp |
|:-----|:-------|:----------|
| Offset Inconsistency | Die lâu > Oplog retention → mất offset | Signal Table: `debezium_signal` → trigger ad-hoc snapshot |
| Schema History | Schema change during downtime | Schema History Topic tự rebuild |

### Layer 3: Kafka
| Case | Mô tả | Giải pháp |
|:-----|:-------|:----------|
| Log Deletion | retention.ms expire → messages mất | **`cleanup.policy=compact`** cho CDC topics (giữ latest per key) |
| Consumer Lag spike | Worker chậm → messages tích lũy | Monitor lag + auto-scale Workers |

### Layer 4: Worker
| Case | Mô tả | Giải pháp |
|:-----|:-------|:----------|
| Silent Crash | Record lỗi type/schema → skip | **DLQ** (`failed_sync_logs`) + alert |
| Data Type Mismatch | VARCHAR(255) nhận string dài | SchemaAdapter check + auto ALTER |
| Committed offset + failed INSERT | Kafka offset ack nhưng DB fail | DLQ capture + Recon heal |

### Layer 5: Destination (Postgres)
| Case | Mô tả | Giải pháp |
|:-----|:-------|:----------|
| Schema Incompatibility | New field, type change | Schema Evolution (auto-alter) + Validation Agent |

### Layer 6: Recon vs CDC Race Condition
| Case | Mô tả | Giải pháp |
|:-----|:-------|:----------|
| Heal ghi đè data mới | Recon write old record đúng lúc CDC write new | **Version-aware Heal**: so sánh timestamp trước UPSERT |

---

## 2. Architecture: Recon Core + Agent (Advanced)

```
┌────────────────────────────────────────────────────────────┐
│  Recon Core (The Brain)                                     │
│                                                            │
│  - So sánh Merkle Tree Hash từ 2 Agents                    │
│  - Nếu hash lệch → yêu cầu Agent gửi ID chi tiết          │
│  - Version-aware Heal (compare timestamp trước UPSERT)      │
│  - Audit Log mọi heal action                                │
│  - Ghi report → cdc_reconciliation_report                   │
│  - Cung cấp API cho CMS                                     │
└────────────────────┬───────────────────────────────────────┘
                     │
          ┌──────────┴──────────┐
          ▼                     ▼
┌──────────────────┐   ┌──────────────────┐
│  Source Agent     │   │  Dest Agent      │
│  (MongoDB)       │   │  (Postgres)      │
│                  │   │                  │
│  Tier 1: Count   │   │  Tier 1: Count   │
│  Tier 2: ID Set  │   │  Tier 2: ID Set  │
│    (batch 10K)   │   │    (batch 10K)   │
│  Tier 3: Merkle  │   │  Tier 3: Merkle  │
│    Tree Hash     │   │    Tree Hash     │
│    (per chunk)   │   │    (per chunk)   │
│                  │   │                  │
│  Trả: count,    │   │  Trả: count,    │
│  ID set, hash   │   │  ID set, hash   │
│  per chunk      │   │  per chunk      │
└──────────────────┘   └──────────────────┘
```

### Merkle Tree Hash
- Chia records thành chunks (10K records mỗi chunk, sort by _id)
- Mỗi chunk: hash = MD5(concat(all record hashes in chunk))
- So sánh chunk hashes giữa source + dest
- Hash lệch → drill down vào chunk đó tìm records cụ thể
- **Hiệu quả**: 1M records = 100 chunks = 100 hash comparisons thay vì 1M ID comparisons

---

## 3. Tiered Approach (with Actions)

| Tier | Method | Frequency | Action khi lệch |
|:-----|:-------|:----------|:-----------------|
| Tier 1 | Count Check | 5 min | Alert CMS + trigger Tier 2 |
| Tier 2 | ID Set/Boundary (batch 10K) | On demand / 1h | Tìm dải ID missing → report |
| Tier 3 | Merkle Tree Hash (per chunk) | 24h | Detect stale data (đủ count nhưng sai nội dung) |

---

## 4. Version-aware Heal

```
Core nhận missing IDs từ Agent
    ↓
Fetch full document từ MongoDB (với timestamp)
    ↓
Query Postgres: SELECT _synced_at FROM table WHERE _id = ?
    ↓
Compare: MongoDB timestamp > Postgres _synced_at?
    ↓
  YES → UPSERT (data source mới hơn)
  NO  → SKIP (Postgres đã có data mới hơn từ CDC stream)
    ↓
Audit Log: ghi action + reason
```

---

## 5. Kafka Hardening

### cleanup.policy=compact
```
Thay vì: cleanup.policy=delete (mặc định — xoá messages cũ)
Đổi thành: cleanup.policy=compact (giữ latest message per key)
```
- CDC topics chứa change events per record ID (key = record _id)
- Compact = giữ latest state per ID → không bao giờ mất latest data
- Worker chậm bao lâu cũng không mất data (compact giữ latest)

### Config Kafka
```
docker exec gpay-kafka kafka-configs --alter \
  --entity-type topics \
  --entity-name cdc.goopay.centralized-export-service.export-jobs \
  --add-config cleanup.policy=compact \
  --bootstrap-server localhost:9092
```

---

## 6. Worker Hardening

### 6.1 Idempotent Write
- ON CONFLICT DO UPDATE (đã có)
- Version check: `WHERE _version <= EXCLUDED._version` (thêm mới)

### 6.2 DLQ → `failed_sync_logs`
- Mọi record lỗi → ghi DB table kèm: raw JSON, error, topic, offset
- CMS UI: view + retry + resolve
- KHÔNG crash Worker, KHÔNG skip silently

### 6.3 Observability
- Prometheus: `cdc_sync_success_total`, `cdc_sync_failed_total` (per table, per op)
- OTel traces (đã có): span per event

### 6.4 Schema Registry Validation
- Worker check schema version từ Kafka message header
- Version lạ (không trong cache) → fetch + validate
- Nếu incompatible → DLQ + alert (KHÔNG process sai)

---

## 7. Debezium Signal Table

Thay vì manual reset offset, dùng Debezium Signal:

```sql
-- Tạo signal collection trong MongoDB
db.debezium_signal.insertOne({
    "type": "execute-snapshot",
    "data": {
        "data-collections": ["payment-bill-service.export-jobs"],
        "type": "incremental"
    }
})
```
→ Debezium đọc signal → re-snapshot table cụ thể → messages re-published → Worker re-consume → data healed

**Lợi ích**: Không cần stop Worker, không cần reset Kafka offset.

---

## 8. Tasks (FINAL — merged)

### Phase 1: Database + Models + Kafka Config
- [ ] T1: Migration `008_reconciliation.sql` (cdc_reconciliation_report + failed_sync_logs)
- [ ] T2: Models Go (Worker + CMS)
- [ ] T3: Kafka topic `cleanup.policy=compact` cho CDC topics
- [ ] T4: Debezium signal collection setup (MongoDB)

### Phase 2: Agents
- [ ] T5: Go MongoDB driver + `pkgs/mongodb/client.go` + config
- [ ] T6: `recon_source_agent.go` (count + ID set batch + Merkle Tree hash)
- [ ] T7: `recon_dest_agent.go` (count + ID set batch + Merkle Tree hash)

### Phase 3: Core + Heal
- [ ] T8: `recon_core.go` (orchestrate + compare + tiered approach + schedule)
- [ ] T9: Version-aware Heal (timestamp compare before UPSERT)
- [ ] T10: Audit Log cho mọi heal action

### Phase 4: Worker Hardening
- [ ] T11: BatchBuffer error → `failed_sync_logs` table
- [ ] T12: Prometheus counters (success/failed per table per op)
- [ ] T13: Schema version validation (từ Kafka message + Schema Registry)

### Phase 5: CMS API + FE
- [ ] T14: API (report, check tiers, heal, failed logs, retry)
- [ ] T15: FE Data Integrity Dashboard (tổng quan + chi tiết + tools)
- [ ] T16: FE failed_sync_logs viewer + retry button

### Phase 6: Verify + Heal Current Drift
- [ ] T17: Detect lệch hiện tại (Tier 1 + Tier 2)
- [ ] T18: Heal via Debezium signal hoặc direct MongoDB fetch
- [ ] T19: Verify count match after heal

---

## 9. Files (FINAL)

### Worker
| File | Purpose |
|:-----|:--------|
| `migrations/008_reconciliation.sql` | Tables |
| `internal/model/reconciliation_report.go` | Model |
| `internal/model/failed_sync_log.go` | Model |
| `internal/service/recon_source_agent.go` | MongoDB agent (count + IDs + Merkle) |
| `internal/service/recon_dest_agent.go` | Postgres agent (count + IDs + Merkle) |
| `internal/service/recon_core.go` | Core (compare + version-aware heal + audit) |
| `pkgs/mongodb/client.go` | MongoDB Go driver |
| `config/config.go` | MongoDBConfig |
| `internal/handler/batch_buffer.go` | Error → failed_sync_logs |
| `pkgs/metrics/prometheus.go` | Sync counters |

### CMS
| File | Purpose |
|:-----|:--------|
| `internal/api/reconciliation_handler.go` | API endpoints |
| `internal/model/reconciliation_report.go` | Model |
| `internal/model/failed_sync_log.go` | Model |

### FE
| File | Purpose |
|:-----|:--------|
| `src/pages/DataIntegrity.tsx` | Dashboard |

---

## 10. Definition of Done

- [ ] Merkle Tree hash comparison hoạt động (source vs dest)
- [ ] Version-aware Heal: KHÔNG ghi đè data mới bằng data cũ
- [ ] failed_sync_logs capture mọi lỗi (không skip, không crash)
- [ ] Kafka CDC topics: cleanup.policy=compact
- [ ] Debezium signal re-snapshot hoạt động
- [ ] Prometheus counters: success + failed per table
- [ ] CMS Dashboard: status per table, actions, failed logs
- [ ] Data lệch hiện tại → healed → count match

---

## 11. Bổ sung — Coverage check từ doc 1

### 11.1 Schema History Topic (Debezium)
- Debezium lưu schema history vào Kafka topic riêng (`__debezium-schema-history`)
- Khi Debezium restart → đọc history topic → tái cấu trúc schema
- **Config cần**: Topic retention = unlimited (không được xoá)
- **Task thêm**: T20 — Verify schema history topic retention config

### 11.2 Kafka Consumer Lag Monitoring
- Monitor consumer lag realtime → alert khi lag > threshold
- Redpanda Console đã hiện lag nhưng cần:
  - Prometheus metric từ Kafka: `kafka_consumer_group_lag`
  - Alert rule: lag > 1000 → warning, lag > 10000 → critical
- **Task thêm**: T21 — Consumer lag metric + alert rule

### 11.3 CMS Tools: Reset Offset + Trigger Snapshot
- FE buttons trong Data Integrity Dashboard:
  - "Reset Debezium Offset" → gọi API → ghi signal vào MongoDB debezium_signal collection
  - "Trigger Snapshot" → gọi API → ghi signal type=execute-snapshot
  - "Reset Kafka Offset" → gọi API → Worker pause + kafka-consumer-groups reset + resume
- **Task thêm**: T22 — CMS API + FE cho 3 tools trên

### 11.4 Updated Task List (T20-T22)
- [ ] T20: Schema History Topic — verify retention config unlimited
- [ ] T21: Consumer lag Prometheus metric + alert threshold
- [ ] T22: CMS tools — Reset offset, Trigger snapshot, Reset Kafka offset

### 11.5 Updated Definition of Done
- [ ] Schema History Topic retention = unlimited
- [ ] Consumer lag metric visible + alert configured
- [ ] CMS tools: Reset Debezium offset + Trigger snapshot + Reset Kafka offset

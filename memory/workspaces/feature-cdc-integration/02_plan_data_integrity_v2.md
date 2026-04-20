# Plan: Data Integrity — Reconciliation System (v2)

> Date: 2026-04-16
> Phase: data_integrity
> Source: `01_requirements_data_integrity_solution.md` (User provided)

---

## 1. Architecture: Recon Core + Agent

```
┌──────────────────────────────────────────────────────────┐
│  Recon Core (Orchestrator) — trong CDC Worker             │
│                                                          │
│  - Quản lý config đối soát (table, frequency)            │
│  - Nhận report từ Source Agent + Dest Agent               │
│  - So sánh kết quả → quyết định Heal                     │
│  - Cung cấp API cho CMS Dashboard                        │
│  - Ghi report → cdc_reconciliation_report                │
└────────────────────┬─────────────────────────────────────┘
                     │
          ┌──────────┴──────────┐
          ▼                     ▼
┌─────────────────┐   ┌─────────────────┐
│  Source Agent    │   │  Dest Agent     │
│  (MongoDB)      │   │  (Postgres)     │
│                 │   │                 │
│  - Count docs   │   │  - Count rows   │
│  - Distinct IDs │   │  - Distinct IDs │
│  - Hash sample  │   │  - Hash sample  │
│                 │   │                 │
│  Trả: count,    │   │  Trả: count,   │
│  ID set, hash   │   │  ID set, hash  │
└─────────────────┘   └─────────────────┘
```

## 2. Tiered Approach

| Tier | Method | Frequency | Action khi lệch |
|:-----|:-------|:----------|:-----------------|
| Tier 1 (Fast) | Count Check | 5 min | Alert CMS + trigger Tier 2 |
| Tier 2 (Medium) | ID Set/Boundary (batch 10K) | On demand / 1h | Tìm dải ID missing → report |
| Tier 3 (Deep) | Field Hash (MD5 per record) | 24h | Detect stale data (đủ count nhưng sai nội dung) |

## 3. Action Plan — Xử lý lệch hiện tại

### Bước 1: Monitor
- Check Kafka consumer lag
- Check Worker error log → failed_sync_logs

### Bước 2: Scan
- Source Agent: `countDocuments()` + `distinct("_id")`
- Dest Agent: `SELECT COUNT(*)` + `SELECT DISTINCT _id`
- Diff → missing IDs

### Bước 3: Heal
- Manual trigger từ CMS: "Re-sync IDs"
- Repair Worker: truy vấn MongoDB bằng missing IDs → UPSERT Postgres
- **KHÔNG đi qua Kafka** — trực tiếp MongoDB → Postgres

### Bước 4: Dashboard
- Tổng quan: tables, status (Matched/Drifted), diff count
- Chi tiết: records lỗi, lý do (schema mismatch, timeout)
- Công cụ: Reset offset, Trigger snapshot, Re-sync IDs

## 4. Worker Hardening

### 4.1 Idempotency
- Tất cả INSERT dùng `ON CONFLICT (id) DO UPDATE` — replay safe (đã có)

### 4.2 Dead Letter Queue → `failed_sync_logs`
- Record lỗi → ghi vào `failed_sync_logs` table
- KHÔNG crash Worker, KHÔNG skip silently
- Kèm: raw JSON, error message, timestamp, topic, offset, table name

### 4.3 Observability
- Prometheus counters: `cdc_sync_success_total`, `cdc_sync_failed_total`
- Labels: table, operation (c/u/d), source (debezium/airbyte)

## 5. Tasks

### Phase 1: Database + Models
- [ ] T1: Migration `008_reconciliation.sql` (cdc_reconciliation_report + failed_sync_logs)
- [ ] T2: Models Go (reconciliation_report + failed_sync_log) — Worker + CMS

### Phase 2: Agents
- [ ] T3: Go MongoDB driver + config
- [ ] T4: `recon_source_agent.go` — count + distinct IDs + hash sample
- [ ] T5: `recon_dest_agent.go` — count + distinct IDs + hash sample

### Phase 3: Core + Heal
- [ ] T6: `recon_core.go` — orchestrate agents, compare, report, schedule
- [ ] T7: Auto-heal — fetch missing from MongoDB → upsert Postgres (bypass Kafka)

### Phase 4: Worker Hardening
- [ ] T8: `failed_sync_logs` — BatchBuffer catch errors → ghi DB
- [ ] T9: Prometheus counters (success/failed per table per op)

### Phase 5: CMS API + FE
- [ ] T10: CMS API (report, check now, heal, failed logs)
- [ ] T11: CMS FE — Data Integrity Dashboard
- [ ] T12: FE — failed_sync_logs viewer

### Phase 6: Verify
- [ ] T13: Detect lệch hiện tại → heal → count match
- [ ] T14: Progress + docs update

## 6. Files

### Worker (centralized-data-service)
| File | New/Edit | Purpose |
|:-----|:---------|:--------|
| `migrations/008_reconciliation.sql` | New | cdc_reconciliation_report + failed_sync_logs |
| `internal/model/reconciliation_report.go` | New | Model |
| `internal/model/failed_sync_log.go` | New | Model |
| `internal/service/recon_source_agent.go` | New | MongoDB agent |
| `internal/service/recon_dest_agent.go` | New | Postgres agent |
| `internal/service/recon_core.go` | New | Orchestrator + heal |
| `pkgs/mongodb/client.go` | New | MongoDB Go driver |
| `config/config.go` | Edit | MongoDBConfig |
| `config/config-local.yml` | Edit | mongodb section |
| `internal/handler/batch_buffer.go` | Edit | Error → failed_sync_logs |
| `internal/server/worker_server.go` | Edit | Init MongoDB + recon |
| `pkgs/metrics/prometheus.go` | Edit | Add success/failed counters |

### CMS (cdc-cms-service)
| File | New/Edit | Purpose |
|:-----|:---------|:--------|
| `internal/api/reconciliation_handler.go` | New | Endpoints |
| `internal/router/router.go` | Edit | Routes |
| `internal/model/reconciliation_report.go` | New | Model |
| `internal/model/failed_sync_log.go` | New | Model |

### FE (cdc-cms-web)
| File | New/Edit | Purpose |
|:-----|:---------|:--------|
| `src/pages/DataIntegrity.tsx` | New | Dashboard |
| `src/App.tsx` | Edit | Route + menu |

## 7. Definition of Done
- [ ] Source Agent query MongoDB thành công
- [ ] Dest Agent query Postgres thành công
- [ ] Recon Core detect diff → report missing IDs
- [ ] Auto-heal: fetch missing → upsert → count match
- [ ] failed_sync_logs ghi records lỗi
- [ ] Prometheus counters active
- [ ] CMS Dashboard hiện status + actions
- [ ] Data lệch hiện tại → heal → match

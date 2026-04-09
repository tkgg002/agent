# Phase 1: Full System - Airbyte Primary Path

> **Workspace**: feature-cdc-integration
> **Phase**: 1 of 2
> **Focus**: Hệ thống hoàn chỉnh. Airbyte là data path chính. CDC Worker full. Chỉ Dynamic Mapper chưa code (dùng static mapping). Debezium chỉ khởi tạo config.

---

## Scope Phase 1

### Full Implementation

| Component | Source (03_implementation.md) |
|-----------|-------------------------------|
| PostgreSQL Schema (CDC tables + JSONB Landing Zone) | Section 2.1, 2.2 |
| Management Tables (mapping_rules, pending_fields, schema_log) | Section 2.3 |
| Upsert Functions | Section 2.4 |
| **CDC Worker** (NATS consumer, worker pool, batch, event handler) | Section 3.1, 3.4 |
| Schema Inspector (drift detection, type inference, NATS alert) | Section 3.2 |
| CMS Backend API (approve/reject, ALTER TABLE, mapping CRUD) | Section 3.5 |
| CMS Frontend React (PendingChangesTable, ApprovalModal) | Section 3.6 |
| Airbyte API Client (RefreshSchema, TriggerSync, UpdateConnection) | Section 3.7 |
| Config Reload (NATS `schema.config.reload`) | Section 3.3 (listener part) |
| Prometheus Metrics | Section 6.1 |
| Docker Compose (local dev) | Section 4.4 |
| K8s Deployment (Worker + CMS) | Section 4.1, 4.2 |
| ConfigMap | Section 4.3 |
| Unit Tests (Inspector + Handler) | Section 5.1 |

### Khởi tạo (Chưa Code Logic)

| Component | Source | Notes |
|-----------|--------|-------|
| **Dynamic Mapper** | Section 3.3 | Structure + interfaces ready. **Chưa code**: dynamic query builder, rule loading, hot reload logic. Worker dùng **static mapping** (hardcoded columns per table) thay thế |
| Debezium connectors | - | Config files ready, chưa deploy production |

### Phase 2

| Component | Notes |
|-----------|-------|
| **Dynamic Mapper full** | Load rules từ DB, dynamic query builder, hot reload |
| Event Bridge (Postgres → NATS → Moleculer) | Trigger + Listener + Poller |
| Data Reconciliation | Checksum, count, drift detection |
| DLQ & advanced error handling | Dead letter queue, replay |
| Enrichment Service | Computed fields, business logic |
| Integration & Performance Tests | E2E, 50K events/sec |
| K8s production scaling | HPA, replicas=5 |

---

## Architecture Phase 1

```
┌────────────────────────────────────────────────────────────────┐
│                      SOURCE DATABASES                          │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐            │
│  │  MongoDB     │  │   MySQL     │  │ PostgreSQL  │            │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘            │
└─────────┼────────────────┼────────────────┼────────────────────┘
          │                │                │
          │ Oplog          │ Binlog         │ WAL
          ▼                ▼                ▼
┌──────────────────────────────────────────────────┐
│  DEBEZIUM CONNECTORS (config ready, chưa active) │
└──────────────────┬───────────────────────────────┘
                   │ (Phase 2 mới active)
                   ▼
┌──────────────────────────────────────────────────┐
│   NATS JETSTREAM CLUSTER                          │
│  Topics:                                          │
│  - cdc.goopay.{table}     (CDC events)            │
│  - schema.drift.detected  (drift alerts)          │
│  - schema.config.reload   (config reload)         │
└──────────────────┬───────────────────────────────┘
                   │ Pull Subscribe
                   ▼
┌──────────────────────────────────────────────────────────────┐
│  CDC WORKER SERVICE (Go) - Full Implementation                │
│                                                               │
│  ┌─────────────────────────────────────────────────────┐     │
│  │  NATS Consumer Pool (10 workers/pod)                 │     │
│  └──────────────┬──────────────────────────────────────┘     │
│                 ▼                                             │
│  ┌─────────────────────────────────────────────────────┐     │
│  │  Schema Inspector (Full)                             │     │
│  │  - Detect new fields    - Infer data types           │     │
│  │  - Save pending_fields  - Publish drift alerts       │     │
│  └──────────────┬──────────────────────────────────────┘     │
│                 ▼                                             │
│  ┌─────────────────────────────────────────────────────┐     │
│  │  Static Mapping (Phase 1)                            │     │
│  │  - Hardcoded columns per table                       │     │
│  │  - Phase 2: replace với Dynamic Mapper               │     │
│  └──────────────┬──────────────────────────────────────┘     │
│                 ▼                                             │
│  ┌─────────────────────────────────────────────────────┐     │
│  │  Batch Buffer (500 records / 2s timeout)             │     │
│  │  → PostgreSQL Batch Upsert                           │     │
│  │  → ALWAYS save _raw_data JSONB                       │     │
│  └─────────────────────────────────────────────────────┘     │
└──────────────────────────────────────────────────────────────┘
                   │
                   ▼
┌──────────────────────────────────────────────────────────────┐
│   POSTGRESQL CLUSTER                                          │
│   Tables: wallet_transactions, payments, orders               │
│   + _raw_data JSONB (Landing Zone)                            │
│   + cdc_mapping_rules, pending_fields, schema_changes_log     │
└──────────────────────────────────────────────────────────────┘
          ↑                              │
          │                              ▼
┌─────────┴────────────┐   ┌──────────────────────────────────┐
│  AIRBYTE              │   │  CMS SERVICE                     │
│  (Primary Data Path)  │   │  Backend: Go/Gin                 │
│  Batch Sync 15min-1hr │   │  - Pending changes API           │
│  Direct Postgres Write│   │  - Approve → ALTER TABLE         │
│                       │   │  - Mapping rules CRUD            │
│                       │   │  - Airbyte API integration       │
│                       │   │  Frontend: React + Ant Design    │
│                       │   │  - PendingChangesTable           │
│                       │   │  - ApprovalModal                 │
└───────────────────────┘   └──────────────────────────────────┘
```

### Luồng chính Phase 1

**Airbyte path** (primary):
1. Airbyte sync Source DBs → PostgreSQL (mapped columns + `_raw_data`)

**CDC Worker path** (active, sẵn sàng cho Debezium Phase 2):
1. CDC Worker subscribe NATS `cdc.goopay.{table}`
2. Nhận event → Schema Inspector detect new fields → lưu `pending_fields` → publish `schema.drift.detected`
3. **Static mapping** extract known columns + luôn save `_raw_data`
4. Batch upsert → PostgreSQL

**CMS path**:
1. CMS UI hiển thị pending changes
2. DevOps/Dev approve → ALTER TABLE + tạo mapping rule
3. Publish `schema.config.reload` → (Phase 2: Dynamic Mapper reload)

---

## Implementation Reference

Code chi tiết trong `03_implementation.md`. Phase 1 implement **gần như toàn bộ**, trừ Dynamic Mapper logic.

### Full Implementation (theo 03_implementation.md)

| What | Section | Lines (approx) |
|------|---------|-----------------|
| System Architecture | 1 | 1-130 |
| DB Schema (CDC tables + JSONB) | 2.1, 2.2 | 133-216 |
| Management Tables | 2.3 | 220-361 |
| Upsert Function | 2.4 | 365-432 |
| Project Structure | 3.1 | 436-521 |
| Schema Inspector | 3.2 | 525-745 |
| **CDC Worker Event Handler** | **3.4** | **1055-1275** |
| CMS Backend Handlers | 3.5 | 1279-1646 |
| CMS Frontend (React) | 3.6 | 1650-2024 |
| Airbyte API Client | 3.7 | 2028-2202 |
| K8s Deployment (Worker) | 4.1 | 2206-2290 |
| K8s Deployment (CMS) | 4.2 | 2294-2368 |
| ConfigMap | 4.3 | 2372-2397 |
| Docker Compose | 4.4 | 2400-2462 |
| Unit Tests | 5.1 | 2466-2561 |
| Prometheus Metrics | 6.1 | 2744-2799 |

### Khởi tạo Only (Structure + Interfaces, chưa code logic)

| What | Section | Notes |
|------|---------|-------|
| Dynamic Mapper | 3.3 (lines 749-1051) | Tạo file `dynamic_mapper.go` với struct + interfaces. **Chưa code**: `LoadRules()`, `MapData()`, `BuildUpsertQuery()`, `convertType()`, `StartConfigReloadListener()`. Worker dùng static mapping thay. |

### Sự khác biệt Phase 1 vs 03_implementation.md

**CDC Worker Event Handler (Section 3.4)** - Phase 1 thay đổi:
- `DynamicEventHandler` → dùng **static column list** thay vì gọi `dynamicMapper.MapData()`
- `getKnownColumns(tableName)` method mới: return hardcoded columns per table
- Vẫn giữ nguyên: Schema Inspector call, `_raw_data` save, hash calculation, batch upsert, delete handling, metrics

```go
// Phase 1: Static mapping thay cho Dynamic Mapper
func (h *EventHandler) getKnownColumns(tableName string) []string {
    switch tableName {
    case "wallet_transactions":
        return []string{
            "user_id", "wallet_id", "transaction_type", "amount",
            "currency", "balance_before", "balance_after", "status",
            "reference_id", "description", "metadata", "created_at", "completed_at",
        }
    case "payments":
        return []string{
            "user_id", "order_id", "amount", "currency", "method",
            "status", "provider_ref", "created_at", "completed_at",
        }
    case "orders":
        return []string{
            "user_id", "merchant_id", "total_amount", "currency",
            "status", "items", "created_at", "updated_at",
        }
    default:
        return nil // Fallback: chỉ save _raw_data
    }
}
```

> **Phase 2**: Replace `getKnownColumns()` bằng `dynamicMapper.MapData()` - load columns từ `cdc_mapping_rules` table.

---

## Phase 1 - Project Structure

```
cdc-worker-service/
├── cmd/
│   ├── worker/
│   │   └── main.go                    # ✅ Full (NATS consumer pool, batch)
│   └── cms-service/
│       └── main.go                    # ✅ Full
├── internal/
│   ├── config/
│   │   └── config.go                  # ✅ Full
│   ├── domain/
│   │   ├── entities/
│   │   │   ├── wallet_transaction.go  # ✅ Full
│   │   │   ├── mapping_rule.go        # ✅ Full
│   │   │   └── pending_field.go       # ✅ Full
│   │   └── repositories/
│   │       ├── transaction_repo.go    # ✅ Full
│   │       ├── mapping_rule_repo.go   # ✅ Full
│   │       └── pending_field_repo.go  # ✅ Full
│   ├── application/
│   │   ├── services/
│   │   │   ├── schema_inspector.go    # ✅ Full (Section 3.2)
│   │   │   └── dynamic_mapper.go      # ⏳ Khởi tạo (struct + interfaces only)
│   │   └── handlers/
│   │       └── event_handler.go       # ✅ Full (Section 3.4, static mapping)
│   └── infrastructure/
│       ├── nats/
│       │   ├── consumer.go            # ✅ Full (worker pool, fetch)
│       │   └── client.go              # ✅ Full
│       ├── postgres/
│       │   ├── repository.go          # ✅ Full
│       │   └── connection.go          # ✅ Full
│       ├── redis/
│       │   └── cache.go               # ✅ Full
│       └── airbyte/
│           └── client.go              # ✅ Full (Section 3.7)
├── interfaces/
│   └── api/
│       ├── health.go                  # ✅ Full
│       └── cms_handlers.go            # ✅ Full (Section 3.5)
├── web/                               # ✅ CMS Frontend
│   └── src/
│       ├── components/
│       │   ├── PendingChangesTable.tsx # ✅ Full (Section 3.6)
│       │   ├── ApprovalModal.tsx       # ✅ Full (Section 3.6)
│       │   └── MappingRulesManager.tsx # ✅ Full
│       ├── pages/
│       │   ├── Dashboard.tsx           # ✅ Full
│       │   └── SchemaChanges.tsx       # ✅ Full
│       └── App.tsx
├── pkg/
│   ├── airbyte/
│   │   └── client.go                  # ✅ Full
│   ├── logger/
│   │   └── logger.go                  # ✅ Full
│   ├── metrics/
│   │   └── prometheus.go              # ✅ Full (Section 6.1)
│   └── utils/
│       ├── hash.go                    # ✅ Full
│       ├── retry.go                   # ✅ Full
│       └── type_inference.go          # ✅ Full
├── deployments/
│   ├── k8s/
│   │   ├── cdc-worker-deployment.yaml # ✅ Full (Section 4.1)
│   │   ├── cms-deployment.yaml        # ✅ Full (Section 4.2)
│   │   └── configmap.yaml             # ✅ Full (Section 4.3)
│   ├── docker/
│   │   ├── Dockerfile.worker          # ✅ Full
│   │   └── Dockerfile.cms             # ✅ Full
│   └── debezium/                      # ⏳ Config files only
│       ├── mongodb-connector.json
│       └── mysql-connector.json
├── migrations/
│   └── 001_init_schema.sql            # ✅ All tables from Section 2
└── tests/
    └── unit/
        ├── schema_inspector_test.go   # ✅ Full (Section 5.1)
        └── event_handler_test.go      # ✅ Full
```

Legend: ✅ = Full implementation | ⏳ = Khởi tạo (structure only)

---

## Execution Order

```
Step 1: Database Setup
  └─ Run migrations (CDC tables + management tables + JSONB + indexes)
  └─ Seed cdc_mapping_rules (initial mappings cho static columns)

Step 2: CDC Worker
  └─ NATS consumer pool (10 workers/pod)
  └─ Event handler với static mapping + Schema Inspector
  └─ Batch buffer (500 records / 2s)
  └─ PostgreSQL batch upsert (_raw_data + mapped columns)
  └─ Health + metrics endpoints

Step 3: Airbyte Configuration
  └─ Setup connections (MongoDB → PostgreSQL, MySQL → PostgreSQL)
  └─ Configure incremental + dedup sync mode
  └─ First sync → verify data

Step 4: CMS Service
  └─ Backend API (approve/reject, ALTER TABLE, CRUD)
  └─ Frontend UI (PendingChangesTable, ApprovalModal)
  └─ Airbyte API integration (RefreshSchema on approve)
  └─ NATS: publish schema.config.reload

Step 5: Integration Test
  └─ Airbyte sync → new field appears in _raw_data
  └─ Schema Inspector detects → pending_fields
  └─ CMS approve → ALTER TABLE → mapping rule created
  └─ Next sync → field mapped to dedicated column
  └─ CDC Worker: publish test event → processes correctly

Step 6: Debezium (Khởi tạo)
  └─ Create connector config files
  └─ Test locally with Docker Compose (optional)
  └─ NOT deployed to production
```

---

## Checklist Phase 1

### Database
- [ ] CDC tables created (wallet_transactions, payments, orders)
- [ ] Management tables created (mapping_rules, pending_fields, schema_log)
- [ ] Upsert function deployed
- [ ] Initial mapping rules seeded
- [ ] GIN indexes on _raw_data

### CDC Worker (Full)
- [ ] NATS consumer pool (10 workers/pod) running
- [ ] Event handler processes events with static mapping
- [ ] Schema Inspector detects new fields
- [ ] Schema Inspector publishes drift alerts to NATS
- [ ] Batch buffer (500 records / 2s timeout) working
- [ ] PostgreSQL batch upsert succeeds
- [ ] `_raw_data` always populated
- [ ] Delete handling (soft delete `_deleted = TRUE`)
- [ ] Health/ready endpoints respond
- [ ] Prometheus metrics exposed
- [ ] Unit tests pass

### Dynamic Mapper (Khởi tạo)
- [ ] `dynamic_mapper.go` file created with struct + interfaces
- [ ] `DynamicMapper` struct defined
- [ ] `MapData()`, `LoadRules()`, `BuildUpsertQuery()` interfaces defined
- [ ] Chưa code logic implementation

### Airbyte
- [ ] Connections configured (MongoDB + MySQL → PostgreSQL)
- [ ] First sync successful
- [ ] Data verified (_raw_data + mapped columns)
- [ ] Airbyte API Client tested

### Schema Inspector
- [ ] Detect new fields from CDC events
- [ ] Infer data types correctly
- [ ] Save to pending_fields table
- [ ] Publish `schema.drift.detected` to NATS
- [ ] Redis cache for table schema
- [ ] Unit tests pass

### CMS
- [ ] GET /api/schema-changes/pending
- [ ] POST /api/schema-changes/:id/approve (ALTER TABLE + mapping rule)
- [ ] POST /api/schema-changes/:id/reject
- [ ] GET/POST /api/mapping-rules
- [ ] Airbyte schema refresh on approve
- [ ] Frontend: PendingChangesTable renders
- [ ] Frontend: ApprovalModal works
- [ ] End-to-end: detect → CMS → approve → ALTER TABLE

### Deployment
- [ ] Docker Compose works locally
- [ ] K8s Worker deployment
- [ ] K8s CMS deployment
- [ ] Debezium config files created

---

## Transition to Phase 2

Phase 1 done = hệ thống chạy hoàn chỉnh với Airbyte + static mapping.

Phase 2 upgrades:
1. **Dynamic Mapper full**: Code `LoadRules()`, `MapData()`, `BuildUpsertQuery()`, `convertType()`, `StartConfigReloadListener()`. Replace static mapping trong event handler.
2. **Activate Debezium**: Deploy connectors, CDC events flow qua NATS → Worker
3. **Event Bridge**: Postgres → NATS → Moleculer
4. **Enrichment Service**: Computed fields, business logic
5. **Data Reconciliation**: Debezium vs Airbyte consistency
6. **DLQ**: Dead letter queue, retry, replay
7. **K8s scaling**: replicas=5, HPA
8. **Testing**: Integration + Performance (50K events/sec)

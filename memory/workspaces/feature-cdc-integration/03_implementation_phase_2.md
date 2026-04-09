# Phase 2: Dynamic Mapper Full + Debezium + Event Bridge + Production

> **Workspace**: feature-cdc-integration
> **Phase**: 2 of 2
> **Prerequisite**: Phase 1 completed (Airbyte running, CDC Worker running with static mapping, CMS working, Schema Inspector active)
> **Focus**: Dynamic Mapper full, activate Debezium, Event Bridge, Reconciliation, production scaling

---

## Scope Phase 2

| Component | Source (03_implementation.md) | Notes |
|-----------|-------------------------------|-------|
| **Dynamic Mapper full** | Section 3.3 | Code logic: `LoadRules()`, `MapData()`, `BuildUpsertQuery()`, `convertType()`, `StartConfigReloadListener()`. Replace static mapping trong event handler |
| Enrichment Service | Section 3.1 (enrichment_service.go) | Computed fields, business logic |
| Event Bridge (Postgres → NATS → Moleculer) | ADR-001 | Trigger cho critical tables, Poller cho non-critical |
| Data Reconciliation | 01_requirements.md Task 4 | Checksum, count, drift detection |
| DLQ & Error Handling | ADR-007 | Dead letter queue, retry 3x, replay mechanism |
| K8s Production Scaling | Section 4.1 | replicas=5, HPA, resource tuning |
| Integration Tests | Section 5.2 | End-to-end schema change workflow |
| Performance Tests | Section 5.3 | 50K events/sec throughput |

---

## Architecture Phase 2 (Full System)

```
Phase 1 (đang chạy):
  Airbyte → PostgreSQL
  CDC Worker (static mapping) → Schema Inspector → CMS → approve
  Dynamic Mapper: structure only, chưa có logic

Phase 2 (upgrade + activate):
  ┌─────────────────────────────────────────────────────────┐
  │  SOURCE DBs                                              │
  │  MongoDB (Oplog) / MySQL (Binlog)                        │
  └──────────┬──────────────────────────────────────────────┘
             │
             ▼
  ┌─────────────────────────────────────┐
  │  DEBEZIUM CONNECTORS (Activate)     │
  │  - MongoDB Source Connector          │
  │  - MySQL Source Connector            │
  └──────────┬──────────────────────────┘
             │ CDC Events (CloudEvents)
             ▼
  ┌─────────────────────────────────────┐
  │  NATS JETSTREAM                      │
  │  - cdc.goopay.{table} (active!)     │
  │  - cdc.dlq (NEW)                     │
  └──────────┬──────────────────────────┘
             │ Pull Subscribe
             ▼
  ┌──────────────────────────────────────────────────────┐
  │  CDC WORKER (Full - Phase 2)                          │
  │                                                       │
  │  ┌─────────────────────────────────────────────┐     │
  │  │  NATS Consumer Pool (10 workers/pod)         │     │
  │  │  → Fetch 1000 msgs/pull                      │     │
  │  └──────────────┬──────────────────────────────┘     │
  │                 ▼                                     │
  │  ┌─────────────────────────────────────────────┐     │
  │  │  Schema Inspector (Phase 1, already running) │     │
  │  └──────────────┬──────────────────────────────┘     │
  │                 ▼                                     │
  │  ┌─────────────────────────────────────────────┐     │
  │  │  Dynamic Mapper (FULL - Phase 2)             │     │
  │  │  - Load rules from cdc_mapping_rules         │     │
  │  │  - Build queries dynamically                 │     │
  │  │  - Hot reload on NATS event                  │     │
  │  └──────────────┬──────────────────────────────┘     │
  │                 ▼                                     │
  │  ┌─────────────────────────────────────────────┐     │
  │  │  Enrichment Service (NEW)                    │     │
  │  │  - Computed fields (balance_after calc)      │     │
  │  │  - Business logic validation                 │     │
  │  └──────────────┬──────────────────────────────┘     │
  │                 ▼                                     │
  │  ┌─────────────────────────────────────────────┐     │
  │  │  Batch Buffer (500 records / 2s timeout)     │     │
  │  │  → PostgreSQL Batch Upsert                   │     │
  │  └─────────────────────────────────────────────┘     │
  └──────────────────────────────────────────────────────┘
             │
             ▼
  ┌──────────────────────────────────────────────────────┐
  │  POSTGRESQL (Phase 1 tables, now dual-write)          │
  │  _source = 'debezium' | 'airbyte'                     │
  └──────────┬───────────────────────────────────────────┘
             │
             ▼
  ┌──────────────────────────────────────────────────────┐
  │  EVENT BRIDGE (NEW)                                   │
  │  Critical tables: Trigger + LISTEN → Go Listener      │
  │  Non-critical: Polling changelog (1-5s interval)       │
  │  → Publish NATS events for Moleculer services          │
  └──────────┬───────────────────────────────────────────┘
             │
             ▼
  ┌──────────────────────────────────────────────────────┐
  │  MOLECULER SERVICES (Node.js)                         │
  │  Subscribe NATS events → Business logic               │
  └──────────────────────────────────────────────────────┘

  ┌──────────────────────────────────────────────────────┐
  │  RECONCILIATION JOB (NEW - Cron)                      │
  │  Critical tables: every 15min (full checksum)          │
  │  Non-critical: every 1-4hr (count only)                │
  └──────────────────────────────────────────────────────┘
```

---

## Implementation Details

### 1. Dynamic Mapper Full (Upgrade from Phase 1 stub)

**Reference**: `03_implementation.md` Section 3.3 (lines 749-1051)

Code full logic cho:
- `LoadRules()` - query `cdc_mapping_rules` table, group by table
- `GetRulesForTable()` - return cached rules
- `MapData()` - extract fields theo active rules, split enriched vs normal
- `BuildUpsertQuery()` - dynamic INSERT...ON CONFLICT with parameterized values
- `convertType()` - VARCHAR, INTEGER, BIGINT, DECIMAL, BOOLEAN, TIMESTAMP, JSONB
- `StartConfigReloadListener()` - subscribe `schema.config.reload`, reload rules + invalidate cache

Replace `getKnownColumns()` static mapping trong event handler bằng `dynamicMapper.MapData()`.

### 2. CDC Worker Upgrades

**Reference**: `03_implementation.md` Section 3.4

| Aspect | Phase 1 | Phase 2 (Upgrade) |
|--------|---------|-------------------|
| Mapping | Static (hardcoded columns) | **Dynamic Mapper** (load from `cdc_mapping_rules`) |
| Enrichment | None | EnrichmentService (computed fields) |
| Error handling | Log & skip | Retry 3x → DLQ |
| Debezium | Config only | Active production connectors |
| Source tag | `_source = 'airbyte'` primary | `_source = 'debezium'` cho CDC path |

**Key upgrade**: Replace `getKnownColumns()` static mapping bằng `dynamicMapper.MapData()` - Section 3.3

### 3. Event Bridge

**Reference**: ADR-001 (Hybrid Approach)

```
Critical Tables (Trigger-based, <10ms):
  wallet_transactions, payments, orders
  → PostgreSQL NOTIFY → Go Listener → NATS publish

Non-Critical Tables (Polling-based, 1-5s):
  logs, analytics, reports
  → Go Poller reads changelog table → NATS publish (batched)
```

**New files needed**:
```
internal/
├── application/
│   └── services/
│       └── event_bridge.go            # Trigger listener + Poller
├── infrastructure/
│   └── postgres/
│       └── trigger_listener.go        # LISTEN/NOTIFY handler
└── interfaces/
    └── dto/
        └── moleculer_event.go         # Moleculer-compatible event format
```

**Moleculer event format** (ADR-002):
```json
{
  "specversion": "1.0",
  "source": "/airbyte/postgres/goopay/{table}",
  "type": "io.goopay.datachangeevent",
  "data": {
    "op": "c|u|d",
    "before": {...},
    "after": {...}
  }
}
```

### 4. Data Reconciliation

**Reference**: `01_requirements.md` Task 4, ADR-004

```
New files:
cmd/
└── reconciliation/
    └── main.go                        # Cron job entry point

internal/
└── application/
    └── services/
        └── reconciliation_service.go  # Compare source vs target
```

**Schedule** (ADR-004):

| Tier | Tables | Frequency | Method |
|------|--------|-----------|--------|
| Critical | wallet_transactions, payments | 15 min | Full checksum |
| High | users, merchants | 1 hour | Count + 10% sampling |
| Medium | logs, reports | 4 hours | Count only |
| Low | analytics | Daily | Count only |

### 5. DLQ & Error Handling

**Reference**: ADR-007

```
Event fail → Retry 3x (exponential backoff)
  → Still fail → Push to NATS stream: cdc.dlq
  → Alert DevOps (Slack/PagerDuty)
  → Admin CLI: cdc-admin replay --dlq-id=<id>
```

### 6. K8s Production Scaling

**Reference**: `03_implementation.md` Section 4.1

Phase 1 → Phase 2 changes:

| Setting | Phase 1 | Phase 2 |
|---------|---------|---------|
| replicas | 3 | 5 |
| WORKER_POOL_SIZE | 10 | 10 |
| BATCH_SIZE | 500 | 500 |
| HPA | No | Yes (CPU 70%) |
| Memory request | 256Mi | 256Mi |
| Memory limit | 512Mi | 512Mi |

### 7. Testing

**Integration test reference**: `03_implementation.md` Section 5.2
- End-to-end schema change workflow test
- Debezium → NATS → Worker → PostgreSQL flow

**Performance test reference**: `03_implementation.md` Section 5.3
- Target: 50K events/sec
- Test: 50000 events over 10 seconds

---

## Phase 2 Project Structure (Additions)

```
cdc-worker-service/
├── cmd/
│   ├── worker/
│   │   └── main.go                    # ⬆ Upgrade: use Dynamic Mapper
│   ├── cms-service/                   # (Phase 1, no change)
│   ├── event-bridge/                  # 🆕
│   │   └── main.go
│   └── reconciliation/               # 🆕
│       └── main.go
├── internal/
│   ├── application/
│   │   ├── handlers/
│   │   │   └── dynamic_event_handler.go  # ⬆ Use Dynamic Mapper instead of static
│   │   └── services/
│   │       ├── schema_inspector.go       # (Phase 1, no change)
│   │       ├── dynamic_mapper.go         # ⬆ FULL implementation (Section 3.3)
│   │       ├── enrichment_service.go     # 🆕
│   │       ├── event_bridge.go           # 🆕
│   │       └── reconciliation_service.go # 🆕
│   └── infrastructure/
│       ├── nats/
│       │   ├── consumer.go               # (Phase 1, no change)
│       │   └── client.go
│       └── postgres/
│           └── trigger_listener.go       # 🆕
├── deployments/
│   ├── k8s/
│   │   ├── cdc-worker-deployment.yaml    # ⬆ replicas=5, HPA
│   │   ├── event-bridge-deployment.yaml  # 🆕
│   │   └── reconciliation-cronjob.yaml   # 🆕
│   └── debezium/                         # ⬆ Deploy to production
│       ├── mongodb-connector.json
│       └── mysql-connector.json
└── tests/
    ├── integration/
    │   ├── cdc_worker_test.go            # 🆕 (Section 5.2)
    │   └── event_bridge_test.go          # 🆕
    └── performance/
        └── throughput_test.go            # 🆕 (Section 5.3)
```

Legend: 🆕 = New in Phase 2 | ⬆ = Upgraded from Phase 1

---

## Execution Order

```
Step 1: Dynamic Mapper Full
  └─ Code LoadRules() - load from cdc_mapping_rules
  └─ Code MapData() - extract fields theo rules
  └─ Code BuildUpsertQuery() - dynamic query builder
  └─ Code convertType() - type conversion
  └─ Code StartConfigReloadListener() - NATS hot reload
  └─ Replace getKnownColumns() static mapping trong event handler
  └─ Redis cache cho mapping rules
  └─ Unit tests

Step 2: Enrichment + DLQ
  └─ Add enrichment service (computed fields)
  └─ Add DLQ error handling (retry 3x → dead letter queue)

Step 3: Debezium Activation
  └─ Deploy Debezium connectors (MongoDB, MySQL)
  └─ Verify CDC events flowing to NATS
  └─ CDC Worker processes events → PostgreSQL

Step 4: Event Bridge
  └─ PostgreSQL triggers for critical tables
  └─ Go Listener (LISTEN/NOTIFY)
  └─ Go Poller for non-critical tables
  └─ NATS publish → Moleculer services verify

Step 5: Data Reconciliation
  └─ Implement reconciliation service
  └─ Setup cron schedule (15min/1hr/4hr/daily)
  └─ Verify Debezium vs Airbyte consistency

Step 6: Production Scaling
  └─ K8s: replicas=5, HPA, resource limits
  └─ Performance testing (50K events/sec target)
  └─ Monitoring dashboards (Grafana)

Step 7: Integration Testing
  └─ End-to-end tests
  └─ Failure scenarios (network, DB down, schema drift)
  └─ DLQ replay testing
```

---

## Checklist Phase 2

### Dynamic Mapper (Core upgrade)
- [ ] `LoadRules()` loads from `cdc_mapping_rules` table
- [ ] `MapData()` extracts fields theo active rules
- [ ] `BuildUpsertQuery()` builds dynamic INSERT...ON CONFLICT
- [ ] `convertType()` handles all types (VARCHAR, INT, DECIMAL, TIMESTAMP, JSONB, BOOLEAN)
- [ ] `StartConfigReloadListener()` subscribes `schema.config.reload`
- [ ] In-memory cache + Redis cache for rules
- [ ] Hot reload: NATS event → reload rules without restart
- [ ] Static mapping replaced in event handler
- [ ] Unit tests: dynamic_mapper_test.go pass

### CDC Worker (Upgrades)
- [ ] Enrichment service computing fields correctly
- [ ] DLQ handling: retry 3x → dead letter queue
- [ ] Source tag `_source = 'debezium'` for Debezium CDC events
- [ ] Throughput >= 5K events/sec per pod

### Debezium
- [ ] MongoDB connector deployed & streaming
- [ ] MySQL connector deployed & streaming
- [ ] CDC events flowing to NATS topics
- [ ] Events processed by CDC Worker successfully

### Event Bridge
- [ ] Triggers created on critical tables
- [ ] Go Listener receiving NOTIFY events
- [ ] Poller running for non-critical tables
- [ ] NATS events compatible with Moleculer format
- [ ] Moleculer services receiving events

### Reconciliation
- [ ] Critical tables: checksum validation every 15 min
- [ ] Drift detection alerts firing correctly
- [ ] Reconciliation reports generated

### Production
- [ ] K8s replicas=5 with HPA
- [ ] Performance test: 50K events/sec passed
- [ ] Grafana dashboards showing key metrics
- [ ] Alerting configured (Slack/PagerDuty)

### Testing
- [ ] Integration tests: end-to-end schema change workflow
- [ ] Integration tests: Debezium → NATS → Worker → PostgreSQL
- [ ] Performance tests: throughput & latency benchmarks
- [ ] DLQ replay test passed

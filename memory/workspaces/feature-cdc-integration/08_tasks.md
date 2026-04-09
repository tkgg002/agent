# Phase 1 - Jira Tasks

> **Project**: CDC Integration
> **Phase**: 1 - Airbyte Primary + Full System (config-driven)
> **Scale**: ~30 source databases (MongoDB + MySQL), ~200 tables/collections
> **Total**: ~3-4 weeks

---

## Epic: CDC-INFRA - Infrastructure Setup

### CDC-D1: PostgreSQL Infrastructure Setup
- **Type**: Task
- **Assignee**: DevOps
- **Priority**: P0
- **Estimate**: 1 day
- **Dependencies**: None
- **Description**: Provision PostgreSQL cluster cho CDC data warehouse. Hệ thống sẽ phục vụ ~200 tables từ ~30 source databases.
- **Acceptance Criteria**:
  - [ ] PostgreSQL cluster running (Primary + Read Replica)
  - [ ] Database `goopay_dw` created
  - [ ] Users configured: `airbyte_user` (write), `cdc_worker` (write), `cms_service` (write + DDL), `readonly_user`
  - [ ] DDL permission cho CMS service user (ALTER TABLE, CREATE TABLE)
  - [ ] Storage capacity cho ~200 tables + JSONB data
  - [ ] Connectivity verified from K8s pods

### CDC-D2: Airbyte Configuration
- **Type**: Task
- **Assignee**: DevOps
- **Priority**: P0
- **Estimate**: 3-5 days
- **Dependencies**: CDC-D1, CDC-M1
- **Description**: Setup Airbyte connections cho ~30 source databases → PostgreSQL. Tables được cấu hình theo `cdc_table_registry`.
- **Acceptance Criteria**:
  - [ ] Airbyte instance deployed (abctl hoặc K8s)
  - [ ] Source connectors: ~30 databases (MongoDB replica sets + MySQL binlog)
  - [ ] Destination connector: PostgreSQL (`goopay_dw`)
  - [ ] Connections created theo `cdc_table_registry` (tables có `sync_engine = 'airbyte'` hoặc `'both'`)
  - [ ] Sync schedules configured theo `sync_interval` trong registry (15min cho critical, 1hr cho non-critical)
  - [ ] Sync mode: incremental + dedup cho tất cả tables
  - [ ] First sync successful cho batch đầu tiên (~10-20 tables pilot)
  - [ ] Data verified: `_raw_data` JSONB populated, `_source = 'airbyte'`
  - [ ] Airbyte connection IDs được cập nhật vào `cdc_table_registry.airbyte_connection_id`

### CDC-D3: NATS + Redis Infrastructure
- **Type**: Task
- **Assignee**: DevOps
- **Priority**: P0
- **Estimate**: 1 day
- **Dependencies**: None
- **Description**: Setup NATS JetStream + Redis cho CDC system
- **Acceptance Criteria**:
  - [ ] NATS JetStream cluster running
  - [ ] Redis cluster running
  - [ ] NATS streams created: `cdc.goopay.>` (wildcard cho mọi table), `schema.drift.detected`, `schema.config.reload`
  - [ ] Retention policy: 7 days
  - [ ] Connectivity verified

### CDC-D4: K8s Deployment
- **Type**: Task
- **Assignee**: DevOps
- **Priority**: P1
- **Estimate**: 1-2 days
- **Dependencies**: CDC-M2, CDC-M6, CDC-F1
- **Description**: Deploy CDC Worker + CMS Service to K8s
- **Acceptance Criteria**:
  - [ ] Namespace `goopay` ready
  - [ ] Secrets created (postgres, airbyte, cms JWT)
  - [ ] ConfigMap applied
  - [ ] CDC Worker deployed (replicas=3)
  - [ ] CMS Service deployed (replicas=2)
  - [ ] Health endpoints respond 200
  - [ ] Prometheus scraping `/metrics`

### CDC-D5: Debezium Config (Init Only)
- **Type**: Task
- **Assignee**: DevOps
- **Priority**: P2
- **Estimate**: 0.5 day
- **Dependencies**: None
- **Description**: Tạo Debezium connector config templates. KHÔNG deploy production (Phase 2). Config sẽ dùng `cdc_table_registry` (tables có `sync_engine = 'debezium'` hoặc `'both'`) để xác định tables cần capture.
- **Acceptance Criteria**:
  - [ ] `mongodb-connector-template.json` created (parameterized cho source DB)
  - [ ] `mysql-connector-template.json` created (parameterized cho source DB)
  - [ ] Config tham chiếu `cdc_table_registry` cho table include list
  - [ ] Tested locally with Docker Compose (optional)

---

## Epic: CDC-DB - Database Schema & Table Registry

### CDC-M1: Database Migration + Table Registry
- **Type**: Story
- **Assignee**: Muscle (Dev)
- **Priority**: P0
- **Estimate**: 2-3 days
- **Dependencies**: CDC-D1
- **Description**: Tạo PostgreSQL schema: **Table Registry** (quản lý ~200 tables), management tables, JSONB Landing Zone template, dynamic table creation function. KHÔNG hardcode CDC tables - tất cả được driven bởi registry.
- **Acceptance Criteria**:
  - [x] Migration file `001_init_schema.sql` created *(2026-03-30)*
  - [x] **Registry table**: `cdc_table_registry` (quản lý toàn bộ ~200 tables)
    - source_db, source_type (mongodb/mysql), source_table, target_table
    - sync_engine (`airbyte` | `debezium` | `both`), sync_interval, priority
    - primary_key_field, primary_key_type
    - is_active, airbyte_connection_id, airbyte_source_id
  - [x] **Management table**: `cdc_mapping_rules` (field-level mapping per table)
  - [x] **Management table**: `pending_fields` (schema drift tracking)
  - [x] **Management table**: `schema_changes_log` (audit trail)
  - [x] **Function**: `create_cdc_table(table_name, primary_key_field, primary_key_type)` — tạo CDC table động với `_raw_data JSONB NOT NULL` + metadata columns + indexes
  - [ ] ~~**Function**: `upsert_with_jsonb_landing()`~~ — **REMOVED**: upsert handled by Go Worker directly, không cần PG function
  - [x] CDC metadata columns template: `_raw_data`, `_source`, `_synced_at`, `_version`, `_hash`, `_deleted`
  - [x] GIN index on `_raw_data` cho mọi CDC table (tạo tự động bởi `create_cdc_table`)
  - [x] Seed data: registry entries cho pilot tables (10 tables)
  - [ ] Migration runs successfully — **chưa chạy** (Docker infra chưa start)
- **Ref**: `03_implementation.md` Section 2.1-2.4

**Key Design: `cdc_table_registry`**:
```sql
CREATE TABLE cdc_table_registry (
    id SERIAL PRIMARY KEY,
    -- Source info
    source_db VARCHAR(100) NOT NULL,       -- e.g. 'goopay_main', 'goopay_wallet', 'goopay_payment'
    source_type VARCHAR(20) NOT NULL,       -- 'mongodb' | 'mysql'
    source_table VARCHAR(200) NOT NULL,     -- Original table/collection name
    target_table VARCHAR(200) NOT NULL,     -- PostgreSQL target table name
    -- Sync config
    sync_engine VARCHAR(20) DEFAULT 'airbyte',  -- 'airbyte' | 'debezium' | 'both'
    sync_interval VARCHAR(20) DEFAULT '1h',      -- '15m', '1h', '4h', '24h'
    priority VARCHAR(10) DEFAULT 'normal',       -- 'critical', 'high', 'normal', 'low'
    -- Primary key config
    primary_key_field VARCHAR(100) DEFAULT 'id',
    primary_key_type VARCHAR(50) DEFAULT 'VARCHAR(36)',
    -- Status
    is_active BOOLEAN DEFAULT TRUE,
    is_table_created BOOLEAN DEFAULT FALSE,     -- CDC table đã được tạo chưa
    -- Airbyte integration
    airbyte_connection_id VARCHAR(100),
    airbyte_source_id VARCHAR(100),
    -- Metadata
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    notes TEXT,
    UNIQUE(source_db, source_table)
);
```

---

## Epic: CDC-WORKER - CDC Worker Service

### CDC-M2: CDC Worker - Core
- **Type**: Story
- **Assignee**: Muscle (Dev)
- **Priority**: P0
- **Estimate**: 3-5 days
- **Dependencies**: CDC-M1, CDC-D3
- **Description**: Go service: NATS consumer pool + **config-driven** event handler + batch upsert to PostgreSQL. Worker hoàn toàn generic - xử lý bất kỳ table nào có trong `cdc_table_registry`, KHÔNG hardcode table/column nào.
- **Sub-tasks**:

#### CDC-M2.1: Project Scaffolding ✅
- **Type**: Sub-task
- **Estimate**: 0.5 day
- **Status**: Done (2026-03-30)
- **Description**: Go module init, directory structure (cmd/internal/pkg), config loader (NATS_URL, POSTGRES_DSN, REDIS_URL, WORKER_POOL_SIZE, BATCH_SIZE)
- **Implementation**: `centralized-data-service/` — go.mod, config/config.go (Viper), config-local.yml, Makefile

#### CDC-M2.2: Infrastructure Layer ✅
- **Type**: Sub-task
- **Estimate**: 1 day
- **Status**: Done (2026-03-30)
- **Description**: PostgreSQL connection pool, NATS JetStream client + pull subscriber, Redis client
- **Implementation**: pkgs/database/postgres.go (GORM+PG), pkgs/natsconn/nats_client.go (JetStream+EnsureStreams), pkgs/rediscache/redis_client.go

#### CDC-M2.3: NATS Consumer Pool ✅
- **Type**: Sub-task
- **Estimate**: 0.5 day
- **Status**: Done (2026-03-30)
- **Description**: Worker pool 10 goroutines/pod, fetch 1000 msgs/pull, graceful shutdown. Subscribe wildcard `cdc.goopay.>` để nhận events từ mọi table.
- **Implementation**: internal/handler/consumer_pool.go

#### CDC-M2.4: Event Handler (Config-Driven) ✅
- **Type**: Sub-task
- **Estimate**: 1-2 days
- **Status**: Done (2026-03-30)
- **Description**: Generic event handler, KHÔNG hardcode table hay columns:
- **Implementation**: internal/handler/event_handler.go, internal/service/registry_service.go
  - Parse CloudEvents JSON, extract table name từ subject/source
  - Lookup `cdc_table_registry` để biết table config (primary key field/type, target table)
  - Lookup `cdc_mapping_rules` để biết known columns cho table đó
  - Extract ID (MongoDB ObjectId + regular + numeric), SHA256 hash
  - Build upsert query dynamically từ mapping rules (mapped columns + `_raw_data` + metadata)
  - Soft delete handling
  - Fallback: nếu table chưa có mapping rules → chỉ save `_raw_data` (zero data loss)

#### CDC-M2.5: Batch Buffer ✅ (partial)
- **Type**: Sub-task
- **Estimate**: 0.5 day
- **Status**: Done (2026-03-30) — **NOTE**: hiện tại dùng single upsert loop, chưa true batch insert
- **Description**: Buffer 500 records / flush 2s timeout, PostgreSQL batch upsert. Group by table name cho efficient batch.
- **Implementation**: internal/handler/batch_buffer.go — group by table OK, flush on size/timeout OK, nhưng batchUpsert chạy từng record

#### CDC-M2.6: Health Endpoints ✅
- **Type**: Sub-task
- **Estimate**: 0.25 day
- **Status**: Done (2026-03-30)
- **Description**: `/health` + `/ready` on `:8080`
- **Implementation**: Trong internal/server/worker_server.go (Fiber endpoints)

#### CDC-M2.7: Unit Tests ✅
- **Type**: Sub-task
- **Estimate**: 0.5 day
- **Status**: Done (2026-03-31)
- **Description**: Tests: extractID, config-driven column lookup, hash calc, dynamic upsert query building.

- **Ref**: `03_implementation.md` Section 3.1, 3.4

---

## Epic: CDC-INSPECT - Schema Drift Detection

### CDC-M3: Schema Inspector
- **Type**: Story
- **Assignee**: Muscle (Dev)
- **Priority**: P0
- **Estimate**: 2-3 days
- **Dependencies**: CDC-M2
- **Description**: Go module: detect new fields in CDC events cho bất kỳ table nào, infer types, save to pending_fields, publish NATS drift alerts. Generic cho ~200 tables.
- **Sub-tasks**:

#### CDC-M3.1: InspectEvent Logic ✅
- **Type**: Sub-task
- **Estimate**: 1 day
- **Status**: Done (2026-03-30)
- **Description**: Extract fields from event, get table schema from cache/DB (`information_schema.columns`), find new fields. Hoạt động generic cho mọi table - không cần biết trước table nào.
- **Implementation**: internal/service/schema_inspector.go — InspectEvent(), getTableSchema(), findNewFields()

#### CDC-M3.2: Type Inference ✅
- **Type**: Sub-task
- **Estimate**: 0.5 day
- **Status**: Done (2026-03-30)
- **Description**: `inferDataType()`: bool→BOOLEAN, float64→INTEGER/BIGINT/DECIMAL, string→TIMESTAMP/VARCHAR/TEXT, map/array→JSONB
- **Implementation**: pkgs/utils/type_inference.go

#### CDC-M3.3: Save Pending Field ✅
- **Type**: Sub-task
- **Estimate**: 0.25 day
- **Status**: Done (2026-03-30)
- **Description**: Upsert to `pending_fields` table, increment `detection_count`, save sample_value
- **Implementation**: internal/repository/pending_field_repo.go — UpsertPendingField()

#### CDC-M3.4: Drift Alert ✅
- **Type**: Sub-task
- **Status**: Done (2026-03-30)
- **Description**: Publish to NATS `schema.drift.detected` with source_db + table name + new fields
- **Implementation**: internal/service/schema_inspector.go — publishDriftAlert()

#### CDC-M3.5: Redis Cache ✅
- **Type**: Sub-task
- **Status**: Done (2026-03-30)
- **Description**: Cache table schema in Redis per table, TTL 5 min. Cache `cdc_table_registry` config, TTL 10 min.
- **Implementation**: schema_inspector.go getTableSchema() — Redis cache 5min. Registry cache in-memory via registry_service.go

#### CDC-M3.6: Integration ✅
- **Type**: Sub-task
- **Status**: Done (2026-03-30)
- **Description**: Wire Schema Inspector into CDC Worker event handler
- **Implementation**: worker_server.go wires SchemaInspector → EventHandler

#### CDC-M3.7: Unit Tests ✅
- **Type**: Sub-task
- **Status**: Done (2026-03-31)
- **Description**: Tests: inferDataType all types, InspectEvent detect new fields.

- **Ref**: `03_implementation.md` Section 3.2

---

## Epic: CDC-MAPPER - Dynamic Mapping Engine

### CDC-M4: Dynamic Mapper (Init Only)
- **Type**: Task
- **Assignee**: Muscle (Dev)
- **Priority**: P1
- **Estimate**: 0.5 day
- **Dependencies**: None
- **Description**: Tạo struct + interfaces cho Dynamic Mapper. KHÔNG code logic (Phase 2). Stub implementations return ErrNotImplemented.
- **Acceptance Criteria**:
  - [ ] File `dynamic_mapper.go` created — ❌ **chưa tạo trong centralized-data-service** (đã tạo trong cdc-cms-service nhưng bị xoá khi tách CMS)
  - [ ] `DynamicMapper` struct defined
  - [ ] `MappedData` struct defined
  - [ ] Interfaces: `LoadRules`, `GetRulesForTable`, `MapData`, `BuildUpsertQuery`, `convertType`, `StartConfigReloadListener`
  - [ ] All methods return `ErrNotImplemented` with `// TODO Phase 2` comment
- **Ref**: `03_implementation.md` Section 3.3

---

## Epic: CDC-AIRBYTE - Airbyte Integration

### CDC-M5: Airbyte API Client
- **Type**: Story
- **Assignee**: Muscle (Dev)
- **Priority**: P0
- **Estimate**: 1 day
- **Dependencies**: CDC-D2
- **Description**: Go client cho Airbyte API. Dùng để refresh schema sau approve, trigger sync, update connection config. Phải hỗ trợ multi-source (~30 DBs).
- **Acceptance Criteria**:
  - [x] `pkg/airbyte/client.go` with `NewClient(baseURL, apiKey, logger)` *(2026-03-30, trong cdc-cms-service)*
  - [x] `RefreshSourceSchema(ctx, sourceID)` - POST discover_schema
  - [x] `UpdateConnection(ctx, connectionID, streams)` - PATCH connection
  - [x] `TriggerSync(ctx, connectionID)` - POST sync
  - [ ] `GetConnectionStatus(ctx, connectionID)` - GET connection status — **chưa implement**
  - [x] Types: `StreamConfig`, `FieldConfig`, response structs
  - [x] Bearer token auth
  - [x] Error handling + zap logging
  - [x] Lookup `airbyte_source_id` / `airbyte_connection_id` từ `cdc_table_registry` *(trong approval_service.go)*
  - [ ] Unit tests (mock HTTP) — **chưa viết**
- **Note**: Airbyte client nằm trong `cdc-cms-service/pkgs/airbyte/` (tách riêng project)
- **Ref**: `03_implementation.md` Section 3.7

---

## Epic: CDC-CMS - CMS Approval Workflow

### CDC-M6: CMS Backend API
- **Type**: Story
- **Assignee**: Muscle (Dev)
- **Priority**: P1
- **Estimate**: 4-5 days
- **Dependencies**: CDC-M1, CDC-M3, CDC-M5
- **Description**: Go/Gin API: schema change approve/reject, mapping rules CRUD, Airbyte integration, **Table Registry management** (CRUD ~200 tables, toggle sync_engine per table).
- **Sub-tasks**:

#### CDC-M6.1: Server Setup ✅
- **Status**: Done (2026-03-30)
- **Description**: ~~Go/Gin~~ **Go/Fiber** HTTP server on `:8080` — **tách riêng project `cdc-cms-service`**
- **Implementation**: cdc-cms-service/internal/server/server.go, cmd/server/main.go

#### CDC-M6.2: Repository Layer ✅
- **Status**: Done (2026-03-30)
- **Description**: PendingFieldRepository, MappingRuleRepository, SchemaChangeLogRepository, **TableRegistryRepository**
- **Implementation**: cdc-cms-service/internal/repository/*.go (4 repos, GORM)

#### CDC-M6.3: List Pending Changes ✅
- **Status**: Done (2026-03-30)
- **Description**: `GET /api/schema-changes/pending` - filter by status, source_db, table_name + pagination
- **Implementation**: cdc-cms-service/internal/api/schema_change_handler.go — GetPending()

#### CDC-M6.4: Approve Schema Change ✅
- **Status**: Done (2026-03-30)
- **Description**: `POST /api/schema-changes/:id/approve` — full transaction flow
- **Implementation**: cdc-cms-service/internal/service/approval_service.go — Approve()

#### CDC-M6.5: Reject Schema Change ✅
- **Status**: Done (2026-03-30)
- **Implementation**: approval_service.go — Reject()

#### CDC-M6.6: Mapping Rules CRUD ✅
- **Status**: Done (2026-03-30)
- **Implementation**: cdc-cms-service/internal/api/mapping_rule_handler.go — List(), Create()

#### CDC-M6.7: Table Registry CRUD ✅
- **Status**: Done (2026-03-30)
- **Implementation**: cdc-cms-service/internal/api/registry_handler.go — List(), Register(), Update(), BulkRegister(), GetStats()

#### CDC-M6.8: Schema History ✅
- **Status**: Done (2026-03-30)
- **Implementation**: schema_change_handler.go — GetHistory()

#### CDC-M6.9: Auth + Tests ⚠️ (partial)
- **Status**: JWT middleware done, unit tests **NOT STARTED**
- **Implementation**: cdc-cms-service/internal/middleware/jwt.go
- **Note per update-sytem-design.md**: Cần Auth Service riêng biệt + RBAC, hiện tại chỉ JWT middleware đơn giản

#### CDC-M6 EXTRA: Swagger Documentation ✅ (không có trong task gốc)
- **Status**: Done (2026-03-30)
- **Description**: Swagger annotations cho tất cả 13 endpoints, swaggo/swag generated docs, Swagger UI tại /swagger/*
- **Implementation**: cdc-cms-service/docs/*.go, swagger.json, swagger.yaml

- **Ref**: `03_implementation.md` Section 3.5

### CDC-F1: CMS Frontend
- **Type**: Story
- **Assignee**: Frontend Dev
- **Priority**: P1
- **Estimate**: 3-4 days
- **Dependencies**: CDC-M6
- **Description**: React + Ant Design UI: pending changes table, approval modal, dashboard, **Table Registry manager**
- **Sub-tasks**:

#### CDC-F1.1: Project Setup
- **Type**: Sub-task
- **Estimate**: 0.25 day
- **Description**: React + Ant Design + axios + routing

#### CDC-F1.2: PendingChangesTable
- **Type**: Sub-task
- **Estimate**: 1 day
- **Description**: Table (source_db, table_name, field_name, sample_value, suggested_type, detected_at, detection_count, status), filters (source_db, table, status), Approve/Reject buttons, auto-refresh 30s, pagination

#### CDC-F1.3: ApprovalModal
- **Type**: Sub-task
- **Estimate**: 0.5 day
- **Description**: Field info display, inputs (target_column_name, final_type dropdown, approval_notes), submit POST approve, loading + error states

#### CDC-F1.4: Reject Modal
- **Type**: Sub-task
- **Estimate**: 0.25 day
- **Description**: TextArea rejection_reason, submit POST reject

#### CDC-F1.5: Dashboard
- **Type**: Sub-task
- **Estimate**: 0.5 day
- **Description**: Summary stats: pending count, approved today, tables with drift, total registered tables, breakdown by source_db + sync_engine

#### CDC-F1.6: TableRegistryManager
- **Type**: Sub-task
- **Estimate**: 1 day
- **Description**: Table Registry UI:
  - Table list: source_db, source_table, target_table, sync_engine, sync_interval, priority, is_active, is_table_created
  - Filters: source_db dropdown, sync_engine, priority, is_active toggle
  - Inline edit: switch sync_engine (airbyte↔debezium↔both), change priority, toggle active
  - Register new table form
  - Bulk import (CSV/JSON upload)
  - Stats cards: total by engine, by priority

#### CDC-F1.7: MappingRulesManager (Optional)
- **Type**: Sub-task
- **Estimate**: 0.5 day
- **Description**: Table view + create form for mapping rules, filter by source_db + table

- **Ref**: `03_implementation.md` Section 3.6

---

## Epic: CDC-OPS - Monitoring & Deployment

### CDC-M7: Monitoring + Docker + K8s Manifests
- **Type**: Story
- **Assignee**: Muscle (Dev)
- **Priority**: P1
- **Estimate**: 1 day
- **Dependencies**: CDC-M2
- **Description**: Prometheus metrics, Dockerfiles, K8s manifests
- **Acceptance Criteria**:
  - [ ] Prometheus metrics: `cdc_events_processed_total` (labels: source_db, table, operation, status), `cdc_processing_duration_seconds`, `schema_drift_detected_total`, `mapping_rules_loaded`, `pending_fields_count`, `registered_tables_total`
  - [ ] `/metrics` endpoint on Worker + CMS
  - [ ] `Dockerfile.worker` (multi-stage Go build)
  - [ ] `Dockerfile.cms` (Go backend + React static)
  - [ ] K8s: `cdc-worker-deployment.yaml` (replicas=3)
  - [ ] K8s: `cms-deployment.yaml` (replicas=2)
  - [ ] K8s: `configmap.yaml`
- **Ref**: `03_implementation.md` Section 4, 6

---

## Epic: CDC-QA - Testing & Validation

### CDC-M8: Integration Test (End-to-End)
- **Type**: Story
- **Assignee**: Muscle (Dev)
- **Priority**: P1
- **Estimate**: 1-2 days
- **Dependencies**: All tasks above
- **Description**: End-to-end validation với dynamic table setup (không hardcode table cụ thể)
- **Acceptance Criteria**:
  - [ ] Docker Compose full stack runs locally
  - [ ] Register test table qua CMS API → `create_cdc_table()` auto creates PostgreSQL table
  - [ ] Airbyte sync → data appears in PostgreSQL (dynamic table)
  - [ ] CDC Worker receives NATS event → config-driven mapping → upsert OK
  - [ ] Schema Inspector detects new field → saved in `pending_fields`
  - [ ] CMS approve → ALTER TABLE → mapping rule created
  - [ ] NATS `schema.config.reload` published
  - [ ] `_raw_data` always contains full JSON
  - [ ] Test with both MongoDB source + MySQL source
  - [ ] Test table registry: register → create → sync → detect drift → approve
  - [ ] Test toggle sync_engine (airbyte→debezium) updates registry correctly
  - [ ] Report results for Brain review

---

## Epic: CDC-REVIEW - Architecture Review

### CDC-B1: Architecture Review & Approve
- **Type**: Task
- **Assignee**: Brain (Tech Lead)
- **Priority**: P0
- **Estimate**: Ongoing
- **Dependencies**: None
- **Description**: Review key deliverables before merge
- **Acceptance Criteria**:
  - [ ] Review `cdc_table_registry` design: có scale tốt cho ~200 tables không?
  - [ ] Review `create_cdc_table()` function: dynamic table creation đúng chuẩn
  - [ ] Review CDC Worker config-driven handler: generic cho mọi table
  - [ ] Review Schema Inspector: hoạt động đúng với unknown tables
  - [ ] Review CMS approve flow transaction safety (CDC-M6)
  - [ ] Review Table Registry API: CRUD + bulk import đủ nhu cầu
  - [ ] Approve table classification strategy (sync_engine + priority per table)

### CDC-B2: Coordination & Sign-off
- **Type**: Task
- **Assignee**: Brain (Tech Lead)
- **Priority**: P0
- **Estimate**: Ongoing
- **Dependencies**: None
- **Description**: Coordinate across roles, final sign-off
- **Acceptance Criteria**:
  - [ ] DevOps infra (D1, D3) done before Muscle starts M2
  - [ ] Airbyte config (D2) parallel with Muscle coding, phối hợp từng batch source DBs
  - [ ] Table registry seed data chuẩn bị song song (DevOps cung cấp list ~200 tables)
  - [ ] CMS Backend (M6) done before Frontend (F1)
  - [ ] Integration test (M8) sign-off

---

## Summary

| ID | Title | Type | Assignee | Priority | Estimate | Depends |
|----|-------|------|----------|----------|----------|---------|
| CDC-D1 | PostgreSQL Infrastructure | Task | DevOps | P0 | 1d | - |
| CDC-D2 | Airbyte Configuration (~30 DBs) | Task | DevOps | P0 | 3-5d | D1, M1 |
| CDC-D3 | NATS + Redis Infrastructure | Task | DevOps | P0 | 1d | - |
| CDC-D4 | K8s Deployment | Task | DevOps | P1 | 1-2d | M2, M6, F1 |
| CDC-D5 | Debezium Config (Init) | Task | DevOps | P2 | 0.5d | - |
| CDC-M1 | Database Migration + Table Registry | Story | Dev | P0 | 2-3d | D1 |
| CDC-M2 | CDC Worker Core (Config-Driven) | Story | Dev | P0 | 3-5d | M1, D3 |
| CDC-M3 | Schema Inspector | Story | Dev | P0 | 2-3d | M2 |
| CDC-M4 | Dynamic Mapper (Init) | Task | Dev | P1 | 0.5d | - |
| CDC-M5 | Airbyte API Client (Multi-Source) | Story | Dev | P0 | 1d | D2 |
| CDC-M6 | CMS Backend API + Registry CRUD | Story | Dev | P1 | 4-5d | M1, M3, M5 |
| CDC-M7 | Monitoring + Docker | Story | Dev | P1 | 1d | M2 |
| CDC-M8 | Integration Test | Story | Dev | P1 | 1-2d | All |
| CDC-F1 | CMS Frontend + Registry UI | Story | Frontend | P1 | 3-4d | M6 |
| CDC-B1 | Architecture Review | Task | Brain | P0 | Ongoing | - |
| CDC-B2 | Coordination & Sign-off | Task | Brain | P0 | Ongoing | - |

---

## Appendix: Scale & Configuration Model

### Hệ thống target

```
~30 Source Databases
├── MongoDB instances (~20 DBs)
│   ├── goopay_main (users, merchants, ...)
│   ├── goopay_wallet (wallet_transactions, wallets, ...)
│   ├── goopay_payment (payments, refunds, ...)
│   ├── goopay_order (orders, order_items, ...)
│   └── ... (~16 more)
└── MySQL instances (~10 DBs)
    ├── goopay_legacy (legacy_payments, ...)
    ├── goopay_report (reports, analytics, ...)
    └── ... (~8 more)

~200 Tables/Collections total
```

### Configuration Model

Mỗi table trong hệ thống được quản lý qua `cdc_table_registry`:

```
┌─────────────────────────────────────────────────────────┐
│                  cdc_table_registry                       │
│                                                           │
│  source_db: goopay_wallet                                │
│  source_type: mongodb                                     │
│  source_table: wallet_transactions                        │
│  target_table: wallet_transactions                        │
│  sync_engine: airbyte          ← Phase 1: airbyte        │
│               debezium         ← Phase 2: switch          │
│               both             ← hoặc chạy cả hai        │
│  sync_interval: 15m                                       │
│  priority: critical                                       │
│  primary_key_field: _id                                   │
│  primary_key_type: VARCHAR(36)                            │
│  is_active: true                                          │
│  is_table_created: true                                   │
│  airbyte_connection_id: conn_abc123                       │
│  airbyte_source_id: src_xyz789                            │
└─────────────────────────────────────────────────────────┘
```

### Không hardcode nguyên tắc

| Aspect | Cũ (hardcoded) | Mới (config-driven) |
|--------|----------------|---------------------|
| CDC tables | 3 tables cố định | ~200 tables từ registry |
| Column mapping | `getKnownColumns()` switch/case | `cdc_mapping_rules` từ DB |
| Sync engine | Airbyte cho tất cả | Per-table: airbyte / debezium / both |
| Table creation | Hardcode CREATE TABLE | `create_cdc_table()` function |
| Primary key | Luôn `id VARCHAR(36)` | Configurable per table |
| Sync schedule | 15min / 1hr cố định | Per-table `sync_interval` |
| Source DB | 1 MongoDB + 1 MySQL | ~30 databases |

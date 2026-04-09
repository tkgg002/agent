# Phase 1 - Task Solutions (Chi tiết giải pháp)

> **Project**: CDC Integration
> **Phase**: 1 - Airbyte Primary + Full System (config-driven)
> **Scale**: ~30 source databases (MongoDB + MySQL), ~200 tables/collections
> **Document**: Giải pháp chi tiết cho từng task trong `08_tasks.md`
> **Created**: 2026-03-25
> **Updated**: 2026-03-26 - Chuyển sang config-driven, table registry, generic cho ~200 tables

---

## Mục lục

1. [CDC-D1: PostgreSQL Infrastructure Setup](#cdc-d1-postgresql-infrastructure-setup)
2. [CDC-D2: Airbyte Configuration (~30 DBs)](#cdc-d2-airbyte-configuration-30-dbs)
3. [CDC-D3: NATS + Redis Infrastructure](#cdc-d3-nats--redis-infrastructure)
4. [CDC-D4: K8s Deployment](#cdc-d4-k8s-deployment)
5. [CDC-D5: Debezium Config (Init Only)](#cdc-d5-debezium-config-init-only)
6. [CDC-M1: Database Migration + Table Registry](#cdc-m1-database-migration--table-registry)
7. [CDC-M2: CDC Worker Core (Config-Driven)](#cdc-m2-cdc-worker-core-config-driven)
8. [CDC-M3: Schema Inspector](#cdc-m3-schema-inspector)
9. [CDC-M4: Dynamic Mapper (Init Only)](#cdc-m4-dynamic-mapper-init-only)
10. [CDC-M5: Airbyte API Client (Multi-Source)](#cdc-m5-airbyte-api-client-multi-source)
11. [CDC-M6: CMS Backend API + Registry CRUD](#cdc-m6-cms-backend-api--registry-crud)
12. [CDC-F1: CMS Frontend + Registry UI](#cdc-f1-cms-frontend--registry-ui)
13. [CDC-M7: Monitoring + Docker + K8s Manifests](#cdc-m7-monitoring--docker--k8s-manifests)
14. [CDC-M8: Integration Test (End-to-End)](#cdc-m8-integration-test-end-to-end)
15. [CDC-B1: Architecture Review & Approve](#cdc-b1-architecture-review--approve)
16. [CDC-B2: Coordination & Sign-off](#cdc-b2-coordination--sign-off)

---

## CDC-D1: PostgreSQL Infrastructure Setup

### Mục tiêu
Provision PostgreSQL cluster cho CDC data warehouse phục vụ ~200 tables từ ~30 source databases.

### Giải pháp chi tiết

#### 1. PostgreSQL Cluster Setup

**Topology**: Primary + Read Replica

```
┌────────────────────┐     ┌────────────────────┐
│  PostgreSQL Primary│────▶│  PostgreSQL Replica │
│  (Read/Write)      │     │  (Read Only)        │
│  Port: 5432        │     │  Port: 5433         │
└────────────────────┘     └────────────────────┘
```

**Option A: Managed Service (Khuyến nghị cho Production)**
- AWS RDS PostgreSQL 15 hoặc Google Cloud SQL
- Instance type: `db.r6g.xlarge` (4 vCPU, 32GB RAM) — scale lớn hơn cho ~200 tables + JSONB
- Storage: 500GB gp3 SSD, auto-scaling enabled (dự trù JSONB data ~200 tables)
- Multi-AZ deployment cho HA
- Automated backups: 7 ngày retention

**Option B: Self-hosted trên K8s**
- CloudNativePG operator hoặc Zalando Postgres Operator
- Helm chart deploy với StatefulSet
- PersistentVolumeClaim: 500GB+

#### 2. Database Creation

```sql
CREATE DATABASE goopay_dw
    ENCODING = 'UTF8'
    LC_COLLATE = 'en_US.UTF-8'
    LC_CTYPE = 'en_US.UTF-8';
```

#### 3. User & Permissions

```sql
-- User cho Airbyte (write CDC data từ ~30 sources)
CREATE USER airbyte_user WITH PASSWORD '<strong_password>';
GRANT CONNECT ON DATABASE goopay_dw TO airbyte_user;
GRANT USAGE ON SCHEMA public TO airbyte_user;
GRANT SELECT, INSERT, UPDATE ON ALL TABLES IN SCHEMA public TO airbyte_user;
ALTER DEFAULT PRIVILEGES IN SCHEMA public
    GRANT SELECT, INSERT, UPDATE ON TABLES TO airbyte_user;

-- User cho CDC Worker service
CREATE USER cdc_worker WITH PASSWORD '<strong_password>';
GRANT CONNECT ON DATABASE goopay_dw TO cdc_worker;
GRANT USAGE ON SCHEMA public TO cdc_worker;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO cdc_worker;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO cdc_worker;
ALTER DEFAULT PRIVILEGES IN SCHEMA public
    GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO cdc_worker;
ALTER DEFAULT PRIVILEGES IN SCHEMA public
    GRANT USAGE, SELECT ON SEQUENCES TO cdc_worker;

-- User cho CMS service (cần DDL: ALTER TABLE + CREATE TABLE cho dynamic table creation)
CREATE USER cms_service WITH PASSWORD '<strong_password>';
GRANT CONNECT ON DATABASE goopay_dw TO cms_service;
GRANT USAGE, CREATE ON SCHEMA public TO cms_service;
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO cms_service;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO cms_service;
ALTER DEFAULT PRIVILEGES IN SCHEMA public
    GRANT ALL PRIVILEGES ON TABLES TO cms_service;

-- Read-only user cho analytics/monitoring
CREATE USER readonly_user WITH PASSWORD '<strong_password>';
GRANT CONNECT ON DATABASE goopay_dw TO readonly_user;
GRANT USAGE ON SCHEMA public TO readonly_user;
GRANT SELECT ON ALL TABLES IN SCHEMA public TO readonly_user;
ALTER DEFAULT PRIVILEGES IN SCHEMA public
    GRANT SELECT ON TABLES TO readonly_user;
```

#### 4. PostgreSQL Configuration Tuning

```ini
# postgresql.conf — tuned cho ~200 tables + heavy JSONB writes
shared_buffers = 8GB                    # 25% of 32GB RAM
effective_cache_size = 24GB             # 75% of RAM
work_mem = 64MB
maintenance_work_mem = 2GB              # Cho VACUUM nhiều tables
wal_buffers = 64MB
checkpoint_timeout = 15min
max_wal_size = 8GB
min_wal_size = 2GB

max_connections = 300                   # Nhiều connections cho ~30 Airbyte sources + worker pool
idle_in_transaction_session_timeout = 60s

synchronous_commit = on
wal_level = replica
max_wal_senders = 5

jit = on
random_page_cost = 1.1                 # SSD
```

#### 5. Connectivity Verification

```bash
# Test từ K8s pod
kubectl run pg-test --rm -it --image=postgres:15 --restart=Never -- \
  psql "postgresql://cdc_worker:<pw>@postgres-primary.goopay.svc:5432/goopay_dw" \
  -c "SELECT version();"

# Test DDL + CREATE TABLE cho CMS service
kubectl run pg-test --rm -it --image=postgres:15 --restart=Never -- \
  psql "postgresql://cms_service:<pw>@postgres-primary.goopay.svc:5432/goopay_dw" \
  -c "CREATE TABLE _test_ddl (id INT); DROP TABLE _test_ddl;"
```

---

## CDC-D2: Airbyte Configuration (~30 DBs)

### Mục tiêu
Setup Airbyte connections cho ~30 source databases → PostgreSQL. Tables được cấu hình theo `cdc_table_registry`.

### Giải pháp chi tiết

#### 1. Airbyte Deployment

```bash
# K8s deployment (recommended cho scale lớn)
helm repo add airbyte https://airbytehq.github.io/helm-charts
helm install airbyte airbyte/airbyte \
  --namespace goopay \
  --set global.database.host=postgres-primary.goopay.svc \
  --set global.database.port=5432 \
  --set worker.replicaCount=4          # Nhiều workers cho ~30 sources
```

#### 2. Source Connectors (~30 databases)

Tạo source connectors theo batch, mỗi source DB 1 connector:

**MongoDB Sources (~20)**:

```json
// Template cho mỗi MongoDB source
{
  "sourceType": "mongodb-v2",
  "name": "mongo-{source_db_name}",
  "connectionConfiguration": {
    "instance_type": {
      "instance": "replica_set",
      "server_addresses": "{host}:{port}",
      "replica_set": "rs0"
    },
    "database": "{database_name}",
    "auth_source": "admin",
    "user": "<user>",
    "password": "<password>",
    "schema_enforced": false
  }
}
```

Danh sách MongoDB sources:

| # | Source Name | Database | Collections (approx) |
|---|-----------|----------|---------------------|
| 1 | mongo-goopay-main | goopay_main | ~15 |
| 2 | mongo-goopay-wallet | goopay_wallet | ~8 |
| 3 | mongo-goopay-payment | goopay_payment | ~10 |
| 4 | mongo-goopay-order | goopay_order | ~12 |
| ... | ... | ... | ... |
| 20 | mongo-goopay-xxx | goopay_xxx | ~N |

**MySQL Sources (~10)**:

```json
{
  "sourceType": "mysql",
  "name": "mysql-{source_db_name}",
  "connectionConfiguration": {
    "host": "{host}",
    "port": 13306,
    "database": "{database_name}",
    "username": "<user>",
    "password": "<password>",
    "replication_method": {
      "method": "CDC",
      "server_id": "{unique_server_id}",
      "initial_waiting_seconds": 300
    }
  }
}
```

#### 3. Destination Connector: PostgreSQL (shared)

```json
{
  "destinationType": "postgres",
  "name": "pg-goopay-dw",
  "connectionConfiguration": {
    "host": "postgres-primary.goopay.svc",
    "port": 5432,
    "database": "goopay_dw",
    "schema": "public",
    "username": "airbyte_user",
    "password": "<password>"
  }
}
```

#### 4. Connections — driven bởi `cdc_table_registry`

Mỗi table trong registry có `sync_engine IN ('airbyte', 'both')` sẽ được tạo 1 Airbyte connection (hoặc grouped theo source DB).

**Grouping strategy**: 1 connection per source DB, chứa tất cả tables/collections của DB đó.

```
Connection: mongo-goopay-wallet → pg-goopay-dw
  Streams:
    - wallet_transactions (incremental + dedup, 15min)
    - wallets (incremental + dedup, 1hr)
    - wallet_history (incremental + append, 4hr)
    ...

Connection: mysql-goopay-legacy → pg-goopay-dw
  Streams:
    - legacy_payments (incremental + dedup, 1hr)
    - legacy_refunds (incremental + dedup, 1hr)
    ...
```

**Schedule theo `priority` trong registry**:

| Priority | sync_interval | Tables (approx) |
|----------|--------------|-----------------|
| critical | 15m | ~20 tables |
| high | 30m | ~30 tables |
| normal | 1h | ~100 tables |
| low | 4h-24h | ~50 tables |

#### 5. Rollout Plan (batched)

Không setup ~30 DBs cùng lúc. Chia batch:

| Batch | When | Sources | Tables |
|-------|------|---------|--------|
| Pilot | Week 1 | 3-5 critical DBs | ~20-30 tables |
| Batch 2 | Week 2 | +10 DBs | +60-80 tables |
| Batch 3 | Week 3 | +remaining | +remaining |

#### 6. Post-sync: Update Registry

Sau khi tạo connections, cập nhật `cdc_table_registry`:

```sql
UPDATE cdc_table_registry SET
    airbyte_connection_id = '<connection_id>',
    airbyte_source_id = '<source_id>'
WHERE source_db = 'goopay_wallet'
  AND sync_engine IN ('airbyte', 'both');
```

#### 7. Verification

```sql
-- Verify data from any synced table
SELECT COUNT(*) as total,
       COUNT(_raw_data) as has_raw_data,
       COUNT(CASE WHEN _source = 'airbyte' THEN 1 END) as airbyte_source
FROM {any_cdc_table};

-- Verify registry sync status
SELECT source_db, COUNT(*) as tables,
       COUNT(airbyte_connection_id) as connected
FROM cdc_table_registry
WHERE sync_engine IN ('airbyte', 'both')
GROUP BY source_db;
```

---

## CDC-D3: NATS + Redis Infrastructure

### Mục tiêu
Setup NATS JetStream + Redis cho CDC system messaging và caching (~200 tables).

### Giải pháp chi tiết

#### 1. NATS JetStream Deployment

```bash
helm repo add nats https://nats-io.github.io/k8s/helm/charts/
helm install nats nats/nats \
  --namespace goopay \
  --set nats.jetstream.enabled=true \
  --set nats.jetstream.memStorage.size=2Gi \
  --set nats.jetstream.fileStorage.size=50Gi \
  --set cluster.enabled=true \
  --set cluster.replicas=3
```

#### 2. NATS Streams & Subjects

```bash
# Stream: CDC Events — wildcard cho mọi source_db + table
nats stream add CDC_EVENTS \
  --subjects "cdc.goopay.>" \
  --retention limits \
  --max-age 7d \
  --max-bytes 50GB \
  --storage file \
  --replicas 3 \
  --discard old

# Stream: Schema Drift Alerts
nats stream add SCHEMA_DRIFT \
  --subjects "schema.drift.detected" \
  --retention limits \
  --max-age 7d \
  --storage file \
  --replicas 3

# Stream: Config Reload Events
nats stream add SCHEMA_CONFIG \
  --subjects "schema.config.reload" \
  --retention limits \
  --max-age 1d \
  --storage file \
  --replicas 3

# Consumer cho CDC Worker (pull-based, durable)
nats consumer add CDC_EVENTS cdc-worker-group \
  --filter "cdc.goopay.>" \
  --deliver all \
  --ack explicit \
  --max-deliver 5 \
  --max-pending 1000 \
  --pull
```

**Subject naming convention** (dynamic, bất kỳ table nào):

```
cdc.goopay.{source_db}.{table_name}
  ví dụ: cdc.goopay.goopay_wallet.wallet_transactions
         cdc.goopay.goopay_payment.payments
         cdc.goopay.goopay_legacy.legacy_refunds

schema.drift.detected             → Schema drift alerts (mọi table)
schema.config.reload              → Config reload triggers
```

#### 3. Redis Deployment

```bash
helm install redis bitnami/redis \
  --namespace goopay \
  --set architecture=standalone \
  --set auth.enabled=true \
  --set master.resources.requests.memory=512Mi \
  --set master.persistence.size=10Gi
```

**Redis key patterns** (scale cho ~200 tables):

| Key Pattern | Value | TTL | Purpose |
|------------|-------|-----|---------|
| `schema:{target_table}` | `{"col1":true,...}` | 5 min | Cache table columns |
| `registry:{target_table}` | JSON table config | 10 min | Cache registry entry |
| `mapping:{target_table}` | JSON mapping rules | 10 min | Cache mapping rules |
| `drift:lock:{table}:{field}` | lock token | 1 min | Dedup drift alerts |

#### 4. Connectivity Verification

```bash
# NATS
kubectl run nats-test --rm -it --image=natsio/nats-box --restart=Never -- \
  nats stream ls --server nats://nats.goopay.svc:4222

# Redis
kubectl run redis-test --rm -it --image=redis:7-alpine --restart=Never -- \
  redis-cli -h redis-master.goopay.svc -a <pw> ping
```

---

## CDC-D4: K8s Deployment

### Mục tiêu
Deploy CDC Worker + CMS Service lên Kubernetes.

### Giải pháp chi tiết

#### 1. Namespace & Secrets

```bash
kubectl create namespace goopay

kubectl create secret generic postgres-secret \
  --namespace goopay \
  --from-literal=dsn="postgresql://cdc_worker:<pw>@postgres-primary.goopay.svc:5432/goopay_dw?sslmode=require"

kubectl create secret generic postgres-cms-secret \
  --namespace goopay \
  --from-literal=dsn="postgresql://cms_service:<pw>@postgres-primary.goopay.svc:5432/goopay_dw?sslmode=require"

kubectl create secret generic airbyte-secret \
  --namespace goopay \
  --from-literal=api_key="<airbyte_api_key>"

kubectl create secret generic cms-secret \
  --namespace goopay \
  --from-literal=jwt_secret="<jwt_secret_key>"
```

#### 2. ConfigMap

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: cdc-config
  namespace: goopay
data:
  nats_url: "nats://nats.goopay.svc.cluster.local:4222"
  redis_url: "redis://:password@redis-master.goopay.svc.cluster.local:6379"
  airbyte_api_url: "http://airbyte-server.goopay.svc.cluster.local:8001"
```

#### 3. CDC Worker Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: cdc-worker
  namespace: goopay
spec:
  replicas: 3
  selector:
    matchLabels:
      app: cdc-worker
  strategy:
    rollingUpdate:
      maxSurge: 1
      maxUnavailable: 0
  template:
    metadata:
      labels:
        app: cdc-worker
      annotations:
        prometheus.io/scrape: "true"
        prometheus.io/port: "8080"
        prometheus.io/path: "/metrics"
    spec:
      containers:
      - name: cdc-worker
        image: goopay/cdc-worker:latest
        env:
        - name: NATS_URL
          valueFrom:
            configMapKeyRef: { name: cdc-config, key: nats_url }
        - name: POSTGRES_DSN
          valueFrom:
            secretKeyRef: { name: postgres-secret, key: dsn }
        - name: REDIS_URL
          valueFrom:
            configMapKeyRef: { name: cdc-config, key: redis_url }
        - name: WORKER_POOL_SIZE
          value: "10"
        - name: BATCH_SIZE
          value: "500"
        - name: BATCH_TIMEOUT_SECONDS
          value: "2"
        ports:
        - containerPort: 8080
        resources:
          requests: { memory: "256Mi", cpu: "500m" }
          limits: { memory: "512Mi", cpu: "1000m" }
        livenessProbe:
          httpGet: { path: /health, port: 8080 }
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet: { path: /ready, port: 8080 }
          initialDelaySeconds: 10
          periodSeconds: 5
---
apiVersion: v1
kind: Service
metadata:
  name: cdc-worker
  namespace: goopay
spec:
  selector:
    app: cdc-worker
  ports:
  - { name: http, port: 8080, targetPort: 8080 }
  type: ClusterIP
```

#### 4. CMS Service Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: cdc-cms
  namespace: goopay
spec:
  replicas: 2
  selector:
    matchLabels:
      app: cdc-cms
  template:
    metadata:
      labels:
        app: cdc-cms
      annotations:
        prometheus.io/scrape: "true"
        prometheus.io/port: "8081"
    spec:
      containers:
      - name: cms-backend
        image: goopay/cdc-cms:latest
        env:
        - name: POSTGRES_DSN
          valueFrom:
            secretKeyRef: { name: postgres-cms-secret, key: dsn }
        - name: NATS_URL
          valueFrom:
            configMapKeyRef: { name: cdc-config, key: nats_url }
        - name: AIRBYTE_API_URL
          valueFrom:
            configMapKeyRef: { name: cdc-config, key: airbyte_api_url }
        - name: AIRBYTE_API_KEY
          valueFrom:
            secretKeyRef: { name: airbyte-secret, key: api_key }
        - name: JWT_SECRET
          valueFrom:
            secretKeyRef: { name: cms-secret, key: jwt_secret }
        ports:
        - containerPort: 8081
        resources:
          requests: { memory: "128Mi", cpu: "250m" }
          limits: { memory: "256Mi", cpu: "500m" }
---
apiVersion: v1
kind: Service
metadata:
  name: cdc-cms
  namespace: goopay
spec:
  selector:
    app: cdc-cms
  ports:
  - { name: http, port: 8081, targetPort: 8081 }
  type: LoadBalancer
```

---

## CDC-D5: Debezium Config (Init Only)

### Mục tiêu
Tạo Debezium connector config **templates** parameterized cho mọi source DB. KHÔNG deploy production (Phase 2).

### Giải pháp chi tiết

#### 1. MongoDB Connector Template

```json
// deployments/debezium/mongodb-connector-template.json
// Parameterized: ${SOURCE_DB}, ${HOST}, ${PORT}, ${REPLICA_SET}, ${TABLE_INCLUDE_LIST}
{
  "name": "goopay-mongo-${SOURCE_DB}",
  "config": {
    "connector.class": "io.debezium.connector.mongodb.MongoDbConnector",
    "mongodb.connection.string": "mongodb://debezium_user:<pw>@${HOST}:${PORT}/?replicaSet=${REPLICA_SET}&authSource=admin",
    "topic.prefix": "cdc.goopay.${SOURCE_DB}",
    "database": "${SOURCE_DB}",
    "collection.include.list": "${TABLE_INCLUDE_LIST}",

    "capture.mode": "change_streams_update_full_with_pre_image",
    "snapshot.mode": "initial",

    "key.converter": "org.apache.kafka.connect.json.JsonConverter",
    "key.converter.schemas.enable": false,
    "value.converter": "org.apache.kafka.connect.json.JsonConverter",
    "value.converter.schemas.enable": false,

    "transforms": "unwrap",
    "transforms.unwrap.type": "io.debezium.connector.mongodb.transforms.ExtractNewDocumentState",
    "transforms.unwrap.drop.tombstones": false,
    "transforms.unwrap.delete.handling.mode": "rewrite",

    "heartbeat.interval.ms": 10000,
    "errors.tolerance": "all",
    "errors.log.enable": true
  }
}
```

**Sử dụng**: Script đọc `cdc_table_registry` WHERE `sync_engine IN ('debezium', 'both')`, group by `source_db`, render template:

```bash
# generate-debezium-config.sh
# Query registry → build TABLE_INCLUDE_LIST per source_db
# Render template → output to deployments/debezium/generated/
```

#### 2. MySQL Connector Template

```json
// deployments/debezium/mysql-connector-template.json
{
  "name": "goopay-mysql-${SOURCE_DB}",
  "config": {
    "connector.class": "io.debezium.connector.mysql.MySqlConnector",
    "database.hostname": "${HOST}",
    "database.port": "${PORT}",
    "database.user": "debezium_user",
    "database.password": "<pw>",
    "database.server.id": "${SERVER_ID}",
    "topic.prefix": "cdc.goopay.${SOURCE_DB}",
    "database.include.list": "${SOURCE_DB}",
    "table.include.list": "${TABLE_INCLUDE_LIST}",

    "include.schema.changes": true,
    "snapshot.mode": "initial",

    "key.converter": "org.apache.kafka.connect.json.JsonConverter",
    "key.converter.schemas.enable": false,
    "value.converter": "org.apache.kafka.connect.json.JsonConverter",
    "value.converter.schemas.enable": false,

    "transforms": "unwrap",
    "transforms.unwrap.type": "io.debezium.transforms.ExtractNewRecordState",
    "transforms.unwrap.drop.tombstones": false,
    "transforms.unwrap.delete.handling.mode": "rewrite",

    "heartbeat.interval.ms": 10000,
    "errors.tolerance": "all"
  }
}
```

---

## CDC-M1: Database Migration + Table Registry

### Mục tiêu
Tạo PostgreSQL schema: **Table Registry** (quản lý ~200 tables), management tables, dynamic table creation function. KHÔNG hardcode bất kỳ CDC table nào.

### Giải pháp chi tiết

#### Migration File: `001_init_schema.sql`

```sql
BEGIN;

-- ============================================================
-- 1. TABLE REGISTRY — quản lý toàn bộ ~200 tables
-- ============================================================

CREATE TABLE IF NOT EXISTS cdc_table_registry (
    id SERIAL PRIMARY KEY,

    -- Source info
    source_db VARCHAR(100) NOT NULL,        -- 'goopay_main', 'goopay_wallet', ...
    source_type VARCHAR(20) NOT NULL,        -- 'mongodb' | 'mysql'
    source_table VARCHAR(200) NOT NULL,      -- Original table/collection name
    target_table VARCHAR(200) NOT NULL,      -- PostgreSQL target table name

    -- Sync config
    sync_engine VARCHAR(20) NOT NULL DEFAULT 'airbyte',
        -- 'airbyte': chỉ sync qua Airbyte (Phase 1 default)
        -- 'debezium': chỉ sync qua Debezium (Phase 2)
        -- 'both': chạy cả hai (transition period)
    sync_interval VARCHAR(20) DEFAULT '1h',  -- '15m', '30m', '1h', '4h', '24h'
    priority VARCHAR(10) DEFAULT 'normal',   -- 'critical', 'high', 'normal', 'low'

    -- Primary key config (mỗi table có thể khác nhau)
    primary_key_field VARCHAR(100) DEFAULT 'id',       -- '_id' cho MongoDB, 'id' cho MySQL
    primary_key_type VARCHAR(50) DEFAULT 'VARCHAR(36)', -- 'VARCHAR(36)', 'VARCHAR(24)', 'BIGINT', ...

    -- Status
    is_active BOOLEAN DEFAULT TRUE,
    is_table_created BOOLEAN DEFAULT FALSE,  -- CDC table đã được tạo trong PostgreSQL chưa

    -- Airbyte integration
    airbyte_connection_id VARCHAR(100),
    airbyte_source_id VARCHAR(100),

    -- Metadata
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    notes TEXT,

    UNIQUE(source_db, source_table),
    CONSTRAINT ctr_check_source_type CHECK (source_type IN ('mongodb', 'mysql', 'postgresql')),
    CONSTRAINT ctr_check_sync_engine CHECK (sync_engine IN ('airbyte', 'debezium', 'both')),
    CONSTRAINT ctr_check_priority CHECK (priority IN ('critical', 'high', 'normal', 'low'))
);

CREATE INDEX IF NOT EXISTS idx_registry_source_db ON cdc_table_registry(source_db);
CREATE INDEX IF NOT EXISTS idx_registry_sync_engine ON cdc_table_registry(sync_engine);
CREATE INDEX IF NOT EXISTS idx_registry_priority ON cdc_table_registry(priority);
CREATE INDEX IF NOT EXISTS idx_registry_active ON cdc_table_registry(is_active);
CREATE INDEX IF NOT EXISTS idx_registry_target ON cdc_table_registry(target_table);

-- ============================================================
-- 2. MANAGEMENT TABLES
-- ============================================================

-- 2.1 CDC Mapping Rules (field-level mapping per table)
CREATE TABLE IF NOT EXISTS cdc_mapping_rules (
    id SERIAL PRIMARY KEY,
    source_table VARCHAR(200) NOT NULL,      -- Matches registry.target_table
    source_field VARCHAR(100) NOT NULL,
    target_column VARCHAR(100) NOT NULL,
    data_type VARCHAR(50) NOT NULL,          -- 'INTEGER', 'VARCHAR(255)', 'DECIMAL(18,6)', 'JSONB', 'TIMESTAMP'
    is_active BOOLEAN DEFAULT TRUE,
    is_enriched BOOLEAN DEFAULT FALSE,
    is_nullable BOOLEAN DEFAULT TRUE,
    default_value TEXT,
    enrichment_function VARCHAR(100),
    enrichment_params JSONB,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    created_by VARCHAR(100),
    updated_by VARCHAR(100),
    notes TEXT,
    UNIQUE(source_table, source_field)
);

CREATE INDEX IF NOT EXISTS idx_mapping_rules_table ON cdc_mapping_rules(source_table);
CREATE INDEX IF NOT EXISTS idx_mapping_rules_active ON cdc_mapping_rules(is_active);

-- 2.2 Pending Fields (Schema Drift Detection — generic cho mọi table)
CREATE TABLE IF NOT EXISTS pending_fields (
    id SERIAL PRIMARY KEY,
    table_name VARCHAR(200) NOT NULL,        -- target_table name
    source_db VARCHAR(100),                  -- Từ registry, để filter dễ hơn
    field_name VARCHAR(100) NOT NULL,
    sample_value TEXT,
    sample_values_json JSONB,
    suggested_type VARCHAR(50) NOT NULL,
    final_type VARCHAR(50),
    status VARCHAR(20) DEFAULT 'pending',
    detected_at TIMESTAMP DEFAULT NOW(),
    reviewed_at TIMESTAMP,
    approved_at TIMESTAMP,
    applied_at TIMESTAMP,
    reviewed_by VARCHAR(100),
    approval_notes TEXT,
    rejection_reason TEXT,
    target_column_name VARCHAR(100),
    detection_count INTEGER DEFAULT 1,
    UNIQUE(table_name, field_name),
    CONSTRAINT pf_check_status CHECK (status IN ('pending', 'approved', 'rejected', 'applied'))
);

CREATE INDEX IF NOT EXISTS idx_pending_status ON pending_fields(status);
CREATE INDEX IF NOT EXISTS idx_pending_table ON pending_fields(table_name);
CREATE INDEX IF NOT EXISTS idx_pending_source_db ON pending_fields(source_db);
CREATE INDEX IF NOT EXISTS idx_pending_detected ON pending_fields(detected_at DESC);

-- 2.3 Schema Changes Log (Audit Trail)
CREATE TABLE IF NOT EXISTS schema_changes_log (
    id SERIAL PRIMARY KEY,
    table_name VARCHAR(200) NOT NULL,
    source_db VARCHAR(100),
    change_type VARCHAR(50) NOT NULL,        -- 'ADD_COLUMN', 'CREATE_TABLE', 'MODIFY_COLUMN'
    field_name VARCHAR(100),
    old_definition TEXT,
    new_definition TEXT,
    sql_executed TEXT NOT NULL,
    execution_duration_ms INTEGER,
    status VARCHAR(20) DEFAULT 'pending',
    error_message TEXT,
    error_stack TEXT,
    pending_field_id INTEGER REFERENCES pending_fields(id),
    executed_by VARCHAR(100) NOT NULL,
    executed_at TIMESTAMP DEFAULT NOW(),
    rollback_sql TEXT,
    rolled_back_at TIMESTAMP,
    rolled_back_by VARCHAR(100),
    airbyte_source_id VARCHAR(100),
    airbyte_refresh_triggered BOOLEAN DEFAULT FALSE,
    airbyte_refresh_status VARCHAR(50),
    CONSTRAINT scl_check_status CHECK (status IN ('pending', 'executing', 'success', 'failed', 'rolled_back'))
);

CREATE INDEX IF NOT EXISTS idx_schema_log_table ON schema_changes_log(table_name);
CREATE INDEX IF NOT EXISTS idx_schema_log_status ON schema_changes_log(status);
CREATE INDEX IF NOT EXISTS idx_schema_log_executed ON schema_changes_log(executed_at DESC);

-- ============================================================
-- 3. DYNAMIC TABLE CREATION FUNCTION
-- ============================================================

-- Tạo CDC table động cho bất kỳ table nào trong registry
CREATE OR REPLACE FUNCTION create_cdc_table(
    p_target_table VARCHAR,
    p_primary_key_field VARCHAR DEFAULT 'id',
    p_primary_key_type VARCHAR DEFAULT 'VARCHAR(36)'
)
RETURNS VOID AS $$
DECLARE
    v_sql TEXT;
BEGIN
    -- Kiểm tra table đã tồn tại chưa
    IF EXISTS (
        SELECT 1 FROM information_schema.tables
        WHERE table_schema = 'public' AND table_name = p_target_table
    ) THEN
        RAISE NOTICE 'Table % already exists, skipping', p_target_table;
        RETURN;
    END IF;

    -- Tạo CDC table với cấu trúc chuẩn:
    -- Chỉ có primary key + _raw_data + metadata
    -- Business columns sẽ được thêm dần qua Schema Inspector → CMS Approve → ALTER TABLE
    v_sql := format(
        'CREATE TABLE %I (
            %I %s PRIMARY KEY,

            -- JSONB Landing Zone (chứa TOÀN BỘ raw data)
            _raw_data JSONB NOT NULL,

            -- CDC Metadata
            _source VARCHAR(20) NOT NULL DEFAULT ''airbyte'',
            _synced_at TIMESTAMP NOT NULL DEFAULT NOW(),
            _version BIGINT NOT NULL DEFAULT 1,
            _hash VARCHAR(64),
            _deleted BOOLEAN DEFAULT FALSE,
            _created_at TIMESTAMP DEFAULT NOW(),
            _updated_at TIMESTAMP DEFAULT NOW(),

            CONSTRAINT %I CHECK (_source IN (''debezium'', ''airbyte''))
        )',
        p_target_table,
        p_primary_key_field,
        p_primary_key_type,
        p_target_table || '_check_source'
    );

    EXECUTE v_sql;

    -- Tạo CDC indexes
    EXECUTE format('CREATE INDEX %I ON %I(_synced_at)', 'idx_' || p_target_table || '_synced', p_target_table);
    EXECUTE format('CREATE INDEX %I ON %I(_source)', 'idx_' || p_target_table || '_source', p_target_table);
    EXECUTE format('CREATE INDEX %I ON %I USING GIN(_raw_data)', 'idx_' || p_target_table || '_raw_data', p_target_table);
    EXECUTE format('CREATE INDEX %I ON %I(_deleted) WHERE _deleted = TRUE', 'idx_' || p_target_table || '_deleted', p_target_table);

    -- Update registry
    UPDATE cdc_table_registry SET is_table_created = TRUE WHERE target_table = p_target_table;

    RAISE NOTICE 'Created CDC table: %', p_target_table;
END;
$$ LANGUAGE plpgsql;

-- ============================================================
-- 4. BATCH TABLE CREATION FUNCTION
-- ============================================================

-- Tạo tất cả CDC tables chưa được tạo từ registry
CREATE OR REPLACE FUNCTION create_all_pending_cdc_tables()
RETURNS INTEGER AS $$
DECLARE
    v_record RECORD;
    v_count INTEGER := 0;
BEGIN
    FOR v_record IN
        SELECT target_table, primary_key_field, primary_key_type
        FROM cdc_table_registry
        WHERE is_active = TRUE AND is_table_created = FALSE
    LOOP
        PERFORM create_cdc_table(v_record.target_table, v_record.primary_key_field, v_record.primary_key_type);
        v_count := v_count + 1;
    END LOOP;

    RETURN v_count;
END;
$$ LANGUAGE plpgsql;

-- ============================================================
-- 5. GENERIC UPSERT FUNCTION
-- ============================================================

CREATE OR REPLACE FUNCTION upsert_with_jsonb_landing(
    p_table_name VARCHAR,
    p_pk_field VARCHAR,           -- Tên primary key field (dynamic)
    p_pk_value VARCHAR,
    p_mapped_data JSONB,          -- Data đã mapped theo rules (có thể empty)
    p_raw_data JSONB,             -- Raw JSON từ CDC event
    p_source VARCHAR,
    p_hash VARCHAR
)
RETURNS VOID AS $$
DECLARE
    v_columns TEXT[];
    v_values TEXT[];
    v_update_sets TEXT[];
    v_sql TEXT;
    v_key TEXT;
    v_value TEXT;
BEGIN
    -- Start with primary key
    v_columns := ARRAY[p_pk_field];
    v_values := ARRAY[quote_literal(p_pk_value)];

    -- Add mapped data columns (nếu có)
    IF p_mapped_data IS NOT NULL AND p_mapped_data != '{}'::JSONB THEN
        FOR v_key, v_value IN SELECT * FROM jsonb_each_text(p_mapped_data)
        LOOP
            v_columns := array_append(v_columns, v_key);
            v_values := array_append(v_values, quote_literal(v_value));
            v_update_sets := array_append(v_update_sets,
                format('%I = EXCLUDED.%I', v_key, v_key));
        END LOOP;
    END IF;

    -- Always add metadata columns
    v_columns := v_columns || ARRAY['_raw_data', '_source', '_synced_at', '_version', '_hash'];
    v_values := v_values || ARRAY[
        quote_literal(p_raw_data::TEXT),
        quote_literal(p_source),
        'NOW()',
        '1',
        quote_literal(p_hash)
    ];

    -- Build INSERT ... ON CONFLICT
    v_sql := format(
        'INSERT INTO %I (%s) VALUES (%s)
         ON CONFLICT (%I) DO UPDATE SET
            %s
            _raw_data = EXCLUDED._raw_data,
            _synced_at = NOW(),
            _version = %I._version + 1,
            _hash = EXCLUDED._hash,
            _updated_at = NOW()
         WHERE %I._hash IS DISTINCT FROM EXCLUDED._hash',
        p_table_name,
        array_to_string(v_columns, ', '),
        array_to_string(v_values, ', '),
        p_pk_field,
        CASE WHEN array_length(v_update_sets, 1) > 0
             THEN array_to_string(v_update_sets, ', ') || ','
             ELSE '' END,
        p_table_name,
        p_table_name
    );

    EXECUTE v_sql;
END;
$$ LANGUAGE plpgsql;

-- ============================================================
-- 6. SEED DATA — Pilot batch (~10-20 tables)
-- ============================================================

-- Ví dụ seed cho pilot batch. Production sẽ bulk import ~200 tables.
INSERT INTO cdc_table_registry
    (source_db, source_type, source_table, target_table, sync_engine, sync_interval, priority, primary_key_field, primary_key_type)
VALUES
    -- MongoDB: goopay_wallet
    ('goopay_wallet', 'mongodb', 'wallet_transactions', 'wallet_transactions', 'airbyte', '15m', 'critical', '_id', 'VARCHAR(24)'),
    ('goopay_wallet', 'mongodb', 'wallets', 'wallets', 'airbyte', '1h', 'high', '_id', 'VARCHAR(24)'),

    -- MongoDB: goopay_payment
    ('goopay_payment', 'mongodb', 'payments', 'payments', 'airbyte', '15m', 'critical', '_id', 'VARCHAR(24)'),
    ('goopay_payment', 'mongodb', 'refunds', 'refunds', 'airbyte', '1h', 'high', '_id', 'VARCHAR(24)'),

    -- MongoDB: goopay_order
    ('goopay_order', 'mongodb', 'orders', 'orders', 'airbyte', '15m', 'critical', '_id', 'VARCHAR(24)'),
    ('goopay_order', 'mongodb', 'order_items', 'order_items', 'airbyte', '1h', 'normal', '_id', 'VARCHAR(24)'),

    -- MySQL: goopay_legacy
    ('goopay_legacy', 'mysql', 'legacy_payments', 'legacy_payments', 'airbyte', '1h', 'normal', 'id', 'BIGINT'),
    ('goopay_legacy', 'mysql', 'legacy_refunds', 'legacy_refunds', 'airbyte', '4h', 'low', 'id', 'BIGINT'),

    -- MongoDB: goopay_main
    ('goopay_main', 'mongodb', 'users', 'users', 'airbyte', '1h', 'high', '_id', 'VARCHAR(24)'),
    ('goopay_main', 'mongodb', 'merchants', 'merchants', 'airbyte', '1h', 'normal', '_id', 'VARCHAR(24)')
ON CONFLICT (source_db, source_table) DO NOTHING;

-- Auto-create CDC tables cho pilot batch
SELECT create_all_pending_cdc_tables();

COMMIT;
```

#### Verification

```sql
-- Verify registry
SELECT source_db, source_type, COUNT(*) as tables,
       COUNT(CASE WHEN is_table_created THEN 1 END) as created
FROM cdc_table_registry GROUP BY source_db, source_type;

-- Verify CDC tables created
SELECT table_name FROM information_schema.tables
WHERE table_schema = 'public'
  AND table_name IN (SELECT target_table FROM cdc_table_registry)
ORDER BY table_name;

-- Verify all CDC tables have correct structure
SELECT t.target_table,
       EXISTS(SELECT 1 FROM information_schema.columns c
              WHERE c.table_name = t.target_table AND c.column_name = '_raw_data') as has_raw_data,
       EXISTS(SELECT 1 FROM information_schema.columns c
              WHERE c.table_name = t.target_table AND c.column_name = '_source') as has_source
FROM cdc_table_registry t WHERE t.is_table_created = TRUE;

-- Verify functions
SELECT proname FROM pg_proc WHERE proname IN ('create_cdc_table', 'create_all_pending_cdc_tables', 'upsert_with_jsonb_landing');
```

---

## CDC-M2: CDC Worker Core (Config-Driven)

### Mục tiêu
Go service: NATS consumer pool + **config-driven** event handler + batch upsert. Hoàn toàn generic cho mọi table trong registry.

### Giải pháp chi tiết

#### CDC-M2.1: Project Scaffolding

```
cdc-worker-service/
├── cmd/
│   ├── worker/main.go
│   └── cms-service/main.go
├── internal/
│   ├── config/config.go
│   ├── domain/
│   │   ├── entities/
│   │   │   ├── table_registry.go        # NEW: registry entity
│   │   │   ├── mapping_rule.go
│   │   │   └── pending_field.go
│   │   └── repositories/
│   │       ├── registry_repo.go         # NEW: registry repository
│   │       ├── mapping_rule_repo.go
│   │       └── pending_field_repo.go
│   ├── application/
│   │   ├── services/
│   │   │   ├── schema_inspector.go
│   │   │   ├── registry_service.go      # NEW: registry lookup + cache
│   │   │   └── dynamic_mapper.go        # Phase 1: stub
│   │   └── handlers/
│   │       └── event_handler.go         # Config-driven, generic
│   └── infrastructure/
│       ├── nats/{consumer.go, client.go}
│       ├── postgres/{repository.go, connection.go}
│       └── redis/cache.go
├── pkg/
│   ├── airbyte/client.go
│   ├── logger/logger.go
│   ├── metrics/prometheus.go
│   └── utils/{hash.go, type_inference.go}
├── go.mod
└── Makefile
```

#### CDC-M2.2: Infrastructure Layer

Tương tự version trước (PG connection pool, NATS JetStream client, Redis client). Không thay đổi.

#### CDC-M2.3: NATS Consumer Pool

Subscribe wildcard `cdc.goopay.>` để nhận events từ mọi source_db + table:

```go
sub, err := js.PullSubscribe(
    "cdc.goopay.>",                    // Wildcard: mọi table, mọi source DB
    "cdc-worker-group",
    nats.ManualAck(),
    nats.AckWait(30*time.Second),
    nats.MaxDeliver(5),
)
```

Worker pool 10 goroutines/pod, fetch 1000 msgs/pull, graceful shutdown. Không thay đổi logic.

#### CDC-M2.4: Event Handler (Config-Driven) — KEY CHANGE

```go
// internal/application/services/registry_service.go
package services

import (
    "context"
    "encoding/json"
    "fmt"
    "sync"
    "time"

    "go.uber.org/zap"

    "cdc-worker/internal/domain/entities"
    "cdc-worker/internal/domain/repositories"
)

// RegistryService — cache table registry + mapping rules in-memory
type RegistryService struct {
    registryRepo  repositories.TableRegistryRepository
    mappingRepo   repositories.MappingRuleRepository
    redisCache    repositories.CacheRepository
    logger        *zap.Logger

    mu            sync.RWMutex
    registryCache map[string]*entities.TableRegistry   // target_table → registry entry
    mappingCache  map[string][]entities.MappingRule     // target_table → mapping rules
}

func NewRegistryService(
    regRepo repositories.TableRegistryRepository,
    mapRepo repositories.MappingRuleRepository,
    cache repositories.CacheRepository,
    logger *zap.Logger,
) *RegistryService {
    rs := &RegistryService{
        registryRepo:  regRepo,
        mappingRepo:   mapRepo,
        redisCache:    cache,
        logger:        logger,
        registryCache: make(map[string]*entities.TableRegistry),
        mappingCache:  make(map[string][]entities.MappingRule),
    }
    // Initial load
    rs.ReloadAll(context.Background())
    return rs
}

func (rs *RegistryService) ReloadAll(ctx context.Context) error {
    // Load registry
    entries, err := rs.registryRepo.GetAllActive(ctx)
    if err != nil {
        return err
    }
    // Load mapping rules
    rules, err := rs.mappingRepo.GetAllActiveRules(ctx)
    if err != nil {
        return err
    }

    rs.mu.Lock()
    defer rs.mu.Unlock()

    rs.registryCache = make(map[string]*entities.TableRegistry)
    for i := range entries {
        rs.registryCache[entries[i].TargetTable] = &entries[i]
    }

    rs.mappingCache = make(map[string][]entities.MappingRule)
    for _, r := range rules {
        rs.mappingCache[r.SourceTable] = append(rs.mappingCache[r.SourceTable], r)
    }

    rs.logger.Info("Registry loaded",
        zap.Int("tables", len(rs.registryCache)),
        zap.Int("mapping_rules", len(rules)),
    )
    return nil
}

// GetTableConfig returns registry config for a table
func (rs *RegistryService) GetTableConfig(targetTable string) *entities.TableRegistry {
    rs.mu.RLock()
    defer rs.mu.RUnlock()
    return rs.registryCache[targetTable]
}

// GetMappingRules returns known column mappings for a table
func (rs *RegistryService) GetMappingRules(targetTable string) []entities.MappingRule {
    rs.mu.RLock()
    defer rs.mu.RUnlock()
    return rs.mappingCache[targetTable]
}
```

```go
// internal/application/handlers/event_handler.go
package handlers

type EventHandler struct {
    pgRepo          CDCRepository
    registrySvc     *services.RegistryService     // NEW: replaces hardcoded columns
    schemaInspector *services.SchemaInspector
    batchBuffer     *BatchBuffer
    logger          *zap.Logger
    metrics         *metrics.Collector
}

func (h *EventHandler) Handle(ctx context.Context, msg *nats.Msg) error {
    start := time.Now()

    // 1. Parse CDC event
    var event CDCEvent
    if err := json.Unmarshal(msg.Data, &event); err != nil {
        return fmt.Errorf("parse CDC event: %w", err)
    }

    // 2. Extract table name from NATS subject or event source
    //    subject: cdc.goopay.goopay_wallet.wallet_transactions
    //    → source_db = "goopay_wallet", table = "wallet_transactions"
    sourceDB, tableName := h.extractSourceAndTable(msg.Subject, event.Source)

    // 3. Lookup table config from registry
    tableConfig := h.registrySvc.GetTableConfig(tableName)
    if tableConfig == nil {
        // Unknown table — log warning, save raw data only nếu table tồn tại
        h.logger.Warn("Table not in registry, skipping",
            zap.String("table", tableName),
            zap.String("source_db", sourceDB),
        )
        return nil
    }

    // 4. Handle delete (soft delete)
    if event.Data.Op == "d" {
        return h.handleDelete(ctx, &event, tableName, tableConfig.PrimaryKeyField)
    }

    data := event.Data.After
    if data == nil {
        return fmt.Errorf("no 'after' data in event")
    }

    // 5. Schema Inspection (detect drift) — generic cho mọi table
    drift, err := h.schemaInspector.InspectEvent(ctx, tableName, sourceDB, data)
    if err != nil {
        h.logger.Error("Schema inspection failed", zap.Error(err))
    } else if drift != nil && drift.Detected {
        h.metrics.IncrSchemaDrift(sourceDB, tableName)
    }

    // 6. Extract primary key value (dynamic field name)
    pkValue := h.extractPrimaryKey(data, tableConfig.PrimaryKeyField, tableConfig.SourceType)

    // 7. Config-driven mapping: lookup mapping rules for this table
    mappingRules := h.registrySvc.GetMappingRules(tableName)
    mappedData := make(map[string]interface{})

    for _, rule := range mappingRules {
        if !rule.IsActive {
            continue
        }
        if val, ok := data[rule.SourceField]; ok {
            mappedData[rule.TargetColumn] = val
        }
    }
    // Nếu không có mapping rules → mappedData empty → chỉ save _raw_data (zero data loss)

    // 8. Always save _raw_data
    rawDataJSON, _ := json.Marshal(data)

    // 9. Calculate hash
    hash := h.calculateHash(data)

    // 10. Add to batch buffer
    record := &UpsertRecord{
        TableName:      tableName,
        PrimaryKeyField: tableConfig.PrimaryKeyField,
        PrimaryKeyValue: pkValue,
        MappedData:     mappedData,
        RawData:        string(rawDataJSON),
        Source:         "debezium",
        Hash:           hash,
    }
    h.batchBuffer.Add(record)

    // 11. Metrics
    duration := time.Since(start)
    h.metrics.ObserveProcessingDuration(event.Data.Op, sourceDB, tableName, duration)
    h.metrics.IncrEventsProcessed(event.Data.Op, sourceDB, tableName, "success")

    return nil
}

// extractSourceAndTable parses source_db + table from NATS subject
// subject format: cdc.goopay.{source_db}.{table_name}
func (h *EventHandler) extractSourceAndTable(subject, source string) (string, string) {
    parts := strings.Split(subject, ".")
    if len(parts) >= 4 {
        return parts[2], parts[3]     // source_db, table_name
    }
    // Fallback: parse from event source field
    sourceParts := strings.Split(source, "/")
    if len(sourceParts) >= 2 {
        return sourceParts[len(sourceParts)-2], sourceParts[len(sourceParts)-1]
    }
    return "unknown", "unknown"
}

// extractPrimaryKey — dynamic PK extraction based on registry config
func (h *EventHandler) extractPrimaryKey(data map[string]interface{}, pkField string, sourceType string) string {
    // MongoDB: _id có thể là ObjectId object {"$oid": "..."}
    if sourceType == "mongodb" && pkField == "_id" {
        if idMap, ok := data["_id"].(map[string]interface{}); ok {
            if oid, ok := idMap["$oid"].(string); ok {
                return oid
            }
        }
    }

    // Generic: lấy field value
    if val, ok := data[pkField]; ok {
        switch v := val.(type) {
        case string:
            return v
        case float64:
            return fmt.Sprintf("%.0f", v)
        default:
            return fmt.Sprintf("%v", v)
        }
    }
    return ""
}

func (h *EventHandler) handleDelete(ctx context.Context, event *CDCEvent, tableName, pkField string) error {
    before := event.Data.Before
    if before == nil {
        return fmt.Errorf("no 'before' data in delete event")
    }
    pkValue := h.extractPrimaryKey(before, pkField, "")
    query := fmt.Sprintf("UPDATE %s SET _deleted = TRUE, _updated_at = NOW() WHERE %s = $1",
        tableName, pkField)
    return h.pgRepo.ExecuteQuery(ctx, query, pkValue)
}
```

#### CDC-M2.5: Batch Buffer

Tương tự version trước nhưng `UpsertRecord` giờ có `PrimaryKeyField` dynamic:

```go
type UpsertRecord struct {
    TableName       string
    PrimaryKeyField string                 // Dynamic: '_id', 'id', etc.
    PrimaryKeyValue string
    MappedData      map[string]interface{} // Có thể empty (chỉ save _raw_data)
    RawData         string
    Source          string
    Hash            string
}
```

Batch upsert query sử dụng `PrimaryKeyField` thay vì hardcode `id`:

```go
func (bb *BatchBuffer) batchUpsert(tableName string, records []*UpsertRecord) error {
    pkField := records[0].PrimaryKeyField

    // ... build dynamic columns from mapped data ...

    query := fmt.Sprintf(`
        INSERT INTO %s (%s) VALUES %s
        ON CONFLICT (%s) DO UPDATE SET %s
        WHERE %s._hash IS DISTINCT FROM EXCLUDED._hash
    `, tableName, columns, valueRows, pkField, updateSets, tableName)

    return bb.pgRepo.ExecuteQuery(context.Background(), query, args...)
}
```

#### CDC-M2.6: Health Endpoints

Không thay đổi: `/health` + `/ready` on `:8080`.

#### CDC-M2.7: Unit Tests

```go
// Key test cases — all generic, no hardcoded tables
func TestExtractPrimaryKey_MongoObjectId(t *testing.T) { ... }
func TestExtractPrimaryKey_StringID(t *testing.T) { ... }
func TestExtractPrimaryKey_NumericID(t *testing.T) { ... }
func TestExtractSourceAndTable_FromSubject(t *testing.T) { ... }
func TestHandle_UnknownTable_Skipped(t *testing.T) { ... }
func TestHandle_NoMappingRules_SaveRawDataOnly(t *testing.T) { ... }
func TestHandle_WithMappingRules_MapsColumns(t *testing.T) { ... }
func TestHandle_Delete_SoftDelete(t *testing.T) { ... }
func TestBatchUpsert_DynamicPrimaryKey(t *testing.T) { ... }
func TestRegistryService_ReloadAll(t *testing.T) { ... }
func TestRegistryService_GetTableConfig_NotFound(t *testing.T) { ... }
```

#### Config Reload Listener

CDC Worker listens NATS `schema.config.reload` → reload registry + mapping rules:

```go
// Trong main.go startup
nc.Subscribe("schema.config.reload", func(msg *nats.Msg) {
    logger.Info("Config reload triggered", zap.String("payload", string(msg.Data)))
    registrySvc.ReloadAll(context.Background())
    // Invalidate Redis cache
    redisCache.Delete(ctx, "registry:*")
    redisCache.Delete(ctx, "mapping:*")
    redisCache.Delete(ctx, "schema:*")
})
```

---

## CDC-M3: Schema Inspector

### Mục tiêu
Detect new fields trong CDC events cho **bất kỳ table nào** (~200 tables). Generic, config-driven.

### Giải pháp chi tiết

#### Tổng quan hoạt động

```
CDC Event (bất kỳ table nào)
        │
        ▼
┌──────────────────────────────────────────┐
│    Schema Inspector (Generic)             │
│                                           │
│  1. extractFieldNames(eventData)          │
│     → tất cả keys trong JSON              │
│                                           │
│  2. getTableSchema(tableName)             │
│     → Redis cache hit?                    │
│       ├─ YES: return cached columns       │
│       └─ NO: query information_schema     │
│              → cache 5min TTL             │
│                                           │
│  3. findNewFields(event, schema)          │
│     → danh sách fields chưa có column     │
│     → skip internal fields (_id, etc.)    │
│                                           │
│  4. inferDataType(value) per new field    │
│                                           │
│  5. savePendingField(table, sourceDB,     │
│     field, value, type)                   │
│     → UPSERT pending_fields              │
│     → increment detection_count          │
│                                           │
│  6. publishDriftAlert(sourceDB, table,    │
│     fields)                               │
│     → NATS schema.drift.detected         │
└──────────────────────────────────────────┘
```

#### InspectEvent — giờ nhận thêm `sourceDB`

```go
func (si *SchemaInspector) InspectEvent(
    ctx context.Context,
    tableName string,
    sourceDB string,              // NEW: để populate pending_fields.source_db
    eventData map[string]interface{},
) (*SchemaDrift, error) {
    eventFields := si.extractFieldNames(eventData)

    tableSchema, err := si.getTableSchema(ctx, tableName)
    if err != nil {
        return nil, fmt.Errorf("get table schema: %w", err)
    }

    newFields := si.findNewFields(eventFields, tableSchema)
    if len(newFields) == 0 {
        return &SchemaDrift{Detected: false}, nil
    }

    si.logger.Info("Schema drift detected",
        zap.String("source_db", sourceDB),
        zap.String("table", tableName),
        zap.Int("new_fields_count", len(newFields)),
    )

    var detectedFields []DetectedField
    for _, fieldName := range newFields {
        value := eventData[fieldName]
        suggestedType := si.inferDataType(value)

        detectedFields = append(detectedFields, DetectedField{
            FieldName:     fieldName,
            SampleValue:   value,
            SuggestedType: suggestedType,
        })

        si.savePendingField(ctx, tableName, sourceDB, fieldName, value, suggestedType)
    }

    si.publishDriftAlert(sourceDB, tableName, detectedFields)

    return &SchemaDrift{
        Detected:  true,
        TableName: tableName,
        SourceDB:  sourceDB,
        NewFields: detectedFields,
    }, nil
}
```

#### Type Inference — không thay đổi

| Go Type | Condition | PostgreSQL Type |
|---------|-----------|-----------------|
| `bool` | any | `BOOLEAN` |
| `float64` | integer, [-2^31, 2^31] | `INTEGER` |
| `float64` | integer, ngoài range | `BIGINT` |
| `float64` | fractional | `DECIMAL(18,6)` |
| `string` | parseable RFC3339 | `TIMESTAMP` |
| `string` | len ≤ 100 | `VARCHAR(100)` |
| `string` | len ≤ 255 | `VARCHAR(255)` |
| `string` | len > 255 | `TEXT` |
| `map[string]interface{}` | any | `JSONB` |
| `[]interface{}` | any | `JSONB` |
| `nil` | - | `TEXT` |

#### Save Pending Field — thêm `source_db`

```go
func (si *SchemaInspector) savePendingField(
    ctx context.Context,
    tableName, sourceDB, fieldName string,
    sampleValue interface{},
    suggestedType string,
) error {
    sampleJSON, _ := json.Marshal(sampleValue)

    query := `
        INSERT INTO pending_fields
            (table_name, source_db, field_name, sample_value, suggested_type, detected_at, status, detection_count)
        VALUES ($1, $2, $3, $4, $5, NOW(), 'pending', 1)
        ON CONFLICT (table_name, field_name) DO UPDATE SET
            detection_count = pending_fields.detection_count + 1,
            sample_value = EXCLUDED.sample_value,
            suggested_type = CASE
                WHEN pending_fields.detection_count < 5 THEN EXCLUDED.suggested_type
                ELSE pending_fields.suggested_type
            END
        WHERE pending_fields.status = 'pending'
    `
    return si.pgRepo.ExecuteQuery(ctx, query, tableName, sourceDB, fieldName, string(sampleJSON), suggestedType)
}
```

#### Drift Alert — thêm `source_db`

```go
func (si *SchemaInspector) publishDriftAlert(sourceDB, tableName string, fields []DetectedField) error {
    alert := map[string]interface{}{
        "source_db":   sourceDB,
        "table":       tableName,
        "new_fields":  fields,
        "detected_at": time.Now().Format(time.RFC3339),
        "severity":    calculateSeverity(len(fields)),
    }
    alertJSON, _ := json.Marshal(alert)
    return si.natsClient.Publish("schema.drift.detected", alertJSON)
}
```

#### Redis Cache — per table

| Key | TTL | Purpose |
|-----|-----|---------|
| `schema:{target_table}` | 5 min | Table columns từ information_schema |

Cache invalidation: khi `schema.config.reload` được publish → delete `schema:{table}`.

#### Unit Tests

```go
func TestInspectEvent_DetectsNewField_AnyTable(t *testing.T) { ... }
func TestInspectEvent_NoNewFields(t *testing.T) { ... }
func TestInspectEvent_CacheHit_SkipsDBQuery(t *testing.T) { ... }
func TestInspectEvent_UnknownTable_QueriesDB(t *testing.T) { ... }
func TestSavePendingField_IncrementsCount(t *testing.T) { ... }
func TestDriftAlert_IncludesSourceDB(t *testing.T) { ... }
func TestInferDataType_AllTypes(t *testing.T) { ... }
```

---

## CDC-M4: Dynamic Mapper (Init Only)

### Mục tiêu
Stub implementation cho Phase 2. Không thay đổi từ version trước.

### Giải pháp chi tiết

```go
// internal/application/services/dynamic_mapper.go
var ErrNotImplemented = errors.New("not implemented: Dynamic Mapper available in Phase 2")

type DynamicMapper struct { ... }
type MappedData struct { ... }

func (dm *DynamicMapper) LoadRules(ctx context.Context) error          { return ErrNotImplemented }
func (dm *DynamicMapper) GetRulesForTable(tableName string) []entities.MappingRule { return nil }
func (dm *DynamicMapper) MapData(ctx context.Context, tableName string, rawData map[string]interface{}) (*MappedData, error) {
    return nil, ErrNotImplemented
}
func (dm *DynamicMapper) BuildUpsertQuery(...) (string, []interface{}, error) { return "", nil, ErrNotImplemented }
func (dm *DynamicMapper) convertType(value interface{}, targetType string) (interface{}, error) { return nil, ErrNotImplemented }
func (dm *DynamicMapper) StartConfigReloadListener(ctx context.Context) {
    // TODO Phase 2
}
```

---

## CDC-M5: Airbyte API Client (Multi-Source)

### Mục tiêu
Go client cho Airbyte API. Hỗ trợ multi-source (~30 DBs), lookup `airbyte_source_id` / `airbyte_connection_id` từ `cdc_table_registry`.

### Giải pháp chi tiết

```go
// pkg/airbyte/client.go
type Client struct {
    baseURL    string
    apiKey     string
    httpClient *http.Client
    logger     *zap.Logger
}

func NewClient(baseURL, apiKey string, logger *zap.Logger) *Client { ... }

// RefreshSourceSchema — trigger rediscover cho 1 source
func (c *Client) RefreshSourceSchema(ctx context.Context, sourceID string) error { ... }

// UpdateConnection — update sync config cho 1 connection
func (c *Client) UpdateConnection(ctx context.Context, connectionID string, streams []StreamConfig) error { ... }

// TriggerSync — manually trigger sync cho 1 connection
func (c *Client) TriggerSync(ctx context.Context, connectionID string) (string, error) { ... }

// GetConnectionStatus — check connection health
func (c *Client) GetConnectionStatus(ctx context.Context, connectionID string) (*ConnectionStatus, error) { ... }
```

**Lookup từ registry** — CMS handler sẽ query registry để lấy `airbyte_source_id`:

```go
// Trong CMS approve handler
func (h *CMSHandler) triggerAirbyteRefresh(ctx context.Context, tableName string, logID int) {
    // Lookup registry
    registry, err := h.registryRepo.GetByTargetTable(ctx, tableName)
    if err != nil || registry.AirbyteSourceID == nil {
        h.logger.Warn("No Airbyte source for table", zap.String("table", tableName))
        return
    }

    if err := h.airbyteClient.RefreshSourceSchema(ctx, *registry.AirbyteSourceID); err != nil {
        h.logger.Error("Airbyte refresh failed", zap.Error(err))
        h.schemaLogRepo.UpdateAirbyteStatus(ctx, logID, "failed")
        return
    }
    h.schemaLogRepo.UpdateAirbyteStatus(ctx, logID, "success")
}
```

---

## CDC-M6: CMS Backend API + Registry CRUD

### Mục tiêu
Go/Gin API: schema change approve/reject, mapping rules CRUD, **Table Registry management** (CRUD ~200 tables).

### Giải pháp chi tiết

#### CDC-M6.1: Server Setup

```go
func main() {
    // ... setup db, nats, airbyte client ...

    // Repositories
    registryRepo := postgres.NewTableRegistryRepo(db)    // NEW
    pfRepo := postgres.NewPendingFieldRepo(db)
    mrRepo := postgres.NewMappingRuleRepo(db)
    slRepo := postgres.NewSchemaLogRepo(db)

    cmsHandler := api.NewCMSHandler(registryRepo, pfRepo, mrRepo, slRepo, db, nc, airbyteClient, logger)

    r := gin.Default()
    auth := r.Group("/api").Use(api.JWTAuthMiddleware(cfg.JWTSecret))
    {
        // Schema Changes
        auth.GET("/schema-changes/pending", cmsHandler.GetPendingChanges)
        auth.POST("/schema-changes/:id/approve", cmsHandler.ApproveSchemaChange)
        auth.POST("/schema-changes/:id/reject", cmsHandler.RejectSchemaChange)
        auth.GET("/schema-changes/history", cmsHandler.GetSchemaHistory)

        // Mapping Rules
        auth.GET("/mapping-rules", cmsHandler.GetMappingRules)
        auth.POST("/mapping-rules", cmsHandler.CreateMappingRule)

        // Table Registry — NEW
        auth.GET("/registry", cmsHandler.ListRegistry)
        auth.POST("/registry", cmsHandler.RegisterTable)
        auth.PATCH("/registry/:id", cmsHandler.UpdateRegistry)
        auth.POST("/registry/batch", cmsHandler.BulkRegisterTables)
        auth.GET("/registry/stats", cmsHandler.GetRegistryStats)
    }
    r.GET("/health", func(c *gin.Context) { c.JSON(200, gin.H{"status": "ok"}) })
}
```

#### CDC-M6.2: Repository Layer

**TableRegistryRepository** (NEW):

```go
type TableRegistryRepository interface {
    GetAll(ctx context.Context, filter RegistryFilter) ([]entities.TableRegistry, int, error)
    GetByID(ctx context.Context, id int) (*entities.TableRegistry, error)
    GetByTargetTable(ctx context.Context, targetTable string) (*entities.TableRegistry, error)
    GetAllActive(ctx context.Context) ([]entities.TableRegistry, error)
    Create(ctx context.Context, entry *entities.TableRegistry) error
    Update(ctx context.Context, entry *entities.TableRegistry) error
    BulkCreate(ctx context.Context, entries []entities.TableRegistry) (int, error)
    GetStats(ctx context.Context) (*RegistryStats, error)
}

type RegistryFilter struct {
    SourceDB   *string
    SyncEngine *string
    Priority   *string
    IsActive   *bool
    Page       int
    PageSize   int
}

type RegistryStats struct {
    Total         int            `json:"total"`
    BySourceDB    map[string]int `json:"by_source_db"`
    BySyncEngine  map[string]int `json:"by_sync_engine"`
    ByPriority    map[string]int `json:"by_priority"`
    TablesCreated int            `json:"tables_created"`
}
```

#### CDC-M6.3: List Pending Changes — thêm filter `source_db`

```go
func (h *CMSHandler) GetPendingChanges(c *gin.Context) {
    status := c.DefaultQuery("status", "pending")
    sourceDB := c.Query("source_db")
    tableName := c.Query("table")
    page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
    pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

    // ... query with filters + pagination ...
}
```

#### CDC-M6.4: Approve Schema Change

Flow không thay đổi logic, nhưng lookup `airbyte_source_id` từ registry:

```
POST /api/schema-changes/:id/approve
    │
    ▼
1. Get pending_field by ID
2. Validate status == 'pending'
3. BEGIN TRANSACTION
   ├─ ALTER TABLE {target_table} ADD COLUMN IF NOT EXISTS {col} {type}
   ├─ INSERT INTO cdc_mapping_rules
   ├─ UPDATE pending_fields SET status='approved'
   ├─ INSERT INTO schema_changes_log
   └─ COMMIT
4. NATS publish "schema.config.reload"
5. Async: lookup registry → Airbyte RefreshSourceSchema (nếu có airbyte_source_id)
```

#### CDC-M6.7: Table Registry CRUD — NEW

```go
// GET /api/registry — list with filters + pagination
func (h *CMSHandler) ListRegistry(c *gin.Context) {
    filter := RegistryFilter{
        Page:     getPageParam(c),
        PageSize: getPageSizeParam(c),
    }
    if v := c.Query("source_db"); v != "" {
        filter.SourceDB = &v
    }
    if v := c.Query("sync_engine"); v != "" {
        filter.SyncEngine = &v
    }
    if v := c.Query("priority"); v != "" {
        filter.Priority = &v
    }
    if v := c.Query("is_active"); v != "" {
        b := v == "true"
        filter.IsActive = &b
    }

    entries, total, err := h.registryRepo.GetAll(ctx, filter)
    if err != nil {
        c.JSON(500, gin.H{"error": "Failed to fetch registry"})
        return
    }
    c.JSON(200, gin.H{"data": entries, "total": total, "page": filter.Page})
}

// POST /api/registry — register new table + auto-create CDC table
func (h *CMSHandler) RegisterTable(c *gin.Context) {
    var entry entities.TableRegistry
    if err := c.ShouldBindJSON(&entry); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }

    if err := h.registryRepo.Create(ctx, &entry); err != nil {
        c.JSON(500, gin.H{"error": "Failed to register table"})
        return
    }

    // Auto-create CDC table in PostgreSQL
    _, err := h.pgDB.ExecContext(ctx,
        "SELECT create_cdc_table($1, $2, $3)",
        entry.TargetTable, entry.PrimaryKeyField, entry.PrimaryKeyType,
    )
    if err != nil {
        h.logger.Error("Failed to create CDC table", zap.Error(err))
        // Non-fatal: table registration succeeded, creation can be retried
    }

    // Publish config reload
    h.natsClient.Publish("schema.config.reload", []byte(entry.TargetTable))

    c.JSON(201, gin.H{"message": "Table registered", "data": entry})
}

// PATCH /api/registry/:id — update config (sync_engine, priority, etc.)
func (h *CMSHandler) UpdateRegistry(c *gin.Context) {
    id, _ := strconv.Atoi(c.Param("id"))

    existing, err := h.registryRepo.GetByID(ctx, id)
    if err != nil {
        c.JSON(404, gin.H{"error": "Not found"})
        return
    }

    var update struct {
        SyncEngine   *string `json:"sync_engine"`
        SyncInterval *string `json:"sync_interval"`
        Priority     *string `json:"priority"`
        IsActive     *bool   `json:"is_active"`
    }
    if err := c.ShouldBindJSON(&update); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }

    if update.SyncEngine != nil { existing.SyncEngine = *update.SyncEngine }
    if update.SyncInterval != nil { existing.SyncInterval = *update.SyncInterval }
    if update.Priority != nil { existing.Priority = *update.Priority }
    if update.IsActive != nil { existing.IsActive = *update.IsActive }

    if err := h.registryRepo.Update(ctx, existing); err != nil {
        c.JSON(500, gin.H{"error": "Failed to update"})
        return
    }

    h.natsClient.Publish("schema.config.reload", []byte(existing.TargetTable))
    c.JSON(200, gin.H{"message": "Updated", "data": existing})
}

// POST /api/registry/batch — bulk register tables
func (h *CMSHandler) BulkRegisterTables(c *gin.Context) {
    var entries []entities.TableRegistry
    if err := c.ShouldBindJSON(&entries); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }

    created, err := h.registryRepo.BulkCreate(ctx, entries)
    if err != nil {
        c.JSON(500, gin.H{"error": "Bulk register failed"})
        return
    }

    // Create CDC tables for all new entries
    h.pgDB.ExecContext(ctx, "SELECT create_all_pending_cdc_tables()")

    h.natsClient.Publish("schema.config.reload", []byte("*"))
    c.JSON(201, gin.H{"message": fmt.Sprintf("%d tables registered", created), "created": created})
}

// GET /api/registry/stats — summary statistics
func (h *CMSHandler) GetRegistryStats(c *gin.Context) {
    stats, err := h.registryRepo.GetStats(ctx)
    if err != nil {
        c.JSON(500, gin.H{"error": "Failed to get stats"})
        return
    }
    c.JSON(200, stats)
}
```

#### CDC-M6.8: Schema History — thêm filter `source_db`

```go
func (h *CMSHandler) GetSchemaHistory(c *gin.Context) {
    table := c.Query("table")
    sourceDB := c.Query("source_db")
    // ... query with filters ...
}
```

---

## CDC-F1: CMS Frontend + Registry UI

### Mục tiêu
React + Ant Design UI: pending changes, approval, dashboard, **Table Registry manager**.

### Giải pháp chi tiết

#### CDC-F1.2: PendingChangesTable

Thêm column `source_db`, thêm filter `source_db`:

```tsx
const columns = [
    { title: 'Source DB', dataIndex: 'source_db', key: 'source_db',
      filters: uniqueSourceDBs.map(d => ({ text: d, value: d })) },
    { title: 'Table', dataIndex: 'table_name', key: 'table_name' },
    { title: 'Field', dataIndex: 'field_name', key: 'field_name' },
    { title: 'Sample Value', dataIndex: 'sample_value', ... },
    { title: 'Suggested Type', dataIndex: 'suggested_type', ... },
    { title: 'Detection Count', dataIndex: 'detection_count', ... },
    { title: 'Status', dataIndex: 'status', ... },
    { title: 'Actions', ... },
];
```

Pagination vì có thể nhiều pending fields từ ~200 tables.

#### CDC-F1.5: Dashboard

```tsx
// Stats cards
<Row gutter={16}>
    <Col span={4}><Card title="Pending Changes"><Statistic value={stats.pendingCount} /></Card></Col>
    <Col span={4}><Card title="Approved Today"><Statistic value={stats.approvedToday} /></Card></Col>
    <Col span={4}><Card title="Tables with Drift"><Statistic value={stats.tablesWithDrift} /></Card></Col>
    <Col span={4}><Card title="Registered Tables"><Statistic value={stats.totalRegistered} /></Card></Col>
    <Col span={4}><Card title="Airbyte Tables"><Statistic value={stats.bySyncEngine.airbyte} /></Card></Col>
    <Col span={4}><Card title="Source DBs"><Statistic value={stats.sourceDBCount} /></Card></Col>
</Row>

// Breakdown charts
<Row>
    <Col span={12}><PieChart data={stats.bySyncEngine} title="By Sync Engine" /></Col>
    <Col span={12}><PieChart data={stats.byPriority} title="By Priority" /></Col>
</Row>
```

#### CDC-F1.6: TableRegistryManager — NEW

```tsx
// pages/TableRegistry.tsx

const TableRegistryManager: React.FC = () => {
    const [data, setData] = useState<RegistryEntry[]>([]);
    const [filters, setFilters] = useState({ source_db: '', sync_engine: '', priority: '' });
    const [registerModalVisible, setRegisterModalVisible] = useState(false);
    const [bulkImportVisible, setBulkImportVisible] = useState(false);

    const columns = [
        { title: 'Source DB', dataIndex: 'source_db', key: 'source_db',
          filters: uniqueSourceDBs },
        { title: 'Source Type', dataIndex: 'source_type', key: 'source_type',
          render: (t: string) => <Tag color={t === 'mongodb' ? 'green' : 'blue'}>{t}</Tag> },
        { title: 'Source Table', dataIndex: 'source_table', key: 'source_table' },
        { title: 'Target Table', dataIndex: 'target_table', key: 'target_table' },
        { title: 'Sync Engine', dataIndex: 'sync_engine', key: 'sync_engine',
          render: (engine: string, record: RegistryEntry) => (
              <Select value={engine} onChange={(v) => updateRegistry(record.id, { sync_engine: v })}>
                  <Option value="airbyte">Airbyte</Option>
                  <Option value="debezium">Debezium</Option>
                  <Option value="both">Both</Option>
              </Select>
          )},
        { title: 'Priority', dataIndex: 'priority', key: 'priority',
          render: (p: string, record: RegistryEntry) => (
              <Select value={p} onChange={(v) => updateRegistry(record.id, { priority: v })}>
                  <Option value="critical">Critical</Option>
                  <Option value="high">High</Option>
                  <Option value="normal">Normal</Option>
                  <Option value="low">Low</Option>
              </Select>
          )},
        { title: 'Interval', dataIndex: 'sync_interval', key: 'sync_interval' },
        { title: 'PK Field', dataIndex: 'primary_key_field', key: 'primary_key_field' },
        { title: 'Table Created', dataIndex: 'is_table_created', key: 'is_table_created',
          render: (v: boolean) => v ? <Tag color="green">Yes</Tag> : <Tag color="red">No</Tag> },
        { title: 'Active', dataIndex: 'is_active', key: 'is_active',
          render: (v: boolean, record: RegistryEntry) => (
              <Switch checked={v} onChange={(checked) => updateRegistry(record.id, { is_active: checked })} />
          )},
    ];

    return (
        <>
            {/* Stats bar */}
            <RegistryStatsBar />

            {/* Filters */}
            <Space style={{ marginBottom: 16 }}>
                <Select placeholder="Source DB" onChange={v => setFilters({...filters, source_db: v})} allowClear>
                    {sourceDBs.map(db => <Option key={db} value={db}>{db}</Option>)}
                </Select>
                <Select placeholder="Sync Engine" onChange={v => setFilters({...filters, sync_engine: v})} allowClear>
                    <Option value="airbyte">Airbyte</Option>
                    <Option value="debezium">Debezium</Option>
                    <Option value="both">Both</Option>
                </Select>
                <Button type="primary" onClick={() => setRegisterModalVisible(true)}>Register Table</Button>
                <Button onClick={() => setBulkImportVisible(true)}>Bulk Import</Button>
            </Space>

            {/* Table */}
            <Table columns={columns} dataSource={data} rowKey="id" pagination={{ pageSize: 20 }} />

            {/* Register Modal */}
            <RegisterTableModal visible={registerModalVisible} onClose={...} onSuccess={...} />

            {/* Bulk Import Modal */}
            <BulkImportModal visible={bulkImportVisible} onClose={...} onSuccess={...} />
        </>
    );
};
```

**BulkImportModal**: Upload JSON/CSV → POST /api/registry/batch

```tsx
// JSON format for bulk import:
[
    {"source_db": "goopay_wallet", "source_type": "mongodb", "source_table": "transfers", "target_table": "transfers", "sync_engine": "airbyte", "priority": "high", "primary_key_field": "_id", "primary_key_type": "VARCHAR(24)"},
    {"source_db": "goopay_legacy", "source_type": "mysql", "source_table": "audit_logs", "target_table": "audit_logs", "sync_engine": "airbyte", "priority": "low", "primary_key_field": "id", "primary_key_type": "BIGINT"},
    ...
]
```

---

## CDC-M7: Monitoring + Docker + K8s Manifests

### Mục tiêu
Prometheus metrics, Dockerfiles, K8s manifests.

### Giải pháp chi tiết

#### Prometheus Metrics — thêm `source_db` label

```go
var (
    CDCEventsProcessed = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "cdc_events_processed_total",
            Help: "Total CDC events processed",
        },
        []string{"operation", "source_db", "table", "status"},   // thêm source_db
    )

    CDCProcessingDuration = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "cdc_processing_duration_seconds",
            Buckets: prometheus.ExponentialBuckets(0.001, 2, 10),
        },
        []string{"operation", "source_db", "table"},
    )

    SchemaDriftDetected = promauto.NewCounterVec(
        prometheus.CounterOpts{Name: "schema_drift_detected_total"},
        []string{"source_db", "table"},
    )

    RegisteredTablesTotal = promauto.NewGaugeVec(         // NEW
        prometheus.GaugeOpts{Name: "registered_tables_total"},
        []string{"source_db", "sync_engine", "priority"},
    )

    MappingRulesLoaded = promauto.NewGauge(
        prometheus.GaugeOpts{Name: "mapping_rules_loaded"},
    )

    PendingFieldsCount = promauto.NewGaugeVec(
        prometheus.GaugeOpts{Name: "pending_fields_count"},
        []string{"status"},
    )
)
```

#### Dockerfiles

Không thay đổi từ version trước (multi-stage Go build + React static).

#### K8s Manifests

Xem [CDC-D4](#cdc-d4-k8s-deployment).

---

## CDC-M8: Integration Test (End-to-End)

### Mục tiêu
E2E validation với dynamic table setup — không hardcode table cụ thể.

### Giải pháp chi tiết

#### Test Scenarios

**Scenario 1: Register table → auto-create CDC table**

```go
func TestE2E_RegisterTable_AutoCreate(t *testing.T) {
    // 1. POST /api/registry — register new table
    entry := map[string]string{
        "source_db": "goopay_test", "source_type": "mongodb",
        "source_table": "test_collection", "target_table": "test_collection",
        "sync_engine": "airbyte", "priority": "normal",
        "primary_key_field": "_id", "primary_key_type": "VARCHAR(24)",
    }
    resp := postJSON("/api/registry", entry)
    assert.Equal(t, 201, resp.StatusCode)

    // 2. Verify PostgreSQL table created
    var exists bool
    db.QueryRow(`SELECT EXISTS (
        SELECT 1 FROM information_schema.tables
        WHERE table_name = 'test_collection'
    )`).Scan(&exists)
    assert.True(t, exists)

    // 3. Verify table has correct structure
    var rawDataExists bool
    db.QueryRow(`SELECT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'test_collection' AND column_name = '_raw_data'
    )`).Scan(&rawDataExists)
    assert.True(t, rawDataExists)

    // 4. Verify registry updated
    var isCreated bool
    db.QueryRow("SELECT is_table_created FROM cdc_table_registry WHERE target_table = 'test_collection'").Scan(&isCreated)
    assert.True(t, isCreated)
}
```

**Scenario 2: CDC Worker processes event for dynamic table**

```go
func TestE2E_Worker_DynamicTable(t *testing.T) {
    // Pre-condition: table "test_collection" registered + created (Scenario 1)

    // 1. Publish CDC event
    event := CloudEvent{
        Source: "/debezium/mongodb/goopay_test/test_collection",
        Data: CDCData{
            Op: "c",
            After: map[string]interface{}{
                "_id": map[string]interface{}{"$oid": "65f1a2b3c4d5e6f7a8b9c0d1"},
                "name": "test item",
                "value": 42.5,
            },
        },
    }
    publishToNATS("cdc.goopay.goopay_test.test_collection", event)
    time.Sleep(5 * time.Second)

    // 2. Verify data in PostgreSQL — only _raw_data (no mapping rules yet)
    var rawData string
    err := db.QueryRow("SELECT _raw_data FROM test_collection WHERE _id = '65f1a2b3c4d5e6f7a8b9c0d1'").Scan(&rawData)
    assert.NoError(t, err)
    assert.Contains(t, rawData, "test item")
}
```

**Scenario 3: Schema Inspector detects drift → CMS approve → column added**

```go
func TestE2E_DriftDetect_Approve_Dynamic(t *testing.T) {
    // 1. Event with fields → Schema Inspector detects "name", "value" as new fields
    // (from Scenario 2)

    // 2. Verify pending_fields
    var count int
    db.QueryRow("SELECT COUNT(*) FROM pending_fields WHERE table_name = 'test_collection' AND status = 'pending'").Scan(&count)
    assert.GreaterOrEqual(t, count, 1)

    // 3. Approve "name" field
    var pendingID int
    db.QueryRow("SELECT id FROM pending_fields WHERE table_name = 'test_collection' AND field_name = 'name'").Scan(&pendingID)

    resp := postJSON(fmt.Sprintf("/api/schema-changes/%d/approve", pendingID), map[string]string{
        "target_column_name": "name",
        "final_type": "VARCHAR(255)",
    })
    assert.Equal(t, 200, resp.StatusCode)

    // 4. Verify column added
    var colExists bool
    db.QueryRow(`SELECT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'test_collection' AND column_name = 'name'
    )`).Scan(&colExists)
    assert.True(t, colExists)

    // 5. Verify mapping rule created
    var ruleCount int
    db.QueryRow("SELECT COUNT(*) FROM cdc_mapping_rules WHERE source_table = 'test_collection' AND source_field = 'name'").Scan(&ruleCount)
    assert.Equal(t, 1, ruleCount)
}
```

**Scenario 4: Toggle sync_engine on registry**

```go
func TestE2E_ToggleSyncEngine(t *testing.T) {
    // PATCH /api/registry/:id — change sync_engine from 'airbyte' to 'debezium'
    resp := patchJSON("/api/registry/1", map[string]string{"sync_engine": "debezium"})
    assert.Equal(t, 200, resp.StatusCode)

    var engine string
    db.QueryRow("SELECT sync_engine FROM cdc_table_registry WHERE id = 1").Scan(&engine)
    assert.Equal(t, "debezium", engine)
}
```

**Scenario 5: Bulk register tables**

```go
func TestE2E_BulkRegister(t *testing.T) {
    entries := []map[string]string{
        {"source_db": "goopay_test", "source_type": "mongodb", "source_table": "bulk_1", "target_table": "bulk_1", ...},
        {"source_db": "goopay_test", "source_type": "mongodb", "source_table": "bulk_2", "target_table": "bulk_2", ...},
        {"source_db": "goopay_test", "source_type": "mysql", "source_table": "bulk_3", "target_table": "bulk_3", ...},
    }
    resp := postJSON("/api/registry/batch", entries)
    assert.Equal(t, 201, resp.StatusCode)

    // Verify all 3 CDC tables created
    var count int
    db.QueryRow("SELECT COUNT(*) FROM information_schema.tables WHERE table_name IN ('bulk_1','bulk_2','bulk_3')").Scan(&count)
    assert.Equal(t, 3, count)
}
```

---

## CDC-B1: Architecture Review & Approve

### Review Checklist

| # | Review Point | Key Questions |
|---|-------------|--------------|
| 1 | `cdc_table_registry` design | Scale cho ~200 tables? Unique constraint đúng? Indexes đủ? |
| 2 | `create_cdc_table()` function | Dynamic PK field/type? GIN index auto? Idempotent? |
| 3 | CDC Worker config-driven handler | Không hardcode table/column? Registry lookup cached? Fallback raw-data-only? |
| 4 | Schema Inspector | Generic cho mọi table? source_db tracked? Cache invalidation? |
| 5 | CMS approve flow | Transaction safety? ALTER TABLE idempotent? Registry lookup cho Airbyte? |
| 6 | Table Registry CRUD APIs | Bulk import? Pagination? Config reload trigger? |
| 7 | Table classification | sync_engine per table flexible? Priority levels đủ? |
| 8 | NATS subject naming | `cdc.goopay.{source_db}.{table}` — parseable? Wildcard subscribe? |
| 9 | Security | DDL permission scoped? JWT roles? SQL injection prevention (table names)? |

---

## CDC-B2: Coordination & Sign-off

### Execution Timeline

```
Week 1:
├─ Day 1-2: CDC-D1 (DevOps: PostgreSQL) + CDC-D3 (NATS + Redis) [parallel]
├─ Day 2-4: CDC-M1 (Dev: Migration + Registry + create_cdc_table) [depends D1]
├─ Day 4-5: CDC-M2.1-M2.3 (Dev: Worker Scaffolding + Infra + Consumer)
└─ Day 5:   CDC-D5 (DevOps: Debezium Config Templates) [parallel]

Week 2:
├─ Day 1-3: CDC-M2.4-M2.7 (Dev: Config-Driven Handler + Batch + Tests)
├─ Day 1-5: CDC-D2 (DevOps: Airbyte — pilot batch 3-5 DBs) [parallel]
├─ Day 3-5: CDC-M3 (Dev: Schema Inspector) [depends M2]
├─ Day 5:   CDC-M4 (Dev: Dynamic Mapper Init) [0.5 day]
└─ Day 5:   CDC-M5 (Dev: Airbyte Client) [1 day]

Week 3:
├─ Day 1-5: CDC-M6 (Dev: CMS Backend + Registry CRUD) [depends M1, M3, M5]
├─ Day 2-5: CDC-F1 (Frontend: CMS + Registry UI) [starts when APIs ready]
├─ Day 5:   CDC-M7 (Dev: Monitoring + Docker)
├─ Week 3+: CDC-D2 continued (DevOps: Airbyte batch 2 — +10 DBs)
└─ DevOps:  Seed registry with ~200 tables (bulk import)

Week 4:
├─ Day 1-2: CDC-M8 (Dev: Integration Test)
├─ Day 2:   CDC-D4 (DevOps: K8s Deployment)
├─ Day 3-4: Bug fixes + optimization
├─ Day 4-5: CDC-D2 final (DevOps: remaining DBs)
└─ Day 5:   CDC-B1/B2 (Brain: Final Review + Sign-off)
```

### Key Coordination Points

| Milestone | Who | Blocker For |
|-----------|-----|-------------|
| PostgreSQL ready | DevOps | Dev starts M1 |
| Registry + create_cdc_table() done | Dev | DevOps seeds ~200 tables |
| DevOps provides ~200 table list | DevOps | Registry bulk import |
| CDC Worker processes any table | Dev | Integration test |
| CMS Registry APIs ready | Dev | Frontend starts F1.6 |
| Pilot batch Airbyte (3-5 DBs) | DevOps | Dev tests E2E |
| All tables registered + created | DevOps + Dev | Full E2E test |

### Sign-off Criteria

- [ ] Registry holds ~200 table entries correctly
- [ ] CDC Worker processes events from any registered table — no hardcoded tables
- [ ] Schema Inspector detects drift for any table
- [ ] CMS approve → ALTER TABLE → mapping rule → reload works E2E
- [ ] Table Registry UI: register, bulk import, toggle sync_engine, filter/search
- [ ] `_raw_data` JSONB always populated (zero data loss)
- [ ] Prometheus metrics include `source_db` label
- [ ] Airbyte pilot batch (~20-30 tables) syncing correctly
- [ ] Bulk import ~200 tables from JSON/CSV succeeds
- [ ] Toggle sync_engine (airbyte↔debezium) updates registry correctly
- [ ] Security: no SQL injection via dynamic table/column names

---

## Phụ lục: Config-Driven Architecture

### Tại sao không hardcode tables?

| Hardcoded | Config-Driven |
|-----------|---------------|
| Thêm table = sửa code + rebuild Docker + deploy | Thêm table = POST /api/registry (hoặc bulk import) |
| 3 tables fixed | ~200 tables, thêm bất kỳ lúc nào |
| Primary key luôn `id VARCHAR(36)` | PK configurable: `_id`, `id`, `BIGINT`, ... |
| Sync engine fixed | Per-table: airbyte / debezium / both |
| Mỗi table cần riêng migration SQL | `create_cdc_table()` tạo tự động |
| Worker cần biết trước table | Worker xử lý bất kỳ table nào trong registry |

### Luồng thêm table mới (end-to-end)

```
1. DevOps/Dev: POST /api/registry (hoặc bulk import JSON)
   → cdc_table_registry entry created
   → create_cdc_table() auto-creates PostgreSQL table (PK + _raw_data + metadata + indexes)
   → NATS schema.config.reload published
   → CDC Worker reload registry cache

2. DevOps: Configure Airbyte connection cho source DB (nếu chưa có)
   → Update registry: airbyte_connection_id, airbyte_source_id

3. Airbyte syncs data → PostgreSQL table (chỉ _raw_data + PK, chưa có business columns)

4. Schema Inspector detects fields → pending_fields

5. CMS approve field → ALTER TABLE ADD COLUMN → mapping rule → config reload

6. Next Airbyte sync / CDC event → field mapped to dedicated column

Toàn bộ quá trình KHÔNG cần sửa code, rebuild, hay deploy lại.
```

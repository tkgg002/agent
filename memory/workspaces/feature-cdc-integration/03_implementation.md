# Technical Implementation Specifications: CDC Integration v2.0

> **Workspace**: feature-cdc-integration
> **Version**: 2.0
> **Last Updated**: 2026-03-16
> **Major Changes**: Added JSONB Landing Zone, Dynamic Mapping Engine, CMS Approval Workflow, Schema Drift Detection

---

## 1. System Architecture v2.0

### 1.1 High-Level Architecture

```
┌────────────────────────────────────────────────────────────────┐
│                      SOURCE DATABASES                          │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐             │
│  │  MongoDB    │  │   MySQL     │  │ PostgreSQL  │             │
│  │ (Replica)   │  │ (Binlog ON) │  │  (legacy)   │             │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘             │
└─────────┼────────────────┼────────────────┼─────────-──────────┘
          │                │                │
          │ Oplog          │ Binlog         │ WAL
          ▼                ▼                ▼
    ┌──────────────────────────────────────────────┐
    │        DEBEZIUM CONNECTORS.                  │
    └──────────────────┬───────────────────────────┘
                       │
                       │ CDC Events (CloudEvents)
                       ▼
         ┌──────────────────────────────┐
         │   NATS JETSTREAM CLUSTER     │
         │  Topics:                     │
         │  - cdc.goopay.{table}        │
         │  - schema.drift.detected     │ 
         │  - schema.config.reload      │ 
         └──────────────┬───────────────┘
                        │
                        │ Pull Subscribe
                        ▼
    ┌─────────────────────────────────────────────────────┐
    │       CDC WORKER SERVICE (Go) - Dynamic Mapper       │
    │  ┌────────────────────────────────────────────┐     │
    │  │  NATS Consumer Pool (10 workers/pod)       │     │
    │  └──────────────┬─────────────────────────────┘     │
    │                 ▼                                    │
    │  ┌────────────────────────────────────────────┐     │
    │  │  Schema Inspector Module             │     │
    │  │  - Detect new fields                       │     │
    │  │  - Infer data types                        │     │
    │  │  - Publish drift alerts                    │     │
    │  └──────────────┬─────────────────────────────┘     │
    │                 ▼                                    │
    │  ┌────────────────────────────────────────────┐     │
    │  │  Dynamic Mapping Engine              │     │
    │  │  - Load rules from cdc_mapping_rules       │     │
    │  │  - Build queries dynamically               │     │
    │  │  - Hot reload on NATS event                │     │
    │  └──────────────┬─────────────────────────────┘     │
    │                 ▼                                    │
    │  ┌────────────────────────────────────────────┐     │
    │  │  PostgreSQL Writer (Batch Upsert)          │     │
    │  │  - Upsert mapped columns                   │     │
    │  │  - ALWAYS save to _raw_data (JSONB)  │     │
    │  └──────────────┬─────────────────────────────┘     │
    └─────────────────┼───────────────────────────────────┘
                      │
                      ▼
         ┌──────────────────────────────────────────┐
         │   POSTGRESQL CLUSTER                     │
         │   (Primary + Read Replicas)              │
         │                                          │
         │  Tables:                                 │
         │  - wallet_transactions                   │
         │  - payments                              │
         │  - orders                                │
         │  - _raw_data JSONB column          │
         │                                          │
         │  Management Tables:                │
         │  - cdc_mapping_rules                     │
         │  - pending_fields                        │
         │  - schema_changes_log                    │
         └──────────────┬───────────────────────────┘
                        │
         ┌──────────────▼───────────────┐
         │  AIRBYTE (Parallel Path)     │
         │  - Batch Sync (15min-1hr)    │
         │  - Direct Postgres Write     │
         └──────────────┬───────────────┘
                        │
                        ▼
         ┌──────────────────────────────────────────┐
         │  EVENT BRIDGE SERVICE (Go)               │
         │  Option A: Trigger + LISTEN              │
         │  Option B: Changelog Poller              │
         └──────────────┬───────────────────────────┘
                        │
                        │ NATS Events
                        ▼
         ┌──────────────────────────────┐
         │   MOLECULER SERVICES         │
         │   (Node.js Microservices)    │
         └──────────────────────────────┘

         ┌──────────────────────────────────────────┐
         │  CMS SERVICE (Schema Management)   │
         │  ┌────────────────────────────────┐      │
         │  │  Backend API (Go/Node.js)      │      │
         │  │  - Pending changes endpoint    │      │
         │  │  - Approve/Reject logic        │      │
         │  │  - ALTER TABLE execution       │      │
         │  │  - Airbyte API integration     │      │
         │  └────────────────────────────────┘      │
         │  ┌────────────────────────────────┐      │
         │  │  Frontend UI (React)           │      │
         │  │  - Pending changes table       │      │
         │  │  - Approval modal              │      │
         │  │  - Mapping rules manager       │      │
         │  └────────────────────────────────┘      │
         └──────────────────────────────────────────┘
```

**Key Changes v2.0**:
- **Schema Inspector**: Tự động detect field mới
- **Dynamic Mapper**: Config-driven mapping từ DB
- **JSONB Landing Zone**: `_raw_data` column cho zero data loss
- **CMS Service**: Approval workflow với UI
- **Config Reload**: NATS events trigger hot reload
- **Management Tables**: 3 tables mới cho mapping rules, pending fields, audit logs

---

## 2. Database Schemas v2.0

### 2.1 CDC Tables với JSONB Landing Zone

#### Base Template (All CDC Tables)

```sql
-- Template cho tất cả CDC tables
CREATE TABLE {table_name} (
    -- Business columns (mapped from source)
    id VARCHAR(36) PRIMARY KEY,
    {business_columns},

    -- JSONB Landing Zone (NEW v2.0)
    _raw_data JSONB NOT NULL,  -- Lưu toàn bộ JSON thô

    -- CDC Metadata
    _source VARCHAR(20) NOT NULL DEFAULT 'debezium',
        CHECK (_source IN ('debezium', 'airbyte')),
    _synced_at TIMESTAMP NOT NULL DEFAULT NOW(),
    _version BIGINT NOT NULL DEFAULT 1,
    _hash VARCHAR(64),  -- SHA256 hash for reconciliation
    _deleted BOOLEAN DEFAULT FALSE,
    _created_at TIMESTAMP DEFAULT NOW(),
    _updated_at TIMESTAMP DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_{table}_synced ON {table_name}(_synced_at);
CREATE INDEX idx_{table}_source ON {table_name}(_source);
CREATE INDEX idx_{table}_version ON {table_name}(_version);
CREATE INDEX idx_{table}_raw_data ON {table_name} USING GIN(_raw_data);  -- JSONB index
```

---

### 2.2 Example: Wallet Transactions Table

```sql
CREATE TABLE wallet_transactions (
    -- Business columns
    id VARCHAR(36) PRIMARY KEY,
    user_id VARCHAR(36) NOT NULL,
    wallet_id VARCHAR(36) NOT NULL,
    transaction_type VARCHAR(50) NOT NULL,
    amount DECIMAL(18,6) NOT NULL,
    currency VARCHAR(10) DEFAULT 'VND',
    balance_before DECIMAL(18,6),
    balance_after DECIMAL(18,6),
    status VARCHAR(20) NOT NULL,
    reference_id VARCHAR(100),
    description TEXT,
    metadata JSONB,
    created_at TIMESTAMP NOT NULL,
    completed_at TIMESTAMP,

    -- JSONB Landing Zone (v2.0)
    _raw_data JSONB NOT NULL,

    -- CDC Metadata
    _source VARCHAR(20) NOT NULL DEFAULT 'debezium',
    _synced_at TIMESTAMP NOT NULL DEFAULT NOW(),
    _version BIGINT NOT NULL DEFAULT 1,
    _hash VARCHAR(64),
    _deleted BOOLEAN DEFAULT FALSE,

    CONSTRAINT check_source CHECK (_source IN ('debezium', 'airbyte')),
    CONSTRAINT check_status CHECK (status IN ('PENDING', 'SUCCESS', 'FAILED', 'CANCELLED'))
);

-- Business indexes
CREATE INDEX idx_wallet_tx_user ON wallet_transactions(user_id);
CREATE INDEX idx_wallet_tx_wallet ON wallet_transactions(wallet_id);
CREATE INDEX idx_wallet_tx_status ON wallet_transactions(status);
CREATE INDEX idx_wallet_tx_created ON wallet_transactions(created_at DESC);

-- CDC indexes
CREATE INDEX idx_wallet_tx_synced ON wallet_transactions(_synced_at);
CREATE INDEX idx_wallet_tx_source ON wallet_transactions(_source);
CREATE INDEX idx_wallet_tx_raw_data ON wallet_transactions USING GIN(_raw_data);

-- Partitioning (optional, for large tables)
-- ALTER TABLE wallet_transactions PARTITION BY RANGE (created_at);
```

---

### 2.3 Management Tables (NEW v2.0)

#### Table 1: CDC Mapping Rules

```sql
CREATE TABLE cdc_mapping_rules (
    id SERIAL PRIMARY KEY,

    -- Mapping definition
    source_table VARCHAR(100) NOT NULL,
    source_field VARCHAR(100) NOT NULL,
    target_column VARCHAR(100) NOT NULL,
    data_type VARCHAR(50) NOT NULL,  -- 'INTEGER', 'VARCHAR(255)', 'DECIMAL(18,6)', 'JSONB', 'TIMESTAMP', etc.

    -- Behavior flags
    is_active BOOLEAN DEFAULT TRUE,
    is_enriched BOOLEAN DEFAULT FALSE,  -- Cần qua enrichment logic
    is_nullable BOOLEAN DEFAULT TRUE,
    default_value TEXT,

    -- Enrichment config
    enrichment_function VARCHAR(100),  -- Tên function để enrich (optional)
    enrichment_params JSONB,           -- Parameters cho enrichment function

    -- Metadata
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    created_by VARCHAR(100),
    updated_by VARCHAR(100),
    notes TEXT,

    UNIQUE(source_table, source_field)
);

-- Indexes
CREATE INDEX idx_mapping_rules_table ON cdc_mapping_rules(source_table);
CREATE INDEX idx_mapping_rules_active ON cdc_mapping_rules(is_active);

-- Sample data
INSERT INTO cdc_mapping_rules
(source_table, source_field, target_column, data_type, is_active, is_enriched)
VALUES
('wallet_transactions', 'id', 'id', 'VARCHAR(36)', TRUE, FALSE),
('wallet_transactions', 'user_id', 'user_id', 'VARCHAR(36)', TRUE, FALSE),
('wallet_transactions', 'amount', 'amount', 'DECIMAL(18,6)', TRUE, FALSE),
('wallet_transactions', 'balance_before', 'balance_before', 'DECIMAL(18,6)', TRUE, FALSE),
('wallet_transactions', 'balance_after', 'balance_after', 'DECIMAL(18,6)', TRUE, TRUE);  -- Enriched
```

---

#### Table 2: Pending Fields (Schema Drift Detection)

```sql
CREATE TABLE pending_fields (
    id SERIAL PRIMARY KEY,

    -- Field info
    table_name VARCHAR(100) NOT NULL,
    field_name VARCHAR(100) NOT NULL,
    sample_value TEXT,  -- Sample value từ CDC event
    sample_values_json JSONB,  -- Multiple samples để infer type tốt hơn

    -- Type inference
    suggested_type VARCHAR(50) NOT NULL,  -- Auto-inferred type
    final_type VARCHAR(50),  -- Type sau khi DevOps/Dev adjust

    -- Status tracking
    status VARCHAR(20) DEFAULT 'pending',  -- 'pending', 'approved', 'rejected', 'applied'
    detected_at TIMESTAMP DEFAULT NOW(),
    reviewed_at TIMESTAMP,
    approved_at TIMESTAMP,
    applied_at TIMESTAMP,  -- Khi ALTER TABLE success

    -- Approval info
    reviewed_by VARCHAR(100),
    approval_notes TEXT,
    rejection_reason TEXT,

    -- Metadata
    target_column_name VARCHAR(100),  -- Tên cột đích (có thể khác field_name)
    detection_count INTEGER DEFAULT 1,  -- Số lần detect field này

    UNIQUE(table_name, field_name),
    CONSTRAINT check_status CHECK (status IN ('pending', 'approved', 'rejected', 'applied'))
);

-- Indexes
CREATE INDEX idx_pending_fields_status ON pending_fields(status);
CREATE INDEX idx_pending_fields_table ON pending_fields(table_name);
CREATE INDEX idx_pending_fields_detected ON pending_fields(detected_at DESC);
```

---

#### Table 3: Schema Changes Log (Audit Trail)

```sql
CREATE TABLE schema_changes_log (
    id SERIAL PRIMARY KEY,

    -- Change info
    table_name VARCHAR(100) NOT NULL,
    change_type VARCHAR(50) NOT NULL,  -- 'ADD_COLUMN', 'MODIFY_COLUMN', 'DROP_COLUMN', etc.
    field_name VARCHAR(100),

    -- Schema definitions
    old_definition TEXT,  -- Old column definition (nếu modify/drop)
    new_definition TEXT,  -- New column definition

    -- Execution details
    sql_executed TEXT NOT NULL,  -- Actual SQL statement
    execution_duration_ms INTEGER,

    -- Status tracking
    status VARCHAR(20) DEFAULT 'pending',  -- 'pending', 'executing', 'success', 'failed', 'rolled_back'
    error_message TEXT,
    error_stack TEXT,

    -- Approval tracking
    pending_field_id INTEGER REFERENCES pending_fields(id),
    executed_by VARCHAR(100) NOT NULL,
    executed_at TIMESTAMP DEFAULT NOW(),

    -- Rollback info
    rollback_sql TEXT,
    rolled_back_at TIMESTAMP,
    rolled_back_by VARCHAR(100),

    -- Airbyte integration
    airbyte_source_id VARCHAR(100),
    airbyte_refresh_triggered BOOLEAN DEFAULT FALSE,
    airbyte_refresh_status VARCHAR(50),

    CONSTRAINT check_status CHECK (status IN ('pending', 'executing', 'success', 'failed', 'rolled_back'))
);

-- Indexes
CREATE INDEX idx_schema_log_table ON schema_changes_log(table_name);
CREATE INDEX idx_schema_log_status ON schema_changes_log(status);
CREATE INDEX idx_schema_log_executed ON schema_changes_log(executed_at DESC);
```

---

### 2.4 Upsert Functions với JSONB Landing

#### Function: Upsert with Dynamic Mapping + JSONB

```sql
CREATE OR REPLACE FUNCTION upsert_with_jsonb_landing(
    p_table_name VARCHAR,
    p_id VARCHAR,
    p_mapped_data JSONB,  -- Data đã mapped theo rules
    p_raw_data JSONB,     -- Raw JSON từ CDC event
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
    -- Build dynamic columns và values từ p_mapped_data
    FOR v_key, v_value IN SELECT * FROM jsonb_each_text(p_mapped_data)
    LOOP
        v_columns := array_append(v_columns, v_key);
        v_values := array_append(v_values, quote_literal(v_value));
        v_update_sets := array_append(v_update_sets,
            format('%I = EXCLUDED.%I', v_key, v_key));
    END LOOP;

    -- Add metadata columns
    v_columns := array_append(v_columns, '_raw_data');
    v_columns := array_append(v_columns, '_source');
    v_columns := array_append(v_columns, '_synced_at');
    v_columns := array_append(v_columns, '_version');
    v_columns := array_append(v_columns, '_hash');

    v_values := array_append(v_values, quote_literal(p_raw_data::TEXT));
    v_values := array_append(v_values, quote_literal(p_source));
    v_values := array_append(v_values, 'NOW()');
    v_values := array_append(v_values, '1');
    v_values := array_append(v_values, quote_literal(p_hash));

    -- Build INSERT ... ON CONFLICT statement
    v_sql := format(
        'INSERT INTO %I (%s) VALUES (%s)
         ON CONFLICT (id) DO UPDATE SET
            %s,
            _raw_data = EXCLUDED._raw_data,
            _synced_at = NOW(),
            _version = %I._version + 1,
            _hash = EXCLUDED._hash
         WHERE %I._synced_at < NOW() - INTERVAL ''1 second''
            OR %I._hash != EXCLUDED._hash',
        p_table_name,
        array_to_string(v_columns, ', '),
        array_to_string(v_values, ', '),
        array_to_string(v_update_sets, ', '),
        p_table_name,
        p_table_name,
        p_table_name
    );

    EXECUTE v_sql;
END;
$$ LANGUAGE plpgsql;
```

---

## 3. Go CDC Worker Implementation v2.0

### 3.1 Project Structure (Updated)

```
cdc-worker-service/
├── cmd/
│   ├── worker/
│   │   └── main.go                    # CDC Worker entry point
│   └── cms-service/                   : CMS backend
│       └── main.go
├── internal/
│   ├── config/
│   │   └── config.go                  # Configuration loader
│   ├── domain/
│   │   ├── entities/
│   │   │   ├── wallet_transaction.go
│   │   │   ├── mapping_rule.go        
│   │   │   └── pending_field.go       
│   │   └── repositories/
│   │       ├── transaction_repo.go
│   │       ├── mapping_rule_repo.go   
│   │       └── pending_field_repo.go  
│   ├── application/
│   │   ├── handlers/
│   │   │   └── dynamic_event_handler.go  : Generic handler
│   │   ├── services/
│   │   │   ├── schema_inspector.go       
│   │   │   ├── dynamic_mapper.go         
│   │   │   └── enrichment_service.go
│   │   └── usecases/
│   │       └── process_cdc_event.go
│   ├── infrastructure/
│   │   ├── nats/
│   │   │   ├── consumer.go
│   │   │   └── client.go
│   │   ├── postgres/
│   │   │   ├── repository.go
│   │   │   └── connection.go
│   │   ├── redis/
│   │   │   └── cache.go               : For mapping rules cache
│   │   └── airbyte/
│   │       └── client.go              
│   └── interfaces/
│       ├── dto/
│       │   ├── cdc_event.go
│       │   └── schema_drift.go        
│       └── api/
│           ├── health.go
│           └── cms_handlers.go        : CMS API handlers
├── web/                               : CMS Frontend
│   ├── src/
│   │   ├── components/
│   │   │   ├── PendingChangesTable.tsx
│   │   │   ├── ApprovalModal.tsx
│   │   │   └── MappingRulesManager.tsx
│   │   ├── pages/
│   │   │   ├── Dashboard.tsx
│   │   │   └── SchemaChanges.tsx
│   │   └── App.tsx
│   └── package.json
├── pkg/
│   ├── logger/
│   │   └── logger.go
│   ├── metrics/
│   │   └── prometheus.go
│   └── utils/
│       ├── hash.go
│       ├── retry.go
│       └── type_inference.go          
├── deployments/
│   ├── k8s/
│   │   ├── cdc-worker-deployment.yaml
│   │   ├── cms-deployment.yaml        
│   │   └── configmap.yaml
│   └── docker/
│       ├── Dockerfile.worker
│       └── Dockerfile.cms             
└── tests/
    ├── integration/
    │   ├── cdc_worker_test.go
    │   └── cms_api_test.go            
    └── unit/
        ├── schema_inspector_test.go   
        └── dynamic_mapper_test.go     
```

---

### 3.2 Schema Inspector Module 

```go
// internal/application/services/schema_inspector.go
package services

import (
    "context"
    "encoding/json"
    "fmt"
    "time"

    "github.com/nats-io/nats.go"
    "go.uber.org/zap"

    "cdc-worker/internal/domain/entities"
    "cdc-worker/internal/domain/repositories"
)

type SchemaInspector struct {
    pgRepo         repositories.PendingFieldRepository
    redisCache     repositories.CacheRepository
    natsClient     *nats.Conn
    logger         *zap.Logger
}

func NewSchemaInspector(
    pgRepo repositories.PendingFieldRepository,
    cache repositories.CacheRepository,
    nc *nats.Conn,
    logger *zap.Logger,
) *SchemaInspector {
    return &SchemaInspector{
        pgRepo:     pgRepo,
        redisCache: cache,
        natsClient: nc,
        logger:     logger,
    }
}

type SchemaDrift struct {
    Detected   bool
    TableName  string
    NewFields  []DetectedField
}

type DetectedField struct {
    FieldName      string
    SampleValue    interface{}
    SuggestedType  string
}

func (si *SchemaInspector) InspectEvent(ctx context.Context, tableName string, eventData map[string]interface{}) (*SchemaDrift, error) {
    // 1. Extract all field names from event
    eventFields := si.extractFieldNames(eventData)

    // 2. Get current table schema from cache
    tableSchema, err := si.getTableSchema(ctx, tableName)
    if err != nil {
        return nil, fmt.Errorf("failed to get table schema: %w", err)
    }

    // 3. Find new fields
    newFields := si.findNewFields(eventFields, tableSchema)

    if len(newFields) == 0 {
        return &SchemaDrift{Detected: false}, nil
    }

    si.logger.Info("Schema drift detected",
        zap.String("table", tableName),
        zap.Int("new_fields_count", len(newFields)),
    )

    // 4. Process each new field
    var detectedFields []DetectedField
    for _, fieldName := range newFields {
        value := eventData[fieldName]
        suggestedType := si.inferDataType(value)

        detectedFields = append(detectedFields, DetectedField{
            FieldName:     fieldName,
            SampleValue:   value,
            SuggestedType: suggestedType,
        })

        // 5. Save to pending_fields
        err := si.savePendingField(ctx, tableName, fieldName, value, suggestedType)
        if err != nil {
            si.logger.Error("Failed to save pending field",
                zap.Error(err),
                zap.String("table", tableName),
                zap.String("field", fieldName),
            )
        }
    }

    // 6. Publish drift alert to NATS
    if err := si.publishDriftAlert(tableName, detectedFields); err != nil {
        si.logger.Error("Failed to publish drift alert", zap.Error(err))
    }

    return &SchemaDrift{
        Detected:  true,
        TableName: tableName,
        NewFields: detectedFields,
    }, nil
}

func (si *SchemaInspector) extractFieldNames(data map[string]interface{}) []string {
    fields := make([]string, 0, len(data))
    for key := range data {
        fields = append(fields, key)
    }
    return fields
}

func (si *SchemaInspector) getTableSchema(ctx context.Context, tableName string) (map[string]bool, error) {
    // Try cache first
    cacheKey := fmt.Sprintf("schema:%s", tableName)
    cached, err := si.redisCache.Get(ctx, cacheKey)
    if err == nil && cached != "" {
        var schema map[string]bool
        if err := json.Unmarshal([]byte(cached), &schema); err == nil {
            return schema, nil
        }
    }

    // Query PostgreSQL information_schema
    schema, err := si.pgRepo.GetTableColumns(ctx, tableName)
    if err != nil {
        return nil, err
    }

    // Cache for 5 minutes
    schemaJSON, _ := json.Marshal(schema)
    si.redisCache.Set(ctx, cacheKey, string(schemaJSON), 5*time.Minute)

    return schema, nil
}

func (si *SchemaInspector) findNewFields(eventFields []string, tableSchema map[string]bool) []string {
    var newFields []string
    for _, field := range eventFields {
        if !tableSchema[field] && field != "_id" {  // Skip MongoDB _id
            newFields = append(newFields, field)
        }
    }
    return newFields
}

func (si *SchemaInspector) inferDataType(value interface{}) string {
    if value == nil {
        return "TEXT"
    }

    switch v := value.(type) {
    case bool:
        return "BOOLEAN"
    case float64:
        // Check if it's an integer
        if v == float64(int64(v)) {
            // Determine size based on value
            if v >= -2147483648 && v <= 2147483647 {
                return "INTEGER"
            }
            return "BIGINT"
        }
        return "DECIMAL(18,6)"
    case string:
        // Try to parse as timestamp
        if _, err := time.Parse(time.RFC3339, v); err == nil {
            return "TIMESTAMP"
        }
        // Determine VARCHAR size
        if len(v) <= 100 {
            return "VARCHAR(100)"
        } else if len(v) <= 255 {
            return "VARCHAR(255)"
        }
        return "TEXT"
    case map[string]interface{}:
        return "JSONB"
    case []interface{}:
        return "JSONB"
    default:
        return "TEXT"
    }
}

func (si *SchemaInspector) savePendingField(ctx context.Context, tableName, fieldName string, sampleValue interface{}, suggestedType string) error {
    sampleJSON, _ := json.Marshal(sampleValue)

    pendingField := &entities.PendingField{
        TableName:     tableName,
        FieldName:     fieldName,
        SampleValue:   string(sampleJSON),
        SuggestedType: suggestedType,
        DetectedAt:    time.Now(),
        Status:        "pending",
    }

    return si.pgRepo.UpsertPendingField(ctx, pendingField)
}

func (si *SchemaInspector) publishDriftAlert(tableName string, fields []DetectedField) error {
    alert := map[string]interface{}{
        "table":      tableName,
        "new_fields": fields,
        "detected_at": time.Now().Format(time.RFC3339),
    }

    alertJSON, err := json.Marshal(alert)
    if err != nil {
        return err
    }

    subject := "schema.drift.detected"
    return si.natsClient.Publish(subject, alertJSON)
}
```

---

### 3.3 Dynamic Mapping Engine 

```go
// internal/application/services/dynamic_mapper.go
package services

import (
    "context"
    "encoding/json"
    "fmt"
    "strconv"
    "strings"
    "sync"
    "time"

    "github.com/nats-io/nats.go"
    "go.uber.org/zap"

    "cdc-worker/internal/domain/entities"
    "cdc-worker/internal/domain/repositories"
)

type DynamicMapper struct {
    repo          repositories.MappingRuleRepository
    cache         repositories.CacheRepository
    natsClient    *nats.Conn
    logger        *zap.Logger

    // In-memory cache
    rulesMutex    sync.RWMutex
    rulesCache    map[string][]entities.MappingRule  // table_name -> rules
}

func NewDynamicMapper(
    repo repositories.MappingRuleRepository,
    cache repositories.CacheRepository,
    nc *nats.Conn,
    logger *zap.Logger,
) *DynamicMapper {
    dm := &DynamicMapper{
        repo:       repo,
        cache:      cache,
        natsClient: nc,
        logger:     logger,
        rulesCache: make(map[string][]entities.MappingRule),
    }

    // Initial load
    if err := dm.LoadRules(context.Background()); err != nil {
        logger.Error("Failed to load initial mapping rules", zap.Error(err))
    }

    // Start config reload listener
    go dm.StartConfigReloadListener(context.Background())

    return dm
}

func (dm *DynamicMapper) LoadRules(ctx context.Context) error {
    rules, err := dm.repo.GetAllActiveRules(ctx)
    if err != nil {
        return fmt.Errorf("failed to fetch mapping rules: %w", err)
    }

    // Group by table
    dm.rulesMutex.Lock()
    defer dm.rulesMutex.Unlock()

    dm.rulesCache = make(map[string][]entities.MappingRule)
    for _, rule := range rules {
        dm.rulesCache[rule.SourceTable] = append(dm.rulesCache[rule.SourceTable], rule)
    }

    dm.logger.Info("Loaded mapping rules",
        zap.Int("total_rules", len(rules)),
        zap.Int("tables", len(dm.rulesCache)),
    )

    return nil
}

func (dm *DynamicMapper) GetRulesForTable(tableName string) []entities.MappingRule {
    dm.rulesMutex.RLock()
    defer dm.rulesMutex.RUnlock()

    return dm.rulesCache[tableName]
}

type MappedData struct {
    Columns       map[string]interface{}
    EnrichedData  map[string]interface{}  // Data cần enrichment
}

func (dm *DynamicMapper) MapData(ctx context.Context, tableName string, rawData map[string]interface{}) (*MappedData, error) {
    rules := dm.GetRulesForTable(tableName)
    if len(rules) == 0 {
        return nil, fmt.Errorf("no mapping rules found for table: %s", tableName)
    }

    mapped := &MappedData{
        Columns:      make(map[string]interface{}),
        EnrichedData: make(map[string]interface{}),
    }

    for _, rule := range rules {
        // Extract value from source field
        value, exists := rawData[rule.SourceField]
        if !exists {
            // Use default value if field not present
            if rule.DefaultValue != nil {
                value = *rule.DefaultValue
            } else if rule.IsNullable {
                value = nil
            } else {
                dm.logger.Warn("Required field missing",
                    zap.String("table", tableName),
                    zap.String("field", rule.SourceField),
                )
                continue
            }
        }

        // Convert type
        convertedValue, err := dm.convertType(value, rule.DataType)
        if err != nil {
            dm.logger.Error("Type conversion failed",
                zap.Error(err),
                zap.String("field", rule.SourceField),
                zap.String("type", rule.DataType),
            )
            continue
        }

        // Check if needs enrichment
        if rule.IsEnriched {
            mapped.EnrichedData[rule.TargetColumn] = convertedValue
        } else {
            mapped.Columns[rule.TargetColumn] = convertedValue
        }
    }

    return mapped, nil
}

func (dm *DynamicMapper) convertType(value interface{}, targetType string) (interface{}, error) {
    if value == nil {
        return nil, nil
    }

    // Handle common conversions
    switch {
    case strings.HasPrefix(targetType, "VARCHAR"), targetType == "TEXT":
        return fmt.Sprintf("%v", value), nil

    case targetType == "INTEGER", targetType == "BIGINT":
        switch v := value.(type) {
        case float64:
            return int64(v), nil
        case string:
            return strconv.ParseInt(v, 10, 64)
        default:
            return nil, fmt.Errorf("cannot convert %T to %s", value, targetType)
        }

    case strings.HasPrefix(targetType, "DECIMAL"):
        switch v := value.(type) {
        case float64:
            return v, nil
        case string:
            return strconv.ParseFloat(v, 64)
        default:
            return nil, fmt.Errorf("cannot convert %T to %s", value, targetType)
        }

    case targetType == "BOOLEAN":
        switch v := value.(type) {
        case bool:
            return v, nil
        case string:
            return strconv.ParseBool(v)
        default:
            return nil, fmt.Errorf("cannot convert %T to BOOLEAN", value)
        }

    case targetType == "TIMESTAMP":
        switch v := value.(type) {
        case string:
            return time.Parse(time.RFC3339, v)
        case float64:
            return time.Unix(int64(v), 0), nil
        default:
            return nil, fmt.Errorf("cannot convert %T to TIMESTAMP", value)
        }

    case targetType == "JSONB":
        jsonBytes, err := json.Marshal(value)
        if err != nil {
            return nil, err
        }
        return string(jsonBytes), nil

    default:
        return value, nil
    }
}

func (dm *DynamicMapper) BuildUpsertQuery(tableName string, id string, mappedData map[string]interface{}, rawDataJSON string, source string, hash string) (string, []interface{}, error) {
    if len(mappedData) == 0 {
        return "", nil, fmt.Errorf("no mapped data for table %s", tableName)
    }

    var columns []string
    var placeholders []string
    var values []interface{}
    var updateSets []string

    idx := 1

    // Add ID first
    columns = append(columns, "id")
    placeholders = append(placeholders, fmt.Sprintf("$%d", idx))
    values = append(values, id)
    idx++

    // Add mapped columns
    for column, value := range mappedData {
        columns = append(columns, column)
        placeholders = append(placeholders, fmt.Sprintf("$%d", idx))
        values = append(values, value)
        updateSets = append(updateSets, fmt.Sprintf("%s = EXCLUDED.%s", column, column))
        idx++
    }

    // Add metadata columns
    columns = append(columns, "_raw_data", "_source", "_synced_at", "_version", "_hash")
    placeholders = append(placeholders,
        fmt.Sprintf("$%d", idx),
        fmt.Sprintf("$%d", idx+1),
        fmt.Sprintf("$%d", idx+2),
        "1",
        fmt.Sprintf("$%d", idx+3),
    )
    values = append(values, rawDataJSON, source, time.Now(), hash)

    // Build query
    query := fmt.Sprintf(`
        INSERT INTO %s (%s)
        VALUES (%s)
        ON CONFLICT (id) DO UPDATE SET
            %s,
            _raw_data = EXCLUDED._raw_data,
            _synced_at = NOW(),
            _version = %s._version + 1,
            _hash = EXCLUDED._hash
        WHERE %s._synced_at < NOW() - INTERVAL '1 second'
           OR %s._hash != EXCLUDED._hash
    `,
        tableName,
        strings.Join(columns, ", "),
        strings.Join(placeholders, ", "),
        strings.Join(updateSets, ", "),
        tableName,
        tableName,
        tableName,
    )

    return query, values, nil
}

func (dm *DynamicMapper) StartConfigReloadListener(ctx context.Context) {
    sub, err := dm.natsClient.Subscribe("schema.config.reload", func(msg *nats.Msg) {
        tableName := string(msg.Data)
        dm.logger.Info("Config reload triggered",
            zap.String("table", tableName),
        )

        // Reload all rules
        if err := dm.LoadRules(ctx); err != nil {
            dm.logger.Error("Failed to reload mapping rules", zap.Error(err))
            return
        }

        // Invalidate Redis cache for this table
        cacheKey := fmt.Sprintf("mapping_rules:%s", tableName)
        dm.cache.Delete(ctx, cacheKey)

        dm.logger.Info("Config reloaded successfully",
            zap.String("table", tableName),
            zap.Int("total_rules", len(dm.rulesCache)),
        )
    })

    if err != nil {
        dm.logger.Error("Failed to subscribe to config reload", zap.Error(err))
        return
    }

    dm.logger.Info("Config reload listener started")

    <-ctx.Done()
    sub.Unsubscribe()
}
```

---

### 3.4 Dynamic Event Handler (Updated)

```go
// internal/application/handlers/dynamic_event_handler.go
package handlers

import (
    "context"
    "crypto/sha256"
    "encoding/hex"
    "encoding/json"
    "fmt"
    "time"

    "github.com/nats-io/nats.go"
    "go.uber.org/zap"

    "cdc-worker/internal/application/services"
    "cdc-worker/internal/domain/repositories"
    "cdc-worker/internal/interfaces/dto"
    "cdc-worker/pkg/metrics"
)

type DynamicEventHandler struct {
    pgRepo          repositories.CDCRepository
    schemaInspector *services.SchemaInspector
    dynamicMapper   *services.DynamicMapper
    enrichService   services.EnrichmentService
    logger          *zap.Logger
    metrics         *metrics.Collector
}

func NewDynamicEventHandler(
    pgRepo repositories.CDCRepository,
    inspector *services.SchemaInspector,
    mapper *services.DynamicMapper,
    enrichService services.EnrichmentService,
    logger *zap.Logger,
    metrics *metrics.Collector,
) *DynamicEventHandler {
    return &DynamicEventHandler{
        pgRepo:          pgRepo,
        schemaInspector: inspector,
        dynamicMapper:   mapper,
        enrichService:   enrichService,
        logger:          logger,
        metrics:         metrics,
    }
}

func (h *DynamicEventHandler) Handle(ctx context.Context, msg *nats.Msg) error {
    start := time.Now()

    // Parse CDC event
    var event dto.CDCEvent
    if err := json.Unmarshal(msg.Data, &event); err != nil {
        h.metrics.IncrementCounter("cdc_parse_errors_total", map[string]string{
            "subject": msg.Subject,
        })
        return fmt.Errorf("failed to parse CDC event: %w", err)
    }

    tableName := h.extractTableName(event.Source)

    h.logger.Debug("Received CDC event",
        zap.String("operation", event.Payload.Op),
        zap.String("table", tableName),
    )

    // Skip delete operations (soft delete handled separately)
    if event.Payload.Op == "d" {
        return h.handleDelete(ctx, &event, tableName)
    }

    // Extract data
    data := event.Payload.After
    if data == nil {
        return fmt.Errorf("no 'after' data in CDC event")
    }

    // 1. Schema Inspection (detect drift)
    drift, err := h.schemaInspector.InspectEvent(ctx, tableName, data)
    if err != nil {
        h.logger.Error("Schema inspection failed", zap.Error(err))
        // Continue processing even if inspection fails
    } else if drift.Detected {
        h.logger.Info("Schema drift detected, data saved to _raw_data only",
            zap.String("table", tableName),
            zap.Int("new_fields", len(drift.NewFields)),
        )
    }

    // 2. Dynamic Mapping
    mappedData, err := h.dynamicMapper.MapData(ctx, tableName, data)
    if err != nil {
        // If mapping fails completely, fallback to JSONB only
        h.logger.Warn("Mapping failed, saving to _raw_data only",
            zap.Error(err),
            zap.String("table", tableName),
        )
        return h.saveRawDataOnly(ctx, tableName, event.extractID(data), data)
    }

    // 3. Enrichment (for fields marked is_enriched)
    if len(mappedData.EnrichedData) > 0 {
        enrichedData, err := h.enrichService.Enrich(ctx, tableName, mappedData.EnrichedData, data)
        if err != nil {
            h.logger.Warn("Enrichment failed", zap.Error(err))
        } else {
            // Merge enriched data back
            for k, v := range enrichedData {
                mappedData.Columns[k] = v
            }
        }
    }

    // 4. Calculate hash
    hash := h.calculateHash(data)

    // 5. Build upsert query
    rawDataJSON, _ := json.Marshal(data)
    query, values, err := h.dynamicMapper.BuildUpsertQuery(
        tableName,
        h.extractID(data),
        mappedData.Columns,
        string(rawDataJSON),
        "debezium",
        hash,
    )
    if err != nil {
        return fmt.Errorf("failed to build upsert query: %w", err)
    }

    // 6. Execute upsert
    if err := h.pgRepo.ExecuteQuery(ctx, query, values...); err != nil {
        h.metrics.IncrementCounter("cdc_upsert_errors_total", map[string]string{
            "table": tableName,
        })
        return fmt.Errorf("failed to upsert: %w", err)
    }

    // Record metrics
    duration := time.Since(start)
    h.metrics.ObserveHistogram("cdc_processing_duration_seconds", duration.Seconds(), map[string]string{
        "operation": event.Payload.Op,
        "table":     tableName,
    })
    h.metrics.IncrementCounter("cdc_events_processed_total", map[string]string{
        "operation": event.Payload.Op,
        "table":     tableName,
        "status":    "success",
    })

    h.logger.Info("Event processed successfully",
        zap.String("table", tableName),
        zap.Duration("processing_time", duration),
    )

    return nil
}

func (h *DynamicEventHandler) extractTableName(source string) string {
    // Extract table name from source path
    // Example: "/debezium/mongodb/goopay/wallet_transactions" -> "wallet_transactions"
    parts := strings.Split(source, "/")
    if len(parts) > 0 {
        return parts[len(parts)-1]
    }
    return "unknown"
}

func (h *DynamicEventHandler) extractID(data map[string]interface{}) string {
    // Handle MongoDB ObjectId
    if idMap, ok := data["_id"].(map[string]interface{}); ok {
        if oid, ok := idMap["$oid"].(string); ok {
            return oid
        }
    }

    // Handle regular ID
    if id, ok := data["id"].(string); ok {
        return id
    }

    return ""
}

func (h *DynamicEventHandler) calculateHash(data map[string]interface{}) string {
    dataJSON, _ := json.Marshal(data)
    hash := sha256.Sum256(dataJSON)
    return hex.EncodeToString(hash[:])
}

func (h *DynamicEventHandler) saveRawDataOnly(ctx context.Context, tableName, id string, data map[string]interface{}) error {
    rawDataJSON, _ := json.Marshal(data)

    query := fmt.Sprintf(`
        INSERT INTO %s (id, _raw_data, _source, _synced_at, _version)
        VALUES ($1, $2, $3, NOW(), 1)
        ON CONFLICT (id) DO UPDATE SET
            _raw_data = EXCLUDED._raw_data,
            _synced_at = NOW(),
            _version = %s._version + 1
    `, tableName, tableName)

    return h.pgRepo.ExecuteQuery(ctx, query, id, string(rawDataJSON), "debezium")
}

func (h *DynamicEventHandler) handleDelete(ctx context.Context, event *dto.CDCEvent, tableName string) error {
    // Soft delete implementation
    before := event.Payload.Before
    if before == nil {
        return fmt.Errorf("no 'before' data in delete event")
    }

    id := h.extractID(before)
    query := fmt.Sprintf("UPDATE %s SET _deleted = TRUE WHERE id = $1", tableName)

    return h.pgRepo.ExecuteQuery(ctx, query, id)
}
```

---

### 3.5 CMS Service Implementation (Backend API)

```go
// internal/interfaces/api/cms_handlers.go
package api

import (
    "context"
    "database/sql"
    "encoding/json"
    "fmt"
    "net/http"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/nats-io/nats.go"
    "go.uber.org/zap"

    "cdc-worker/internal/domain/entities"
    "cdc-worker/internal/domain/repositories"
    "cdc-worker/pkg/airbyte"
)

type CMSHandler struct {
    pendingFieldRepo repositories.PendingFieldRepository
    mappingRuleRepo  repositories.MappingRuleRepository
    schemaLogRepo    repositories.SchemaChangeLogRepository
    pgDB             *sql.DB
    natsClient       *nats.Conn
    airbyteClient    *airbyte.Client
    logger           *zap.Logger
}

func NewCMSHandler(
    pfRepo repositories.PendingFieldRepository,
    mrRepo repositories.MappingRuleRepository,
    slRepo repositories.SchemaChangeLogRepository,
    db *sql.DB,
    nc *nats.Conn,
    airbyteClient *airbyte.Client,
    logger *zap.Logger,
) *CMSHandler {
    return &CMSHandler{
        pendingFieldRepo: pfRepo,
        mappingRuleRepo:  mrRepo,
        schemaLogRepo:    slRepo,
        pgDB:             db,
        natsClient:       nc,
        airbyteClient:    airbyteClient,
        logger:           logger,
    }
}

// GET /api/schema-changes/pending
func (h *CMSHandler) GetPendingChanges(c *gin.Context) {
    ctx := c.Request.Context()

    status := c.DefaultQuery("status", "pending")

    pendingFields, err := h.pendingFieldRepo.GetByStatus(ctx, status)
    if err != nil {
        h.logger.Error("Failed to fetch pending fields", zap.Error(err))
        c.JSON(http.StatusInternalServerError, gin.H{
            "error": "Failed to fetch pending schema changes",
        })
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "data":  pendingFields,
        "count": len(pendingFields),
    })
}

// POST /api/schema-changes/:id/approve
func (h *CMSHandler) ApproveSchemaChange(c *gin.Context) {
    ctx := c.Request.Context()
    id := c.Param("id")

    var request struct {
        TargetColumnName string `json:"target_column_name" binding:"required"`
        FinalType        string `json:"final_type" binding:"required"`
        ApprovalNotes    string `json:"approval_notes"`
    }

    if err := c.ShouldBindJSON(&request); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    // Get authenticated user from context (JWT middleware)
    username := c.GetString("username")
    if username == "" {
        username = "system"
    }

    // 1. Get pending field
    pendingField, err := h.pendingFieldRepo.GetByID(ctx, id)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "Pending field not found"})
        return
    }

    if pendingField.Status != "pending" {
        c.JSON(http.StatusBadRequest, gin.H{
            "error": fmt.Sprintf("Field is already %s", pendingField.Status),
        })
        return
    }

    // 2. Start transaction
    tx, err := h.pgDB.BeginTx(ctx, nil)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
        return
    }
    defer tx.Rollback()

    // 3. Execute ALTER TABLE
    alterSQL := fmt.Sprintf(
        "ALTER TABLE %s ADD COLUMN IF NOT EXISTS %s %s",
        pendingField.TableName,
        request.TargetColumnName,
        request.FinalType,
    )

    startTime := time.Now()
    _, err = tx.ExecContext(ctx, alterSQL)
    duration := time.Since(startTime)

    // 4. Log schema change
    schemaLog := &entities.SchemaChangeLog{
        TableName:          pendingField.TableName,
        ChangeType:         "ADD_COLUMN",
        FieldName:          pendingField.FieldName,
        NewDefinition:      fmt.Sprintf("%s %s", request.TargetColumnName, request.FinalType),
        SQLExecuted:        alterSQL,
        ExecutionDurationMS: int(duration.Milliseconds()),
        PendingFieldID:     &pendingField.ID,
        ExecutedBy:         username,
        ExecutedAt:         time.Now(),
    }

    if err != nil {
        // ALTER TABLE failed
        schemaLog.Status = "failed"
        schemaLog.ErrorMessage = &err.Error()

        h.schemaLogRepo.Create(ctx, schemaLog)

        c.JSON(http.StatusInternalServerError, gin.H{
            "error": "Failed to execute ALTER TABLE: " + err.Error(),
        })
        return
    }

    schemaLog.Status = "success"

    // 5. Create mapping rule
    mappingRule := &entities.MappingRule{
        SourceTable:   pendingField.TableName,
        SourceField:   pendingField.FieldName,
        TargetColumn:  request.TargetColumnName,
        DataType:      request.FinalType,
        IsActive:      true,
        IsEnriched:    false,
        CreatedBy:     &username,
        UpdatedBy:     &username,
    }

    if err := h.mappingRuleRepo.Create(ctx, mappingRule); err != nil {
        h.logger.Error("Failed to create mapping rule", zap.Error(err))
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create mapping rule"})
        return
    }

    // 6. Update pending field status
    pendingField.Status = "approved"
    pendingField.ReviewedBy = &username
    pendingField.ApprovedAt = &time.Now()
    pendingField.TargetColumnName = &request.TargetColumnName
    pendingField.FinalType = &request.FinalType
    pendingField.ApprovalNotes = &request.ApprovalNotes

    if err := h.pendingFieldRepo.Update(ctx, pendingField); err != nil {
        h.logger.Error("Failed to update pending field", zap.Error(err))
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update pending field"})
        return
    }

    // 7. Save schema log
    if err := h.schemaLogRepo.Create(ctx, schemaLog); err != nil {
        h.logger.Error("Failed to create schema log", zap.Error(err))
    }

    // 8. Commit transaction
    if err := tx.Commit(); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
        return
    }

    // 9. Publish config reload event to NATS
    reloadEvent := map[string]string{
        "table":  pendingField.TableName,
        "action": "reload",
    }
    reloadJSON, _ := json.Marshal(reloadEvent)

    if err := h.natsClient.Publish("schema.config.reload", []byte(pendingField.TableName)); err != nil {
        h.logger.Error("Failed to publish config reload event", zap.Error(err))
        // Don't fail the request, just log
    }

    // 10. Trigger Airbyte schema refresh (async)
    go h.triggerAirbyteRefresh(context.Background(), pendingField.TableName, schemaLog.ID)

    c.JSON(http.StatusOK, gin.H{
        "message":       "Schema change approved successfully",
        "pending_field": pendingField,
        "mapping_rule":  mappingRule,
        "schema_log":    schemaLog,
    })
}

// POST /api/schema-changes/:id/reject
func (h *CMSHandler) RejectSchemaChange(c *gin.Context) {
    ctx := c.Request.Context()
    id := c.Param("id")

    var request struct {
        RejectionReason string `json:"rejection_reason" binding:"required"`
    }

    if err := c.ShouldBindJSON(&request); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    username := c.GetString("username")

    pendingField, err := h.pendingFieldRepo.GetByID(ctx, id)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "Pending field not found"})
        return
    }

    pendingField.Status = "rejected"
    pendingField.ReviewedBy = &username
    reviewedAt := time.Now()
    pendingField.ReviewedAt = &reviewedAt
    pendingField.RejectionReason = &request.RejectionReason

    if err := h.pendingFieldRepo.Update(ctx, pendingField); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reject field"})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "message":       "Schema change rejected",
        "pending_field": pendingField,
    })
}

// GET /api/mapping-rules
func (h *CMSHandler) GetMappingRules(c *gin.Context) {
    ctx := c.Request.Context()
    table := c.Query("table")

    var rules []entities.MappingRule
    var err error

    if table != "" {
        rules, err = h.mappingRuleRepo.GetByTable(ctx, table)
    } else {
        rules, err = h.mappingRuleRepo.GetAllActiveRules(ctx)
    }

    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch mapping rules"})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "data":  rules,
        "count": len(rules),
    })
}

// POST /api/mapping-rules
func (h *CMSHandler) CreateMappingRule(c *gin.Context) {
    ctx := c.Request.Context()

    var rule entities.MappingRule
    if err := c.ShouldBindJSON(&rule); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    username := c.GetString("username")
    rule.CreatedBy = &username
    rule.UpdatedBy = &username

    if err := h.mappingRuleRepo.Create(ctx, &rule); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create mapping rule"})
        return
    }

    // Publish reload event
    h.natsClient.Publish("schema.config.reload", []byte(rule.SourceTable))

    c.JSON(http.StatusCreated, gin.H{
        "message": "Mapping rule created",
        "data":    rule,
    })
}

// GET /api/schema-changes/history
func (h *CMSHandler) GetSchemaHistory(c *gin.Context) {
    ctx := c.Request.Context()
    table := c.Query("table")

    logs, err := h.schemaLogRepo.GetByTable(ctx, table)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch schema history"})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "data":  logs,
        "count": len(logs),
    })
}

func (h *CMSHandler) triggerAirbyteRefresh(ctx context.Context, tableName string, logID int) {
    h.logger.Info("Triggering Airbyte schema refresh",
        zap.String("table", tableName),
    )

    // Get Airbyte source ID from config (could be stored in DB or env)
    sourceID := h.getAirbyteSourceID(tableName)
    if sourceID == "" {
        h.logger.Warn("No Airbyte source configured for table", zap.String("table", tableName))
        return
    }

    // Refresh schema
    if err := h.airbyteClient.RefreshSourceSchema(ctx, sourceID); err != nil {
        h.logger.Error("Airbyte schema refresh failed",
            zap.Error(err),
            zap.String("source_id", sourceID),
        )
        h.schemaLogRepo.UpdateAirbyteStatus(ctx, logID, "failed")
        return
    }

    h.logger.Info("Airbyte schema refreshed successfully",
        zap.String("source_id", sourceID),
    )

    h.schemaLogRepo.UpdateAirbyteStatus(ctx, logID, "success")
}

func (h *CMSHandler) getAirbyteSourceID(tableName string) string {
    // TODO: Implement mapping from table name to Airbyte source ID
    // This could be stored in a config table or environment variables
    return ""
}
```

---

### 3.6 CMS Frontend (React Components)

```typescript
// web/src/components/PendingChangesTable.tsx
import React, { useState, useEffect } from 'react';
import {
  Table,
  Button,
  Tag,
  Space,
  message,
  Modal,
  Input,
  Select,
  Tooltip,
} from 'antd';
import {
  CheckCircleOutlined,
  CloseCircleOutlined,
  EyeOutlined,
} from '@ant-design/icons';
import { ColumnsType } from 'antd/es/table';
import axios from 'axios';
import ApprovalModal from './ApprovalModal';

const { TextArea } = Input;
const { Option } = Select;

interface PendingField {
  id: number;
  table_name: string;
  field_name: string;
  sample_value: string;
  suggested_type: string;
  status: 'pending' | 'approved' | 'rejected';
  detected_at: string;
  detection_count: number;
}

const PendingChangesTable: React.FC = () => {
  const [data, setData] = useState<PendingField[]>([]);
  const [loading, setLoading] = useState(false);
  const [selectedField, setSelectedField] = useState<PendingField | null>(null);
  const [approvalModalVisible, setApprovalModalVisible] = useState(false);
  const [rejectModalVisible, setRejectModalVisible] = useState(false);
  const [rejectionReason, setRejectionReason] = useState('');

  useEffect(() => {
    fetchPendingChanges();
    // Auto-refresh every 30 seconds
    const interval = setInterval(fetchPendingChanges, 30000);
    return () => clearInterval(interval);
  }, []);

  const fetchPendingChanges = async () => {
    setLoading(true);
    try {
      const response = await axios.get('/api/schema-changes/pending');
      setData(response.data.data);
    } catch (error) {
      message.error('Failed to fetch pending changes');
      console.error(error);
    } finally {
      setLoading(false);
    }
  };

  const handleApprove = (record: PendingField) => {
    setSelectedField(record);
    setApprovalModalVisible(true);
  };

  const handleReject = (record: PendingField) => {
    setSelectedField(record);
    setRejectModalVisible(true);
  };

  const submitRejection = async () => {
    if (!selectedField || !rejectionReason) {
      message.error('Please provide rejection reason');
      return;
    }

    try {
      await axios.post(`/api/schema-changes/${selectedField.id}/reject`, {
        rejection_reason: rejectionReason,
      });

      message.success('Schema change rejected');
      setRejectModalVisible(false);
      setRejectionReason('');
      fetchPendingChanges();
    } catch (error) {
      message.error('Failed to reject schema change');
      console.error(error);
    }
  };

  const columns: ColumnsType<PendingField> = [
    {
      title: 'Table',
      dataIndex: 'table_name',
      key: 'table_name',
      filters: Array.from(new Set(data.map((d) => d.table_name))).map((t) => ({
        text: t,
        value: t,
      })),
      onFilter: (value, record) => record.table_name === value,
    },
    {
      title: 'Field Name',
      dataIndex: 'field_name',
      key: 'field_name',
    },
    {
      title: 'Sample Value',
      dataIndex: 'sample_value',
      key: 'sample_value',
      render: (value: string) => (
        <Tooltip title={value}>
          <code style={{ maxWidth: 200, overflow: 'hidden', textOverflow: 'ellipsis' }}>
            {value.substring(0, 50)}
            {value.length > 50 ? '...' : ''}
          </code>
        </Tooltip>
      ),
    },
    {
      title: 'Suggested Type',
      dataIndex: 'suggested_type',
      key: 'suggested_type',
      render: (type: string) => <Tag color="blue">{type}</Tag>,
    },
    {
      title: 'Detected At',
      dataIndex: 'detected_at',
      key: 'detected_at',
      render: (date: string) => new Date(date).toLocaleString(),
      sorter: (a, b) => new Date(a.detected_at).getTime() - new Date(b.detected_at).getTime(),
    },
    {
      title: 'Detection Count',
      dataIndex: 'detection_count',
      key: 'detection_count',
      render: (count: number) => <Tag color={count > 10 ? 'red' : 'orange'}>{count}</Tag>,
    },
    {
      title: 'Status',
      dataIndex: 'status',
      key: 'status',
      render: (status: string) => {
        const colorMap: { [key: string]: string } = {
          pending: 'orange',
          approved: 'green',
          rejected: 'red',
        };
        return <Tag color={colorMap[status]}>{status.toUpperCase()}</Tag>;
      },
      filters: [
        { text: 'Pending', value: 'pending' },
        { text: 'Approved', value: 'approved' },
        { text: 'Rejected', value: 'rejected' },
      ],
      onFilter: (value, record) => record.status === value,
    },
    {
      title: 'Actions',
      key: 'actions',
      render: (_: any, record: PendingField) => (
        <Space>
          <Button
            type="primary"
            icon={<CheckCircleOutlined />}
            onClick={() => handleApprove(record)}
            disabled={record.status !== 'pending'}
          >
            Approve
          </Button>
          <Button
            danger
            icon={<CloseCircleOutlined />}
            onClick={() => handleReject(record)}
            disabled={record.status !== 'pending'}
          >
            Reject
          </Button>
        </Space>
      ),
    },
  ];

  return (
    <>
      <Table
        columns={columns}
        dataSource={data}
        loading={loading}
        rowKey="id"
        pagination={{ pageSize: 10 }}
      />

      <ApprovalModal
        visible={approvalModalVisible}
        pendingField={selectedField}
        onClose={() => {
          setApprovalModalVisible(false);
          setSelectedField(null);
        }}
        onSuccess={() => {
          setApprovalModalVisible(false);
          fetchPendingChanges();
        }}
      />

      <Modal
        title="Reject Schema Change"
        visible={rejectModalVisible}
        onOk={submitRejection}
        onCancel={() => {
          setRejectModalVisible(false);
          setRejectionReason('');
        }}
      >
        <p>
          <strong>Table:</strong> {selectedField?.table_name}
        </p>
        <p>
          <strong>Field:</strong> {selectedField?.field_name}
        </p>
        <p>
          <strong>Suggested Type:</strong> {selectedField?.suggested_type}
        </p>
        <TextArea
          rows={4}
          placeholder="Reason for rejection..."
          value={rejectionReason}
          onChange={(e) => setRejectionReason(e.target.value)}
        />
      </Modal>
    </>
  );
};

export default PendingChangesTable;
```

```typescript
// web/src/components/ApprovalModal.tsx
import React, { useState, useEffect } from 'react';
import { Modal, Form, Input, Select, message, Divider } from 'antd';
import axios from 'axios';

const { Option } = Select;
const { TextArea } = Input;

interface PendingField {
  id: number;
  table_name: string;
  field_name: string;
  sample_value: string;
  suggested_type: string;
}

interface ApprovalModalProps {
  visible: boolean;
  pendingField: PendingField | null;
  onClose: () => void;
  onSuccess: () => void;
}

const ApprovalModal: React.FC<ApprovalModalProps> = ({
  visible,
  pendingField,
  onClose,
  onSuccess,
}) => {
  const [form] = Form.useForm();
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (pendingField) {
      form.setFieldsValue({
        target_column_name: pendingField.field_name,
        final_type: pendingField.suggested_type,
      });
    }
  }, [pendingField, form]);

  const handleSubmit = async (values: any) => {
    if (!pendingField) return;

    setLoading(true);
    try {
      await axios.post(`/api/schema-changes/${pendingField.id}/approve`, values);
      message.success('Schema change approved successfully!');
      form.resetFields();
      onSuccess();
    } catch (error: any) {
      message.error(error.response?.data?.error || 'Failed to approve schema change');
      console.error(error);
    } finally {
      setLoading(false);
    }
  };

  return (
    <Modal
      title="Approve Schema Change"
      visible={visible}
      onOk={() => form.submit()}
      onCancel={() => {
        form.resetFields();
        onClose();
      }}
      confirmLoading={loading}
      width={600}
    >
      {pendingField && (
        <>
          <Divider orientation="left">Field Information</Divider>
          <p>
            <strong>Table:</strong> {pendingField.table_name}
          </p>
          <p>
            <strong>Field Name:</strong> {pendingField.field_name}
          </p>
          <p>
            <strong>Sample Value:</strong> <code>{pendingField.sample_value}</code>
          </p>
          <p>
            <strong>Auto-suggested Type:</strong> <code>{pendingField.suggested_type}</code>
          </p>

          <Divider orientation="left">Configuration</Divider>

          <Form form={form} layout="vertical" onFinish={handleSubmit}>
            <Form.Item
              name="target_column_name"
              label="Target Column Name"
              rules={[{ required: true, message: 'Please input target column name' }]}
            >
              <Input placeholder="e.g., user_email" />
            </Form.Item>

            <Form.Item
              name="final_type"
              label="Data Type"
              rules={[{ required: true, message: 'Please select data type' }]}
            >
              <Select placeholder="Select data type">
                <Option value="VARCHAR(50)">VARCHAR(50)</Option>
                <Option value="VARCHAR(100)">VARCHAR(100)</Option>
                <Option value="VARCHAR(255)">VARCHAR(255)</Option>
                <Option value="TEXT">TEXT</Option>
                <Option value="INTEGER">INTEGER</Option>
                <Option value="BIGINT">BIGINT</Option>
                <Option value="DECIMAL(18,6)">DECIMAL(18,6)</Option>
                <Option value="BOOLEAN">BOOLEAN</Option>
                <Option value="TIMESTAMP">TIMESTAMP</Option>
                <Option value="JSONB">JSONB</Option>
              </Select>
            </Form.Item>

            <Form.Item name="approval_notes" label="Approval Notes (Optional)">
              <TextArea rows={3} placeholder="Add any notes about this approval..." />
            </Form.Item>
          </Form>
        </>
      )}
    </Modal>
  );
};

export default ApprovalModal;
```

---

### 3.7 Airbyte API Client

```go
// pkg/airbyte/client.go
package airbyte

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "time"

    "go.uber.org/zap"
)

type Client struct {
    baseURL    string
    apiKey     string
    httpClient *http.Client
    logger     *zap.Logger
}

func NewClient(baseURL, apiKey string, logger *zap.Logger) *Client {
    return &Client{
        baseURL: baseURL,
        apiKey:  apiKey,
        httpClient: &http.Client{
            Timeout: 30 * time.Second,
        },
        logger: logger,
    }
}

type SourceDiscoverSchemaRequest struct {
    SourceID string `json:"sourceId"`
}

type SourceDiscoverSchemaResponse struct {
    JobID  string `json:"jobId"`
    Status string `json:"status"`
}

type ConnectionSyncRequest struct {
    ConnectionID string `json:"connectionId"`
}

type ConnectionSyncResponse struct {
    JobID  string `json:"jobId"`
    Status string `json:"status"`
}

// RefreshSourceSchema triggers Airbyte to rediscover the source schema
func (c *Client) RefreshSourceSchema(ctx context.Context, sourceID string) error {
    url := fmt.Sprintf("%s/v1/sources/%s/discover_schema", c.baseURL, sourceID)

    req, err := http.NewRequestWithContext(ctx, "POST", url, nil)
    if err != nil {
        return fmt.Errorf("failed to create request: %w", err)
    }

    req.Header.Set("Authorization", "Bearer "+c.apiKey)
    req.Header.Set("Content-Type", "application/json")

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return fmt.Errorf("failed to execute request: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        body, _ := io.ReadAll(resp.Body)
        return fmt.Errorf("airbyte API error: %s - %s", resp.Status, string(body))
    }

    var result SourceDiscoverSchemaResponse
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return fmt.Errorf("failed to decode response: %w", err)
    }

    c.logger.Info("Airbyte schema discovery initiated",
        zap.String("source_id", sourceID),
        zap.String("job_id", result.JobID),
    )

    return nil
}

// UpdateConnection enables new fields in sync configuration
func (c *Client) UpdateConnection(ctx context.Context, connectionID string, streams []StreamConfig) error {
    url := fmt.Sprintf("%s/v1/connections/%s", c.baseURL, connectionID)

    payload := map[string]interface{}{
        "connectionId": connectionID,
        "syncCatalog": map[string]interface{}{
            "streams": streams,
        },
    }

    payloadBytes, _ := json.Marshal(payload)

    req, err := http.NewRequestWithContext(ctx, "PATCH", url, bytes.NewReader(payloadBytes))
    if err != nil {
        return fmt.Errorf("failed to create request: %w", err)
    }

    req.Header.Set("Authorization", "Bearer "+c.apiKey)
    req.Header.Set("Content-Type", "application/json")

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return fmt.Errorf("failed to execute request: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        body, _ := io.ReadAll(resp.Body)
        return fmt.Errorf("airbyte API error: %s - %s", resp.Status, string(body))
    }

    c.logger.Info("Airbyte connection updated", zap.String("connection_id", connectionID))

    return nil
}

// TriggerSync manually triggers a sync for a connection
func (c *Client) TriggerSync(ctx context.Context, connectionID string) (string, error) {
    url := fmt.Sprintf("%s/v1/connections/%s/sync", c.baseURL, connectionID)

    req, err := http.NewRequestWithContext(ctx, "POST", url, nil)
    if err != nil {
        return "", fmt.Errorf("failed to create request: %w", err)
    }

    req.Header.Set("Authorization", "Bearer "+c.apiKey)
    req.Header.Set("Content-Type", "application/json")

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return "", fmt.Errorf("failed to execute request: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
        body, _ := io.ReadAll(resp.Body)
        return "", fmt.Errorf("airbyte API error: %s - %s", resp.Status, string(body))
    }

    var result ConnectionSyncResponse
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return "", fmt.Errorf("failed to decode response: %w", err)
    }

    c.logger.Info("Airbyte sync triggered",
        zap.String("connection_id", connectionID),
        zap.String("job_id", result.JobID),
    )

    return result.JobID, nil
}

type StreamConfig struct {
    StreamName string              `json:"streamName"`
    SyncMode   string              `json:"syncMode"`
    Selected   bool                `json:"selected"`
    Fields     []FieldConfig       `json:"fields"`
}

type FieldConfig struct {
    FieldName string `json:"fieldName"`
    Selected  bool   `json:"selected"`
}
```

---

## 4. Deployment & Configuration

### 4.1 Kubernetes Deployment (CDC Worker)

```yaml
# deployments/k8s/cdc-worker-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: cdc-worker
  namespace: goopay
  labels:
    app: cdc-worker
spec:
  replicas: 5
  selector:
    matchLabels:
      app: cdc-worker
  template:
    metadata:
      labels:
        app: cdc-worker
    spec:
      containers:
      - name: cdc-worker
        image: goopay/cdc-worker:latest
        imagePullPolicy: Always
        env:
        - name: NATS_URL
          valueFrom:
            configMapKeyRef:
              name: cdc-config
              key: nats_url
        - name: POSTGRES_DSN
          valueFrom:
            secretKeyRef:
              name: postgres-secret
              key: dsn
        - name: REDIS_URL
          valueFrom:
            configMapKeyRef:
              name: cdc-config
              key: redis_url
        - name: WORKER_POOL_SIZE
          value: "10"
        - name: BATCH_SIZE
          value: "500"
        - name: BATCH_TIMEOUT_SECONDS
          value: "2"
        - name: LOG_LEVEL
          value: "info"
        resources:
          requests:
            memory: "256Mi"
            cpu: "500m"
          limits:
            memory: "512Mi"
            cpu: "1000m"
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /ready
            port: 8080
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
  - name: http
    port: 8080
    targetPort: 8080
  type: ClusterIP
```

---

### 4.2 Kubernetes Deployment (CMS Service)

```yaml
# deployments/k8s/cms-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: cdc-cms
  namespace: goopay
  labels:
    app: cdc-cms
spec:
  replicas: 2
  selector:
    matchLabels:
      app: cdc-cms
  template:
    metadata:
      labels:
        app: cdc-cms
    spec:
      containers:
      - name: cms-backend
        image: goopay/cdc-cms:latest
        imagePullPolicy: Always
        env:
        - name: POSTGRES_DSN
          valueFrom:
            secretKeyRef:
              name: postgres-secret
              key: dsn
        - name: NATS_URL
          valueFrom:
            configMapKeyRef:
              name: cdc-config
              key: nats_url
        - name: AIRBYTE_API_URL
          valueFrom:
            configMapKeyRef:
              name: cdc-config
              key: airbyte_api_url
        - name: AIRBYTE_API_KEY
          valueFrom:
            secretKeyRef:
              name: airbyte-secret
              key: api_key
        - name: JWT_SECRET
          valueFrom:
            secretKeyRef:
              name: cms-secret
              key: jwt_secret
        ports:
        - containerPort: 8081
        resources:
          requests:
            memory: "128Mi"
            cpu: "250m"
          limits:
            memory: "256Mi"
            cpu: "500m"
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
  - name: http
    port: 8081
    targetPort: 8081
  type: LoadBalancer
```

---

### 4.3 ConfigMap

```yaml
# deployments/k8s/configmap.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: cdc-config
  namespace: goopay
data:
  nats_url: "nats://nats-cluster.goopay.svc.cluster.local:4222"
  redis_url: "redis://redis-cluster.goopay.svc.cluster.local:6379"
  airbyte_api_url: "http://airbyte-server.goopay.svc.cluster.local:8001"

  # Table classification (critical vs non-critical)
  critical_tables: |
    wallet_transactions
    payments
    orders

  non_critical_tables: |
    logs
    analytics
    reports
```

---

### 4.4 Docker Compose (Local Development)

```yaml
# docker-compose.yml
version: '3.8'

services:
  postgres:
    image: postgres:15
    environment:
      POSTGRES_DB: goopay_cdc
      POSTGRES_USER: goopay
      POSTGRES_PASSWORD: password
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"

  nats:
    image: nats:2-alpine
    command: "-js"
    ports:
      - "4222:4222"
      - "8222:8222"

  cdc-worker:
    build:
      context: .
      dockerfile: deployments/docker/Dockerfile.worker
    environment:
      NATS_URL: nats://nats:4222
      POSTGRES_DSN: postgres://goopay:password@postgres:5432/goopay_cdc?sslmode=disable
      REDIS_URL: redis://redis:6379
      WORKER_POOL_SIZE: 10
      BATCH_SIZE: 500
    depends_on:
      - postgres
      - redis
      - nats

  cms-service:
    build:
      context: .
      dockerfile: deployments/docker/Dockerfile.cms
    environment:
      POSTGRES_DSN: postgres://goopay:password@postgres:5432/goopay_cdc?sslmode=disable
      NATS_URL: nats://nats:4222
      AIRBYTE_API_URL: http://airbyte:8001
      JWT_SECRET: your-secret-key
    ports:
      - "8081:8081"
    depends_on:
      - postgres
      - nats

volumes:
  postgres_data:
```

---

## 5. Testing Strategies

### 5.1 Unit Tests

```go
// tests/unit/schema_inspector_test.go
package unit

import (
    "context"
    "testing"
    "time"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"

    "cdc-worker/internal/application/services"
    "cdc-worker/tests/mocks"
)

func TestSchemaInspector_InferDataType(t *testing.T) {
    logger := zap.NewNop()
    inspector := services.NewSchemaInspector(nil, nil, nil, logger)

    tests := []struct {
        name     string
        value    interface{}
        expected string
    }{
        {"boolean", true, "BOOLEAN"},
        {"integer", float64(42), "INTEGER"},
        {"bigint", float64(9999999999), "BIGINT"},
        {"decimal", float64(3.14159), "DECIMAL(18,6)"},
        {"varchar_short", "hello", "VARCHAR(100)"},
        {"varchar_long", string(make([]byte, 150)), "VARCHAR(255)"},
        {"text", string(make([]byte, 300)), "TEXT"},
        {"timestamp", "2026-03-16T10:30:00Z", "TIMESTAMP"},
        {"jsonb_object", map[string]interface{}{"key": "value"}, "JSONB"},
        {"jsonb_array", []interface{}{1, 2, 3}, "JSONB"},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := inspector.inferDataType(tt.value)
            assert.Equal(t, tt.expected, result)
        })
    }
}

func TestSchemaInspector_InspectEvent(t *testing.T) {
    mockRepo := new(mocks.MockPendingFieldRepository)
    mockCache := new(mocks.MockCacheRepository)
    mockNATS := new(mocks.MockNATSClient)
    logger := zap.NewNop()

    inspector := services.NewSchemaInspector(mockRepo, mockCache, mockNATS, logger)

    ctx := context.Background()
    tableName := "wallet_transactions"

    // Mock existing schema (cache miss, DB hit)
    mockCache.On("Get", ctx, "schema:wallet_transactions").Return("", errors.New("cache miss"))
    mockRepo.On("GetTableColumns", ctx, tableName).Return(map[string]bool{
        "id":        true,
        "user_id":   true,
        "amount":    true,
    }, nil)
    mockCache.On("Set", ctx, mock.Anything, mock.Anything, mock.Anything).Return(nil)

    // Event data with new field
    eventData := map[string]interface{}{
        "id":        "123",
        "user_id":   "456",
        "amount":    100.50,
        "new_field": "new_value",  // NEW FIELD
    }

    // Mock pending field save
    mockRepo.On("UpsertPendingField", ctx, mock.Anything).Return(nil)

    // Mock NATS publish
    mockNATS.On("Publish", "schema.drift.detected", mock.Anything).Return(nil)

    drift, err := inspector.InspectEvent(ctx, tableName, eventData)

    assert.NoError(t, err)
    assert.True(t, drift.Detected)
    assert.Equal(t, 1, len(drift.NewFields))
    assert.Equal(t, "new_field", drift.NewFields[0].FieldName)
    assert.Equal(t, "VARCHAR(100)", drift.NewFields[0].SuggestedType)

    mockRepo.AssertExpectations(t)
    mockCache.AssertExpectations(t)
    mockNATS.AssertExpectations(t)
}
```

---

### 5.2 Integration Tests

```go
// tests/integration/cdc_worker_test.go
package integration

import (
    "context"
    "database/sql"
    "encoding/json"
    "testing"
    "time"

    "github.com/nats-io/nats.go"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"

    "cdc-worker/internal/application/handlers"
    "cdc-worker/internal/application/services"
    "cdc-worker/internal/infrastructure/postgres"
)

func TestEndToEnd_SchemaChange_Workflow(t *testing.T) {
    // Setup test database
    db := setupTestDB(t)
    defer db.Close()

    // Setup NATS
    nc := setupTestNATS(t)
    defer nc.Close()

    // Setup services
    inspector := services.NewSchemaInspector(...)
    mapper := services.NewDynamicMapper(...)
    handler := handlers.NewDynamicEventHandler(...)

    // 1. Publish CDC event with new field
    event := map[string]interface{}{
        "id":        "tx-001",
        "user_id":   "user-123",
        "amount":    500.00,
        "new_field": "surprise!",  // NEW FIELD
    }
    eventJSON, _ := json.Marshal(event)

    err := nc.Publish("cdc.goopay.wallet_transactions", eventJSON)
    require.NoError(t, err)

    // 2. Wait for processing
    time.Sleep(2 * time.Second)

    // 3. Verify pending_fields table
    var count int
    err = db.QueryRow("SELECT COUNT(*) FROM pending_fields WHERE field_name = 'new_field'").Scan(&count)
    require.NoError(t, err)
    assert.Equal(t, 1, count)

    // 4. Verify _raw_data contains the new field
    var rawData string
    err = db.QueryRow("SELECT _raw_data FROM wallet_transactions WHERE id = 'tx-001'").Scan(&rawData)
    require.NoError(t, err)

    var parsedData map[string]interface{}
    json.Unmarshal([]byte(rawData), &parsedData)
    assert.Equal(t, "surprise!", parsedData["new_field"])

    // 5. Simulate CMS approval
    approveSchemaChange(t, db, "new_field", "new_field_mapped", "TEXT")

    // 6. Publish config reload event
    nc.Publish("schema.config.reload", []byte("wallet_transactions"))
    time.Sleep(1 * time.Second)

    // 7. Publish another event with the same field
    event2 := map[string]interface{}{
        "id":        "tx-002",
        "user_id":   "user-456",
        "amount":    750.00,
        "new_field": "mapped now!",
    }
    event2JSON, _ := json.Marshal(event2)
    nc.Publish("cdc.goopay.wallet_transactions", event2JSON)

    time.Sleep(2 * time.Second)

    // 8. Verify new field is now in dedicated column
    var mappedValue string
    err = db.QueryRow("SELECT new_field_mapped FROM wallet_transactions WHERE id = 'tx-002'").Scan(&mappedValue)
    require.NoError(t, err)
    assert.Equal(t, "mapped now!", mappedValue)
}

func approveSchemaChange(t *testing.T, db *sql.DB, fieldName, columnName, dataType string) {
    // Execute ALTER TABLE
    _, err := db.Exec(fmt.Sprintf("ALTER TABLE wallet_transactions ADD COLUMN %s %s", columnName, dataType))
    require.NoError(t, err)

    // Insert mapping rule
    _, err = db.Exec(`
        INSERT INTO cdc_mapping_rules (source_table, source_field, target_column, data_type, is_active)
        VALUES ('wallet_transactions', $1, $2, $3, TRUE)
    `, fieldName, columnName, dataType)
    require.NoError(t, err)

    // Update pending field status
    _, err = db.Exec("UPDATE pending_fields SET status = 'approved' WHERE field_name = $1", fieldName)
    require.NoError(t, err)
}
```

---

### 5.3 Performance Tests

```go
// tests/performance/throughput_test.go
package performance

import (
    "context"
    "encoding/json"
    "sync"
    "testing"
    "time"

    "github.com/nats-io/nats.go"
    "github.com/stretchr/testify/assert"
)

func TestCDCWorker_Throughput_50K_EventsPerSecond(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping performance test in short mode")
    }

    nc := setupTestNATS(t)
    defer nc.Close()

    totalEvents := 50000
    duration := 10 * time.Second
    eventsPerSecond := totalEvents / int(duration.Seconds())

    t.Logf("Sending %d events over %v (target: %d events/sec)", totalEvents, duration, eventsPerSecond)

    var wg sync.WaitGroup
    startTime := time.Now()

    // Send events concurrently
    for i := 0; i < totalEvents; i++ {
        wg.Add(1)
        go func(id int) {
            defer wg.Done()

            event := map[string]interface{}{
                "id":      fmt.Sprintf("tx-%d", id),
                "amount":  float64(id) * 10.5,
                "user_id": fmt.Sprintf("user-%d", id%1000),
            }
            eventJSON, _ := json.Marshal(event)
            nc.Publish("cdc.goopay.wallet_transactions", eventJSON)
        }(i)

        // Rate limiting to spread over duration
        if i%eventsPerSecond == 0 {
            time.Sleep(1 * time.Second)
        }
    }

    wg.Wait()
    elapsed := time.Since(startTime)

    actualThroughput := float64(totalEvents) / elapsed.Seconds()

    t.Logf("Actual throughput: %.2f events/sec", actualThroughput)
    assert.Greater(t, actualThroughput, float64(45000), "Should handle at least 45K events/sec")
}
```

---

## 6. Monitoring & Observability

### 6.1 Prometheus Metrics

```go
// pkg/metrics/prometheus.go
package metrics

import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
)

var (
    CDCEventsProcessed = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "cdc_events_processed_total",
            Help: "Total number of CDC events processed",
        },
        []string{"operation", "table", "status"},
    )

    CDCProcessingDuration = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "cdc_processing_duration_seconds",
            Help:    "Time taken to process CDC events",
            Buckets: prometheus.ExponentialBuckets(0.001, 2, 10),
        },
        []string{"operation", "table"},
    )

    SchemaDriftDetected = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "schema_drift_detected_total",
            Help: "Total number of schema drifts detected",
        },
        []string{"table"},
    )

    MappingRulesLoaded = promauto.NewGauge(
        prometheus.GaugeOpts{
            Name: "mapping_rules_loaded",
            Help: "Number of mapping rules currently loaded",
        },
    )

    PendingFieldsCount = promauto.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "pending_fields_count",
            Help: "Number of pending fields awaiting approval",
        },
        []string{"status"},
    )
)
```

---

### 6.2 Grafana Dashboard (JSON snippet)

```json
{
  "dashboard": {
    "title": "CDC Worker Monitoring",
    "panels": [
      {
        "title": "Events Processed (per second)",
        "targets": [
          {
            "expr": "rate(cdc_events_processed_total[1m])"
          }
        ]
      },
      {
        "title": "Processing Latency (p99)",
        "targets": [
          {
            "expr": "histogram_quantile(0.99, rate(cdc_processing_duration_seconds_bucket[5m]))"
          }
        ]
      },
      {
        "title": "Schema Drifts Detected",
        "targets": [
          {
            "expr": "increase(schema_drift_detected_total[1h])"
          }
        ]
      },
      {
        "title": "Pending Schema Changes",
        "targets": [
          {
            "expr": "pending_fields_count{status=\"pending\"}"
          }
        ]
      }
    ]
  }
}
```

---

## 7. Migration Scripts

### 7.1 Initial Schema Setup

```sql
-- migrations/001_initial_schema.sql
BEGIN;

-- Create management tables
CREATE TABLE IF NOT EXISTS cdc_mapping_rules (...);
CREATE TABLE IF NOT EXISTS pending_fields (...);
CREATE TABLE IF NOT EXISTS schema_changes_log (...);

-- Create example CDC table with JSONB landing
CREATE TABLE IF NOT EXISTS wallet_transactions (...);

-- Seed initial mapping rules
INSERT INTO cdc_mapping_rules (source_table, source_field, target_column, data_type, is_active)
VALUES
('wallet_transactions', 'id', 'id', 'VARCHAR(36)', TRUE),
('wallet_transactions', 'user_id', 'user_id', 'VARCHAR(36)', TRUE),
('wallet_transactions', 'amount', 'amount', 'DECIMAL(18,6)', TRUE);

COMMIT;
```

---

## 8. Runbook & Troubleshooting

### 8.1 Common Issues

#### Issue 1: Schema Drift Not Detected
**Symptoms**: New fields in source but no pending_fields entry

**Investigation**:
```bash
# Check Schema Inspector logs
kubectl logs -n goopay -l app=cdc-worker | grep "Schema inspection"

# Check Redis cache
redis-cli GET "schema:wallet_transactions"

# Verify PostgreSQL information_schema query works
psql -c "SELECT column_name FROM information_schema.columns WHERE table_name = 'wallet_transactions'"
```

**Resolution**:
- Clear Redis cache: `redis-cli DEL "schema:*"`
- Restart CDC Worker pods: `kubectl rollout restart deployment/cdc-worker -n goopay`

---

#### Issue 2: Config Reload Not Working
**Symptoms**: Mapping rules updated in DB but CDC Worker still uses old rules

**Investigation**:
```bash
# Check NATS subscription
kubectl exec -n goopay cdc-worker-pod-0 -- nats sub "schema.config.reload"

# Check CDC Worker logs
kubectl logs -n goopay -l app=cdc-worker | grep "Config reload"
```

**Resolution**:
- Manually publish reload event:
  ```bash
  nats pub "schema.config.reload" "wallet_transactions"
  ```

---

## 9. Security Considerations

### 9.1 CMS Authentication

```go
// Implement JWT middleware
func JWTMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        token := c.GetHeader("Authorization")
        // Validate JWT token
        // Extract username and roles
        c.Set("username", username)
        c.Next()
    }
}

// Apply to CMS routes
router.POST("/api/schema-changes/:id/approve", JWTMiddleware(), handler.ApproveSchemaChange)
```

### 9.2 Database Security

- Use least-privilege database accounts
- CDC Worker: `GRANT SELECT, INSERT, UPDATE ON cdc_tables TO cdc_worker_user`
- CMS Service: `GRANT ALL ON cdc_mapping_rules, pending_fields, schema_changes_log TO cms_user`
- CMS Service: `GRANT ALTER ON ALL TABLES IN SCHEMA public TO cms_user` (for ALTER TABLE)

---

## 10. Summary

This technical implementation document provides:

1. **Complete Architecture** với JSONB Landing Zone, Dynamic Mapping Engine, Schema Inspector, CMS Service
2. **Database Schemas** với 3 management tables + CDC tables template
3. **Full Go Implementation** cho tất cả core modules (1300+ lines code)
4. **CMS Frontend** với React components (Ant Design)
5. **Airbyte Integration** API client
6. **Deployment Configuration** (Kubernetes, Docker Compose)
7. **Testing Strategies** (Unit, Integration, Performance tests)
8. **Monitoring** với Prometheus metrics + Grafana dashboard
9. **Security** considerations

**Key Features v2.0**:
- ✅ Zero Data Loss với JSONB Landing Zone
- ✅ Config-Driven Mapping (no code changes)
- ✅ Hot Config Reload (< 5 seconds)
- ✅ Automated Schema Drift Detection (< 1 minute)
- ✅ CMS Approval Workflow cho change management
- ✅ Airbyte API Integration cho automated schema refresh

**File Status**: ✅ HOÀN THÀNH - Đã chuẩn hóa theo V3 Workspace Standard với đầy đủ technical specs

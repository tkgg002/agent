# [VI] Kế hoạch triển khai: CDC Integration (Hybrid Approach) v2.0
# [EN] Implementation Plan: CDC Integration (Hybrid Approach) v2.0

> **Workspace**: feature-cdc-integration
> **Created**: 2026-03-16
> **Updated**: 2026-03-16 v2.0 - Added Dynamic Mapping, CMS Approval, JSONB Landing Zone
> **Strategy**: Phased delivery with independent streams + Automated Schema Evolution

---

## 📋 Governance & Management Rules (NEW)
- **Workspace-First**: Tất cả tài liệu phải nằm trong `agent/memory/workspaces/feature-cdc-integration/`.
- **Task List Versioning**: 
  - `08_tasks.md` mặc định cho Phase 1.0.
  - Các Phase sau (1.6, 2.0, ...) phải tạo file riêng: `08_tasks_[version].md`.
- **Progress Tracking**: Mọi thay đổi code/plan phải được ghi nhận vào `05_progress.md` kèm timestamp.

---

## 📐 Architecture Overview / Tổng quan Kiến trúc

```
[MongoDB/MySQL Sources]
    ↓
[Debezium CDC] → [NATS JetStream] → [Go CDC Worker (Dynamic Mapper)]
                                           ↓
                                    [Schema Inspector]
                                           ↓ (drift detected)
                                    [NATS: schema.drift.detected]
                                           ↓
                                    [CMS Service (Approval)]
                                           ↓ (approved)
                                    [ALTER TABLE + Update Rules]
                                           ↓
                                    [NATS: schema.config.reload]
                                           ↓
                              [PostgreSQL with JSONB Landing]
                              (_raw_data + mapped columns)
                                           ↑
[Airbyte Batch Sync] ──────────────────────┘
                                           ↓
[Event Bridge (Postgres Triggers/Listener)] → [NATS] → [Moleculer Services]
```

**Key Changes v2.0**:
- **Dynamic Mapper**: CDC Worker sử dụng mapping rules từ CMS thay vì hard-coded struct
- **Schema Inspector**: Tự động detect field mới trong JSON payload
- **CMS Service**: Approval workflow cho schema changes
- **JSONB Landing Zone**: Column `_raw_data` lưu toàn bộ JSON thô → zero data loss
- **Config Reload**: NATS event trigger reload mapping rules without restart

---

## Phase 1: Foundation & Schema Design / Giai đoạn 1: Nền tảng & Thiết kế Schema

**Objective**: Thiết lập cấu trúc cơ sở dữ liệu và xác định table mapping.
**Objective**: Establish database structure and define table mapping.

### 1.1 Table Classification / Phân loại Bảng

**[VI]** DevOps và Dev cùng phân loại các bảng trong hệ thống GooPay:
- **Real-time Tables** (Debezium path): Orders, Payments, Wallet Transactions, Transfer Requests
- **Batch Tables** (Airbyte path): User Activity Logs, Audit Logs, Reports, Product Catalog

**[EN]** DevOps and Dev collaboratively classify tables in GooPay system:
- **Real-time Tables** (Debezium path): Critical transactional data
- **Batch Tables** (Airbyte path): Analytics and historical data

**Deliverable**: `table_classification.md` document

---

### 1.2 PostgreSQL Schema Design / Thiết kế Schema PostgreSQL

**[VI]** Thiết kế schema cho từng nhóm bảng với **JSONB Landing Zone**:
- Primary Keys mapping từ source DBs
- Metadata columns: `_source`, `_synced_at`, `_version`, `_hash`
- **NEW**: `_raw_data JSONB` - Lưu toàn bộ JSON thô để zero data loss
- Upsert constraints: `ON CONFLICT (pk) DO UPDATE SET ...`
- Partitioning strategy (nếu table > 10M rows)
- Indexes cho query performance

**[EN]** Design schema for each table group with **JSONB Landing Zone**:
- Primary Keys mapped from source databases
- Metadata columns + `_raw_data` for zero data loss
- Conflict resolution via upsert constraints
- Partitioning for large tables
- Performance indexes

**Solution**:
```sql
CREATE TABLE wallet_transactions (
    id VARCHAR(36) PRIMARY KEY,
    user_id VARCHAR(36) NOT NULL,
    amount DECIMAL(15,2),
    status VARCHAR(20),
    created_at TIMESTAMP,

    -- Metadata
    _source VARCHAR(20) DEFAULT 'debezium',
    _synced_at TIMESTAMP DEFAULT NOW(),
    _version BIGINT DEFAULT 1,
    _hash VARCHAR(64),

    -- JSONB Landing Zone (NEW)
    _raw_data JSONB,  -- Lưu toàn bộ JSON thô

    CONSTRAINT check_source CHECK (_source IN ('debezium', 'airbyte'))
);

CREATE INDEX idx_wallet_tx_user ON wallet_transactions(user_id);
CREATE INDEX idx_wallet_tx_synced ON wallet_transactions(_synced_at);
CREATE INDEX idx_wallet_tx_raw_data ON wallet_transactions USING GIN(_raw_data);  -- For JSONB queries
```

**Management Tables (NEW)**:
```sql
-- Mapping rules cho dynamic mapper
CREATE TABLE cdc_mapping_rules (
    id SERIAL PRIMARY KEY,
    source_table VARCHAR(100),
    source_field VARCHAR(100),
    target_column VARCHAR(100),
    data_type VARCHAR(50),  -- int, varchar, decimal, jsonb, timestamp, etc.
    is_active BOOLEAN DEFAULT TRUE,
    is_enriched BOOLEAN DEFAULT FALSE,  -- Cần qua enrichment logic
    default_value TEXT,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(source_table, source_field)
);

-- Pending schema changes
CREATE TABLE pending_fields (
    id SERIAL PRIMARY KEY,
    table_name VARCHAR(100),
    field_name VARCHAR(100),
    sample_value TEXT,
    suggested_type VARCHAR(50),
    detected_at TIMESTAMP DEFAULT NOW(),
    status VARCHAR(20) DEFAULT 'pending',  -- pending, approved, rejected
    approved_by VARCHAR(100),
    approved_at TIMESTAMP,
    rejection_reason TEXT,
    UNIQUE(table_name, field_name)
);

-- Schema change audit log
CREATE TABLE schema_changes_log (
    id SERIAL PRIMARY KEY,
    table_name VARCHAR(100),
    change_type VARCHAR(50),  -- ADD_COLUMN, MODIFY_COLUMN, etc.
    field_name VARCHAR(100),
    old_definition TEXT,
    new_definition TEXT,
    sql_executed TEXT,
    executed_by VARCHAR(100),
    executed_at TIMESTAMP DEFAULT NOW(),
    status VARCHAR(20),  -- success, failed, rolled_back
    error_message TEXT
);
```

**Assignee**: Dev (Database Engineer) + DevOps (review)

---

### 1.3 Conflict Resolution Strategy / Chiến lược Xử lý Xung đột

**[VI]** Định nghĩa quy tắc khi cả Debezium và Airbyte cùng ghi vào 1 bảng (edge case):
- Timestamp-based: Record mới nhất thắng (`_synced_at > existing._synced_at`)
- Version-based: Increment `_version` mỗi lần update
- Audit trail: Lưu conflict events vào `cdc_conflicts` table

**[EN]** Define rules when both Debezium and Airbyte write to the same table (edge case):
- Timestamp wins: Latest record takes precedence
- Version increment for tracking changes
- Conflict audit logging

**Deliverable**: `conflict_resolution.sql` (triggers/functions)

---

## Phase 2: Go CDC Worker Implementation / Giai đoạn 2: Triển khai Go CDC Worker

**Objective**: Xây dựng service Go xử lý real-time CDC events từ NATS.
**Objective**: Build Go service to process real-time CDC events from NATS.

### 2.1 Project Structure / Cấu trúc Dự án

**[VI]** Tạo Go service mới theo cấu trúc DDD/Clean Architecture:
```
cdc-worker-service/
├── cmd/
│   └── worker/
│       └── main.go
├── internal/
│   ├── domain/          # Entities, Value Objects
│   ├── application/     # Use cases, Event Handlers
│   ├── infrastructure/  # NATS consumer, Postgres repository
│   └── interfaces/      # DTOs, Config
├── pkg/
│   ├── nats/           # NATS client wrapper
│   ├── postgres/       # Database connection
│   └── logger/         # Structured logging
└── deployments/        # Kubernetes manifests
```

**[EN]** Create new Go service following DDD/Clean Architecture.

**Assignee**: Muscle (Go Developer)

---

### 2.2 NATS Consumer Setup / Cài đặt NATS Consumer

**[VI]** Implement NATS JetStream consumer:
- Subscribe vào topics: `cdc.goopay.{table_name}`
- Consumer group để load balancing (multiple instances)
- Durable consumer với message acknowledgment
- Error handling với dead letter queue (DLQ)

**[EN]** Implement NATS JetStream consumer with:
- Topic subscription pattern
- Consumer groups for scaling
- Durable consumers for reliability
- DLQ for failed messages

**Solution**:
```go
// pkg/nats/consumer.go
func NewCDCConsumer(nc *nats.Conn, subject string, handler EventHandler) (*Consumer, error) {
    js, _ := nc.JetStream()

    sub, err := js.PullSubscribe(subject, "cdc-worker-group",
        nats.Durable("cdc-worker"),
        nats.ManualAck(),
    )

    return &Consumer{sub: sub, handler: handler}, err
}
```

**Assignee**: Muscle (Go Developer)

---

### 2.3 Event Parsing & Enrichment / Phân tích & làm giàu dữ liệu

**[VI]** Parse Debezium CDC events và enrichment:
- Decode Debezium CloudEvents hoặc Avro format
- Extract operation type (INSERT/UPDATE/DELETE)
- Map source fields → target PostgreSQL columns
- Data enrichment:
  - Thêm computed fields (ví dụ: `balance_after_transaction`)
  - Lookup related data từ cache/DB
  - Validate business rules

**[EN]** Parse Debezium CDC events and perform enrichment:
- Decode event payload
- Extract operation metadata
- Map fields to target schema
- Add computed fields and validate business logic

**Solution**:
```go
// internal/application/event_handler.go
func (h *WalletTransactionHandler) Handle(event *CDCEvent) error {
    tx := h.parseTransaction(event.Payload)

    // Enrichment
    tx.BalanceAfter = h.calculateBalance(tx.UserID, tx.Amount)
    tx.EnrichedAt = time.Now()

    // Business validation
    if err := h.validator.Validate(tx); err != nil {
        return h.sendToDLQ(event, err)
    }

    return h.repo.Upsert(tx)
}
```

**Assignee**: Muscle (Go Developer)

---

### 2.4 PostgreSQL Writer / Ghi dữ liệu vào PostgreSQL

**[VI]** Implement repository pattern cho PostgreSQL:
- Upsert operations (`INSERT ... ON CONFLICT UPDATE`)
- Batch writes cho performance (buffer 100-500 records)
- Transaction safety
- Retry logic với exponential backoff

**[EN]** Implement repository pattern with:
- Upsert operations for conflict handling
- Batch writes for performance
- Transaction safety
- Retry mechanism

**Solution**:
```go
// internal/infrastructure/postgres_repository.go
func (r *WalletTxRepository) Upsert(tx *WalletTransaction) error {
    query := `
        INSERT INTO wallet_transactions (id, user_id, amount, ...)
        VALUES ($1, $2, $3, ...)
        ON CONFLICT (id) DO UPDATE SET
            amount = EXCLUDED.amount,
            _synced_at = NOW(),
            _version = wallet_transactions._version + 1
        WHERE wallet_transactions._synced_at < EXCLUDED._synced_at
    `
    _, err := r.db.Exec(query, tx.ID, tx.UserID, tx.Amount, ...)
    return err
}
```

**Assignee**: Muscle (Go Developer)

---

## Phase 3: Event Bridge Implementation / Giai đoạn 3: Triển khai Event Bridge

**Objective**: Phát NATS events từ Postgres cho Moleculer services.
**Objective**: Publish NATS events from Postgres for Moleculer services.

### 3.1 Architecture Decision / Quyết định Kiến trúc

**[VI]** Chọn giữa 2 options:

**Option A: PostgreSQL Triggers + NOTIFY/LISTEN**
- ✅ Real-time (< 10ms latency)
- ✅ Native PostgreSQL feature
- ❌ Tight coupling với database
- ❌ Cần Go Listener service chạy 24/7

**Option B: Polling + Changelog Table**
- ✅ Loosely coupled
- ✅ Dễ scale và maintain
- ❌ Higher latency (1-5s polling interval)
- ✅ Có thể batch events

**Recommendation**: **Option A** cho critical tables (Payments, Wallet), **Option B** cho non-critical tables (Logs, Reports).

**[EN]** Choose between Trigger-based (real-time) or Polling-based (scalable) approach.

**Deliverable**: ADR (Architecture Decision Record) in `04_decisions.md`

---

### 3.2 Trigger-based Bridge (Option A) / Cầu nối dựa trên Trigger

**[VI]** Implement PostgreSQL Triggers:

```sql
CREATE OR REPLACE FUNCTION notify_nats_event()
RETURNS TRIGGER AS $$
BEGIN
    PERFORM pg_notify(
        'cdc_events',
        json_build_object(
            'table', TG_TABLE_NAME,
            'operation', TG_OP,
            'data', row_to_json(NEW)
        )::text
    );
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER wallet_tx_notify
AFTER INSERT OR UPDATE ON wallet_transactions
FOR EACH ROW
WHEN (NEW._source = 'airbyte')  -- Only for Airbyte-synced data
EXECUTE FUNCTION notify_nats_event();
```

**Go Listener Service**:
```go
// cmd/event-bridge/main.go
func listenAndPublish(pgConn *pgx.Conn, nc *nats.Conn) {
    _, err := pgConn.Exec(context.Background(), "LISTEN cdc_events")

    for {
        notification, _ := pgConn.WaitForNotification(context.Background())

        event := parseNotification(notification.Payload)
        natsSubject := fmt.Sprintf("goopay.%s.%s", event.Table, event.Operation)

        nc.Publish(natsSubject, event.Data)
    }
}
```

**Assignee**: Muscle (Go Developer)

---

### 3.3 Polling-based Bridge (Option B) / Cầu nối dựa trên Polling

**[VI]** Implement changelog table và poller:

```sql
CREATE TABLE cdc_changelog (
    id BIGSERIAL PRIMARY KEY,
    table_name VARCHAR(100),
    operation VARCHAR(10),
    record_id VARCHAR(100),
    data JSONB,
    created_at TIMESTAMP DEFAULT NOW(),
    published BOOLEAN DEFAULT FALSE
);

-- Trigger ghi vào changelog
CREATE TRIGGER log_changes_wallet_tx
AFTER INSERT OR UPDATE ON wallet_transactions
FOR EACH ROW
WHEN (NEW._source = 'airbyte')
EXECUTE FUNCTION log_to_changelog();
```

**Go Poller**:
```go
func (p *ChangelogPoller) Poll() {
    ticker := time.NewTicker(2 * time.Second)

    for range ticker.C {
        changes := p.repo.FetchUnpublished(limit: 500)

        for _, change := range changes {
            p.publishToNATS(change)
            p.repo.MarkPublished(change.ID)
        }
    }
}
```

**Assignee**: Muscle (Go Developer)

---

## Phase 4: Data Reconciliation / Giai đoạn 4: Đối soát Dữ liệu

**Objective**: Kiểm tra tính nhất quán giữa Source và Target DBs.
**Objective**: Verify data consistency between Source and Target databases.

### 4.1 Reconciliation Script / Script Đối soát

**[VI]** Viết Go CLI tool để:
- Compare record counts: `SELECT COUNT(*) FROM source_table` vs target
- Checksum validation: Hash toàn bộ row và so sánh
- Detect missing records: Join source và target, tìm gaps
- Detect stale data: Compare timestamps (`source.updated_at` vs `target._synced_at`)

**[EN]** Build Go CLI tool for data reconciliation with multiple validation strategies.

**Solution**:
```go
// cmd/reconcile/main.go
func ReconcileTable(tableName string) (*Report, error) {
    sourceCount := getSourceCount(tableName)
    targetCount := getTargetCount(tableName)

    report := &Report{
        Table: tableName,
        SourceCount: sourceCount,
        TargetCount: targetCount,
        CountMatch: sourceCount == targetCount,
    }

    if !report.CountMatch {
        report.MissingRecords = findMissingRecords(tableName)
    }

    checksumMatch := compareChecksums(tableName)
    report.ChecksumMatch = checksumMatch

    return report, nil
}
```

**Assignee**: Muscle (Go Developer)

---

### 4.2 Automated Scheduling / Lập lịch Tự động

**[VI]** Deploy reconciliation job như Kubernetes CronJob:
- Critical tables: Chạy mỗi 15 phút
- Non-critical tables: Chạy mỗi 4 giờ
- Output: JSON report lưu vào S3/MinIO
- Alerts: Gửi Slack/Email khi phát hiện drift > 1%

**[EN]** Deploy as Kubernetes CronJob with different frequencies for different table tiers.

**Deliverable**: `deployments/reconciliation-cronjob.yaml`

---

### 4.3 Auto-Repair Mechanism (Optional) / Cơ chế Tự sửa chữa

**[VI]** Nếu phát hiện missing records:
- Trigger manual sync từ source DB
- Hoặc publish backfill event vào NATS để CDC Worker xử lý lại

**[EN]** Optional: Implement auto-repair by triggering backfill syncs.

**Assignee**: Muscle (Go Developer)

---

## Phase 5: Schema Drift Detection & CMS Integration / Giai đoạn 5: Phát hiện Schema Drift & CMS

**Objective**: Tự động hóa việc phát hiện và quản lý schema changes.
**Objective**: Automate schema change detection and management.

### 5.1 Schema Inspector Module

**[VI]** Implement module Go để detect schema drift:

```go
// internal/application/schema_inspector.go
type SchemaInspector struct {
    pgRepo      *postgres.Repository
    redisCache  *redis.Client
    natsClient  *nats.Conn
}

func (si *SchemaInspector) InspectEvent(event *CDCEvent) (*SchemaDrift, error) {
    // 1. Extract all field names from JSON
    eventFields := extractFieldNames(event.Payload.After)

    // 2. Get current table schema from cache or DB
    tableSchema := si.getTableSchema(event.Table)

    // 3. Compare
    newFields := findNewFields(eventFields, tableSchema)

    if len(newFields) > 0 {
        // 4. Save to pending_fields
        for _, field := range newFields {
            suggestedType := inferDataType(event.Payload.After[field])
            si.savePendingField(event.Table, field, suggestedType, event.Payload.After[field])
        }

        // 5. Publish alert
        si.publishDriftAlert(event.Table, newFields)

        return &SchemaDrift{Detected: true, NewFields: newFields}, nil
    }

    return &SchemaDrift{Detected: false}, nil
}

func inferDataType(value interface{}) string {
    switch v := value.(type) {
    case float64:
        if v == float64(int64(v)) {
            return "INTEGER"
        }
        return "DECIMAL"
    case string:
        if _, err := time.Parse(time.RFC3339, v); err == nil {
            return "TIMESTAMP"
        }
        return "VARCHAR(255)"
    case bool:
        return "BOOLEAN"
    case map[string]interface{}:
        return "JSONB"
    default:
        return "TEXT"
    }
}
```

**[EN]** Implement Go module for schema drift detection with automatic type inference and alert mechanism.

**Assignee**: Muscle (Go Developer)

---

### 5.2 CMS Backend Service

**[VI]** Xây dựng CMS backend với REST API:

**API Endpoints**:
```go
// cmd/cms-service/main.go
router.GET("/api/schema-changes/pending", listPendingChanges)
router.POST("/api/schema-changes/:id/approve", approveSchemaChange)
router.POST("/api/schema-changes/:id/reject", rejectSchemaChange)
router.GET("/api/mapping-rules", listMappingRules)
router.POST("/api/mapping-rules", createMappingRule)
router.PUT("/api/mapping-rules/:id", updateMappingRule)
```

**Approve Schema Change Logic**:
```go
func approveSchemaChange(c *gin.Context) {
    id := c.Param("id")

    // 1. Get pending field
    pendingField, _ := repo.GetPendingField(id)

    // 2. Execute ALTER TABLE
    sql := fmt.Sprintf(
        "ALTER TABLE %s ADD COLUMN %s %s",
        pendingField.TableName,
        pendingField.FieldName,
        pendingField.SuggestedType,
    )
    _, err := db.Exec(sql)
    if err != nil {
        // Rollback and return error
        return c.JSON(500, gin.H{"error": err.Error()})
    }

    // 3. Insert mapping rule
    rule := &MappingRule{
        SourceTable:  pendingField.TableName,
        SourceField:  pendingField.FieldName,
        TargetColumn: pendingField.FieldName,  // or custom name
        DataType:     pendingField.SuggestedType,
        IsActive:     true,
    }
    repo.CreateMappingRule(rule)

    // 4. Log schema change
    repo.LogSchemaChange(pendingField, sql, c.GetString("user_id"))

    // 5. Publish config reload event
    natsClient.Publish("schema.config.reload", []byte(pendingField.TableName))

    // 6. Mark as approved
    repo.UpdatePendingFieldStatus(id, "approved", c.GetString("user_id"))

    // 7. Trigger Airbyte refresh (if needed)
    if pendingField.Source == "airbyte" {
        airbyteClient.RefreshSchema(pendingField.TableName)
    }

    return c.JSON(200, gin.H{"status": "approved"})
}
```

**[EN]** Build CMS backend service with approval workflow and integration with Airbyte API.

**Assignee**: Muscle (Go/Node.js Developer)

---

### 5.3 CMS Frontend UI

**[VI]** Xây dựng React UI cho CMS:

**Components**:
- `PendingChangesTable`: Hiển thị danh sách pending fields
- `ApprovalModal`: Modal để review và approve/reject
- `MappingRulesManager`: CRUD interface cho mapping rules
- `SchemaChangeHistory`: Audit trail view

**Technology Stack**:
- React + TypeScript
- Ant Design / Material-UI for components
- React Query for API calls
- WebSocket for real-time notifications

**Assignee**: Frontend Developer (hoặc Muscle nếu full-stack)

---

## Phase 6: Dynamic Mapping Engine / Giai đoạn 6: Dynamic Mapping Engine

**Objective**: Chuyển CDC Worker sang generic processor, config-driven mapping.
**Objective**: Convert CDC Worker to generic processor with config-driven mapping.

### 6.1 Dynamic Query Builder

**[VI]** Implement query builder động dựa trên mapping rules:

```go
// internal/application/dynamic_mapper.go
type DynamicMapper struct {
    rules map[string][]MappingRule  // table_name -> rules
    cache *redis.Client
}

func (dm *DynamicMapper) LoadRules() error {
    rules, err := dm.repo.GetAllMappingRules()
    if err != nil {
        return err
    }

    // Group by table
    dm.rules = make(map[string][]MappingRule)
    for _, rule := range rules {
        if rule.IsActive {
            dm.rules[rule.SourceTable] = append(dm.rules[rule.SourceTable], rule)
        }
    }

    return nil
}

func (dm *DynamicMapper) BuildUpsertQuery(tableName string, data map[string]interface{}) (string, []interface{}, error) {
    rules := dm.rules[tableName]
    if len(rules) == 0 {
        return "", nil, fmt.Errorf("no mapping rules for table %s", tableName)
    }

    var columns []string
    var placeholders []string
    var values []interface{}
    var updateSets []string

    idx := 1
    for _, rule := range rules {
        // Extract value from JSON
        value, exists := data[rule.SourceField]
        if !exists {
            value = rule.DefaultValue
        }

        // Convert type
        convertedValue, err := convertType(value, rule.DataType)
        if err != nil {
            continue  // Skip invalid data
        }

        columns = append(columns, rule.TargetColumn)
        placeholders = append(placeholders, fmt.Sprintf("$%d", idx))
        values = append(values, convertedValue)
        updateSets = append(updateSets, fmt.Sprintf("%s = EXCLUDED.%s", rule.TargetColumn, rule.TargetColumn))
        idx++
    }

    // Add _raw_data JSONB
    rawData, _ := json.Marshal(data)
    columns = append(columns, "_raw_data", "_synced_at", "_version")
    placeholders = append(placeholders, fmt.Sprintf("$%d", idx), fmt.Sprintf("$%d", idx+1), "1")
    values = append(values, rawData, time.Now())

    query := fmt.Sprintf(`
        INSERT INTO %s (%s)
        VALUES (%s)
        ON CONFLICT (id) DO UPDATE SET
            %s,
            _raw_data = EXCLUDED._raw_data,
            _synced_at = NOW(),
            _version = %s._version + 1
    `, tableName, strings.Join(columns, ", "), strings.Join(placeholders, ", "),
       strings.Join(updateSets, ", "), tableName)

    return query, values, nil
}
```

**[EN]** Implement dynamic query builder that constructs SQL based on mapping rules loaded from database.

**Assignee**: Muscle (Go Developer)

---

### 6.2 Config Reload Mechanism

**[VI]** Implement hot reload khi có schema changes:

```go
// Subscribe to NATS reload events
func (dm *DynamicMapper) StartConfigReloadListener(ctx context.Context) {
    nc, _ := nats.Connect(natsURL)
    nc.Subscribe("schema.config.reload", func(msg *nats.Msg) {
        tableName := string(msg.Data)
        logger.Info("Config reload triggered", zap.String("table", tableName))

        // Reload rules from DB
        if err := dm.LoadRules(); err != nil {
            logger.Error("Failed to reload rules", zap.Error(err))
            return
        }

        // Invalidate cache
        dm.cache.Del(ctx, "mapping_rules:"+tableName)

        logger.Info("Config reloaded successfully", zap.String("table", tableName))
    })
}
```

**[EN]** Implement hot reload mechanism via NATS events, zero downtime.

**Assignee**: Muscle (Go Developer)

---

## Phase 7: Migration Automation & CI/CD / Giai đoạn 7: Tự động hóa Migration

**Objective**: Automate ALTER TABLE and Airbyte refresh workflow.
**Objective**: Tự động hóa ALTER TABLE và Airbyte refresh workflow.

### 7.1 Airbyte API Integration

**[VI]** Tích hợp Airbyte API:

```go
// pkg/airbyte/client.go
type AirbyteClient struct {
    baseURL string
    apiKey  string
}

func (ac *AirbyteClient) RefreshSourceSchema(sourceID string) error {
    url := fmt.Sprintf("%s/v1/sources/%s/discover_schema", ac.baseURL, sourceID)
    resp, err := ac.httpClient.Post(url, "application/json", nil)
    if err != nil {
        return err
    }
    return nil
}

func (ac *AirbyteClient) UpdateConnection(connectionID string, enabledFields []string) error {
    // Update connection config to enable new fields
    return nil
}

func (ac *AirbyteClient) TriggerSync(connectionID string) error {
    url := fmt.Sprintf("%s/v1/connections/%s/sync", ac.baseURL, connectionID)
    resp, err := ac.httpClient.Post(url, "application/json", nil)
    return err
}
```

**[EN]** Integrate with Airbyte API for automated schema refresh and sync triggering.

**Assignee**: Muscle (Go Developer) + DevOps

---

## Phase 8: Testing & Validation / Giai đoạn 8: Kiểm thử & Xác nhận

### 5.1 Unit Tests / Kiểm thử Đơn vị

**[VI]** Viết unit tests cho:
- Event parsing logic
- Data enrichment functions
- Upsert repository methods
- Conflict resolution logic

**[EN]** Write unit tests covering core business logic.

**Coverage Target**: > 80%

---

### 5.2 Integration Tests / Kiểm thử Tích hợp

**[VI]** Setup test environment:
- Mock NATS broker (embedded NATS server)
- Dockerized PostgreSQL
- Publish test CDC events → verify writes
- Trigger test data in Postgres → verify NATS events published

**[EN]** Integration tests with real dependencies (containerized).

**Assignee**: Muscle + QA Agent

---

### 5.3 Load Testing / Kiểm thử Tải

**[VI]** Simulate production load:
- 10,000 CDC events/second vào NATS
- Đo latency: NATS event → Postgres write (target: < 100ms p99)
- Đo throughput: CDC Worker instances scaling (target: handle 50K events/sec với 5 pods)

**[EN]** Load testing with production-like traffic patterns.

**Tools**: k6, Grafana, Prometheus

---

### 5.4 End-to-End Testing / Kiểm thử Đầu-cuối

**[VI]** Test workflow hoàn chỉnh:
1. Insert record vào MongoDB (source)
2. Debezium capture change → publish NATS
3. CDC Worker nhận event → write Postgres
4. Verify record trong Postgres match với source
5. Airbyte sync batch data → Postgres
6. Event Bridge phát NATS event
7. Moleculer service nhận event → process

**[EN]** Full end-to-end workflow validation.

**Assignee**: QA Agent + Muscle

---

## Phase 6: Deployment & Monitoring / Giai đoạn 6: Triển khai & Giám sát

### 6.1 Kubernetes Deployment / Triển khai Kubernetes

**[VI]** Tạo manifests:
- `cdc-worker-deployment.yaml`: Deployment với 3-5 replicas
- `event-bridge-deployment.yaml`: StatefulSet (nếu dùng LISTEN) hoặc Deployment (polling)
- `reconciliation-cronjob.yaml`: CronJob chạy định kỳ
- ConfigMaps: Table mappings, topic configurations
- Secrets: Database credentials, NATS tokens

**[EN]** Create Kubernetes deployment manifests with proper scaling and secrets management.

**Assignee**: DevOps (với review từ Muscle)

---

### 6.2 Monitoring & Alerting / Giám sát & Cảnh báo

**[VI]** Setup metrics và alerts:

**Metrics**:
- `cdc_events_processed_total` (counter)
- `cdc_processing_latency_seconds` (histogram)
- `postgres_upsert_errors_total` (counter)
- `reconciliation_drift_percentage` (gauge)

**Alerts**:
- CDC Worker lag > 1000 messages
- Processing latency p99 > 200ms
- Reconciliation drift > 5%
- Event Bridge publish failures > 10/min

**[EN]** Comprehensive monitoring with Prometheus metrics and Grafana dashboards.

**Assignee**: DevOps + Muscle

---

## 📊 Success Metrics / Chỉ số Thành công

| Metric | Target | Measurement |
|--------|--------|-------------|
| CDC Event Latency (p99) | < 100ms | NATS timestamp → Postgres write timestamp |
| Throughput | 50K events/sec | With 5 CDC Worker pods |
| Data Accuracy | 99.99% | Reconciliation drift < 0.01% |
| Uptime | 99.9% | CDC Worker + Event Bridge |
| Recovery Time | < 5 min | From pod restart to full processing |

---

## 🚧 Risks & Mitigations / Rủi ro & Biện pháp

| Risk | Impact | Mitigation |
|------|--------|-----------|
| NATS message loss | High | JetStream persistence, message acknowledgment |
| Postgres write conflicts | Medium | Timestamp-based conflict resolution, version tracking |
| CDC Worker pod crash | Medium | Kubernetes auto-restart, durable NATS consumers |
| Schema drift (source vs target) | High | Schema registry, validation pipeline |
| Data reconciliation performance | Low | Partition-based reconciliation, incremental checks |

---

## 📅 Timeline Estimate / Ước lượng Thời gian (Updated v2.0)

| Phase | Duration | Dependencies | Notes |
|-------|----------|--------------|-------|
| Phase 1: Schema Design + JSONB | 4-6 days | DevOps table classification | +1 day for JSONB setup & management tables |
| Phase 2: CDC Worker (Dynamic) | 8-12 days | NATS setup (DevOps) | +2 days for dynamic mapping logic |
| Phase 3: Event Bridge | 5-7 days | Phase 2 complete | No change |
| Phase 4: Reconciliation | 4-6 days | PostgreSQL access | No change |
| **Phase 5: Schema Drift + CMS** | **10-14 days** | Phase 1-2 complete | **NEW**: Inspector + Backend + Frontend |
| **Phase 6: Dynamic Mapping Engine** | **5-7 days** | Phase 5 complete | **NEW**: Query builder + Config reload |
| **Phase 7: Migration Automation** | **3-5 days** | Phase 5-6 complete | **NEW**: Airbyte API + CI/CD |
| Phase 8: Testing | 7-10 days | All phases complete | +2 days for CMS & dynamic mapping tests |
| Phase 9: Deployment | 4-6 days | K8s cluster ready | +1 day for CMS deployment |
| **Total** | **7-10 weeks** | Assuming no major blockers | **+3-4 weeks** compared to v1.0 |

**Critical Path**:
1. Phase 1 → Phase 2 → Phase 5 → Phase 6 → Phase 8
2. CMS Frontend (Phase 5.3) có thể song song với Phase 6-7

**Resource Allocation**:
- **Backend Go Developer**: Full-time cho Phase 2, 5, 6, 7
- **Frontend Developer**: Phase 5.3 (CMS UI) - có thể outsource hoặc part-time
- **DevOps**: Phase 1 (table classification), Phase 7 (CI/CD), Phase 9 (deployment)

---

## ✅ Definition of Done / Định nghĩa Hoàn thành (Updated v2.0)

### Core CDC Functionality
- [ ] PostgreSQL schema với JSONB Landing Zone (`_raw_data` column) cho zero data loss
- [ ] CDC Worker processes NATS events với latency < 100ms (p99)
- [ ] Event Bridge publishes events cho Moleculer services trong đúng format
- [ ] Reconciliation job phát hiện data drift và alert

### Dynamic Mapping & Schema Evolution (NEW)
- [ ] **Schema Drift Detection**: Tự động detect field mới trong < 1 phút
- [ ] **JSONB Fallback**: Mọi data được lưu vào `_raw_data` khi schema chưa ready
- [ ] **CMS Approval Workflow**: End-to-end từ detect → review → approve → ALTER TABLE
- [ ] **Dynamic Mapping**: CDC Worker sử dụng mapping rules từ DB, không hard-code
- [ ] **Config Reload**: Reload mapping rules qua NATS event trong < 5 giây, zero restart
- [ ] **Airbyte Integration**: CMS tự động trigger Airbyte refresh sau schema change
- [ ] **Audit Trail**: Mọi schema change có log trong `schema_changes_log`

### Testing & Quality
- [ ] Unit test coverage > 80% (bao gồm schema inspector, dynamic mapper, CMS logic)
- [ ] Integration tests pass (bao gồm CMS approval workflow end-to-end)
- [ ] Load testing confirms throughput target (50K events/sec)
- [ ] Schema drift detection tested với simulated new fields
- [ ] Dynamic mapping tested với ≥3 different tables
- [ ] Config reload tested không ảnh hưởng ongoing processing

### Deployment & Operations
- [ ] Kubernetes deployments với health checks và resource limits
- [ ] CMS service deployed với authentication & authorization
- [ ] Monitoring dashboards và alerts setup (bao gồm schema drift metrics)
- [ ] Documentation hoàn chỉnh (Architecture, Runbooks, Troubleshooting, CMS User Guide)
- [ ] Security scan pass (`/security-agent`) cho tất cả services
- [ ] Runbook cho schema change approval process
- [ ] Disaster recovery plan cho rollback schema changes

### Project Management
- [ ] Memory updated (`05_progress.md`, `active_plans.md`, `04_decisions.md`)
- [ ] ADRs documented cho JSONB strategy, dynamic mapping approach, CMS architecture

### Lỗi Schema Inspector với Airbyte (Lịch sử Sửa lỗi Phase 1.6)

**[VI]** 
Vấn đề: Airbyte push data vòng qua hệ thống sự kiện NATS do sử dụng batch load trực tiếp vào Postgres, khiến hệ thống Real-Time Schema Drift bằng SchemaInspector hoàn toàn không hoạt động. Ngoài ra API scan Introspect dùng LIMIT 100 cũ quét ngẫu nhiên nên không bao giờ thấy field mới từ Top-latest synced data của Mông Cổ DB.
Giải pháp: Sử dụng nút Introspect/Discover từ CMS UI làm công cụ chuẩn (thay vì chờ NATS), đi kèm với việc bổ sung `ORDER BY _synced_at DESC LIMIT 100` vào query của NATS handler `cdc.cmd.introspect` để đảm bảo bắt trúng tài liệu vừa đồng bộ.

**[EN]**
Problem: Airbyte pushes data directly to DW bypassing NATS Real-time stream, rendering SchemaInspector blind to Airbyte schema drift. Secondly, the Introspect API command uses LIMIT 100 without ordering, hiding newly added columns from Mongo syncs. 
Solution: Emphasize the Discover feature on CMS UI. Modify `cdc.cmd.introspect` handler query to sort by `_synced_at DESC` to correctly retrieve latest synced objects.

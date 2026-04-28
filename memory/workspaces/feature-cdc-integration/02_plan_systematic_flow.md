# Execution Plan — Systematic Connect→Master Flow

> Stage 4 · Phase: `systematic_flow` · 2026-04-24

## 1. Roadmap (5 tracks)

```
Track 0: Migrations        ━━━┓
Track 1: BE Sources        ━━━━┓
Track 2: BE ShadowAutomator━━━━━━┓
Track 3: BE Wizard         ━━━━━━━━┓
Track 4: FE Wizard+Dropdown━━━━━━━━━━━━
```

## 2. Track 0 — Migrations (prereq)

File mới trong `centralized-data-service/migrations/`:

- **027_systematic_sources.sql** — tạo:
  - `cdc_internal.sources` (table)
  - `cdc_internal.cdc_wizard_sessions` (table)
  - Indexes + constraints
- **028_sonyflake_fallback_fn.sql** — tạo:
  - `cdc_internal.gen_sonyflake_id()` SQL function (fallback trigger body)
  - `cdc_internal.ensure_shadow_sonyflake_trigger(text)` helper — idempotent attach trigger cho 1 bảng shadow.

**Rollback path**: nếu deploy sai, `DROP FUNCTION` + `DROP TABLE` trong 1 transaction ngược lại. Không destructive cho bảng `cdc_table_registry` (chỉ thêm mới).

## 3. Track 1 — Backend Sources (Task 1 Boss)

Files:
- `internal/model/source.go` — NEW (GORM model Source)
- `internal/repository/source_repo.go` — NEW (Upsert, GetByConnectorName, List, MarkDeleted)
- `internal/api/sources_handler.go` — NEW (`List`, `Get`)
- `internal/api/system_connectors_handler.go` — EDIT (`Create` tail insert source row; `Delete` tail mark deleted)
- `internal/server/server.go` — EDIT (wire `SourcesHandler`, pass `SourceRepo` vào `SystemConnectorsHandler` + `WizardHandler`)
- `internal/router/router.go` — EDIT (thêm GET /api/v1/sources, GET /api/v1/sources/:id)

### 3.1. Ký constructor mới

```go
// BEFORE
func NewSystemConnectorsHandler(kafkaConnectURL string, logger *zap.Logger)

// AFTER
func NewSystemConnectorsHandler(kafkaConnectURL string, sourceRepo *repository.SourceRepo, logger *zap.Logger)
```

### 3.2. Create flow sau khi forward Kafka Connect thành công

```go
// Forward to kafka-connect
if err := h.doJSON(...); err != nil { return 502 }

// Parse common Debezium fields
fp := parseFingerprint(req.Name, req.Config) // helper
if err := h.sourceRepo.Upsert(c.Context(), &model.Source{
    ConnectorName: req.Name,
    SourceType: fp.SourceType,
    TopicPrefix: fp.TopicPrefix,
    ServerAddress: fp.ServerAddress,
    DatabaseIncludeList: fp.DatabaseIncludeList,
    CollectionIncludeList: fp.CollectionIncludeList,
    ConnectorClass: req.Config["connector.class"],
    RawConfigSanitized: filterSafeConfig(req.Config),
    Status: "created",
    CreatedBy: middleware.GetUsername(c),
}); err != nil {
    // Don't fail the request — log; Kafka Connect is already the authority.
    h.logger.Warn("source fingerprint persist failed", zap.Error(err))
}
```

## 4. Track 2 — Backend ShadowAutomator (Task 2 Boss)

Files:
- `internal/service/shadow_automator.go` — NEW (`EnsureShadowTable`, `ensureSonyflakeFunction`, `attachSonyflakeTrigger`)
- `internal/api/registry_handler.go` — EDIT (`Register` gọi automator **trước** khi return; fallback publish NATS cho legacy path nếu flag bật)
- `internal/server/server.go` — EDIT (wire ShadowAutomator vào RegistryHandler)

### 4.1. Service signature

```go
type ShadowAutomator struct {
    db     *gorm.DB
    logger *zap.Logger
}

// EnsureShadowTable creates cdc_internal.<target>, attaches Sonyflake fallback trigger,
// idempotent. Called synchronously from Register.
func (s *ShadowAutomator) EnsureShadowTable(ctx context.Context, reg *model.TableRegistry) error {
    if err := s.ensureSonyflakeFunction(ctx); err != nil { return err }
    if err := s.createShadowDDL(ctx, reg); err != nil { return err }
    if err := s.attachSonyflakeTrigger(ctx, reg.TargetTable); err != nil { return err }
    return s.markCreated(ctx, reg)
}
```

### 4.2. Schema deployed (8 cols — giữ nguyên 003_sonyflake_schema template)

```sql
CREATE TABLE IF NOT EXISTS cdc_internal.<target> (
    id BIGINT PRIMARY KEY,
    source_id VARCHAR(200) NOT NULL,
    _raw_data JSONB NOT NULL,
    _source VARCHAR(20) NOT NULL DEFAULT 'debezium',
    _synced_at TIMESTAMP NOT NULL DEFAULT NOW(),
    _version BIGINT NOT NULL DEFAULT 1,
    _hash VARCHAR(64),
    _deleted BOOLEAN DEFAULT FALSE,
    _created_at TIMESTAMP DEFAULT NOW(),
    _updated_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(source_id)
);
-- indexes: _synced_at, _source, GIN(_raw_data)
-- trigger: trg_<target>_sonyflake_fallback BEFORE INSERT
```

### 4.3. Sonyflake fallback function

```sql
-- migration 028
CREATE OR REPLACE FUNCTION cdc_internal.gen_sonyflake_id()
RETURNS BIGINT AS $$
DECLARE
  v_ts_ms BIGINT;
  v_machine INT;
  v_seq BIGINT;
BEGIN
  -- ts: ms since 2026-01-01
  v_ts_ms := (EXTRACT(EPOCH FROM NOW()) * 1000)::BIGINT - 1767225600000;
  -- machine_id: reuse cdc_internal.machine_id_seq (migration 018)
  v_machine := COALESCE(current_setting('cdc.machine_id', true)::INT, 0) & 65535;
  -- seq: fallback sequence
  v_seq := nextval('cdc_internal.fencing_token_seq') & 65535;
  RETURN (v_ts_ms << 22) | (v_machine::BIGINT << 16) | v_seq;
END;
$$ LANGUAGE plpgsql;

-- trigger body (attached per-table by ensure_shadow_sonyflake_trigger)
CREATE OR REPLACE FUNCTION cdc_internal.tg_sonyflake_fallback()
RETURNS TRIGGER AS $$
BEGIN
  IF NEW.id IS NULL OR NEW.id = 0 THEN
    NEW.id := cdc_internal.gen_sonyflake_id();
  END IF;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;
```

### 4.4. Register handler change

```go
// registry_handler.go:Register
if err := h.repo.Create(ctx, &entry); err != nil { return 500 }
if err := h.automator.EnsureShadowTable(ctx, &entry); err != nil {
    // rollback registry row to avoid orphan
    h.repo.Delete(ctx, entry.ID)
    return 500 "shadow DDL failed: " + err.Error()
}
// keep legacy NATS publish as idempotent no-op (worker skip if is_table_created=true)
h.natsClient.Conn.Publish("cdc.cmd.create-default-columns", createColsPayload)
```

## 5. Track 3 — Backend Wizard (Task 3 Boss)

Files:
- `internal/model/wizard_session.go` — NEW
- `internal/repository/wizard_repo.go` — NEW
- `internal/service/master_swap.go` — NEW (AtomicSwap)
- `internal/api/wizard_handler.go` — NEW (Create, Get, Patch, Execute, Progress)
- `internal/api/master_registry_handler.go` — EDIT thêm `Swap` handler
- `internal/router/router.go` — EDIT (wire wizard + master swap routes)
- `internal/server/server.go` — EDIT (wire handlers)

### 5.1. Wizard session model

```go
type WizardSession struct {
    ID           string    `gorm:"column:id;primaryKey" json:"id"`           // UUID
    SourceName   string    `gorm:"column:source_name" json:"source_name"`
    ConnectorID  *string   `gorm:"column:connector_id" json:"connector_id"`
    RegistryID   *uint     `gorm:"column:registry_id" json:"registry_id"`
    MasterName   *string   `gorm:"column:master_name" json:"master_name"`
    CurrentStep  int       `gorm:"column:current_step" json:"current_step"`
    Status       string    `gorm:"column:status" json:"status"`              // draft|running|done|failed
    StepPayload  JSONB     `gorm:"column:step_payload;type:jsonb" json:"step_payload"`
    ProgressLog  JSONB     `gorm:"column:progress_log;type:jsonb" json:"progress_log"`
    CreatedBy    string    `gorm:"column:created_by" json:"created_by"`
    CreatedAt    time.Time `gorm:"column:created_at" json:"created_at"`
    UpdatedAt    time.Time `gorm:"column:updated_at" json:"updated_at"`
}
func (WizardSession) TableName() string { return "cdc_internal.cdc_wizard_sessions" }
```

### 5.2. Execute "Automate Everything" pipeline

Khi `POST /api/v1/wizard/sessions/:id/execute` với payload `{connector_config, source_table, target_table, primary_key_field, master_name, master_spec}`:

1. CreateConnector → upsert source (Track 1) → update session.connector_id + progress_log append.
2. InsertRegistry + EnsureShadowTable (Track 2) → update session.registry_id.
3. Publish `cdc.cmd.debezium-signal` (TriggerSnapshot via existing handler) → progress log.
4. Poll `SELECT COUNT(*) FROM cdc_internal.<target>` mỗi 3s max 60s — wait first row.
5. If config.auto_approve_proposals=true → auto-approve pending proposals (gọi `SchemaProposalHandler.Approve` internal).
6. Create master via MasterRegistryHandler internal → auto-approve.
7. Session.status = done.

Mỗi step append vào `progress_log` JSONB. Lỗi step N → session.status=failed + return 207 Multi-Status với step fail info.

### 5.3. Atomic Swap service

```go
// service/master_swap.go
func (s *MasterSwap) Swap(ctx context.Context, masterName, newTableName, reason string) error {
    return s.db.Transaction(func(tx *gorm.DB) error {
        // 3s lock timeout — master swap must not block OLTP
        if err := tx.Exec("SET LOCAL lock_timeout = '3s'").Error; err != nil { return err }

        ts := time.Now().Unix()
        oldName := fmt.Sprintf("%s_old_%d", masterName, ts)

        // RENAME old → _old_<ts>
        if err := tx.Exec(fmt.Sprintf("ALTER TABLE public.%q RENAME TO %q", masterName, oldName)).Error; err != nil {
            return fmt.Errorf("rename old: %w", err)
        }
        // RENAME new → master
        if err := tx.Exec(fmt.Sprintf("ALTER TABLE public.%q RENAME TO %q", newTableName, masterName)).Error; err != nil {
            return fmt.Errorf("rename new: %w", err)
        }
        // Audit row
        return tx.Exec(`INSERT INTO cdc_activity_log (operation, target_table, status, details, triggered_by)
            VALUES ('master_swap', ?, 'success', ?, 'manual')`,
            masterName, fmt.Sprintf(`{"old_table":"%s","new_table":"%s","reason":"%s"}`, oldName, newTableName, reason)).Error
    })
}
```

## 6. Track 4 — Frontend

### 6.1. Wizard Rewrite (Option A)

File: `cdc-cms-web/src/pages/SourceToMasterWizard.tsx` — REWRITE.

Key pieces:
- `useQuery('wizard-session', sessionId)` load state from F3.2.
- `useMutation` for each step action.
- `URL search param ?session_id=X` persists via `useSearchParams`.
- On mount: if no session_id → CreateSession, update URL.
- "🚀 Automate Everything" button → call F3.4, switch to live progress polling (setInterval 2s).
- `<Steps current={session.current_step} />` driven by server state, not local `useState`.

### 6.2. TableRegistry dropdown

File: `cdc-cms-web/src/pages/TableRegistry.tsx` — EDIT modal:

```tsx
const { data: sources } = useQuery(['sources'], () => cmsApi.get('/api/v1/sources').then(r => r.data.data));
const [selectedSourceId, setSelectedSourceId] = useState<string>();
const selectedSource = sources?.find(s => s.id === selectedSourceId);
const availableCollections = selectedSource?.collection_include_list?.split(',').map(s => s.trim()) ?? [];

// In modal:
<Form.Item name="source_id" label="Source" rules={[{ required: true }]}>
  <Select onChange={(id) => {
    setSelectedSourceId(id);
    const src = sources.find(s => s.id === id);
    form.setFieldsValue({
      source_db: src.database_include_list,
      source_type: src.source_type,
    });
  }}>
    {sources?.map(s => <Option key={s.id} value={s.id}>{s.connector_name}</Option>)}
  </Select>
</Form.Item>
<Form.Item name="source_table" label="Collection" rules={[{ required: true }]}>
  <Select disabled={!selectedSource}>
    {availableCollections.map(c => <Option key={c} value={c}>{c}</Option>)}
  </Select>
</Form.Item>
<Form.Item name="source_db" label="Source DB (auto)"><Input disabled /></Form.Item>
<Form.Item name="source_type" label="Source Type (auto)"><Input disabled /></Form.Item>
```

## 7. Verification Plan (for Stage 6)

| Check | Command | Expected |
|:-|:-|:-|
| BE build | `cd cdc-cms-service && go build ./...` | exit 0 |
| FE typecheck | `cd cdc-cms-web && npx tsc --noEmit` | exit 0 |
| Migration dryrun | `psql -f 027_*.sql -f 028_*.sql -v ON_ERROR_STOP=1` (on test DB) | exit 0 |
| Fallback trigger | `INSERT INTO cdc_internal.test_t (source_id,_raw_data) VALUES ('s1','{}'); SELECT id FROM cdc_internal.test_t` | id > 0 (BIGINT) |
| Atomic swap | `BEGIN; ... COMMIT;` test with deliberate rollback | original table intact |
| Idempotency | Call POST /api/v1/system/connectors twice with same name | row count in sources = 1 |
| Wizard resume | Create session → F5 browser → session_id in URL still loads state | Current_step preserved |

## 8. Rollout

1. Merge migrations 027, 028 first. Verify on staging.
2. Deploy cdc-cms-service với feature flag `ENABLE_SHADOW_AUTOMATOR` (default `true`, fallback `false` → NATS async legacy).
3. Deploy cdc-cms-web.
4. Smoke test: create 1 dummy source + registry + automate → validate AC1-AC6.
5. Remove feature flag in next release nếu không regression.

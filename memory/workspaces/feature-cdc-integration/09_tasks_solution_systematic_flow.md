# Technical Solution — Systematic Flow

> Stage 4 · Phase: `systematic_flow` · 2026-04-24
> Ready for Stage 5 (Implementation). Each block is the exact source to land.

## 1. SQL — Migration 027

File: `centralized-data-service/migrations/027_systematic_sources.sql`

```sql
-- Migration 027: Systematic Flow — Sources registry + Wizard state machine
-- Adds: cdc_internal.sources, cdc_internal.cdc_wizard_sessions
-- Depends on: 018_sonyflake_v125_foundation.sql (creates cdc_internal schema)

BEGIN;

CREATE TABLE IF NOT EXISTS cdc_internal.sources (
  id                      BIGSERIAL PRIMARY KEY,
  connector_name          VARCHAR(200) NOT NULL UNIQUE,
  source_type             VARCHAR(32)  NOT NULL,
  connector_class         VARCHAR(200) NOT NULL,
  topic_prefix            VARCHAR(200),
  server_address          VARCHAR(500),
  database_include_list   VARCHAR(500),
  collection_include_list TEXT,
  raw_config_sanitized    JSONB,
  status                  VARCHAR(32)  NOT NULL DEFAULT 'created',
  created_by              VARCHAR(100),
  created_at              TIMESTAMP NOT NULL DEFAULT NOW(),
  updated_at              TIMESTAMP NOT NULL DEFAULT NOW(),
  CONSTRAINT sources_status_check CHECK (status IN ('created','running','paused','failed','deleted'))
);

CREATE INDEX IF NOT EXISTS idx_sources_status ON cdc_internal.sources(status);
CREATE INDEX IF NOT EXISTS idx_sources_type   ON cdc_internal.sources(source_type);

COMMENT ON TABLE  cdc_internal.sources IS
  'Systematic Flow: Connection Fingerprint. Persisted after POST /api/v1/system/connectors succeeds on Kafka Connect. Registry dropdown reads from here.';
COMMENT ON COLUMN cdc_internal.sources.status IS
  'Lifecycle: created -> running (after verified) -> paused/failed. Soft-delete: deleted.';

CREATE TABLE IF NOT EXISTS cdc_internal.cdc_wizard_sessions (
  id             UUID PRIMARY KEY,
  source_name    VARCHAR(200),
  connector_id   BIGINT REFERENCES cdc_internal.sources(id) ON DELETE SET NULL,
  registry_id    BIGINT,
  master_name    VARCHAR(200),
  current_step   INTEGER NOT NULL DEFAULT 0,
  status         VARCHAR(32) NOT NULL DEFAULT 'draft',
  step_payload   JSONB NOT NULL DEFAULT '{}'::jsonb,
  progress_log   JSONB NOT NULL DEFAULT '[]'::jsonb,
  created_by     VARCHAR(100),
  created_at     TIMESTAMP NOT NULL DEFAULT NOW(),
  updated_at     TIMESTAMP NOT NULL DEFAULT NOW(),
  CONSTRAINT wizard_status_check CHECK (status IN ('draft','running','done','failed'))
);

CREATE INDEX IF NOT EXISTS idx_wizard_sessions_status ON cdc_internal.cdc_wizard_sessions(status);
CREATE INDEX IF NOT EXISTS idx_wizard_sessions_created_by ON cdc_internal.cdc_wizard_sessions(created_by);

COMMENT ON TABLE cdc_internal.cdc_wizard_sessions IS
  'Systematic Flow: Wizard State Machine. Persists draft + progress of Source→Master automation.';

COMMIT;
```

## 2. SQL — Migration 028

File: `centralized-data-service/migrations/028_sonyflake_fallback_fn.sql`

```sql
-- Migration 028: Sonyflake Fallback Trigger function
-- Fallback-only: triggers sinh ID khi client INSERT không cung cấp id (NULL hoặc 0).
-- Authoritative path: Go Worker pkgs/idgen/sonyflake.go.
-- Depends on: 018_sonyflake_v125_foundation.sql (fencing_token_seq exists)

BEGIN;

-- Custom epoch: 2026-01-01 UTC (ms since)
CREATE OR REPLACE FUNCTION cdc_internal.gen_sonyflake_id()
RETURNS BIGINT AS $$
DECLARE
  v_ts_ms   BIGINT;
  v_machine INTEGER;
  v_seq     BIGINT;
BEGIN
  v_ts_ms := (EXTRACT(EPOCH FROM clock_timestamp()) * 1000)::BIGINT - 1767225600000;
  -- Machine ID: session-set via SET LOCAL cdc.machine_id = '...' by the Go worker.
  -- Fallback to 0 for psql inserts / manual use.
  BEGIN
    v_machine := COALESCE(NULLIF(current_setting('cdc.machine_id', true), '')::INTEGER, 0) & 65535;
  EXCEPTION WHEN OTHERS THEN
    v_machine := 0;
  END;
  v_seq := nextval('cdc_internal.fencing_token_seq') & 65535;
  RETURN ((v_ts_ms & 4398046511103) << 22) | ((v_machine::BIGINT & 65535) << 6) | (v_seq & 63);
END;
$$ LANGUAGE plpgsql VOLATILE;

COMMENT ON FUNCTION cdc_internal.gen_sonyflake_id() IS
  'Fallback Sonyflake-like ID. Shape: [42 bits ts ms since 2026-01-01][16 bits machine_id][6 bits seq]. Go worker overrides with authoritative IDs.';

-- Trigger body — attached per shadow table by Go automator.
CREATE OR REPLACE FUNCTION cdc_internal.tg_sonyflake_fallback()
RETURNS TRIGGER AS $$
BEGIN
  IF NEW.id IS NULL OR NEW.id = 0 THEN
    NEW.id := cdc_internal.gen_sonyflake_id();
  END IF;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION cdc_internal.tg_sonyflake_fallback() IS
  'BEFORE INSERT trigger body — fallback id gen. Attached by cdc_internal.ensure_shadow_sonyflake_trigger().';

-- Helper: idempotent attach trigger cho 1 table
CREATE OR REPLACE FUNCTION cdc_internal.ensure_shadow_sonyflake_trigger(p_table TEXT)
RETURNS VOID AS $$
DECLARE
  v_trigger_name TEXT;
BEGIN
  v_trigger_name := 'trg_' || p_table || '_sonyflake_fallback';
  -- Drop then recreate (idempotent; guards against old signature)
  EXECUTE format('DROP TRIGGER IF EXISTS %I ON cdc_internal.%I', v_trigger_name, p_table);
  EXECUTE format(
    'CREATE TRIGGER %I BEFORE INSERT ON cdc_internal.%I FOR EACH ROW EXECUTE FUNCTION cdc_internal.tg_sonyflake_fallback()',
    v_trigger_name, p_table
  );
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION cdc_internal.ensure_shadow_sonyflake_trigger(TEXT) IS
  'Idempotent: drop+recreate fallback trigger on cdc_internal.<p_table>. Called by Go ShadowAutomator after DDL.';

COMMIT;
```

## 3. Go — Source model + repo

`internal/model/source.go`:

```go
package model

import "time"

type Source struct {
    ID                     int64     `gorm:"column:id;primaryKey" json:"id"`
    ConnectorName          string    `gorm:"column:connector_name;uniqueIndex;not null" json:"connector_name"`
    SourceType             string    `gorm:"column:source_type;not null" json:"source_type"`
    ConnectorClass         string    `gorm:"column:connector_class;not null" json:"connector_class"`
    TopicPrefix            string    `gorm:"column:topic_prefix" json:"topic_prefix"`
    ServerAddress          string    `gorm:"column:server_address" json:"server_address"`
    DatabaseIncludeList    string    `gorm:"column:database_include_list" json:"database_include_list"`
    CollectionIncludeList  string    `gorm:"column:collection_include_list" json:"collection_include_list"`
    RawConfigSanitized     []byte    `gorm:"column:raw_config_sanitized;type:jsonb" json:"raw_config_sanitized,omitempty"`
    Status                 string    `gorm:"column:status;not null;default:created" json:"status"`
    CreatedBy              string    `gorm:"column:created_by" json:"created_by"`
    CreatedAt              time.Time `gorm:"column:created_at" json:"created_at"`
    UpdatedAt              time.Time `gorm:"column:updated_at" json:"updated_at"`
}

func (Source) TableName() string { return "cdc_internal.sources" }
```

`internal/repository/source_repo.go`:

```go
package repository

import (
    "context"

    "cdc-cms-service/internal/model"

    "gorm.io/gorm"
    "gorm.io/gorm/clause"
)

type SourceRepo struct{ db *gorm.DB }

func NewSourceRepo(db *gorm.DB) *SourceRepo { return &SourceRepo{db: db} }

// Upsert by connector_name.
func (r *SourceRepo) Upsert(ctx context.Context, s *model.Source) error {
    return r.db.WithContext(ctx).Clauses(clause.OnConflict{
        Columns: []clause.Column{{Name: "connector_name"}},
        DoUpdates: clause.AssignmentColumns([]string{
            "source_type", "connector_class", "topic_prefix", "server_address",
            "database_include_list", "collection_include_list", "raw_config_sanitized",
            "status", "updated_at",
        }),
    }).Create(s).Error
}

func (r *SourceRepo) List(ctx context.Context) ([]model.Source, error) {
    var out []model.Source
    err := r.db.WithContext(ctx).Where("status != ?", "deleted").Order("created_at DESC").Find(&out).Error
    return out, err
}

func (r *SourceRepo) GetByID(ctx context.Context, id int64) (*model.Source, error) {
    var s model.Source
    err := r.db.WithContext(ctx).First(&s, id).Error
    return &s, err
}

func (r *SourceRepo) MarkDeleted(ctx context.Context, connectorName string) error {
    return r.db.WithContext(ctx).Model(&model.Source{}).
        Where("connector_name = ?", connectorName).
        Update("status", "deleted").Error
}
```

## 4. Go — ShadowAutomator

`internal/service/shadow_automator.go`:

```go
package service

import (
    "context"
    "fmt"

    "cdc-cms-service/internal/model"

    "go.uber.org/zap"
    "gorm.io/gorm"
)

type ShadowAutomator struct {
    db     *gorm.DB
    logger *zap.Logger
}

func NewShadowAutomator(db *gorm.DB, logger *zap.Logger) *ShadowAutomator {
    return &ShadowAutomator{db: db, logger: logger}
}

// EnsureShadowTable creates cdc_internal.<target>, deploys Sonyflake fallback fn if missing,
// attaches per-table trigger. Idempotent. Synchronous. Safe to call on every Register.
func (s *ShadowAutomator) EnsureShadowTable(ctx context.Context, reg *model.TableRegistry) error {
    if err := validateIdent(reg.TargetTable); err != nil {
        return fmt.Errorf("invalid target_table: %w", err)
    }
    if err := s.ensureSonyflakeFunction(ctx); err != nil {
        return fmt.Errorf("bootstrap sonyflake fn: %w", err)
    }
    if err := s.createShadowDDL(ctx, reg); err != nil {
        return fmt.Errorf("create shadow ddl: %w", err)
    }
    if err := s.attachSonyflakeTrigger(ctx, reg.TargetTable); err != nil {
        return fmt.Errorf("attach trigger: %w", err)
    }
    return s.markCreated(ctx, reg)
}

// ensureSonyflakeFunction deploys cdc_internal.gen_sonyflake_id() + tg body + helper
// if they don't exist. Runs inline so the automator is self-contained (Boss directive).
func (s *ShadowAutomator) ensureSonyflakeFunction(ctx context.Context) error {
    const ddl = `
    CREATE OR REPLACE FUNCTION cdc_internal.gen_sonyflake_id()
    RETURNS BIGINT AS $fn$
    DECLARE
      v_ts_ms BIGINT; v_machine INTEGER; v_seq BIGINT;
    BEGIN
      v_ts_ms := (EXTRACT(EPOCH FROM clock_timestamp()) * 1000)::BIGINT - 1767225600000;
      BEGIN
        v_machine := COALESCE(NULLIF(current_setting('cdc.machine_id', true), '')::INTEGER, 0) & 65535;
      EXCEPTION WHEN OTHERS THEN v_machine := 0; END;
      v_seq := nextval('cdc_internal.fencing_token_seq') & 65535;
      RETURN ((v_ts_ms & 4398046511103) << 22) | ((v_machine::BIGINT & 65535) << 6) | (v_seq & 63);
    END;
    $fn$ LANGUAGE plpgsql VOLATILE;

    CREATE OR REPLACE FUNCTION cdc_internal.tg_sonyflake_fallback()
    RETURNS TRIGGER AS $tg$
    BEGIN
      IF NEW.id IS NULL OR NEW.id = 0 THEN
        NEW.id := cdc_internal.gen_sonyflake_id();
      END IF;
      RETURN NEW;
    END;
    $tg$ LANGUAGE plpgsql;

    CREATE OR REPLACE FUNCTION cdc_internal.ensure_shadow_sonyflake_trigger(p_table TEXT)
    RETURNS VOID AS $h$
    DECLARE v_trigger_name TEXT;
    BEGIN
      v_trigger_name := 'trg_' || p_table || '_sonyflake_fallback';
      EXECUTE format('DROP TRIGGER IF EXISTS %I ON cdc_internal.%I', v_trigger_name, p_table);
      EXECUTE format('CREATE TRIGGER %I BEFORE INSERT ON cdc_internal.%I FOR EACH ROW EXECUTE FUNCTION cdc_internal.tg_sonyflake_fallback()', v_trigger_name, p_table);
    END;
    $h$ LANGUAGE plpgsql;`
    return s.db.WithContext(ctx).Exec(ddl).Error
}

func (s *ShadowAutomator) createShadowDDL(ctx context.Context, reg *model.TableRegistry) error {
    // Schema must exist (migration 018 creates it). Safe to CREATE IF NOT EXISTS.
    ddl := fmt.Sprintf(`
        CREATE SCHEMA IF NOT EXISTS cdc_internal;
        CREATE TABLE IF NOT EXISTS cdc_internal.%[1]q (
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
        CREATE INDEX IF NOT EXISTS %[2]q ON cdc_internal.%[1]q (_synced_at);
        CREATE INDEX IF NOT EXISTS %[3]q ON cdc_internal.%[1]q (_source);
        CREATE INDEX IF NOT EXISTS %[4]q ON cdc_internal.%[1]q USING GIN(_raw_data);
    `,
        reg.TargetTable,
        "idx_"+reg.TargetTable+"_synced_at",
        "idx_"+reg.TargetTable+"_source",
        "idx_"+reg.TargetTable+"_raw",
    )
    return s.db.WithContext(ctx).Exec(ddl).Error
}

func (s *ShadowAutomator) attachSonyflakeTrigger(ctx context.Context, table string) error {
    return s.db.WithContext(ctx).Exec("SELECT cdc_internal.ensure_shadow_sonyflake_trigger(?)", table).Error
}

func (s *ShadowAutomator) markCreated(ctx context.Context, reg *model.TableRegistry) error {
    return s.db.WithContext(ctx).Model(&model.TableRegistry{}).
        Where("id = ?", reg.ID).Update("is_table_created", true).Error
}

// validateIdent guards against SQL injection via table name.
// Accept: [a-z0-9_]{1,63}
func validateIdent(s string) error {
    if len(s) == 0 || len(s) > 63 {
        return fmt.Errorf("identifier length")
    }
    for _, c := range s {
        if !(c == '_' || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9')) {
            return fmt.Errorf("identifier char: %q", c)
        }
    }
    return nil
}
```

## 5. Go — System Connectors Create edit

`internal/api/system_connectors_handler.go` (key edits):

```go
// Constructor
type SystemConnectorsHandler struct {
    kafkaConnectURL string
    httpClient      *http.Client
    sourceRepo      *repository.SourceRepo // NEW
    logger          *zap.Logger
}

func NewSystemConnectorsHandler(kafkaConnectURL string, sourceRepo *repository.SourceRepo, logger *zap.Logger) *SystemConnectorsHandler {
    return &SystemConnectorsHandler{
        kafkaConnectURL: strings.TrimRight(kafkaConnectURL, "/"),
        httpClient:      &http.Client{Timeout: 10 * time.Second},
        sourceRepo:      sourceRepo,
        logger:          logger,
    }
}

// Append at end of Create(), after kafka-connect returned 201:
fp := parseFingerprint(req.Name, req.Config)
rawCfg, _ := json.Marshal(filterSafeConfig(req.Config))
src := &model.Source{
    ConnectorName:         req.Name,
    SourceType:            fp.sourceType,
    ConnectorClass:        req.Config["connector.class"],
    TopicPrefix:           fp.topicPrefix,
    ServerAddress:         fp.serverAddress,
    DatabaseIncludeList:   fp.dbList,
    CollectionIncludeList: fp.collectionList,
    RawConfigSanitized:    rawCfg,
    Status:                "created",
    CreatedBy:             middleware.GetUsername(c),
}
if err := h.sourceRepo.Upsert(c.Context(), src); err != nil {
    // best-effort — connector already live on Kafka Connect; don't fail the request
    h.logger.Warn("source fingerprint persist failed",
        zap.String("connector", req.Name), zap.Error(err))
}

// helper (private)
type fingerprint struct {
    sourceType, topicPrefix, serverAddress, dbList, collectionList string
}

func parseFingerprint(name string, cfg map[string]string) fingerprint {
    fp := fingerprint{topicPrefix: cfg["topic.prefix"]}
    cls := cfg["connector.class"]
    switch {
    case strings.Contains(cls, "MongoDb"):
        fp.sourceType = "mongodb"
        fp.serverAddress = cfg["mongodb.connection.string"]
        fp.dbList = cfg["database.include.list"]
        fp.collectionList = cfg["collection.include.list"]
    case strings.Contains(cls, "MySql"):
        fp.sourceType = "mysql"
        fp.serverAddress = cfg["database.hostname"] + ":" + cfg["database.port"]
        fp.dbList = cfg["database.include.list"]
        fp.collectionList = cfg["table.include.list"]
    case strings.Contains(cls, "Postgres"):
        fp.sourceType = "postgres"
        fp.serverAddress = cfg["database.hostname"] + ":" + cfg["database.port"]
        fp.dbList = cfg["database.dbname"]
        fp.collectionList = cfg["table.include.list"]
    default:
        fp.sourceType = "unknown"
    }
    return fp
}
```

Append end of `Delete(c *fiber.Ctx)` (after kafka-connect 202):

```go
_ = h.sourceRepo.MarkDeleted(c.Context(), name) // best-effort
```

## 6. Go — Sources handler

`internal/api/sources_handler.go`:

```go
package api

import (
    "strconv"

    "cdc-cms-service/internal/repository"

    "github.com/gofiber/fiber/v2"
    "go.uber.org/zap"
)

type SourcesHandler struct {
    repo   *repository.SourceRepo
    logger *zap.Logger
}

func NewSourcesHandler(repo *repository.SourceRepo, logger *zap.Logger) *SourcesHandler {
    return &SourcesHandler{repo: repo, logger: logger}
}

func (h *SourcesHandler) List(c *fiber.Ctx) error {
    items, err := h.repo.List(c.Context())
    if err != nil {
        return c.Status(500).JSON(fiber.Map{"error": "list sources: " + err.Error()})
    }
    return c.JSON(fiber.Map{"data": items, "count": len(items)})
}

func (h *SourcesHandler) Get(c *fiber.Ctx) error {
    id, err := strconv.ParseInt(c.Params("id"), 10, 64)
    if err != nil {
        return c.Status(400).JSON(fiber.Map{"error": "invalid id"})
    }
    s, err := h.repo.GetByID(c.Context(), id)
    if err != nil {
        return c.Status(404).JSON(fiber.Map{"error": "not found"})
    }
    return c.JSON(s)
}
```

## 7. Go — Registry handler Register() edit

`internal/api/registry_handler.go`:

```go
// Constructor: add automator *service.ShadowAutomator
// (same pattern as other fields).

// Inside Register, replace the section between repo.Create and NATS publish:
if err := h.repo.Create(c.Context(), &entry); err != nil {
    return c.Status(500).JSON(fiber.Map{"error": "failed to register table: " + err.Error()})
}

// Synchronous Shadow DDL (Systematic Flow). Rollback registry row on fail.
if err := h.automator.EnsureShadowTable(c.Context(), &entry); err != nil {
    if delErr := h.db.Delete(&model.TableRegistry{}, entry.ID).Error; delErr != nil {
        h.logger.Error("registry rollback failed after shadow err",
            zap.Uint("id", entry.ID), zap.Error(delErr))
    }
    return c.Status(500).JSON(fiber.Map{"error": "shadow DDL failed: " + err.Error()})
}

// Legacy NATS path still published — Worker treats is_table_created=true as no-op.
createColsPayload, _ := json.Marshal(map[string]interface{}{...}) // unchanged
h.natsClient.Conn.Publish("cdc.cmd.create-default-columns", createColsPayload)
```

## 8. Go — Wizard session + handler

Full listings in Stage 5 pass. Structure summary:

- `model/wizard_session.go`: UUID PK, JSONB step_payload + progress_log.
- `repository/wizard_repo.go`: Create, Get, Update, AppendProgress.
- `api/wizard_handler.go`:
  - `Create(c)` — body `{source_name}`, generate UUID, INSERT, return 201.
  - `Get(c)` — SELECT by id.
  - `Patch(c)` — update step_payload + current_step.
  - `Execute(c)` — kicks off goroutine running pipeline; return 202 with session_id.
  - `Progress(c)` — returns current row (step, status, progress_log).

## 9. Go — Atomic swap

`internal/service/master_swap.go`:

```go
package service

import (
    "context"
    "fmt"
    "time"

    "go.uber.org/zap"
    "gorm.io/gorm"
)

type MasterSwap struct {
    db     *gorm.DB
    logger *zap.Logger
}

func NewMasterSwap(db *gorm.DB, logger *zap.Logger) *MasterSwap {
    return &MasterSwap{db: db, logger: logger}
}

// Swap atomically renames public.<masterName> → <masterName>_old_<ts>
// and public.<newTableName> → public.<masterName>, inside one TX.
// Uses SET LOCAL lock_timeout to avoid long API hangs.
func (s *MasterSwap) Swap(ctx context.Context, masterName, newTableName, reason string) error {
    if err := validateIdent(masterName); err != nil { return err }
    if err := validateIdent(newTableName); err != nil { return err }

    return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
        if err := tx.Exec("SET LOCAL lock_timeout = '3s'").Error; err != nil { return err }

        ts := time.Now().Unix()
        oldName := fmt.Sprintf("%s_old_%d", masterName, ts)

        if err := tx.Exec(fmt.Sprintf(`ALTER TABLE public.%q RENAME TO %q`, masterName, oldName)).Error; err != nil {
            return fmt.Errorf("rename current: %w", err)
        }
        if err := tx.Exec(fmt.Sprintf(`ALTER TABLE public.%q RENAME TO %q`, newTableName, masterName)).Error; err != nil {
            return fmt.Errorf("rename new: %w", err)
        }
        return tx.Exec(`
            INSERT INTO cdc_activity_log (operation, target_table, status, details, triggered_by, started_at, completed_at)
            VALUES ('master_swap', ?, 'success', ?::jsonb, 'manual', NOW(), NOW())`,
            masterName, fmt.Sprintf(`{"old_table":%q,"new_table":%q,"reason":%q}`, oldName, newTableName, reason),
        ).Error
    })
}
```

## 10. FE — TableRegistry dropdown

Minimal diff at `TableRegistry.tsx:444-470` (Register Modal). Replace direct `<Input>` for source fields with:
- Fetch: `useEffect` load `/api/v1/sources` once.
- Select source_id → auto-fill source_db + source_type (disabled inputs).
- Select source_table from parsed `collection_include_list`.

(Exact code in Stage 5 when file is in context.)

## 11. FE — Wizard rewrite

Minimum contract in Stage 5:
- Mount: read `?session_id=` → if missing, `POST /api/v1/wizard/sessions` → set URL.
- Load: `GET /api/v1/wizard/sessions/:id` every 2s while `status === 'running'`.
- Action buttons call `PATCH` to update step_payload + call `Execute`.
- 11 steps hydrated from `progress_log`.

## 12. Done definition (code-complete)

- 2 migration files.
- 4 new Go files (model/source, repo/source, service/shadow_automator, service/master_swap).
- 3 edited Go files (system_connectors_handler, registry_handler, server).
- 1 router edit.
- 3 new Go files for wizard (model/wizard_session, repo/wizard, api/wizard_handler).
- 1 edited master_registry_handler (add Swap).
- 1 new api/sources_handler.
- 2 FE files (rewrite wizard + edit TableRegistry).

Total: ~13 new + 5 edits + 2 migrations. Estimated 4-6h coding.

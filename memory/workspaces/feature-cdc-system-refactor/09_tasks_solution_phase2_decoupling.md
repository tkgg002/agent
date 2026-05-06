# Phase 2 v2 — Tasks Solution — Code sketch per task

> **Tasks ref**: `08_tasks_phase2_decoupling.md`
> **Plan ref**: `02_plan_phase2_decoupling.md`

---

## P1 — Code sketch

### T1.1 — `internal/domain/job/job.go` (entity)

```go
package job

import "time"

type Status string

const (
    StatusPending Status = "pending"
    StatusRunning Status = "running"
    StatusSuccess Status = "success"
    StatusFailed  Status = "failed"
)

type Job struct {
    ID             string
    Type           string
    Status         Status
    Payload        []byte // JSON
    Result         []byte // JSON, populated when finished
    ErrorMessage   string
    IdempotencyKey string
    CreatedBy      string
    CorrelationID  string
    CreatedAt      time.Time
    StartedAt      *time.Time
    FinishedAt     *time.Time
}

func New(jtype, payload string, createdBy string) *Job {
    return &Job{
        Type:      jtype,
        Status:    StatusPending,
        Payload:   []byte(payload),
        CreatedBy: createdBy,
        CreatedAt: time.Now().UTC(),
    }
}
```

### T1.2 — `internal/app/ports/command_bus.go`

```go
package ports

import "context"

type Command interface {
    Type() string
    Validate() error
}

type CommandResult struct {
    JobID    string
    Accepted bool
}

type CommandBus interface {
    Dispatch(ctx context.Context, c Command) (CommandResult, error)
}
```

### T1.4 — `cmd/server/main.go` wiring (excerpt)

```go
// after db, nats, redis setup
mappingRuleRepo := persistence.NewMappingRuleRepo(db)
sourceRepo := persistence.NewSourceRepo(db)
masterRepo := persistence.NewMasterRepo(db)
jobRepo := persistence.NewJobRepo(db)

publisher := messaging.NewNATSPublisher(natsClient.Conn, logger)
cmdBus := messaging.NewNATSCommandBus(natsClient.Conn, jobRepo, logger)

queryBus := app.NewQueryBus(/* register query handlers from queries/ */)

apiServer := server.New(server.Deps{
    QueryBus:   queryBus,
    CommandBus: cmdBus,
    Publisher:  publisher,
    JobRepo:    jobRepo,
    // ... per-handler injection
})
```

---

## P2 — Code sketch

### T2.1 — `internal/app/queries/list_mapping_rules.go`

```go
package queries

import (
    "context"

    "cdc-cms-service/internal/app/ports"
    "cdc-cms-service/internal/domain/mapping"
)

type ListMappingRulesQuery struct {
    Filter mapping.Filter
    Limit  int
    Offset int
}

func (q ListMappingRulesQuery) Type() string { return "mapping.list" }

type ListMappingRulesHandler struct {
    repo ports.MappingRuleRepo
}

func NewListMappingRulesHandler(r ports.MappingRuleRepo) *ListMappingRulesHandler {
    return &ListMappingRulesHandler{repo: r}
}

func (h *ListMappingRulesHandler) Handle(ctx context.Context, q ListMappingRulesQuery) ([]mapping.Rule, error) {
    return h.repo.List(ctx, q.Filter)
}
```

### T2.1 — `internal/infra/persistence/mapping_rule_repo_gorm.go` (placeholder cho P2, full ở P4)

```go
package persistence

import (
    "context"

    "cdc-cms-service/internal/app/ports"
    "cdc-cms-service/internal/domain/mapping"
    "gorm.io/gorm"
)

type mappingRuleRepoGorm struct {
    db *gorm.DB
}

func NewMappingRuleRepo(db *gorm.DB) ports.MappingRuleRepo {
    return &mappingRuleRepoGorm{db: db}
}

func (r *mappingRuleRepoGorm) List(ctx context.Context, f mapping.Filter) ([]mapping.Rule, error) {
    var rows []mappingRuleRow // flat scan struct per Lesson #1253
    q := r.db.WithContext(ctx).Table("cdc_system.mapping_rule_v2")
    if f.Status != "" { q = q.Where("status = ?", f.Status) }
    if f.TargetTable != "" { q = q.Where("target_table = ?", f.TargetTable) }
    if err := q.Order("id DESC").Find(&rows).Error; err != nil {
        return nil, err
    }
    out := make([]mapping.Rule, len(rows))
    for i, row := range rows {
        out[i] = row.toDomain()
    }
    return out, nil
}

type mappingRuleRow struct {
    ID            int64  `gorm:"column:id"`
    SourceObjectID int64 `gorm:"column:source_object_id"`
    TargetTable   string `gorm:"column:target_table"`
    SourceField   string `gorm:"column:source_field"`
    TargetColumn  string `gorm:"column:target_column"`
    DataType      string `gorm:"column:data_type"`
    Status        string `gorm:"column:status"`
}

func (r mappingRuleRow) toDomain() mapping.Rule {
    return mapping.Rule{
        ID: r.ID, SourceObjectID: r.SourceObjectID,
        TargetTable: r.TargetTable, SourceField: r.SourceField,
        TargetColumn: r.TargetColumn, DataType: r.DataType,
        Status: mapping.Status(r.Status),
    }
}
```

### T2.1 — `internal/api/mapping_rule_handler.go` thinning

```go
// BEFORE
func (h *MappingRuleHandler) List(c *fiber.Ctx) error {
    var rows []map[string]any
    err := h.db.Raw(`SELECT ... FROM cdc_system.mapping_rule_v2 WHERE ...`).Scan(&rows).Error
    if err != nil { return c.Status(500).JSON(...) }
    return c.JSON(rows)
}

// AFTER
func (h *MappingRuleHandler) List(c *fiber.Ctx) error {
    q := queries.ListMappingRulesQuery{
        Filter: parseFilter(c),
        Limit:  parseLimit(c, 100),
    }
    res, err := h.queryBus.Ask(c.Context(), q)
    if err != nil { return c.Status(500).JSON(fiber.Map{"error": err.Error()}) }
    return c.JSON(res)
}
```

---

## P3 — Code sketch

### T3.1 — `centralized-data-service/migrations/cdc/036_create_cdc_jobs.sql`

```sql
-- Phase 2 v2 — P3.1 — cdc_jobs job tracking table
-- Apply order: BEFORE deploying CMS v2 to prod (CMS Dispatch sẽ INSERT vào table này).
BEGIN;

CREATE TABLE IF NOT EXISTS cdc_system.cdc_jobs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    type            TEXT NOT NULL,
    status          TEXT NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending','running','success','failed')),
    payload         JSONB NOT NULL,
    result          JSONB,
    error_message   TEXT,
    idempotency_key TEXT UNIQUE,
    created_by      TEXT NOT NULL,
    correlation_id  TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    started_at      TIMESTAMPTZ,
    finished_at     TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_cdc_jobs_type_status
    ON cdc_system.cdc_jobs(type, status, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_cdc_jobs_correlation
    ON cdc_system.cdc_jobs(correlation_id);

COMMIT;
```

### T3.2 — `internal/infra/persistence/job_repo_gorm.go`

```go
package persistence

import (
    "context"
    "time"

    "cdc-cms-service/internal/app/ports"
    "cdc-cms-service/internal/domain/job"
    "gorm.io/gorm"
)

type jobRepoGorm struct{ db *gorm.DB }

func NewJobRepo(db *gorm.DB) ports.JobRepo {
    return &jobRepoGorm{db: db}
}

func (r *jobRepoGorm) Create(ctx context.Context, j *job.Job) error {
    return r.db.WithContext(ctx).Exec(`
        INSERT INTO cdc_system.cdc_jobs
            (id, type, status, payload, idempotency_key, created_by, correlation_id, created_at)
        VALUES (gen_random_uuid(), ?, 'pending', ?::jsonb, NULLIF(?, ''), ?, NULLIF(?, ''), NOW())
        ON CONFLICT (idempotency_key) DO NOTHING
        RETURNING id
    `, j.Type, string(j.Payload), j.IdempotencyKey, j.CreatedBy, j.CorrelationID).
        Scan(&j.ID).Error
}

func (r *jobRepoGorm) GetByID(ctx context.Context, id string) (*job.Job, error) {
    var row jobRow
    err := r.db.WithContext(ctx).Table("cdc_system.cdc_jobs").
        Where("id = ?", id).First(&row).Error
    if err != nil { return nil, err }
    j := row.toDomain()
    return &j, nil
}

func (r *jobRepoGorm) UpdateStatus(ctx context.Context, id string, s job.Status, result, errMsg string) error {
    return r.db.WithContext(ctx).Exec(`
        UPDATE cdc_system.cdc_jobs
           SET status = ?,
               result = NULLIF(?, '')::jsonb,
               error_message = NULLIF(?, ''),
               finished_at = NOW()
         WHERE id = ?
           AND status IN ('pending','running')
    `, string(s), result, errMsg, id).Error
}

type jobRow struct {
    ID, Type, Status, CreatedBy, CorrelationID string
    Payload, Result                              []byte
    ErrorMessage                                 string `gorm:"column:error_message"`
    IdempotencyKey                               string `gorm:"column:idempotency_key"`
    CreatedAt                                    time.Time `gorm:"column:created_at"`
    StartedAt, FinishedAt                        *time.Time
}

func (r jobRow) toDomain() job.Job {
    return job.Job{
        ID: r.ID, Type: r.Type, Status: job.Status(r.Status),
        Payload: r.Payload, Result: r.Result,
        ErrorMessage: r.ErrorMessage, IdempotencyKey: r.IdempotencyKey,
        CreatedBy: r.CreatedBy, CorrelationID: r.CorrelationID,
        CreatedAt: r.CreatedAt, StartedAt: r.StartedAt, FinishedAt: r.FinishedAt,
    }
}
```

### T3.3 — `internal/infra/messaging/nats_command_bus.go`

```go
package messaging

import (
    "context"
    "encoding/json"
    "fmt"

    "cdc-cms-service/internal/app/ports"
    "cdc-cms-service/internal/domain/job"
    "github.com/nats-io/nats.go"
    "go.uber.org/zap"
)

type natsCommandBus struct {
    nc      *nats.Conn
    jobRepo ports.JobRepo
    log     *zap.Logger
}

func NewNATSCommandBus(nc *nats.Conn, jobRepo ports.JobRepo, log *zap.Logger) ports.CommandBus {
    return &natsCommandBus{nc: nc, jobRepo: jobRepo, log: log}
}

func (b *natsCommandBus) Dispatch(ctx context.Context, c ports.Command) (ports.CommandResult, error) {
    if err := c.Validate(); err != nil {
        return ports.CommandResult{}, err
    }
    payload, err := json.Marshal(c)
    if err != nil { return ports.CommandResult{}, err }

    j := &job.Job{
        Type:          c.Type(),
        Status:        job.StatusPending,
        Payload:       payload,
        CreatedBy:     ctxUserID(ctx),
        CorrelationID: ctxCorrelationID(ctx),
    }
    if ck, ok := c.(commandWithIdempotency); ok {
        j.IdempotencyKey = ck.IdempotencyKey()
    }
    if err := b.jobRepo.Create(ctx, j); err != nil {
        return ports.CommandResult{}, fmt.Errorf("create job: %w", err)
    }

    subject := mapTypeToSubject(c.Type())
    natsPayload, _ := json.Marshal(map[string]any{
        "job_id":         j.ID,
        "type":           c.Type(),
        "payload":        json.RawMessage(payload),
        "correlation_id": j.CorrelationID,
    })
    if err := b.nc.Publish(subject, natsPayload); err != nil {
        // Job row stays as 'pending'. Recovery cron có thể re-publish.
        b.log.Warn("nats publish failed; job remains pending",
            zap.String("subject", subject),
            zap.String("job_id", j.ID),
            zap.Error(err))
        return ports.CommandResult{}, fmt.Errorf("publish: %w", err)
    }
    return ports.CommandResult{JobID: j.ID, Accepted: true}, nil
}

type commandWithIdempotency interface {
    IdempotencyKey() string
}

func mapTypeToSubject(t string) string {
    return map[string]string{
        "master.swap":         "cdc.cmd.master-swap",
        "master.create":       "cdc.cmd.master-create",
        "source.v2-sync":      "cdc.cmd.v2-sync",
        "source.standardize":  "cdc.cmd.standardize",
        "source.scan-fields":  "cdc.cmd.scan-fields",
        "source.create-default-columns": "cdc.cmd.create-default-columns",
        "source.detect-timestamp":       "cdc.cmd.detect-timestamp-field",
        "mapping.alter-column":          "cdc.cmd.alter-column",
        "mapping.backfill":              "cdc.cmd.backfill",
        "recon.check":                   "cdc.cmd.recon-check",
        "recon.heal":                    "cdc.cmd.recon-heal",
        "recon.retry-failed":            "cdc.cmd.retry-failed",
        "recon.debezium-signal":         "cdc.cmd.debezium-signal",
        "recon.debezium-snapshot":       "cdc.cmd.debezium-snapshot",
        "recon.backfill-source-ts":      "cdc.cmd.recon-backfill-source-ts",
        "transmute.run":                 "cdc.cmd.transmute",
        "connector.restart-debezium":    "cdc.cmd.restart-debezium",
    }[t]
}
```

### T3.5 — `internal/app/commands/trigger_recon_check.go`

```go
package commands

type TriggerReconCheckCommand struct {
    Tier  string `json:"tier"`
    Table string `json:"table"`
}

func (c TriggerReconCheckCommand) Type() string { return "recon.check" }

func (c TriggerReconCheckCommand) Validate() error {
    if c.Table == "" { return fmt.Errorf("table required") }
    return nil
}
```

Handler API thin (`reconciliation_handler.go:TriggerCheck`):
```go
func (h *ReconciliationHandler) TriggerCheck(c *fiber.Ctx) error {
    var req struct { Tier, Table string }
    if err := c.BodyParser(&req); err != nil { return badRequest(c, err) }
    cmd := commands.TriggerReconCheckCommand{Tier: req.Tier, Table: req.Table}
    res, err := h.cmdBus.Dispatch(c.Context(), cmd)
    if err != nil { return c.Status(500).JSON(fiber.Map{"error": err.Error()}) }
    return c.Status(202).JSON(fiber.Map{"job_id": res.JobID, "accepted": true})
}
```

### T3.6 — Worker: `master_swap_handler.go`

```go
// centralized-data-service/internal/handler/master_swap_handler.go
package handler

import (
    "context"
    "encoding/json"
    "fmt"

    "github.com/nats-io/nats.go"
    "go.uber.org/zap"
    "gorm.io/gorm"
)

const (
    SubjectMasterSwapCmd       = "cdc.cmd.master-swap"
    SubjectMasterSwapCompleted = "cdc.evt.master-swap.completed"
)

type MasterSwapHandler struct {
    db   *gorm.DB
    nats *nats.Conn
    log  *zap.Logger
}

func NewMasterSwapHandler(db *gorm.DB, nc *nats.Conn, log *zap.Logger) *MasterSwapHandler {
    return &MasterSwapHandler{db: db, nats: nc, log: log}
}

type masterSwapPayload struct {
    JobID    string          `json:"job_id"`
    Type     string          `json:"type"`
    Payload  json.RawMessage `json:"payload"`
    CorrID   string          `json:"correlation_id"`
}

type masterSwapInner struct {
    MasterTable string `json:"master_table"`
    ShadowTable string `json:"shadow_table"`
}

func (h *MasterSwapHandler) Handle(msg *nats.Msg) {
    var p masterSwapPayload
    if err := json.Unmarshal(msg.Data, &p); err != nil {
        h.log.Warn("master-swap: bad payload", zap.Error(err))
        return
    }
    var inner masterSwapInner
    _ = json.Unmarshal(p.Payload, &inner)

    ctx := context.Background()
    err := h.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
        if err := tx.Exec(`SET LOCAL lock_timeout = '3s'`).Error; err != nil { return err }
        // dùng pgIdent quote
        sql := fmt.Sprintf(`ALTER TABLE %s RENAME TO %s_old; ALTER TABLE %s RENAME TO %s`,
            quoteIdent(inner.MasterTable),
            quoteIdent(inner.MasterTable),
            quoteIdent(inner.ShadowTable),
            quoteIdent(inner.MasterTable))
        return tx.Exec(sql).Error
    })

    h.publishCompleted(p.JobID, err)
}

func (h *MasterSwapHandler) publishCompleted(jobID string, err error) {
    status, errStr := "success", ""
    if err != nil {
        status, errStr = "failed", err.Error()
    }
    evt, _ := json.Marshal(map[string]any{
        "job_id":       jobID,
        "type":         "master.swap",
        "status":       status,
        "error":        errStr,
        "completed_at": time.Now().UTC().Format(time.RFC3339Nano),
    })
    if perr := h.nats.Publish(SubjectMasterSwapCompleted, evt); perr != nil {
        h.log.Warn("publish master-swap.completed failed",
            zap.String("job_id", jobID), zap.Error(perr))
    }
}
```

Worker boot (`worker_server.go`):
```go
masterSwapH := handler.NewMasterSwapHandler(db, natsClient.Conn, logger)
if _, err := natsClient.Conn.Subscribe(handler.SubjectMasterSwapCmd, masterSwapH.Handle); err != nil {
    return fmt.Errorf("subscribe %s: %w", handler.SubjectMasterSwapCmd, err)
}
```

### T3.9 — `JobMonitor` extend wildcard

```go
// centralized-data-service/internal/service/job_monitor.go
const SubjectAllCompleted = "cdc.evt.*.completed"

func (m *JobMonitor) Subscribe(nc *nats.Conn) error {
    _, err := nc.Subscribe(SubjectAllCompleted, m.HandleCompleted)
    return err
}

type completedEvent struct {
    JobID       string          `json:"job_id"`
    Type        string          `json:"type"`
    Status      string          `json:"status"`
    Result      json.RawMessage `json:"result"`
    Error       string          `json:"error"`
    CompletedAt string          `json:"completed_at"`
}

func (m *JobMonitor) HandleCompleted(msg *nats.Msg) {
    var ev completedEvent
    if err := json.Unmarshal(msg.Data, &ev); err != nil {
        m.logger.Warn("job monitor: bad payload", zap.Error(err))
        return
    }
    if ev.JobID == "" {
        // legacy event không có job_id (e.g. transmute scheduler ad-hoc) — skip
        return
    }
    err := m.db.Exec(`
        UPDATE cdc_system.cdc_jobs
           SET status = ?, result = NULLIF(?, '')::jsonb,
               error_message = NULLIF(?, ''),
               finished_at = NOW()
         WHERE id = ?
           AND status IN ('pending','running')
    `, ev.Status, string(ev.Result), ev.Error, ev.JobID).Error
    if err != nil {
        m.logger.Warn("job monitor: update failed",
            zap.String("job_id", ev.JobID), zap.Error(err))
    }
}
```

### T3.10 — `GET /api/jobs/:id` thin handler

```go
// internal/api/jobs_handler.go
package api

type JobsHandler struct {
    queryBus ports.QueryBus
}

func (h *JobsHandler) Get(c *fiber.Ctx) error {
    id := c.Params("id")
    if id == "" { return c.Status(400).JSON(fiber.Map{"error": "id required"}) }
    res, err := h.queryBus.Ask(c.Context(), queries.GetJobQuery{ID: id})
    if err != nil { return c.Status(404).JSON(fiber.Map{"error": err.Error()}) }
    return c.JSON(res)
}
```

---

## P4 — Code sketch

### T4.6 — `pkgs/utils/pg_ident.go`

```go
package utils

import "strings"

// PgIdent quotes a Postgres identifier safely.
// Preserves existing pgIdent semantics from reconciliation_handler.go.
func PgIdent(s string) string {
    return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
}
```

### T4.7 — Service file deletion checklist

```bash
# Pre-delete audit (no caller from non-internal packages)
for f in internal/service/*.go; do
    base=$(basename "$f" .go)
    callers=$(grep -rn "service\.${base}" internal/ --include='*.go' | grep -v "internal/service/")
    if [ -z "$callers" ]; then
        echo "SAFE TO DELETE: $f"
    else
        echo "STILL USED: $f -> $callers"
    fi
done
```

### T4.8 — `wc -l` validation

```bash
wc -l internal/api/*.go | awk '$1 > 100 && $2 != "total" { print "VIOLATION:", $0 }'
# DoD: empty output
```

### T4.9 — Coverage gate

```bash
go test -cover ./internal/infra/persistence/... | tee /tmp/cov.txt
awk '/coverage:/ { gsub("%",""); if ($2 < 50) { print "BELOW 50%:", $0; exit 1 } }' /tmp/cov.txt
```

---

## Pre-commit checklist (mỗi pillar)

```bash
#!/usr/bin/env bash
set -euo pipefail

echo "=== Build ==="
go build ./...

echo "=== Test ==="
go test ./... -count=1

echo "=== Coverage gate (P2/P3/P4) ==="
case "${PILLAR:-}" in
  P2) go test -cover ./internal/app/queries/... | tee /tmp/cov.txt
      awk '/coverage:/ { gsub("%",""); if ($2 < 60) { exit 1 } }' /tmp/cov.txt ;;
  P3) go test -cover ./internal/app/commands/... ./internal/infra/messaging/... | tee /tmp/cov.txt
      awk '/coverage:/ { gsub("%",""); if ($2 < 50) { exit 1 } }' /tmp/cov.txt ;;
  P4) go test -cover ./internal/infra/persistence/... | tee /tmp/cov.txt
      awk '/coverage:/ { gsub("%",""); if ($2 < 50) { exit 1 } }' /tmp/cov.txt ;;
esac

echo "=== Endpoint smoke (8 + applicable POST) ==="
TOKEN=$(curl -s -X POST http://localhost:8081/auth/login -d '{"username":"admin","password":"admin"}' | jq -r .access_token)
for path in /health /api/system/health /api/sync/health /api/v1/source-objects /api/mapping-rules /api/v1/system/connectors /api/reconciliation/report /api/v1/masters; do
    code=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer $TOKEN" "http://localhost:8083$path")
    [[ "$code" == "200" || ("$path" == "/api/v1/system/connectors" && "$code" == "502") ]] || { echo "FAIL $path: $code"; exit 1; }
done

echo "=== /security-agent ==="
# user-invoked

echo "=== APPEND 05_progress.md ==="
echo "| $(date -u +'%Y-%m-%d %H:%M ICT') | Muscle | claude-opus-4-7 | Pillar ${PILLAR} commit ${COMMIT_HASH:-pending}: ${SUMMARY} |" \
  >> agent/memory/workspaces/feature-cdc-system-refactor/05_progress.md
```

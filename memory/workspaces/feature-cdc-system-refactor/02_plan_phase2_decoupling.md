# Phase 2 v2 — Plan — 4 Pillar CQRS Decoupling

> **Ref**: `01_requirements_phase2_decoupling.md`
> **Order**: P1 → P2 → P3 → P4 (strict, theo user mandate)

---

## Pillar P1 — Khởi tạo cấu trúc `domain/` và `app/` + Interfaces

### Mục tiêu
Tạo skeleton + interface contract. KHÔNG migrate logic. KHÔNG thay đổi handler hiện tại. Sau P1, code base build PASS, mọi endpoint vẫn chạy bằng path cũ.

### Cấu trúc tạo mới

```
internal/
├── domain/
│   ├── mapping/
│   │   ├── rule.go                  # MappingRule entity (id, source, target, status)
│   │   ├── status.go                # MappingStatus value object
│   │   └── errors.go                # ErrInvalidScope, ErrDuplicate
│   ├── source/
│   │   ├── object.go                # SourceObject entity
│   │   └── scope.go                 # Scope value object (database, table)
│   ├── master/
│   │   ├── binding.go               # MasterBinding entity
│   │   └── schema_status.go
│   ├── reconciliation/
│   │   ├── report.go                # ReconReport entity (read model)
│   │   └── drift.go                 # DriftStatus value object
│   └── job/
│       ├── job.go                   # Job entity (id, type, status, result, error)
│       └── status.go                # JobStatus enum: pending|running|success|failed
└── app/
    ├── ports/
    │   ├── repository.go            # interfaces — see below
    │   ├── command_bus.go
    │   ├── query_bus.go
    │   └── publisher.go
    ├── queries/                     # empty placeholder cho P2
    └── commands/                    # empty placeholder cho P3
```

### Interface contracts — `internal/app/ports/`

**`repository.go`**:
```go
package ports

type MappingRuleRepo interface {
    List(ctx context.Context, f mapping.Filter) ([]mapping.Rule, error)
    GetByID(ctx context.Context, id int64) (*mapping.Rule, error)
    Save(ctx context.Context, r *mapping.Rule) error
    UpdateStatus(ctx context.Context, id int64, status mapping.Status) error
}

type SourceRepo interface {
    List(ctx context.Context, f source.Filter) ([]source.Object, error)
    GetByID(ctx context.Context, id int64) (*source.Object, error)
    GetByRegistryID(ctx context.Context, rid int64) (*source.Object, error)
    Save(ctx context.Context, o *source.Object) error
    ResolveScope(ctx context.Context, id int64) (source.Scope, error)
}

type MasterRepo interface {
    List(ctx context.Context, f master.Filter) ([]master.Binding, error)
    GetByName(ctx context.Context, name string) (*master.Binding, error)
    Save(ctx context.Context, b *master.Binding) error
    UpdateSchemaStatus(ctx context.Context, name string, s master.SchemaStatus) error
}

type JobRepo interface {
    Create(ctx context.Context, j *job.Job) error
    GetByID(ctx context.Context, id string) (*job.Job, error)
    UpdateStatus(ctx context.Context, id string, s job.Status, result, errMsg string) error
    ListPending(ctx context.Context, jtype string, limit int) ([]job.Job, error)
}

type ReconReportRepo interface {
    Latest(ctx context.Context, table string) (*reconciliation.Report, error)
    List(ctx context.Context, f reconciliation.Filter) ([]reconciliation.Report, error)
}

type FailedSyncLogRepo interface {
    List(ctx context.Context, f reconciliation.LogFilter) ([]reconciliation.FailedLog, error)
    GetByID(ctx context.Context, id int64) (*reconciliation.FailedLog, error)
    UpdateStatus(ctx context.Context, id int64, status string) error
}

// ... AlertRepo, ConnectionRepo, WizardRepo, ActivityLogRepo
```

**`command_bus.go`**:
```go
package ports

type Command interface {
    Type() string                     // e.g. "master.swap"
    Validate() error
}

type CommandResult struct {
    JobID    string
    Accepted bool
    Reason   string
}

type CommandBus interface {
    // Dispatch creates a Job row (status=pending), publishes the NATS cmd
    // subject, returns the JobID. Idempotent on (Type, IdempotencyKey).
    Dispatch(ctx context.Context, c Command) (CommandResult, error)
}
```

**`query_bus.go`**:
```go
package ports

type Query interface {
    Type() string
}

type QueryBus interface {
    Ask(ctx context.Context, q Query) (any, error)
}
```

**`publisher.go`**:
```go
package ports

type Publisher interface {
    Publish(ctx context.Context, subject string, payload []byte) error
}
```

### Tasks
- T1.1 — Tạo struct directory + entity files (no logic).
- T1.2 — Define interface contract (repository.go, command_bus.go, query_bus.go, publisher.go).
- T1.3 — Tạo placeholder `infra/persistence/`, `infra/messaging/` directory.
- T1.4 — Wire `server.go` để inject port (chưa swap impl).
- T1.5 — Build PASS.

### DoD P1
- `ls internal/domain internal/app/ports internal/app/queries internal/app/commands internal/infra` returns directories.
- `go build ./...` PASS.
- 8 endpoint smoke PASS (chưa thay đổi logic, chỉ thêm skeleton).
- Effort: 2d.

---

## Pillar P2 — Migrate Read paths → `app/queries/`

### Mục tiêu
Mỗi endpoint READ trong handler → 1 query handler trong `app/queries/`. Handler API chỉ marshal HTTP, gọi `queryBus.Ask(...)`. Read paths chọn TRƯỚC vì không thay đổi state, blast radius nhỏ.

### Endpoint READ đối tượng (priority order)

| # | HTTP | Handler hiện tại | Query mới | Domain |
|---|---|---|---|---|
| 1 | `GET /api/mapping-rules` | mapping_rule_handler.go:List | `ListMappingRulesQuery` | mapping |
| 2 | `GET /api/mapping-rules/:id` | mapping_rule_handler.go:GetByID | `GetMappingRuleQuery` | mapping |
| 3 | `GET /api/v1/source-objects` | registry_handler.go:List | `ListSourceObjectsQuery` | source |
| 4 | `GET /api/v1/source-objects/:id` | registry_handler.go:GetByID | `GetSourceObjectQuery` | source |
| 5 | `GET /api/v1/masters` | master_registry_handler.go:List | `ListMastersQuery` | master |
| 6 | `GET /api/v1/masters/:name` | master_registry_handler.go:GetByName | `GetMasterByNameQuery` | master |
| 7 | `GET /api/reconciliation/report` | reconciliation_handler.go:LatestReport | `GetReconReportQuery` | reconciliation |
| 8 | `GET /api/failed-sync-logs` | reconciliation_handler.go:ListFailedLogs | `ListFailedLogsQuery` | reconciliation |
| 9 | `GET /api/sync/health` | system_health_handler.go:SyncHealth | `GetSyncHealthQuery` | reconciliation |
| 10 | `GET /api/system/health` | system_health_handler.go:Snapshot | `GetSystemHealthSnapshotQuery` | (cross) |
| 11 | `GET /api/v1/system/connectors` | connector_handler.go:List | `ListConnectorsQuery` | (cross) |
| 12 | `GET /api/wizard/sessions/:id` | wizard_handler.go:GetByID | `GetWizardSessionQuery` | wizard |
| 13 | `GET /api/alerts` | alerts_handler.go:List | `ListAlertsQuery` | alert |
| 14 | `GET /api/users` | users_handler.go:List | `ListUsersQuery` | user |
| 15 | `GET /api/admin/audit` | audit_handler.go:List | `ListAdminAuditQuery` | admin |

### Pattern per query

```go
// internal/app/queries/list_mapping_rules.go
package queries

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

### Pattern handler API thin
```go
// internal/api/mapping_rule_handler.go (sau P2)
func (h *MappingRuleHandler) List(c *fiber.Ctx) error {
    q := queries.ListMappingRulesQuery{Filter: parseFilter(c), Limit: 100}
    res, err := h.queryBus.Ask(c.Context(), q)
    if err != nil { return err }
    return c.JSON(res)
}
```

### Tasks
- T2.1 — `ListMappingRules` query + repo impl tạm (delegate raw SQL hiện tại).
- T2.2 — `ListSourceObjects` + `GetSourceObject`.
- T2.3 — `ListMasters` + `GetMasterByName`.
- T2.4 — `GetReconReport` + `ListFailedLogs`.
- T2.5 — `GetSyncHealth` + `GetSystemHealthSnapshot`.
- T2.6 — `ListConnectors`.
- T2.7 — `GetWizardSession`, `ListAlerts`, `ListUsers`, `ListAdminAudit`.
- T2.8 — Test mỗi query handler qua sqlmock.

### DoD P2
- `grep -r "h.db.Raw\|h.db.Exec" internal/api/*Handler*Get*\|*List*` = 0 trong 15 endpoint READ.
- 15 endpoint smoke PASS với token thật.
- Test coverage `app/queries/` ≥ 60%.
- Effort: 3d.

---

## Pillar P3 — Migrate Action → `app/commands/` + NATS Command Dispatcher

### Mục tiêu
1. Mỗi endpoint WRITE trong handler → 1 command handler trong `app/commands/`. Handler API gọi `commandBus.Dispatch(...)` → 202 Accepted + `job_id`.
2. Command Bus implement: tạo Job row (`cdc_system.cdc_jobs`) + publish NATS cmd subject + return job_id.
3. Worker side: subscribe các cmd subject mới (master-swap, v2-sync), publish `cdc.evt.X.completed`. JobMonitor (đã có cho transmute) extend để cover các evt mới.
4. Move 2 INLINE heavy:
   - **Master Swap** (master_registry_handler.go:630) → `cdc.cmd.master-swap`.
   - **V2 SyncFromLegacy** (gọi inline tại registry_handler.go:148/268/316) → `cdc.cmd.v2-sync`.

### Migration table — Commands

| # | HTTP | Handler hiện tại | Command mới | NATS cmd | NATS evt | Job tracking |
|---|---|---|---|---|---|---|
| 1 | `POST /api/v1/source-objects` (Register) | registry_handler.go:Register | `RegisterSourceCommand` | (existing) `cdc.cmd.create-default-columns` | NEW `cdc.evt.create-default-columns.completed` | jobs.type=`source.register` |
| 2 | `PUT /api/v1/source-objects/:id` (Update) | registry_handler.go:Update | `UpdateSourceCommand` | (existing) | NEW evt | |
| 3 | `POST /api/v1/source-objects/bulk` (BulkRegister) | registry_handler.go:BulkRegister | `BulkRegisterSourceCommand` | (existing) | NEW evt | |
| 4 | `POST /api/source-objects/:id/sync` (V2 Sync — INLINE) | registry_handler.go:148 inline | `V2SyncCommand` | **NEW** `cdc.cmd.v2-sync` | **NEW** `cdc.evt.v2-sync.completed` | jobs.type=`source.v2-sync` |
| 5 | `POST /api/source-objects/:id/standardize` | source_object_actions_handler.go:Standardize | `StandardizeFieldsCommand` | (existing) `cdc.cmd.standardize` | **NEW** `cdc.evt.standardize.completed` | jobs.type=`source.standardize` |
| 6 | `POST /api/source-objects/:id/scan-fields` | scan-fields handler | `ScanFieldsCommand` | (existing) `cdc.cmd.scan-fields` | **NEW** `cdc.evt.scan-fields.completed` | |
| 7 | `POST /api/source-objects/:id/detect-timestamp` | DetectTimestampFieldV2 | `DetectTimestampCommand` | (existing) | **NEW** evt | |
| 8 | `POST /api/source-objects/:id/create-default-columns` | CreateDefaultColumns | `CreateDefaultColumnsCommand` | (existing) | **NEW** evt | |
| 9 | `POST /api/mapping-rules` | mapping_rule_handler.go:Create | `CreateMappingRuleCommand` | (none — pure metadata) | (none) | sync write |
| 10 | `PATCH /api/mapping-rules/:id` | mapping_rule_handler.go:Update | `UpdateMappingRuleCommand` | (none) | (none) | sync write |
| 11 | `POST /api/mapping-rules/batch-update` | mapping_rule_handler.go:BatchUpdate | `BatchUpdateMappingRuleCommand` | (existing) `cdc.cmd.alter-column` + `cdc.cmd.backfill` | **NEW** evt cho từng cái | jobs.type=`mapping.batch-update` (parent) + child |
| 12 | `POST /api/mapping-rules/:id/backfill` | mapping_rule_handler.go:Backfill | `BackfillMappingRuleCommand` | (existing) | **NEW** evt | |
| 13 | `POST /api/v1/masters` (Create) | master_registry_handler.go:Create | `CreateMasterCommand` | (none — pure metadata) | (none) | sync |
| 14 | `POST /api/v1/masters/:name/approve` | master_registry_handler.go:Approve | `ApproveMasterCommand` | (existing) `cdc.cmd.master-create` | **NEW** evt (đã 1 phần qua provisioning step_completed — chuẩn hóa) | jobs.type=`master.create` |
| 15 | `POST /api/v1/masters/:name/swap` (INLINE → MOVE) | master_registry_handler.go:630 inline | `MasterSwapCommand` | **NEW** `cdc.cmd.master-swap` | **NEW** `cdc.evt.master-swap.completed` | jobs.type=`master.swap` |
| 16 | `POST /api/v1/masters/:name/reject` | Reject | `RejectMasterCommand` | (none) | (none) | sync |
| 17 | `POST /api/reconciliation/check` | TriggerCheck | `TriggerReconCheckCommand` | (existing) `cdc.cmd.recon-check` | **NEW** `cdc.evt.recon-check.completed` | jobs.type=`recon.check` |
| 18 | `POST /api/reconciliation/heal` | TriggerHeal | `TriggerReconHealCommand` | (existing) | **NEW** evt | |
| 19 | `POST /api/failed-sync-logs/:id/retry` | RetryFailedLog | `RetryFailedLogCommand` | (existing) `cdc.cmd.retry-failed` | **NEW** evt | jobs.type=`recon.retry` |
| 20 | `POST /api/recon/backfill-source-ts` | TriggerBackfillSourceTs | `BackfillSourceTsCommand` | (existing) | (existing recon_runs.status — reuse) | reuse |
| 21 | `POST /api/transmute-schedules/:id/run` | transmute_schedule_handler:Run | `RunTransmuteCommand` | (existing) `cdc.cmd.transmute` | (existing) `cdc.evt.transmute.completed` ✅ | reuse `transmute_schedule.last_status` |
| 22 | `POST /api/connector-ops/restart-debezium` | system_health_handler.go:128 | `RestartDebeziumCommand` | (existing) | **NEW** evt | |
| 23 | `POST /api/wizard/sessions` (Create + Patch) | wizard_handler.go | `CreateWizardCommand` + `PatchWizardCommand` | (none — metadata) | (none) | sync |
| 24 | `POST /api/alerts/:id/ack` | alerts_handler.go:Ack | `AckAlertCommand` | (none) | (none) | sync |

### Tổng kết:
- **Sync metadata write** (no NATS): 7 command (mapping create/update, master create/reject, wizard create/patch, alert ack)
- **Async NATS cmd** existing + companion event: 14 command
- **Async NATS cmd** mới hoàn toàn (cmd subject MỚI): 2 command (master-swap, v2-sync)
- **Companion evt subject** cần thêm: 12 (vì transmute và provisioning đã có)

### `cdc_system.cdc_jobs` — DDL mới (P3)

```sql
-- Phải coordinate với centralized-data-service/migrations/
CREATE TABLE IF NOT EXISTS cdc_system.cdc_jobs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    type            TEXT NOT NULL,           -- e.g. "master.swap", "recon.check"
    status          TEXT NOT NULL DEFAULT 'pending',  -- pending|running|success|failed
    payload         JSONB NOT NULL,
    result          JSONB,
    error_message   TEXT,
    idempotency_key TEXT UNIQUE,             -- optional dedup
    created_by      TEXT NOT NULL,
    correlation_id  TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    started_at      TIMESTAMPTZ,
    finished_at     TIMESTAMPTZ
);
CREATE INDEX idx_cdc_jobs_type_status ON cdc_system.cdc_jobs(type, status, created_at DESC);
CREATE INDEX idx_cdc_jobs_correlation ON cdc_system.cdc_jobs(correlation_id);
```

### `NATSCommandBus` impl — `internal/infra/messaging/nats_command_bus.go`

```go
type natsCommandBus struct {
    nc      *nats.Conn
    jobRepo ports.JobRepo
    log     *zap.Logger
}

func (b *natsCommandBus) Dispatch(ctx context.Context, c ports.Command) (ports.CommandResult, error) {
    if err := c.Validate(); err != nil {
        return ports.CommandResult{}, err
    }
    payload, _ := json.Marshal(c)
    j := &job.Job{
        ID:           uuid.NewString(),
        Type:         c.Type(),
        Status:       job.StatusPending,
        Payload:      payload,
        CreatedBy:    auth.UserFrom(ctx),
        CorrelationID: middleware.CorrIDFrom(ctx),
    }
    if err := b.jobRepo.Create(ctx, j); err != nil {
        return ports.CommandResult{}, err
    }
    // map command type → NATS subject
    subject := mapTypeToSubject(c.Type())
    natsPayload := buildNATSPayload(c, j.ID)  // include job_id
    if err := b.nc.Publish(subject, natsPayload); err != nil {
        // KHÔNG rollback job — let JobMonitor flag "stuck" job (recovery cron)
        b.log.Warn("publish failed", zap.String("subject", subject), zap.Error(err))
        return ports.CommandResult{}, err
    }
    return ports.CommandResult{JobID: j.ID, Accepted: true}, nil
}
```

### Tasks
- T3.1 — Migration `cdc_system.cdc_jobs` (coordinate centralized-data-service/migrations).
- T3.2 — `JobRepo` impl + integration test.
- T3.3 — `NATSCommandBus` impl + unit test.
- T3.4 — Migrate 7 sync metadata command (mapping/master/wizard/alert) → `app/commands/`.
- T3.5 — Migrate 14 existing-NATS command → wrap qua `commandBus.Dispatch`.
- T3.6 — Worker: thêm subscribe `cdc.cmd.master-swap` + handler ALTER RENAME + emit `cdc.evt.master-swap.completed`.
- T3.7 — Worker: thêm subscribe `cdc.cmd.v2-sync` + handler UPSERT + emit evt.
- T3.8 — Worker: extend mọi cmd handler hiện có → emit `cdc.evt.X.completed` (12 subject).
- T3.9 — Worker: extend `JobMonitor` subscribe 12 evt subject mới + UPDATE `cdc_system.cdc_jobs.status`.
- T3.10 — CMS: thêm `GET /api/jobs/:id` endpoint qua `GetJobQuery`.
- T3.11 — Test action smoke A1-A3 (recon check + job status + master swap async).

### DoD P3
- `grep "ALTER TABLE.*RENAME" internal/api/ internal/app/` = 0.
- `grep "SyncFromLegacy" internal/api/ internal/app/` = 0 (chỉ ở cmd handler ở worker).
- `cdc_system.cdc_jobs` table created + populated khi POST action.
- `GET /api/jobs/:job_id` returns status.
- 12 new evt subject + JobMonitor subscribe verify qua `nats stream ls` + log.
- Effort: 5d (worker + CMS).

---

## Pillar P4 — Chuẩn hóa `infra/persistence/` — eliminate raw SQL

### Mục tiêu
Sau P2+P3, mọi caller chỉ dùng `ports.XRepo`. P4 đảm bảo: implementation trong `internal/infra/persistence/` là NƠI DUY NHẤT có raw SQL. Các file `internal/api/`, `internal/app/`, `internal/domain/` không còn `db.Raw()` / `db.Exec()`.

### Audit & migrate

```bash
# Phase 0 audit
grep -rn "db.Raw\|db.Exec\|db.Where\|.Find(" internal/api/ internal/app/ internal/domain/
# Mục tiêu: 0 hit sau P4
```

### File cleanup

| File | Action |
|---|---|
| `internal/api/*.go` (handler) | Loại bỏ tất cả `*gorm.DB` field, chỉ giữ `queryBus`, `commandBus` |
| `internal/service/*.go` (16 file) | Hoặc move sang `app/commands/` (use case) hoặc `infra/persistence/` (DB query) — KHÔNG tồn tại layer trung gian sau P4 |
| `internal/repository/*.go` (existing — chỉ V1) | Move sang `internal/infra/persistence/` + rename theo aggregate |

### Tasks
- T4.1 — `MappingRuleRepo` GORM impl — extract toàn bộ SQL từ `mapping_rule_handler.go`.
- T4.2 — `SourceRepo` GORM impl — extract từ `registry_handler.go` + `source_object_actions_handler.go`.
- T4.3 — `MasterRepo` GORM impl — extract từ `master_registry_handler.go`.
- T4.4 — `JobRepo` GORM impl (đã làm phần lớn ở P3, hoàn thiện).
- T4.5 — `ReconReportRepo` + `FailedSyncLogRepo` — extract từ `reconciliation_handler.go`.
- T4.6 — `AlertRepo`, `WizardRepo`, `ConnectionRepo`, `ActivityLogRepo`, `AdminActionRepo`.
- T4.7 — Xóa `internal/service/*.go` cũ (đã migrate).
- T4.8 — `pkgs/utils/pg_ident.go` — move helper `pgIdent` từ `reconciliation_handler.go`.
- T4.9 — Test coverage `infra/persistence/` ≥ 50% qua sqlmock.

### DoD P4
- `grep -rn "db.Raw\|db.Exec" internal/api/ internal/app/ internal/domain/` = 0.
- `wc -l internal/api/*.go` mọi file ≤100.
- `wc -l internal/service/` empty (folder removed).
- 8 endpoint smoke + 3 action smoke PASS.
- Test coverage `internal/infra/persistence/` ≥ 50%.
- Effort: 3d.

---

## Decision log

### Q1: Tạo bảng `cdc_jobs` thống nhất hay reuse per-domain status column?
**Quyết định**: Tạo `cdc_system.cdc_jobs` MỚI. Per-domain column (master_binding.schema_status, transmute_schedule.last_status, recon_runs.status, failed_sync_logs.status) **giữ nguyên** — dùng để track domain-specific state (e.g., master "approved/rejected/failed"). `cdc_jobs` track ASYNC OPERATION state (pending/running/success/failed) — orthogonal concern.
**Lý do**: 
- Ít blast radius — không migrate dữ liệu cũ.
- Per-domain status là semantic khác (domain state machine), khác với job state (execution).
- `job_id` mapping trỏ về row trong `cdc_jobs` cho generic query "đang có job nào treo?".

### Q2: Move INLINE Master Swap sang worker — có cần TX cross-DB không?
**Quyết định**: KHÔNG. Worker chạy ALTER RENAME chỉ trên 1 DB (master DB), không cần distributed TX. CMS chỉ ghi `cdc_jobs.status='pending'` trước publish — nếu worker fail → JobMonitor flag failed, CMS có thể retry qua POST lại endpoint (mới tạo job mới).
**Lý do**: KISS (Lesson #160). Distributed TX phức tạp hơn benefit.

### Q3: Pillar P2 (Read) trước P3 (Write) — sequential bắt buộc?
**Quyết định**: Sequential bắt buộc. Read interface định hình QueryBus pattern; Write reuse same pattern + thêm CommandBus + Job tracking. Nếu làm song song → 2 stream code đụng cùng repo file (race conflict).
**Lý do**: User mandate explicit thứ tự P1→P2→P3→P4.

### Q4: Worker JobMonitor mở rộng cho 12 evt mới — 1 subscription per subject hay wildcard?
**Quyết định**: Wildcard `cdc.evt.*.completed`. Single handler `HandleCompleted(msg)` parse `subject` để route logic. Đỡ tạo 12 subscription riêng.
**Lý do**: Lesson 2026-04-29 line 845 — broker buffers; wildcard subscribe consistent + dễ extend.

### Q5: `internal/service/` có tồn tại sau P4 không?
**Quyết định**: KHÔNG. Mọi service hiện tại di cư:
- Use case logic → `app/commands/` (write) hoặc `app/queries/` (read).
- DB query → `infra/persistence/`.
- External call (Kafka Connect REST, NATS publish) → `infra/http/` hoặc `infra/messaging/`.
- Domain validation → `domain/X/`.
**Lý do**: Clean Architecture nguyên bản — không có "service layer" trung gian giữa app và infra.

### Q6: Phase 2 v1 (`phase2_cms_refactor`) có execute không?
**Quyết định**: KHÔNG. v1 superseded bởi v2. File `phase2_cms_refactor.md` GIỮ NGUYÊN làm lịch sử (CLAUDE.md §11). Pillar P5 (health collector probe split) v1 → integrate vào P4 v2 (move toàn bộ probe sang `infra/health/probes/`). Pillar P6 (V2 sync atomicity) v1 → giải quyết bởi P3 v2 (move sang worker).
**Lý do**: User mandate 4 pillar mới rõ ràng.

---

## Risk matrix

| Pillar | Top risk | Mitigation |
|---|---|---|
| P1 | Tạo skeleton sai pattern, P2-P4 sửa lại | Code review pillar đầu kỹ; ref Clean Architecture book |
| P2 | Read query miss filter / pagination so handler cũ | Test sqlmock so sánh result với handler cũ qua snapshot |
| P3 | Worker handler chưa kịp deploy → cmd lost | NATS JetStream retention 7d (có sẵn); deploy worker TRƯỚC CMS |
| P3 | `cdc_jobs` migration race với worker subscribe trước table tồn tại | Migration apply TRƯỚC khi rebuild + deploy worker |
| P3 | Companion event flooding NATS (12 cmd × every action) | Đã có JetStream backpressure; max_msgs limit |
| P4 | Service layer xóa nhưng còn external dependency import path | grep + auto fix import; build PASS gate |

## Test strategy

| Layer | Test type | Tool |
|---|---|---|
| `domain/` | Unit table-driven | `testing` |
| `app/queries/` | Unit + sqlmock | `testing` + `sqlmock` |
| `app/commands/` | Unit + mock CommandBus | `testing` + interface mock |
| `infra/persistence/` | Integration (real DB) | `testing` + `dockertest` (optional) |
| `infra/messaging/` | Integration NATS | `testing` + embedded NATS server |
| End-to-end | Smoke 8+3 endpoint | `curl` + scripted token |

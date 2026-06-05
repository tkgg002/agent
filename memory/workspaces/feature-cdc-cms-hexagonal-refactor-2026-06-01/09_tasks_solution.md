# 09_tasks_solution.md — Hồ sơ giải pháp kỹ thuật

> Doc này định danh **giải pháp cụ thể** cho từng Gap → tham chiếu từ `02_plan.md` (roadmap) và `03_implementation.md` (code demo).

## 1. Map Gap → Solution

| Gap | Solution | Phase | Doc tham chiếu |
|---|---|---|---|
| A. God Interface | **S1**: Port-per-aggregate, 2-4 interface hẹp ngữ nghĩa mỗi file | Phase 1 | §3 + ADR-01 |
| B. 18 commands raw gorm | **S2**: Wrap raw SQL thành port hẹp, command chỉ phụ thuộc port | Phase 3 | §5 + ADR-02 |
| C. Composition root phình | **S3**: Pure-function split-file (KHÔNG receiver-state) | Phase 2 | §4 + ADR-03 |
| D. Flat layer | **S4** (optional): Vertical Slice theo 8 BC + `platform/` cross-cutting | Phase 4 | §6 + ADR-05 |
| E. Bootstrap 0 test | **S5**: Test trước refactor, target ≥ 70% coverage `internal/bootstrap/` | Phase 0 | §2 |
| F. Naming chồng model/domain | **S6**: Move `model/` về BC tương ứng trong Phase 4 | Phase 4 | §6 |
| G. Audit logic xen lẫn | **S7**: Tách middleware → `platform/audit_mw/`, domain query → `bc/observability/` | Phase 4 | §6 |

---

## 2. Bounded Context map chi tiết (8 BC)

> Đây là **kết quả gom 18 chức năng** đã list ở session trước. Mỗi BC = 1 lý do thay đổi (cohesion theo Conway).

### 2.1 `source` BC
- **Chức năng**: Discovery + Registration + Provisioning Lifecycle.
- **Domain**: `Object`, `Scope`, `ProvisioningState` (registered/provisioned/active/disabled/failed).
- **Commands gom vào** (8):
  - `register_registry.go`
  - `bulk_register_registry.go`
  - `update_registry.go`
  - `update_source_object_v2.go`
  - `update_shadow_binding.go`
  - `v2_sync.go`
  - `source_async.go`
  - (phần "standardize/scan-fields/create-default-columns/detect-timestamp-field" hiện ở handler — sẽ thành command)
- **Queries gom vào** (6): `list_source_objects.go`, `list_sources.go`, `get_source_object_mapping_context.go`, `source_object_reader.go`, `source_objects_read_models.go`, `bridge_status_reader.go`
- **API handler**: `sources_handler.go`, `source_objects_handler.go`, `source_object_actions_handler.go`, `shadow_binding_actions_handler.go`, `registry_handler*.go`
- **Infra**: `source_object_read_repo_gorm.go`, `source_object_v2_sync.go`, `shadow_automator.go`, `provisioning_orchestrator.go`, `provisioning_state_machine.go`, `system_connector_repo_gorm.go`
- **NATS subjects sở hữu**: `cdc.cmd.discover`, `.introspect.*`, `.scan-fields`, `.scan-raw-data`, `.standardize`, `.create-default-columns`, `.detect-timestamp-field`

### 2.2 `mapping` BC
- **Chức năng**: Mapping Rule CRUD + Schema Drift Approval.
- **Domain**: `Rule`, `Status`, `PendingField`, `SchemaChangeLog`, `errors`.
- **Commands** (4): `create_mapping_rule.go`, `update_mapping_rule.go`, `approve_schema_proposal.go`, `reject_schema_proposal.go`.
- **Queries** (4): `list_mapping_rules.go`, `resolve_mapping_scope.go`.
- **API**: `mapping_rule_handler*.go`, `mapping_preview_handler.go`, `schema_change_handler.go`, `schema_proposal_handler.go`.
- **Infra**: `mapping_rule_repo_gorm.go`, `schema_log_repo_gorm.go`, `pending_field_repo_gorm.go`, `approval_service.go`.
- **NATS subjects sở hữu**: `cdc.cmd.alter-column`, `.backfill`.

### 2.3 `master` BC
- **Chức năng**: Master Binding CRUD + Approve + Swap.
- **Domain**: `Binding`, `SchemaStatus` (draft/approved/rejected).
- **Commands** (5): `create_master.go`, `approve_master.go`, `reject_master.go`, `master_swap.go`, `toggle_master_active.go`.
- **Queries** (1): `list_masters.go`.
- **API**: `master_registry_handler_*.go` (5 file split-verb).
- **Infra**: `master_read_repo_gorm.go`, `master_swap.go`.
- **NATS subjects sở hữu**: `cdc.cmd.master-create`, `.master-swap`, `.master.bind`.

### 2.4 `transform` BC (transmute + snapshot + backfill)
- **Chức năng**: Transmute Schedule + Snapshot V2 + Backfill.
- **Domain**: `Schedule`, `SnapshotProgress`, `BackfillPlan`.
- **Commands** (4): `create_schedule.go`, `update_schedule.go`, `toggle_schedule.go`, `transmute_run.go`.
- **Queries** (3): `list_transmute_schedules.go`, `list_snapshot_progress.go`.
- **API**: `transmute_schedule_handler.go`, `snapshot_progress_handler.go`, `schedule_handler.go`.
- **Infra**: `transmute_schedule_read_repo_gorm.go`, `snapshot_progress_read_repo_gorm.go`.
- **NATS subjects sở hữu**: `cdc.cmd.transmute`, `.batch-transform`, `.snapshot.v2`, `.schedule.enable`.

### 2.5 `reconciliation` BC
- **Chức năng**: Recon check + Heal + Failed Log.
- **Domain**: `Report`, `FailedLog`, `FailedLogStatus`.
- **Commands** (3): `recon_check.go`, `recon_async.go`, `mark_failed_log_retrying.go`.
- **Queries** (4): `list_latest_reports.go`, `get_table_history.go`, `list_failed_logs.go`, `list_gaps.go`, `recon_enrichment.go`, `recon_read_models.go`, `recon_reader.go`.
- **API**: `reconciliation_handler*.go` (6 file split-verb).
- **Infra**: `recon_read_repo_gorm.go`.
- **NATS subjects sở hữu**: `cdc.cmd.recon-check`, `.recon-heal`, `.recon-backfill-source-ts`, `.retry-failed`.

### 2.6 `wizard` BC (cross-BC saga)
- **Chức năng**: Source→Master automation 11-step.
- **Domain**: `WizardSession`, `Step`.
- **Commands** (3): `create_wizard.go`, `patch_wizard.go`, `wizard_execute.go`.
- **Queries** (2): `get_wizard_session.go`, `get_wizard_progress.go`.
- **API**: `wizard_handler.go`.
- **Infra**: `wizard_repo_gorm.go`.
- **Đặc biệt**: BC duy nhất được phép import port của BC khác (`source`, `mapping`, `master`) — Saga pattern.

### 2.7 `system_control` BC
- **Chức năng**: Kafka Connect / Debezium connector lifecycle + Worker Schedule.
- **Domain**: `ConnectorState`, `WorkerScheduleSpec`.
- **Commands** (4): `system_connector.go`, `system_async.go`, `create_worker_schedule.go` (+ debezium signal/snapshot/restart commands).
- **Queries** (3): `list_connectors.go`, `list_worker_schedules.go`.
- **API**: `system_connectors_handler.go`, `schedule_handler.go`.
- **Infra**: `kafka_connect.go` (HTTP client), `system_connector_repo_gorm.go`, `worker_schedule_read_repo_gorm.go`.
- **NATS subjects sở hữu**: `cdc.cmd.debezium-signal`, `.debezium-snapshot`, `.restart-debezium`.

### 2.8 `observability` BC
- **Chức năng**: System Health + Alerts + Activity Log + Audit/QA.
- **Domain**: `Alert`, `HealthSnapshot`, `ActivityEntry`.
- **Commands** (2): `ack_alert.go`, `silence_alert.go`.
- **Queries** (5): `get_qa_summary.go`, `list_gaps.go`, `get_metric_health.go`, `list_activity_logs.go`, `get_activity_stats.go`, `get_sync_health.go`, `activity_log_read_models.go`.
- **API**: `health_handler.go`, `system_health_handler.go`, `alerts_handler.go`, `audit_handler.go`, `activity_log_handler.go`, `action_trace.go`.
- **Infra**: `alert_manager.go`, `activity_logger.go`, `system_health_collector.go`, `system_health_compute.go`, `system_health_alerts.go`, `system_health_queries.go`, `probes/*.go`, `sync_health_read_repo_gorm.go`, `activity_log_read_repo_gorm.go`.

### 2.9 KHÔNG phải BC — chuyển sang `platform/`
- `Job` (cdc_jobs row + lifecycle) → `platform/bus/` (backbone của CommandBus).
- `Auth/JWT/RBAC` → `platform/auth/`.
- `Idempotency/RateLimit` → `platform/ratelimit/`.
- `Audit middleware` (chỉ middleware tier — domain query thì ở `bc/observability/`) → `platform/audit_mw/`.
- `Deprecation middleware` → `platform/deprecation/`.
- `OTel + Prom client` → `platform/observability/`.

### 2.10 KHÔNG phải BC — chuyển sang `shared/`
- `naming/` (canonical schema/topic naming) — cross-BC, KHÔNG domain.
- `pg/` (pg_ident, type_inference) — technical primitives.
- `id/` (Sonyflake wrapper, hash) — technical primitives.

---

## 3. S1 — Port split (Phase 1) chi tiết

### 3.1 Strategy
- Mỗi aggregate có 1 file port: `internal/app/ports/<aggregate>_port.go`.
- Interface theo **vai trò** (Reader / Writer / Approver / Swapper / Resolver), KHÔNG theo CRUD.
- Mỗi interface ≤ 4 method.

### 3.2 Ví dụ split `MasterRepo`
**Trước** (`ports/repository.go`):
```go
type MasterRepo interface {
    List(ctx context.Context, f master.Filter) ([]master.Binding, error)
    GetByName(ctx context.Context, name string) (*master.Binding, error)
    Save(ctx context.Context, b *master.Binding) error
    UpdateSchemaStatus(ctx context.Context, name string, s master.SchemaStatus) error
}
```

**Sau** (`ports/master_port.go`):
```go
type MasterReader interface {
    List(ctx context.Context, f master.Filter) ([]master.Binding, error)
    GetByName(ctx context.Context, name string) (*master.Binding, error)
}

type MasterWriter interface {
    Save(ctx context.Context, b *master.Binding) error
}

type MasterApprover interface {
    UpdateSchemaStatus(ctx context.Context, name string, s master.SchemaStatus) error
}
```

### 3.3 Migration step (per aggregate)
1. Tạo file port mới với interface hẹp.
2. Implement của persistence vẫn struct cũ — chỉ thêm các method (đã có sẵn).
3. Update từng handler/command/query: đổi type annotation từ `ports.MasterRepo` → `ports.MasterReader` (hoặc Writer / Approver tương ứng).
4. Build + test sau mỗi handler.
5. Khi 0 reference legacy → xóa `MasterRepo` cũ khỏi `repository.go`.
6. Khi `repository.go` empty → DELETE file.

### 3.4 Số lượng port file dự kiến
| File | Interface count | LOC ước |
|---|---|---|
| `master_port.go` | 3 (Reader, Writer, Approver) | ~30 |
| `mapping_port.go` | 4 (Reader, Writer, BatchUpdater, Previewer) | ~40 |
| `source_port.go` | 3 (Reader, Writer, Resolver) | ~30 |
| `job_port.go` | 4 (Creator, Reader, Updater, ListPending) | ~30 |
| `recon_port.go` | 3 (ReportReader, FailedLogReader, FailedLogUpdater) | ~25 |
| `schema_port.go` | 4 (SchemaLogCreator, SchemaLogReader, PendingFieldReader, PendingFieldUpdater) | ~30 |
| `wizard_port.go` | 3 (Creator, Reader, ProgressAppender) | ~25 |
| `system_connector_port.go` | 3 (Upserter, Reader, Cleaner) | ~30 |
| `registry_port.go` | 2 (Reader, StatsReader) | ~20 |

**Total**: ~260 LOC tách thành 9 file thay vì 151 LOC trong 1 file God.

---

## 4. S3 — Composition Root pure-function (Phase 2) chi tiết

### 4.1 Anti-pattern phải tránh
```go
// ❌ ANTI-PATTERN: receiver state mutation
func (s *Server) setupInfrastructure() {
    s.db, _ = database.NewPostgresConnection(s.cfg.DB)
    s.nats, _ = natsconn.NewNatsClient(s.cfg, s.logger)
    // ... mutate s
}
```
**Lý do reject**: hidden state coupling, order phụ thuộc nhưng không nhìn thấy.

### 4.2 Pattern đúng
```go
// ✅ PURE FUNCTION: input/output explicit
type Infra struct {
    DB       *gorm.DB
    ShadowDB *gorm.DB
    NATS     *natsconn.NatsClient
    Redis    *rediscache.RedisCache
}

func buildInfra(cfg *config.AppConfig, logger *zap.Logger) (*Infra, error) {
    db, err := database.NewPostgresConnection(cfg.DB)
    if err != nil { return nil, fmt.Errorf("postgres: %w", err) }
    // ...
    return &Infra{DB: db, ShadowDB: shadowDB, NATS: nats, Redis: redis}, nil
}
```

### 4.3 File breakdown
| File | Responsibility | LOC ước |
|---|---|---|
| `server.go` | `New()` orchestrate + `Start()` + `Shutdown()` | ≤ 80 |
| `infra.go` | `buildInfra()` + bootstrap data sync | ~70 |
| `repos.go` | `buildRepos()` + `Repos` struct typed | ~80 |
| `bus.go` | `buildCommandBus()` + `registerCommandHandlers()` (100 dòng cũ) | ~150 |
| `routes.go` | `registerRoutes()` + `Handlers` struct | ~100 |
| `workers.go` | `buildBackgroundWorkers()` + start/stop helper | ~50 |

**Total**: ~530 LOC tách thành 6 file thay vì 333 LOC trong 1 file. Mỗi file 1 trách nhiệm.

---

## 5. S2 — Refactor 18 commands raw gorm (Phase 3) chi tiết

### 5.1 Pattern refactor
**Trước** (vd `approve_master.go`):
```go
type ApproveMasterHandler struct {
    db   *gorm.DB
    nats *natsconn.NatsClient
    log  *zap.Logger
}

func (h *ApproveMasterHandler) Handle(...) {
    // raw SQL via db.Exec / db.Model().Updates() ...
}
```

**Sau**:
```go
type ApproveMasterHandler struct {
    approver MasterApprover     // interface hẹp từ ports/master_port.go
    pub      bus.Publisher
    log      *zap.Logger
}

func (h *ApproveMasterHandler) Handle(...) {
    if err := h.approver.Approve(ctx, name, approverEmail); err != nil {...}
    return h.pub.Publish(ctx, "cdc.cmd.master-create", payload)
}
```

### 5.2 Thứ tự refactor (18 commands, order theo độ phức tạp)
| Order | Command | Độ phức tạp | Port mới |
|---|---|---|---|
| 1 | `mark_failed_log_retrying.go` | Trivial | `FailedLogUpdater` |
| 2 | `toggle_master_active.go` | Low | `MasterToggler` |
| 3 | `toggle_schedule.go` | Low | `ScheduleToggler` |
| 4 | `reject_master.go` | Low | `MasterRejecter` |
| 5 | `reject_schema_proposal.go` | Low | `SchemaProposalRejecter` |
| 6 | `create_worker_schedule.go` | Low | `WorkerScheduleCreator` |
| 7 | `update_shadow_binding.go` | Med | `ShadowBindingUpdater` |
| 8 | `update_schedule.go` | Med | `ScheduleUpdater` |
| 9 | `update_source_object_v2.go` | Med | `SourceObjectV2Updater` |
| 10 | `create_master.go` | Med | `MasterWriter` |
| 11 | `create_schedule.go` | Med | `ScheduleWriter` |
| 12 | `update_registry.go` | Med | `RegistryUpdater` |
| 13 | `create_mapping_rule.go` | Med | `MappingRuleWriter` |
| 14 | `update_mapping_rule.go` | Med | `MappingRuleUpdater` |
| 15 | `register_registry.go` | High | `RegistryRegister` + saga deps |
| 16 | `approve_master.go` | High | `MasterApprover` + Publisher |
| 17 | `approve_schema_proposal.go` | High | `SchemaApprover` + ApprovalService |
| 18 | `bulk_register_registry.go` | High | `RegistryBulkRegister` (cross-aggregate) |

**Rule**: 1 commit / 1 command. Stop khi 3 commit fail liên tiếp (rule §8).

---

## 6. S4 — Vertical Slice (Phase 4 OPTIONAL) chi tiết

### 6.1 Target tree (đầy đủ)
```
internal/
├── server/                # composition root (đã split ở Phase 2)
│   ├── server.go
│   ├── infra.go
│   ├── repos.go
│   ├── bus.go
│   ├── routes.go
│   └── workers.go
│
├── platform/              # 🆕 cross-cutting
│   ├── bus/
│   │   ├── command.go     # ← ports/command_bus.go
│   │   ├── nats_bus.go    # ← infra/messaging/nats_command_bus.go
│   │   ├── job.go         # ← domain/job/job.go
│   │   └── reaper.go      # ← infra/messaging/stuck_job_reaper.go
│   ├── auth/              # ← middleware/jwt.go + rbac.go
│   ├── ratelimit/         # ← middleware/idempotency.go + ratelimit.go
│   ├── audit_mw/          # ← middleware/audit.go
│   ├── deprecation/       # ← middleware/deprecation.go
│   └── observability/
│       └── prom_client.go # ← infra/http/prom_client.go
│
├── bc/                    # 🆕 bounded contexts
│   ├── source/
│   ├── mapping/
│   ├── master/
│   ├── transform/
│   ├── reconciliation/
│   ├── wizard/
│   ├── system_control/
│   └── observability/
│
├── shared/                # 🆕 technical primitives (KHÔNG domain)
│   ├── naming/            # ← internal/naming/
│   ├── pg/                # ← pkgs/utils/pg_ident + type_inference
│   └── id/                # ← pkgs/utils/hash
│
├── bootstrap/             # giữ nguyên — init data sync
└── migrate/               # giữ nguyên
```

### 6.2 Mỗi BC structure
```
bc/master/
├── domain/                # pure Go aggregate
│   ├── binding.go         # ← internal/domain/master/binding.go
│   └── status.go          # SchemaStatus enum
├── ports.go               # MasterReader, Writer, Approver, Swapper, Toggler
├── commands/              # ← cherry-pick từ internal/app/commands/*.go
│   ├── create.go
│   ├── approve.go
│   ├── reject.go
│   ├── swap.go
│   └── toggle.go
├── queries/               # ← cherry-pick từ internal/app/queries/*.go
│   └── list.go
├── infra/                 # ← cherry-pick từ internal/infra/persistence/*.go
│   ├── master_read_repo_gorm.go
│   └── master_swap.go
└── api/                   # ← cherry-pick từ internal/api/master_registry_handler_*.go
    ├── handler.go
    ├── handler_create.go
    ├── handler_approve.go
    ├── handler_swap.go
    └── handler_toggle.go
```

### 6.3 Linter rule
File `.go-arch-lint.yml`:
```yaml
version: 1
allow:
  depOnAnyVendor: true

components:
  source_bc:    { in: internal/bc/source/** }
  mapping_bc:   { in: internal/bc/mapping/** }
  master_bc:    { in: internal/bc/master/** }
  transform_bc: { in: internal/bc/transform/** }
  recon_bc:     { in: internal/bc/reconciliation/** }
  wizard_bc:    { in: internal/bc/wizard/** }
  system_bc:    { in: internal/bc/system_control/** }
  obs_bc:       { in: internal/bc/observability/** }
  platform:     { in: internal/platform/** }
  shared:       { in: internal/shared/** }
  server:       { in: internal/server/** }
  persistence:  { in: internal/bc/*/infra/** }

deps:
  source_bc:    { mayDependOn: [platform, shared] }
  mapping_bc:   { mayDependOn: [platform, shared] }
  master_bc:    { mayDependOn: [platform, shared] }
  transform_bc: { mayDependOn: [platform, shared] }
  recon_bc:     { mayDependOn: [platform, shared] }
  wizard_bc:    { mayDependOn: [platform, shared, source_bc, mapping_bc, master_bc] }  # SAGA exception
  system_bc:    { mayDependOn: [platform, shared] }
  obs_bc:       { mayDependOn: [platform, shared] }
  platform:     { mayDependOn: [shared] }
  shared:       { mayDependOn: [] }                  # leaf — không phụ thuộc ai
  server:       { mayDependOn: [platform, shared, source_bc, mapping_bc, master_bc, transform_bc, recon_bc, wizard_bc, system_bc, obs_bc] }

  # ONLY infra/ được import gorm
  source_bc.infra:    { mayDependOn: [vendor:gorm.io/gorm] }
  # ... tương tự cho các BC khác
```

---

## 7. S5 — Bootstrap test (Phase 0) chi tiết

### 7.1 Hiện trạng
- `internal/bootstrap/registry_mirror.go` — 0 test
- `internal/bootstrap/shadow_connection.go` — 0 test

### 7.2 Test plan
| Test | Cover | Tool |
|---|---|---|
| `TestSyncLegacyToV2Bootstrap_EmptyDB` | Empty V1 → no-op | `sqlmock` |
| `TestSyncLegacyToV2Bootstrap_V1HasRows_V2Empty` | Mirror happy path | `sqlmock` + golden file |
| `TestSyncLegacyToV2Bootstrap_BothPopulated` | Skip mirror (đã sync) | `sqlmock` |
| `TestSyncLegacyToV2Bootstrap_V1CorruptRow` | Skip + log warn | `sqlmock` |
| `TestEnsureDefaultShadowConnection_NotExist` | Insert | `sqlmock` |
| `TestEnsureDefaultShadowConnection_AlreadyExist` | Idempotent | `sqlmock` |

Target coverage: ≥ 70%.

---

## 8. ADR (đầy đủ ở `04_decisions.md`)

| ADR | Quyết định |
|---|---|
| ADR-01 | Tách God Interface theo aggregate, KHÔNG đẩy interface tới từng handler |
| ADR-02 | Command chỉ phụ thuộc port hẹp, KHÔNG import `gorm.io/gorm` |
| ADR-03 | Composition Root pure-function, KHÔNG receiver-state |
| ADR-04 | **REJECT** Shared Kernel domain (`internal/domain/shared/`) — False Cognate |
| ADR-05 | Vertical Slice là Phase 4 OPTIONAL — phụ thuộc Phase 0-3 xong |
| ADR-06 | Wizard là BC duy nhất được phép cross-BC import (Saga pattern) |
| ADR-07 | `internal/shared/` chỉ chứa technical primitives, KHÔNG enum/lifecycle |
| ADR-08 | KHÔNG dùng DI framework — pure function đủ |

---

## 9. Câu hỏi pending (cần user trả lời trước khi Muscle thực thi)

| # | Câu hỏi | Default nếu user không trả |
|---|---|---|
| Q1 | Có làm Phase 4 (Vertical Slice) không, hay chỉ Phase 0-3? | Chỉ 0-3 (80% benefit, 50% effort) |
| Q2 | Có install `go-arch-lint` không, hay dùng `depguard` của golangci-lint? | `depguard` (đã có sẵn) |
| Q3 | Phase 0 nếu coverage hiện < 60% — bỏ qua hay viết bổ sung? | Viết bổ sung, không bypass |
| Q4 | Tên folder `internal/bc/` hay `internal/modules/` (v1 dùng `modules/`)? | `internal/bc/` (rõ DDD hơn) |
| Q5 | `pkgs/utils/` move vào `internal/shared/` luôn ở Phase 4, hay Phase 5 riêng? | Phase 5 riêng (optional) |
| Q6 | Mỗi PR phase yêu cầu user manual review, hay auto-merge sau test PASS? | Manual review (rule §8) |

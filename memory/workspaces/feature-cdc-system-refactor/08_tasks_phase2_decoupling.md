# Phase 2 v2 — Tasks — 4 Pillar CQRS Decoupling

> **Plan ref**: `02_plan_phase2_decoupling.md`

## Task graph (dependency)

```
P1 (interface skeleton)
  ├─ T1.1 → T1.2 → T1.3 → T1.4 → T1.5 (sequential)
  │
  └─→ P2 (Read migration)
       ├─ T2.1 (mapping list demo)
       │   └─→ T2.2 ... T2.7 (parallel-able)
       │              └─→ T2.8 (test all)
       │
       └─→ P3 (Write migration + bus + 2 INLINE move)
            ├─ T3.1 (migration cdc_jobs) ────────────┐
            ├─ T3.2 (JobRepo impl)                    │
            ├─ T3.3 (NATSCommandBus impl)             │
            │     └─→ T3.4 (7 sync metadata cmd)      │
            │     └─→ T3.5 (14 existing NATS cmd)     │
            │             └─→ T3.6 (worker master-swap)
            │             └─→ T3.7 (worker v2-sync)
            │             └─→ T3.8 (worker emit 12 evt)
            │                     └─→ T3.9 (JobMonitor extend)
            │                             └─→ T3.10 (GET /jobs/:id)
            │                                     └─→ T3.11 (smoke A1-A3)
            │
            └─→ P4 (infra persistence cleanup)
                 ├─ T4.1 ... T4.6 (parallel — 6 repo)
                 │      └─→ T4.7 (delete service/)
                 │      └─→ T4.8 (move pg_ident)
                 │      └─→ T4.9 (coverage gate)
```

---

## Pillar P1 — Setup interface skeleton (2d)

### T1.1 — Tạo `internal/domain/` directory + entity files
- **Subject**: Stub entity files (no logic)
- **DoD**:
  - `internal/domain/{mapping,source,master,reconciliation,job}/` mỗi folder có entity.go (struct + tag) + value-object.go
  - `go build ./...` PASS
- **Effort**: 4h

### T1.2 — Tạo `internal/app/ports/` interfaces
- **Subject**: Define interfaces
- **DoD**:
  - `internal/app/ports/repository.go` chứa: `MappingRuleRepo`, `SourceRepo`, `MasterRepo`, `JobRepo`, `ReconReportRepo`, `FailedSyncLogRepo`, `AlertRepo`, `WizardRepo`, `ConnectionRepo`, `ActivityLogRepo`, `AdminActionRepo`
  - `internal/app/ports/command_bus.go` chứa `Command`, `CommandResult`, `CommandBus`
  - `internal/app/ports/query_bus.go` chứa `Query`, `QueryBus`
  - `internal/app/ports/publisher.go` chứa `Publisher`
  - `go build ./...` PASS
- **Effort**: 4h

### T1.3 — Stub `internal/app/{queries,commands}/` + `internal/infra/{persistence,messaging,http,cache}/`
- **Subject**: Empty placeholder dir + 1 dummy file mỗi folder
- **DoD**: `ls` returns directory; `go build ./...` PASS
- **Effort**: 2h

### T1.4 — Wire `cmd/server/main.go` + `internal/server/server.go` để inject port
- **Subject**: Inject placeholder repo (delegate to existing service tạm) → handler vẫn dùng path cũ
- **DoD**:
  - `server.go` create instances of repo (impl tạm gọi internal/repository hoặc service hiện tại)
  - 8 endpoint smoke PASS — chưa thay đổi behavior
- **Effort**: 4h

### T1.5 — Build + smoke gate
- **Subject**: Validate P1 không phá hành vi
- **DoD**: `go build ./...` PASS, `go test ./... -count=1` PASS, 8 endpoint smoke PASS
- **Effort**: 2h
- **BlockedBy**: T1.1-T1.4

---

## Pillar P2 — Read migration → app/queries/ (3d)

### T2.1 — `ListMappingRulesQuery` demo (validate pattern)
- **DoD**:
  - File `internal/app/queries/list_mapping_rules.go` chứa struct + handler
  - File `internal/infra/persistence/mapping_rule_repo_gorm.go` impl `MappingRuleRepo.List`
  - Handler `mapping_rule_handler.go:List` đổi sang gọi `queryBus.Ask(ListMappingRulesQuery{...})`
  - Unit test sqlmock golden path
  - Endpoint `GET /api/mapping-rules` smoke PASS
- **Effort**: 6h
- **BlockedBy**: T1.5

### T2.2 — `ListSourceObjectsQuery` + `GetSourceObjectQuery`
- **DoD**:
  - File `list_source_objects.go`, `get_source_object.go`
  - `source_repo_gorm.go` impl
  - 2 endpoint smoke PASS
- **Effort**: 4h
- **BlockedBy**: T2.1

### T2.3 — `ListMastersQuery` + `GetMasterByNameQuery`
- **DoD**: 2 endpoint smoke PASS
- **Effort**: 4h
- **BlockedBy**: T2.1

### T2.4 — `GetReconReportQuery` + `ListFailedLogsQuery`
- **Subject**: Reconciliation read paths (2 endpoint)
- **DoD**:
  - `recon_report_repo_gorm.go` chứa LATERAL JOIN + ComputeDriftStatus extracted
  - `failed_sync_log_repo_gorm.go` impl
  - 2 endpoint smoke PASS với token
- **Effort**: 6h
- **BlockedBy**: T2.1

### T2.5 — `GetSyncHealthQuery` + `GetSystemHealthSnapshotQuery`
- **DoD**: 2 endpoint smoke PASS
- **Effort**: 4h
- **BlockedBy**: T2.1

### T2.6 — `ListConnectorsQuery`
- **Subject**: Kafka Connect REST → `infra/http/connector_client.go`
- **DoD**: 1 endpoint smoke PASS (502 OK nếu Kafka Connect down)
- **Effort**: 3h
- **BlockedBy**: T2.1

### T2.7 — Wizard, Alerts, Users, AdminAudit query
- **DoD**: 4 endpoint smoke PASS
- **Effort**: 4h
- **BlockedBy**: T2.1

### T2.8 — Test coverage gate
- **DoD**: `go test -cover ./internal/app/queries/...` ≥ 60%
- **Effort**: 2h
- **BlockedBy**: T2.2-T2.7

---

## Pillar P3 — Write migration + Command Bus + 2 INLINE move (5d)

### T3.1 — Migration `cdc_system.cdc_jobs`
- **Subject**: DDL apply tới `cdc_dw` (gpay-postgres-cdc)
- **DoD**:
  - File `centralized-data-service/migrations/cdc/036_create_cdc_jobs.sql`
  - Apply qua docker exec → psql → table created
  - Index `idx_cdc_jobs_type_status` + `idx_cdc_jobs_correlation` exist
- **Effort**: 2h
- **BlockedBy**: T2.8

### T3.2 — `JobRepo` GORM impl
- **DoD**:
  - `internal/infra/persistence/job_repo_gorm.go` impl: Create, GetByID, UpdateStatus, ListPending
  - Test sqlmock golden + idempotency-key dedup
- **Effort**: 4h
- **BlockedBy**: T3.1

### T3.3 — `NATSCommandBus` impl
- **DoD**:
  - `internal/infra/messaging/nats_command_bus.go`
  - Test với embedded NATS server
  - Validate publish PASS, validate Job row created TRƯỚC publish
- **Effort**: 6h
- **BlockedBy**: T3.2

### T3.4 — Migrate 7 sync metadata commands
- **List**: `CreateMappingRule`, `UpdateMappingRule`, `CreateMaster`, `RejectMaster`, `CreateWizard`, `PatchWizard`, `AckAlert`
- **DoD**:
  - File `app/commands/{create_mapping_rule,update_mapping_rule,...}.go`
  - Handler API gọi `commandBus.Dispatch` (CommandBus tạm dispatch sync — vì không có NATS subject)
  - 7 endpoint smoke PASS
- **Effort**: 6h
- **BlockedBy**: T3.3

### T3.5 — Migrate 14 existing-NATS commands
- **List**: 14 trigger đã dispatch NATS (recon-check, recon-heal, retry-failed, debezium-signal, recon-backfill-source-ts, debezium-snapshot, create-default-columns, standardize, scan-fields, detect-timestamp-field, backfill, alter-column, transmute, master-create, restart-debezium)
- **DoD**:
  - 14 file `app/commands/`
  - Handler API gọi `commandBus.Dispatch` thay vì publish NATS trực tiếp
  - Response 202 + `job_id`
  - 14 endpoint smoke (POST → 202 + job_id valid uuid)
- **Effort**: 8h
- **BlockedBy**: T3.3

### T3.6 — Worker: subscribe `cdc.cmd.master-swap` + handler ALTER RENAME
- **Subject**: Move INLINE Master Swap khỏi CMS
- **DoD**:
  - File `centralized-data-service/internal/handler/master_swap_handler.go`
  - Subscribe `cdc.cmd.master-swap` trong `worker_server.go`
  - Handler chạy `ALTER TABLE shadow_X RENAME TO master_X` với `SET LOCAL lock_timeout='3s'`
  - Emit `cdc.evt.master-swap.completed` với job_id + status
  - CMS `MasterSwapCommand` gọi `commandBus.Dispatch` → 202
  - Smoke: POST `/api/v1/masters/orders/swap` → 202 → wait → `GET /api/jobs/:id` → status=success
- **Effort**: 6h
- **BlockedBy**: T3.5

### T3.7 — Worker: subscribe `cdc.cmd.v2-sync` + handler UPSERT V2 metadata
- **Subject**: Move SyncFromLegacy khỏi CMS request thread
- **DoD**:
  - File `centralized-data-service/internal/handler/v2_sync_handler.go`
  - Subscribe `cdc.cmd.v2-sync`
  - Handler chạy `SyncFromLegacy` logic (chuyển từ CMS service)
  - Emit `cdc.evt.v2-sync.completed`
  - CMS `RegisterSourceCommand` chỉ INSERT V1 row, sau đó dispatch v2-sync (async); KHÔNG block request
  - Smoke: POST `/api/v1/source-objects` → 201 (V1 created) + job_id v2-sync trong response
- **Effort**: 6h
- **BlockedBy**: T3.5

### T3.8 — Worker: emit `cdc.evt.X.completed` cho 12 cmd hiện có
- **Subject**: Bổ sung companion event cho 12 cmd handler đã tồn tại trong worker
- **List**: standardize, scan-fields, detect-timestamp-field, create-default-columns, backfill, alter-column, recon-check, recon-heal, retry-failed, debezium-signal, recon-backfill-source-ts, master-create
- **DoD**:
  - Mỗi handler trong `centralized-data-service/internal/handler/command_handler.go`+ `recon_handler.go` thêm publish evt sau khi xong
  - Payload: `{job_id, type, status, result, error, completed_at}`
- **Effort**: 6h
- **BlockedBy**: T3.6, T3.7

### T3.9 — Worker: extend `JobMonitor` subscribe wildcard `cdc.evt.*.completed`
- **DoD**:
  - `internal/service/job_monitor.go` đổi subscribe sang wildcard
  - `HandleCompleted` route theo `msg.Subject`
  - UPDATE `cdc_system.cdc_jobs` SET status, result, finished_at WHERE id = job_id
  - Idempotent guard: WHERE status IN ('pending','running')
- **Effort**: 4h
- **BlockedBy**: T3.8

### T3.10 — CMS: thêm `GET /api/jobs/:id` endpoint
- **DoD**:
  - File `internal/api/jobs_handler.go` (≤50 dòng)
  - `GetJobQuery` + `JobRepo.GetByID`
  - Smoke: GET sau khi POST cmd → status field present
- **Effort**: 2h
- **BlockedBy**: T3.9

### T3.11 — Action smoke A1-A3
- **DoD**:
  - A1: POST `/api/reconciliation/check` → 202 + job_id
  - A2: GET `/api/jobs/:job_id` → status field
  - A3: POST `/api/v1/masters/orders/swap` → 202 → poll → success
- **Effort**: 2h
- **BlockedBy**: T3.10

---

## Pillar P4 — infra/persistence cleanup (3d)

### T4.1 — `MappingRuleRepo` full impl + delete `internal/repository/v2_mapping_rule_repo.go` cũ (nếu có)
- **DoD**:
  - `infra/persistence/mapping_rule_repo_gorm.go` chứa toàn bộ SQL từ `mapping_rule_handler.go`
  - `grep "h.db.Raw\|h.db.Exec" internal/api/mapping_rule_handler.go` = 0
- **Effort**: 4h
- **BlockedBy**: T3.11

### T4.2 — `SourceRepo` full impl
- **DoD**: tương tự T4.1 cho `registry_handler.go` + `source_object_actions_handler.go`
- **Effort**: 6h
- **BlockedBy**: T3.11

### T4.3 — `MasterRepo` full impl
- **DoD**: tương tự cho `master_registry_handler.go`
- **Effort**: 4h
- **BlockedBy**: T3.11

### T4.4 — `ReconReportRepo` + `FailedSyncLogRepo` full impl
- **DoD**: extract LATERAL JOIN từ `reconciliation_handler.go:LatestReport` + `pgIdent` qua `pkgs/utils/pg_ident.go`
- **Effort**: 6h
- **BlockedBy**: T3.11

### T4.5 — `AlertRepo`, `WizardRepo`, `ConnectionRepo`, `ActivityLogRepo`, `AdminActionRepo`
- **DoD**: 5 file impl
- **Effort**: 6h
- **BlockedBy**: T3.11

### T4.6 — Move `pgIdent` helper → `pkgs/utils/pg_ident.go`
- **DoD**: removed from `reconciliation_handler.go`; all callers updated
- **Effort**: 1h
- **BlockedBy**: T4.4

### T4.7 — Delete `internal/service/*.go` cũ (đã migrate)
- **Subject**: Cleanup
- **DoD**:
  - 16 file trong `internal/service/` deleted (ngoại trừ những file thuần utility — case-by-case)
  - `go build ./...` PASS
- **Effort**: 2h
- **BlockedBy**: T4.1-T4.5

### T4.8 — `wc -l internal/api/*.go` mọi file ≤100
- **DoD**: command output validate
- **Effort**: 1h
- **BlockedBy**: T4.7

### T4.9 — Test coverage `internal/infra/persistence/` ≥ 50%
- **DoD**: `go test -cover ./internal/infra/persistence/...` ≥ 0.50
- **Effort**: 6h
- **BlockedBy**: T4.7

---

## Per-pillar gate (BẮT BUỘC trước commit)

1. `go build ./...` PASS
2. `go test ./... -count=1` PASS
3. Endpoint smoke (8 GET + applicable POST cho pillar đó) PASS với token thật
4. `/security-agent` PASS (CLAUDE.md §8)
5. APPEND `05_progress.md` với commit hash
6. Pillar boundary clean: `grep` audit query passing (DoD command)

## Estimate tổng

| Pillar | Effort | Risk |
|---|---|---|
| P1 | 2d | LOW |
| P2 | 3d | LOW |
| P3 | 5d | HIGH |
| P4 | 3d | MEDIUM |
| **Sequential** | **13d** | |
| **+ pre-commit gate overhead 20%** | **~16d** | |
| **Realistic** | **3 tuần** với 1 engineer | |

## Out-of-band (escalate Brain)
- Worker chưa có infrastructure subscribe wildcard `cdc.evt.*.completed` → STOP + escalate.
- `cdc_system.cdc_jobs` migration race với worker → STOP + apply migration on cdc_dw TRƯỚC.
- FE break vì response 202 + job_id thay vì 200 + result → STOP + escalate (FE đã handle pending từ Lesson 2026-04-29).

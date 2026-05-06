# Phase 2 — cdc-cms-service Refactor — Tasks

> **Date**: 2026-05-05 | **Plan ref**: `02_plan_phase2_cms_refactor.md`

## Task graph (dependency)

```
T1 (P0) — dead code prune
   └─→ T2 (P1.0) — repo skeleton + 1 demo (mapping rule)
          ├─→ T3-T8 (P1.1-P1.6) — fan-out 6 repo
          │      └─→ T9 (P2.1) — reconciliation handler split
          │      └─→ T10 (P2.2) — source_object_actions handler split
          │      └─→ T11 (P2.3) — mapping_rule handler split
          │      └─→ T12 (P2.4) — master_registry handler split
          │             └─→ T13 (P3) — ActivityLog helper
          │             └─→ T14 (P4) — V1↔V2 dedup
          │
          ├─→ T15 (P5) — health probe split (parallel với P2)
          ├─→ T16 (P6) — V2 sync atomicity (parallel với P2)
          └─→ T17 (P7) — test uplift (last)
```

---

## T1 — P0: Dead-code prune

- **Subject**: Remove `ReconciliationService` no-op + audit `cdc_event.go`
- **DoD**:
  - `internal/service/reconciliation_service.go` deleted
  - `internal/server/server.go` field + alloc + goroutine removed (verify line 85, 187)
  - `cdc_event.go`: `grep -r "model.CDCEvent\|CDCEventData\|UpsertRecord" --include='*.go' cdc-cms-service/` = 0 → delete; nếu có hit, giữ
  - `go build ./...` PASS
  - 8 endpoint smoke PASS
- **Effort**: 2h
- **Blocks**: T2-T17

## T2 — P1.0: Repo skeleton + MappingRuleV2Repo demo

- **Subject**: Define repo interface pattern + implement first repo + refactor 1 handler call site
- **DoD**:
  - New file `internal/repository/v2_mapping_rule_repo.go` với interface `MappingRuleV2Repo` + struct `mappingRuleV2GormRepo`
  - Method tối thiểu: `List(ctx, filter) ([]MappingRuleV2DTO, error)`, `GetByID`, `Create`, `UpdateStatus`, `BatchUpdateStatus`
  - Flat DTO struct (no nested) per Lesson #1253
  - `mapping_rule_handler.go`: `List` method dùng `h.repo.List(...)` thay raw SQL
  - `server.go` wire repo, inject vào handler
  - Unit test `v2_mapping_rule_repo_test.go` qua sqlmock — golden path
  - Endpoint `GET /api/mapping-rules` smoke PASS với token thật
- **Effort**: 1d
- **BlockedBy**: T1

## T3 — P1.1: SourceObjectRegistryRepo

- **Subject**: Repo cho `cdc_system.source_object_registry` + `shadow_binding`
- **DoD**: methods `List`, `GetByID`, `GetByRegistryID`, `Update`, `ResolveScopeBySourceObjectID`, `ResolveScopeByRegistryID`. Refactor `source_objects_handler.go` read paths.
- **Effort**: 1d
- **BlockedBy**: T2

## T4 — P1.2: MasterBindingRepo

- **DoD**: methods `List`, `GetByName`, `Create`, `Approve`, `Reject`, `ToggleActive`. Refactor `master_registry_handler.go` read path (Section 3.10 endpoints).
- **Effort**: 6h
- **BlockedBy**: T2

## T5 — P1.3: ConnectionRegistryRepo

- **DoD**: methods `Resolve(connectionCode)` (đã có ad-hoc trong `SourceObjectV2SyncService` — extract).
- **Effort**: 4h
- **BlockedBy**: T2

## T6 — P1.4: WizardSessionRepo

- **DoD**: methods `Create`, `GetByID`, `PatchDraft`, `RecordProgress`, `MarkExecuted`. Refactor `wizard_handler.go`.
- **Effort**: 6h
- **BlockedBy**: T2

## T7 — P1.5: AlertRepo

- **DoD**: methods `Upsert`, `Resolve`, `Ack`, `Silence`, `ListActive`, `ListSilenced`, `ListHistory`. Refactor `alert_manager.go` (extract DB layer) + `alerts_handler.go`.
- **Effort**: 6h
- **BlockedBy**: T2

## T8 — P1.6: AdminActionRepo

- **DoD**: method `LogAdminAction(ctx, event)` — replace raw SQL trong `middleware/audit.go`. Add async drainer (đã có channel pattern, just extract).
- **Effort**: 4h
- **BlockedBy**: T2

## T9 — P2.1: Split `reconciliation_handler.go` (894 dòng)

- **Subject**: Extract drift calc + dispatch + queries → `internal/service/reconciliation/`
- **DoD**:
  - File mới: `drift_calculator.go`, `recon_dispatcher.go`, `failed_log_query.go`, `report_query.go`
  - Move `pgIdent` → `pkgs/utils/pg_ident.go`
  - Handler ≤200 dòng
  - 4 endpoint reconciliation smoke PASS (`GET /api/reconciliation/report`, `GET /api/failed-sync-logs`, `POST /api/reconciliation/check`, `POST /api/recon/backfill-source-ts`)
- **Effort**: 1.5d
- **BlockedBy**: T2-T8 (cần repo abstraction)

## T10 — P2.2: Split `source_object_actions_handler.go` (693 dòng)

- **DoD**:
  - File mới: `internal/service/source_object/dispatch_service.go`, `scope_resolver.go`
  - Generic `Dispatch(ctx, subject, scope, payload)` thay 7 method gần giống nhau
  - Handler ≤200 dòng
  - 13 endpoint smoke (xem Section 3.4 inventory)
- **Effort**: 1d
- **BlockedBy**: T3, T5

## T11 — P2.3: Split `mapping_rule_handler.go` (689 dòng)

- **DoD**:
  - File mới: `internal/service/mapping/mapping_rule_service.go`, `scope_resolver.go`, `list_query.go`, `batch_dispatcher.go`
  - Fix N+1 trong `BatchUpdate` (line 615-689) → bulk publish + single SQL update
  - Handler ≤180 dòng
- **Effort**: 1d
- **BlockedBy**: T2, T3

## T12 — P2.4: Split `master_registry_handler.go` (667 dòng)

- **DoD**:
  - File mới: `internal/service/master_registry/master_service.go`
  - Reuse existing `MasterSwap` service (no duplicate)
  - Handler ≤150 dòng
- **Effort**: 1d
- **BlockedBy**: T4

## T13 — P3: ActivityLog helper

- **Subject**: 1 helper, 8+ call site replaced
- **DoD**:
  - New `internal/service/activity_logger.go` với `Log` + `LogAsync`
  - `grep "model.ActivityLog{" internal/api/` = 0
  - Wire trong `server.go`
- **Effort**: 4h
- **BlockedBy**: T9-T12 (handler đã thin)

## T14 — P4: V1↔V2 dedup

- **Subject**: V2 không gọi V1 RegistryHandler methods
- **DoD**:
  - `grep "h.registry\." source_object_actions_handler.go` = 0
  - V1 RegistryHandler + V2 SourceObjectActionsHandler đều gọi `dispatch.Service`
  - Verify cms-fe-overhaul phase 27-32 status: V2 đã direct-write chưa? (Read 03_implementation_phase27, 29 trước khi commit)
- **Effort**: 6h
- **BlockedBy**: T10

## T15 — P5: Health collector probe split (parallel)

- **DoD**:
  - 7 probe file dưới `internal/service/health/probes/`
  - Collector ≤300 dòng
  - Probe gọi parallel qua errgroup → snapshot generation latency giảm
  - `/api/system/health` smoke PASS
- **Effort**: 1d
- **BlockedBy**: T1 (parallel với T2-T14)

## T16 — P6: V2 sync atomicity (parallel)

- **DoD**:
  - `SourceObjectV2SyncService.SyncFromLegacy` đổi sang `SyncFromLegacyTx(tx)`
  - Caller `registry_handler.go:Register` + `BulkRegister` wrap trong `db.Transaction(func(tx *gorm.DB) error {...})`
  - Test: mock SyncFromLegacyTx return err → assert V1 row rollback
- **Effort**: 1d
- **BlockedBy**: T1 (parallel)

## T17 — P7: Test uplift (last)

- **DoD**:
  - Test mới cho mọi service file (target 35% combined)
  - Test mới cho mọi repo file (sqlmock golden path)
  - CI threshold gate
- **Effort**: 2d
- **BlockedBy**: T13, T14, T15, T16

---

## Per-pillar gate (BẮT BUỘC trước commit)

1. `go build ./...` PASS
2. `go test ./... -count=1` PASS (ít nhất golden path test mới + existing test)
3. Endpoint smoke (theo §Verification trong 01_requirements_phase2) — 8 endpoint
4. `/security-agent` PASS (CLAUDE.md §8)
5. APPEND `05_progress.md` log entry với commit hash

## Estimate tổng

- Sequential: ~12-14 work day
- With parallel (T15+T16 || T9-T14): ~10-12 work day
- Pre-commit gate overhead: +20%
- **Realistic: 2-3 tuần** với 1 engineer dedicated

## Out-of-band tasks (nếu phát sinh)

- TS phát hiện migration thêm cần thiết → **STOP** + spawn workspace riêng `feature-cdc-cms-migration-XX`. Phase 2 không touch DB.
- TS phát hiện worker code cần đổi (subject schema thay đổi) → **STOP** + escalate Brain. Phase 2 not touch worker.
- TS phát hiện FE code cần đổi (response shape thay đổi) → **STOP** + escalate. Contract preserve là constraint cứng.

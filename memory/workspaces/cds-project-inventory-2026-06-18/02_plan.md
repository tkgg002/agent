# 02_plan.md — Master Refactor Plan: centralized-data-service

> **Approach**: Layer-first + Sub-domain (ADR-001/002/003)
> **Strategy**: Strangler Fig — từng phase nhỏ, compile+test+commit sau mỗi bước
> **Scope**: 191 Go files, 3 entrypoints (worker, sinkworker, admin-api)
> **God Objects cần tách**: `command_handler.go` (3437L), `recon_core.go` (1900L), `worker_server.go` (1245L)

---

## Cấu trúc mục tiêu

```
internal/
├── handler/                    # Layer Giao tiếp (NATS, Kafka, HTTP)
│   ├── source/                 # 🔲 TODO
│   ├── shadow/                 # 🔲 TODO
│   ├── master/                 # 🔲 TODO
│   ├── recon/                  # 🔲 TODO
│   └── orchestration/          # 🔲 TODO
│
├── service/                    # Layer Business Logic
│   ├── source/                 # 🔲 TODO
│   ├── shadow/                 # 🔲 TODO
│   ├── master/                 # 🔲 TODO
│   ├── governance/             # 🔲 TODO
│   ├── recon/                  # 🔲 TODO
│   └── transmute/              # ✅ Pre-existing (strategy pattern)
│
├── repository/                 # Layer Data Access
│   ├── source/                 # connection_registry, source_object, table_registry
│   ├── shadow/                 # shadow_binding, failed_sync_log
│   ├── master/                 # master_binding, mapping_rule_v2, transmute_schedule
│   └── recon/                  # snapshot_dlq, reconciliation_report
│
└── model/                      # Layer Entities & Structs
    ├── source/                 # ConnectionRegistry, SourceObjectRegistry, TableRegistry, SchemaChangeLog
    ├── shadow/                 # ShadowBinding, CDCEvent, FailedSyncLog, PendingField, SensitiveField
    ├── master/                 # MasterBinding, MappingRuleV2, SyncRuntimeState, WorkerSchedule, TransmuteSchedule
    └── system/                 # ActivityLog, SnapshotDLQ, ReconciliationReport
```

---

## Tổng quan Giai đoạn (Updated 2026-06-19T09:50)

| # | Giai đoạn | Nội dung | Status |
|---|-----------|----------|--------|
| 1 | Model Layer | Move 18 files → 4 sub-folders | ✅ DONE |
| 2 | Repository Layer | Move 11 files → 4 sub-folders | ✅ DONE |
| 3a | Tạo Repos mới | Extract inline GORM → 3 repos mới | ✅ DONE (repos created) |
| 3b | Wire Callers | Migrate callers sang repos mới | 🔲 TODO |
| 4 | Service: governance + source | Move → sub-folders | 🔲 TODO |
| 5a | Service: shadow + master + recon | Move → sub-folders | 🔲 TODO |
| 5b | Tách recon_core.go | Split 1901L → 3 files (engine/tier_a/tier_b) | ✅ DONE |
| 6 | Handler: shadow + recon + master + source + orchestration | Move → sub-folders | 🔲 TODO |
| 7 | Handler: tách command_handler.go | File-split 3441L → 506L + 5 files (ADR-005) | ✅ DONE |
| 8a | Shared Utils: pkgs/sqlutil | Extract QuoteIdent, IsSafeIdent → pkgs/ | ✅ DONE (package created) |
| 8b | Shared Utils: wire callers | Migrate `quoteCommand*` → `sqlutil.QuoteIdent/QualifiedTable` (27 calls) | ✅ DONE |
| 9 | Server DI: split worker_server | Split 1247L → 3 files (core/init/tickers) | ✅ DONE |

---

## Giai đoạn 1: Model Layer (18 files → 4 sub-folders) ✅ DONE

**Mục tiêu**: Tổ chức `internal/model/` theo domain, không phụ thuộc gì — làm trước.

### Thứ tự thực hiện (ít rủi ro → nhiều)

**Batch 1.1: `model/system/`** (3 files)
- `model/activity_log.go` → `model/system/activity_log.go`
- `model/snapshot_dlq.go` → `model/system/snapshot_dlq.go`
- `model/reconciliation_report.go` → `model/system/reconciliation_report.go`
- Package: `package system`
- Gate: `go build ./internal/model/...`

**Batch 1.2: `model/source/`** (4 files)
- `model/connection_registry.go` → `model/source/connection_registry.go`
- `model/source_object_registry.go` → `model/source/source_object_registry.go`
- `model/table_registry.go` → `model/source/table_registry.go`
- `model/schema_change_log.go` → `model/source/schema_change_log.go`
- Package: `package source`
- Gate: `go build ./internal/model/...`

**Batch 1.3: `model/shadow/`** (5 files)
- `model/shadow_binding.go` → `model/shadow/shadow_binding.go`
- `model/cdc_event.go` → `model/shadow/cdc_event.go`
- `model/failed_sync_log.go` → `model/shadow/failed_sync_log.go`
- `model/pending_field.go` → `model/shadow/pending_field.go`
- `model/sensitive_field.go` → `model/shadow/sensitive_field.go`
- Package: `package shadow`
- Gate: `go build ./internal/model/...`

**Batch 1.4: `model/master/`** (6 files)
- `model/master_binding.go` → `model/master/master_binding.go`
- `model/mapping_rule_v2.go` → `model/master/mapping_rule_v2.go`
- `model/mapping_rule.go` → `model/master/mapping_rule.go`
- `model/sync_runtime_state.go` → `model/master/sync_runtime_state.go`
- `model/worker_schedule.go` → `model/master/worker_schedule.go`
- `model/transmute_schedule.go` → `model/master/transmute_schedule.go`
- Package: `package master`
- Gate: `go build ./internal/model/...`

**Root `model/`**: Empty (all files moved to sub-packages)

**Compile gate Phase 1**: `go build ./internal/model/... && go build ./...`
**Commit**: `refactor(model): organize into source/shadow/master/system sub-packages`

---

## Giai đoạn 2: Repository Layer (11 files → 4 sub-folders) ✅ DONE

**Phụ thuộc**: Phase 1 (model imports)

**Batch 2.1: `repository/source/`** (4 files)
- `connection_registry_repo.go` → `repository/source/`
- `source_object_registry_repo.go` → `repository/source/`
- `registry_repo.go` → `repository/source/`
- `schema_log_repo.go` → `repository/source/`

**Batch 2.2: `repository/shadow/`** (2 files)
- `shadow_binding_repo.go` → `repository/shadow/`
- `pending_field_repo.go` → `repository/shadow/`

**Batch 2.3: `repository/master/`** (5 files)
- `master_binding_repo.go` → `repository/master/`
- `mapping_rule_v2_repo.go` → `repository/master/`
- `mapping_rule_repo.go` → `repository/master/`
- `sync_runtime_state_repo.go` → `repository/master/`
- `transmute_schedule_repo.go` → `repository/master/`

**Compile gate Phase 2**: `go build ./internal/repository/... && go build ./...`
**Commit**: `refactor(repository): organize into source/shadow/master sub-packages`

---

## Giai đoạn 3a: Tạo Repository mới ✅ DONE

**Status**: Repos created, callers NOT yet migrated.

- `repository/shadow/failed_sync_log_repo.go` ✅ — Create, GetByID, UpdateByID, ListPending
- `repository/recon/snapshot_dlq_repo.go` ✅ — CreateBatch, GetPendingByProgress
- `repository/recon/reconciliation_report_repo.go` ✅ — Create, GetByID, UpdateByID, GetLatestByTable

---

## Giai đoạn 3b: Wire Callers → Repos mới 🔲 TODO

**Mục tiêu**: Migrate inline GORM calls trong handlers/services → gọi qua repos.

**3b.1: Wire `FailedSyncLogRepo`**
- `batch_buffer.go` L413: `bb.db.Create(...)` → `bb.failedSyncLogRepo.Create(ctx, ...)`
- `dlq_handler.go` L261: `d.db.WithContext(ctx).Model(&shadow.FailedSyncLog{})` → `d.failedSyncLogRepo.UpdateByID(ctx, ...)`
- `dlq_state_machine.go` L436: `sm.db.WithContext(ctx).Where("id = ?", id).First(&row)` → `sm.failedSyncLogRepo.GetByID(ctx, id)`
- `recon_handler.go` L802: `h.db.Model(&shadow.FailedSyncLog{})` → `h.failedSyncLogRepo.UpdateByID(ctx, ...)`
- `kafka_consumer.go` L1351: `kc.db.WithContext(ctx).Create(&row)` → `kc.failedSyncLogRepo.Create(ctx, &row)`

**Gate**: `go build ./... && go test ./...`

---

## ~~Giai đoạn 4: Service Layer — Governance + Source~~ ⏭️ DEFERRED

> **ADR-004**: Go private function sharing barrier. See `04_decisions.md`.
>
> **Blocker**: `MetadataRegistry` interface cycle + `text_sanitizer.go` private helpers.
> **Unblock condition**: Extract `MetadataRegistry` → `internal/ports/` + export shared helpers → `internal/pkgs/`.

---

## ~~Giai đoạn 5: Service Layer — Shadow + Master + Recon~~ ⏭️ PARTIALLY DONE

> **Sub-packages**: DEFERRED (ADR-004, same blocker as Phase 4).

### Phase 5b: Tách recon_core.go ✅ DONE
- `recon_core.go` (1901L) → **deleted**, replaced by:
  - `recon_engine.go` (727L) — Config, constructors, run management, CheckAll, utilities
  - `recon_tier_a.go` (803L) — Source↔Shadow: Tier1/2/3, OrphanPrune, lag helpers
  - `recon_tier_b.go` (419L) — Shadow↔Master: RunSegmentB, RunRowDiffB, diffIDTs

---

## ~~Giai đoạn 6: Handler Layer — Shadow + Recon~~ ⏭️ DEFERRED

> **ADR-004**: Same Go private function sharing barrier as Phase 4-5.

---

## Giai đoạn 7: Tách command_handler.go ✅ DONE (ADR-005: File-Split)

> **Revised strategy**: File-split trong cùng `handler/` package thay vì struct-split.
> See `04_decisions.md` ADR-005.

**Kết quả**:
- `command_handler.go` (506L) — struct + setup + shared helpers
- `command_handler_ddl.go` (767L) — HandleStandardize, HandleCreateDefaultColumns, HandleDropGINIndex, HandleAlterColumn
- `command_handler_discover.go` (899L) — HandleDiscover, HandleDiscoverMongo*, HandleScanFields
- `command_handler_scan.go` (836L) — HandleScanRawData, HandleScanArrayFields, HandlePeriodicScan, HandleBackfill
- `command_handler_transform.go` (340L) — HandleBatchTransform, HandleMasterSwap
- `command_handler_sync.go` (181L) — HandleSyncRegister, HandleSyncState, HandleRestartDebezium

---

## Giai đoạn 8a: Shared Utils — pkgs/sqlutil ✅ DONE

- `pkgs/sqlutil/quote.go` — QuoteIdent, QualifiedTable, IsSafeIdent, IsSafeType
- `pkgs/sqlutil/quote_test.go` — 4/4 tests PASS

---

## Giai đoạn 8b: Wire callers → pkgs/sqlutil ✅ DONE

**Kết quả**: 27 calls migrated, old definitions removed.

| Caller file | Private func | Migration | Status |
|---|---|---|---|
| `command_handler_ddl.go` | `quoteCommandIdent` | → `sqlutil.QuoteIdent` | ✅ Done (16 calls) |
| `command_handler_ddl.go` | `quoteCommandQualifiedTable` | → `sqlutil.QualifiedTable` | ✅ Done |
| `command_handler_scan.go` | `quoteCommandIdent` | → `sqlutil.QuoteIdent` | ✅ Done (7 calls) |
| `command_handler_transform.go` | `quoteCommandIdent` | → `sqlutil.QuoteIdent` | ✅ Done (3 calls) |
| `command_handler_discover.go` | `quoteCommandIdent` | → `sqlutil.QuoteIdent` | ✅ Done (1 call) |
| `command_handler.go` | `isSafeIdent` / `isSafeType` | **KEPT** (semantically different: includes `-`, max 64, allowlist) | ⏭️ N/A |
| `command_handler.go` | `sanitizeAdminError` / `sanitizeAdminResultMap` | **KEPT** (handler-specific logic) | ⏭️ N/A |
| `command_handler.go` | `publishResult` / `writeActivity` | **KEPT** (too invasive to extract cross-handler) | ⏭️ N/A |

**Gate**: `go build ./... && go test ./...` ✅ PASS

---

## Giai đoạn 9: Server DI Wiring ✅ DONE

**Revised**: Thay vì update imports (vì Phase 4-6 deferred), split file size:
- `worker_server.go` (1247L → 337L) — Struct + Start() + Shutdown()
- `worker_server_init.go` (704L) — NewWorkerServer DI wiring
- `worker_server_tickers.go` (243L) — Periodic cycle handlers

---

## Quy tắc chung cho MỌI giai đoạn

1. **Strangler Fig**: Compile gate bắt buộc sau mỗi batch
2. **Named Imports**: Khi collision (handler/master vs service/master) → `handlermaster`, `servicemaster`
3. **Package name = folder name**: `internal/service/master/` → `package master`
4. **KHÔNG đổi logic**: Chỉ move + đổi package declaration + update imports
5. **Verify callers**: `grep -rn '<OldImportPath>' ./` phải = 0 sau mỗi batch
6. **KHÔNG tự commit** (GP-230): Agent KHÔNG tự `git commit`. Accumulate changes → verify build → để User review qua IDE Source Control → User quyết định commit strategy
7. **Git restore-point**: Commit sau mỗi batch thành công (User-controlled)

---

## Files KHÔNG move

| Path | Lý do |
|---|---|
| `internal/sinkworker/` | Entrypoint riêng, có main logic |
| `internal/server/worker_server.go` | DI root — chỉ update imports (Giai đoạn 9) |
| `internal/naming/` | Pure utility, không thuộc domain |
| `internal/activity/` | Taxonomy enums — merge vào `model/system/` sau |
| `internal/admin/` | HTTP server admin — giữ nguyên |
| `pkgs/` | External packages, không thay đổi |
| `config/` | Config layer, không thay đổi |

---

## Chi tiết mapping files

- [02_plan_model.md](./02_plan_model.md) — Phase 1 chi tiết
- [02_plan_repository.md](./02_plan_repository.md) — Phase 2+3 chi tiết
- [02_plan_service.md](./02_plan_service.md) — Phase 4+5 chi tiết
- [02_plan_handler.md](./02_plan_handler.md) — Phase 6+7 chi tiết

## Tham khảo Domain Analysis

- [04_domain_groups.md](./04_domain_groups.md) — Phân tích 7 domains × 4 layers
- [04_decisions.md](./04_decisions.md) — ADR-001/002/003

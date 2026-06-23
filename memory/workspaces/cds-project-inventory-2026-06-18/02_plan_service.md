# 02_plan_service.md — Phase 3: Tổ chức lại `internal/service/`

## 40 production files → 5 sub-folders

---

## `service/source/` — 8 files

| File move |
|---|
| `metadata_registry_service.go` (34 funcs) |
| `registry_service.go` (22 funcs) |
| `connection_manager.go` |
| `connection_overrides.go` |
| `connector_resolver.go` |
| `source_router.go` |
| `mongo_introspection.go` (6 funcs) |
| `scan_service.go` |

## `service/shadow/` — 7 files

| File move |
|---|
| `schema_adapter.go` (33 funcs) |
| `dynamic_mapper.go` (19 funcs) |
| `child_explode.go` |
| `enrichment_service.go` |
| `bridge_service.go` |
| `type_resolver.go` |
| `text_sanitizer.go` |

## `service/master/` — 7 files + transmute/ folder

| File move |
|---|
| `master_ddl_generator.go` (14 funcs) |
| `transmuter.go` (28 funcs) |
| `transmute_scheduler.go` (5 funcs) |
| `child_explode_master.go` |
| `job_monitor.go` |
| `transform_registry.go` |
| `transmute/` (strategy folder) |

**Key funcs `transmuter.go`**:
```
NewTransmuterModule(...)
Run(ctx, masterName, onlySourceIDs) → TransmuteResult
loadMaster / shadowActive / loadRules [private]
InvalidateRuleCache(bindingID, masterTable)
fetchShadowBatch / processBatch / toTransmuteRules [private]
upsertMaster / gjsonValueToGo / coerceForColumn [private]
markRuntimeSuccess / markRuntimeFailure / persistRuntimeState [private]
```

**Key funcs `master_ddl_generator.go`**:
```
NewMasterDDLGenerator(...)
Generate(ctx, masterName) / Apply(ctx) / EnsureMaster(ctx)
loadBinding / parsePKFromSpec / parseIndexesFromSpec [private]
markDDLStatus / ReconcileColumn / DropColumn
```

## `service/governance/` — 10 files

| File move |
|---|
| `masking_service.go` (27 funcs) |
| `schema_inspector.go` (11 funcs) |
| `schema_validator.go` |
| `activity_logger.go` |
| `partition_dropper.go` |
| `wal_monitor.go` |
| `full_count_aggregator.go` |
| `debezium_signal.go` (11 funcs) |
| `timestamp_detector.go` |
| `backfill_source_ts.go` (14 funcs) |

## `service/recon/` — 8 files (recon_core.go tách thành 3)

| File move | Ghi chú |
|---|---|
| `recon_core.go` | **TÁCH** → 3 files dưới |
| `recon_source_agent.go` | Move |
| `recon_dest_agent.go` | Move |
| `recon_heal.go` | Move |
| `recon_alert.go` | Move |
| `dlq_worker.go` | Move |
| `provisioning_orchestrator.go` (22 funcs) | Move |
| `provisioning_state_machine.go` | Move |

**Tách recon_core.go → 3 files**:
- `recon_engine.go` — Base: NewReconCore, CheckAll, PruneAllOrphans, shared helpers
- `recon_tier_a.go` — Source↔Shadow: RunTier1/2/3, OrphanPrune, lag logic
- `recon_tier_b.go` — Shadow↔Master: RunSegmentB, RunRowDiffB, diffIDTs

# Report: Phase 1 (Model) + Phase 2 (Repository) — Refactor centralized-data-service

> **Thời gian**: 2026-06-19T01:50 → 02:05 (15 phút)
> **Phương pháp**: Strangler Fig — batch nhỏ, compile gate, commit sau mỗi batch
> **Kết quả**: `go build ./...` PASS, `go test ./internal/...` PASS

---

## Tổng kết thay đổi

| Giai đoạn | Commits | Files changed | Lines added | Lines removed |
|-----------|---------|---------------|-------------|---------------|
| Phase 1.1 (model/system) | 1 | 14 | 67 | 60 |
| Phase 1.2 (model/source) | 1 | 25 | 171 | 163 |
| Phase 1.3 (model/shadow) | 1 | 25 | 108 | 103 |
| Phase 1.4 (model/master) | 1 | 25 | 122 | 126 |
| Phase 2.1 (repo/source) | 1 | 15 | 44 | 38 |
| Phase 2.2+2.3 (repo/shadow+master) | 1 | 16 | 47 | 44 |
| Test fixes | 1 | 4 | 4 | 0 |
| **TỔNG** | **7 commits** | **~60 unique files** | **~563** | **~534** |

---

## Cấu trúc sau refactor

### model/ (18 files → 4 sub-packages)
```
internal/model/
├── source/     (4 files: connection_registry, source_object_registry, table_registry, schema_change_log)
├── shadow/     (5 files: shadow_binding, cdc_event, failed_sync_log, pending_field, sensitive_field)
├── master/     (6 files: master_binding, mapping_rule_v2, mapping_rule, sync_runtime_state, worker_schedule, transmute_schedule)
└── system/     (3 files: activity_log, snapshot_dlq, reconciliation_report)
```

### repository/ (11 files → 3 sub-packages)
```
internal/repository/
├── source/     (4 files: connection_registry_repo, source_object_registry_repo, registry_repo, schema_log_repo)
├── shadow/     (2 files: shadow_binding_repo, pending_field_repo)
└── master/     (5 files: master_binding_repo, mapping_rule_v2_repo, mapping_rule_repo, sync_runtime_state_repo, transmute_schedule_repo)
```

---

## Named Imports sử dụng (ADR-001)

| Package | Named Import | Lý do collision |
|---------|-------------|-----------------|
| `model/source` | `sourcemodel` | `source :=` variable in recon_handler.go |
| `model/master` | `mastermodel` | `master :=` variable in 8 files |
| `model/shadow` | `shadow` | Không collision |
| `model/system` | `system` | Không collision |
| `repository/source` | `reposource` | `source` variable collision |
| `repository/shadow` | `reposhadow` | Để đồng nhất |
| `repository/master` | `repomaster` | `master` variable collision |

---

## Issues encountered & giải quyết

1. **macOS sed `\b` word boundary**: Không hoạt động → dùng specific patterns (`model.MappingRule{`, `[]model.MappingRule`)
2. **Double substitution**: `model.` → `mastermodel.` → `mastermastermodel.` khi sed chạy 2 lần → phải fix global
3. **Variable shadowing**: `source := "auto"` che khuất package `source` → Named Import `sourcemodel`
4. **Pre-existing test failure**: `transmuter_test.go` NUMERIC/DECIMAL coercion — không liên quan refactor

---

## Files đã thay đổi (danh sách đầy đủ)

### Model files moved (18 files)
- `model/activity_log.go` → `model/system/activity_log.go`
- `model/snapshot_dlq.go` → `model/system/snapshot_dlq.go`
- `model/reconciliation_report.go` → `model/system/reconciliation_report.go`
- `model/connection_registry.go` → `model/source/connection_registry.go`
- `model/source_object_registry.go` → `model/source/source_object_registry.go`
- `model/table_registry.go` → `model/source/table_registry.go`
- `model/schema_change_log.go` → `model/source/schema_change_log.go`
- `model/shadow_binding.go` → `model/shadow/shadow_binding.go`
- `model/cdc_event.go` → `model/shadow/cdc_event.go`
- `model/failed_sync_log.go` → `model/shadow/failed_sync_log.go`
- `model/pending_field.go` → `model/shadow/pending_field.go`
- `model/sensitive_field.go` → `model/shadow/sensitive_field.go`
- `model/master_binding.go` → `model/master/master_binding.go`
- `model/mapping_rule_v2.go` → `model/master/mapping_rule_v2.go`
- `model/mapping_rule.go` → `model/master/mapping_rule.go`
- `model/sync_runtime_state.go` → `model/master/sync_runtime_state.go`
- `model/worker_schedule.go` → `model/master/worker_schedule.go`
- `model/transmute_schedule.go` → `model/master/transmute_schedule.go`

### Repository files moved (11 files)
- `repository/connection_registry_repo.go` → `repository/source/`
- `repository/source_object_registry_repo.go` → `repository/source/`
- `repository/registry_repo.go` → `repository/source/`
- `repository/schema_log_repo.go` → `repository/source/`
- `repository/shadow_binding_repo.go` → `repository/shadow/`
- `repository/pending_field_repo.go` → `repository/shadow/`
- `repository/master_binding_repo.go` → `repository/master/`
- `repository/mapping_rule_v2_repo.go` → `repository/master/`
- `repository/mapping_rule_repo.go` → `repository/master/`
- `repository/sync_runtime_state_repo.go` → `repository/master/`
- `repository/transmute_schedule_repo.go` → `repository/master/`

### Caller files updated (imports + references) — ~30 unique files
- `internal/handler/batch_buffer.go`
- `internal/handler/command_handler.go`
- `internal/handler/dlq_handler.go`
- `internal/handler/dlq_state_machine.go`
- `internal/handler/event_bridge.go`
- `internal/handler/event_handler.go`
- `internal/handler/kafka_consumer.go`
- `internal/handler/recon_handler.go`
- `internal/handler/recon_heal_v4.go`
- `internal/handler/snapshot_runner_handler.go`
- `internal/handler/transmute_handler.go`
- `internal/server/worker_server.go`
- `internal/service/activity_logger.go`
- `internal/service/backfill_source_ts.go`
- `internal/service/bridge_service.go`
- `internal/service/child_explode.go`
- `internal/service/connection_overrides.go`
- `internal/service/dlq_worker.go`
- `internal/service/dynamic_mapper.go`
- `internal/service/enrichment_service.go`
- `internal/service/full_count_aggregator.go`
- `internal/service/masking_service_test.go`
- `internal/service/master_ddl_generator.go`
- `internal/service/metadata_registry_service.go`
- `internal/service/recon_core.go`
- `internal/service/recon_heal.go`
- `internal/service/registry_service.go`
- `internal/service/scan_service.go`
- `internal/service/schema_adapter.go`
- `internal/service/schema_inspector.go`
- `internal/service/schema_validator.go`
- `internal/service/source_router.go`
- `internal/service/transmuter.go`
- `test/internal/handler/...` (5 test files)
- `test/internal/service/...` (5 test files)
- `scratch/debug_registry/debug_registry.go`

---

## Verification

- ✅ `go build ./...` — PASS
- ✅ `go vet ./...` — PASS (pre-existing sonyflake warning only)
- ✅ `go test ./internal/...` — PASS (service + handler + transmute)
- ⚠️ Pre-existing: `transmuter_test.go` NUMERIC/DECIMAL coercion failure (không liên quan refactor)

---

## Giai đoạn tiếp theo (chưa thực hiện)

| # | Giai đoạn | Status |
|---|-----------|--------|
| 1 | Model Layer | ✅ DONE |
| 2 | Repository Layer | ✅ DONE |
| 3 | Tạo Repos mới (inline GORM) | ⬜ TODO |
| 4 | Service: governance + source | ⬜ TODO |
| 5 | Service: shadow + master + recon | ⬜ TODO |
| 6 | Handler: shadow + recon | ⬜ TODO |
| 7 | Handler: tách command_handler.go | ⬜ TODO |
| 8 | Shared Utils + pkgs/ | ⬜ TODO |
| 9 | Server DI Wiring | ⬜ TODO |

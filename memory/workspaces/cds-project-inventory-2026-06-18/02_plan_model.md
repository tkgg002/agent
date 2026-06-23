# 02_plan_model.md — Phase 1: Tổ chức lại `internal/model/`

## Hiện trạng: 18 files, tất cả flat trong `model/`

## Mục tiêu: Tổ chức theo 4 sub-folder

---

## `internal/model/source/` — Entities thuộc Data Sources

| File hiện tại | File mới | Structs chứa |
|---|---|---|
| `model/connection_registry.go` | `model/source/connection_registry.go` | `ConnectionRegistry` |
| `model/source_object_registry.go` | `model/source/source_object_registry.go` | `SourceObjectRegistry` |
| `model/table_registry.go` | `model/source/table_registry.go` ⚠️ deprecated | `TableRegistry` (V1 legacy) |
| `model/schema_change_log.go` | `model/source/schema_change_log.go` | `SchemaChangeLog` |

---

## `internal/model/shadow/` — Entities thuộc Shadow plane

| File hiện tại | File mới | Structs chứa |
|---|---|---|
| `model/shadow_binding.go` | `model/shadow/shadow_binding.go` | `ShadowBinding` |
| `model/cdc_event.go` | `model/shadow/cdc_event.go` | `CDCEvent`, `CDCEventData`, `UpsertRecord` |
| `model/failed_sync_log.go` | `model/shadow/failed_sync_log.go` | `FailedSyncLog` |
| `model/pending_field.go` | `model/shadow/pending_field.go` | `PendingField`, `SensitiveField` |

---

## `internal/model/master/` — Entities thuộc Master plane

| File hiện tại | File mới | Structs chứa |
|---|---|---|
| `model/master_binding.go` | `model/master/master_binding.go` | `MasterBinding` |
| `model/mapping_rule_v2.go` | `model/master/mapping_rule_v2.go` | `MappingRuleV2` |
| `model/mapping_rule.go` | `model/master/mapping_rule.go` ⚠️ V1 deprecated | `MappingRule` |
| `model/sync_runtime_state.go` | `model/master/sync_runtime_state.go` | `SyncRuntimeState` |
| `model/worker_schedule.go` | `model/master/worker_schedule.go` | `WorkerSchedule` |
| `model/transmute_schedule.go` | `model/master/transmute_schedule.go` | `TransmuteSchedule` |

---

## `internal/model/system/` — Cross-cutting entities

| File hiện tại | File mới | Structs chứa |
|---|---|---|
| `model/activity_log.go` | `model/system/activity_log.go` | `ActivityLog` |
| `model/snapshot_dlq.go` | `model/system/snapshot_dlq.go` | `SnapshotDLQ` |
| `model/recon_report.go` | `model/system/recon_report.go` | `ReconciliationReport` |

> ⚠️ `internal/activity/` chứa taxonomy enums → merge vào `model/system/activity_log.go` (constants only)

---

## Quy tắc thực hiện

1. **Move** file, cập nhật package declaration thành `package source` / `package shadow` / `package master` / `package system`
2. Tất cả callers import từ `internal/model/` → cập nhật sang `internal/model/source/`, v.v.
3. Nếu struct có circular dep giữa sub-package → tạo `model/shared/` chứa shared types
4. Compile gate: `go build ./internal/model/...`

---

## File nào KHÔNG move

| File | Lý do |
|---|---|
| `model/event_types.go` | Pure constants — có thể giữ ở root `model/` |
| `model/registry_filter.go` | Filter type, dùng chung nhiều repo |

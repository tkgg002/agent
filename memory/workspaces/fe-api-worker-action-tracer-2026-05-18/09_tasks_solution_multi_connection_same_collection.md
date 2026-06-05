# Solution Summary — Multi connector cùng (db, collection) → tách shadow schema

**Phase**: fe-api-worker-action-tracer-2026-05-18 / multi_connection_same_collection
**Date**: 2026-05-19
**Status**: PLAN ONLY — awaiting user decision

## Nguyên nhân (3 layer cùng đẩy về `(db, table)` identity, bỏ connection)

| Layer | File:line | Code hiện tại | Hậu quả |
|---|---|---|---|
| L0 — Input | `internal/model/table_registry.go` | V1 `TableRegistry` Go struct KHÔNG có `SourceConnectionID`. FE Register form không có dropdown connector. | System không biết user reference connector nào |
| L1 — Identity key | `internal/infra/persistence/source_object_v2_sync.go:80` | `normalizedSourceKey := lower(engine + ":" + sourceDB + ":" + sourceTable)` | UNIQUE `source_object_registry.normalized_source_key` chặn row thứ 2 → 2 connector merge thành 1 row |
| L1 — Object code | `source_object_v2_sync.go:91, 333` | `buildSourceObjectCode(engine, sourceDB, sourceTable)` | `object_code` UNIQUE cũng collision |
| L2 — Resolver | `source_object_v2_sync.go:271-291` | `resolveSourceConnectionID`: ORDER BY id ASC LIMIT 1 → **first-wins** | `(mongodb, centralized-export-service)` luôn return id=1 (`goopay`); id=2 (`goopay1`) bị skip |
| L3 — Shadow schema | `source_object_v2_sync.go:78` | `shadowSchema := "shadow_" + slugify(sourceDB)` | Cùng `sourceDB` → cùng schema → không tách Postgres schema riêng |

**Evidence từ user API**:
- `/sources` trả 3 connector (`goopay`, `goopay1`, `goopay2`) khác physical mongo.
- `/source-objects` trả `total: 2` — `goopay` + `goopay1` (cùng db/collection) merge vào ID=1 với `shadow_schema=shadow_centralized_export_service`. Không có row riêng cho `goopay1`.

## Giải pháp (3 options)

### Option A — Identity-by-Connection ✅ RECOMMENDED

Đưa `connection_id` (qua `connection_code` stable) vào identity tier-1. 1 logical source = `(connection, db, object)` triplet.

**Schema migrations (3)**:
1. `054_v1_add_source_connection_id.sql` — `ALTER TABLE cdc_system.cdc_table_registry ADD COLUMN source_connection_id BIGINT REFERENCES cdc_system.connection_registry(id);` (nullable, backwards compat).
2. `055_backfill_v1_source_connection_id.sql` — backfill từ first-wins lookup cho rows hiện tại; log mỗi row affected để admin review.
3. `056_relax_v1_unique_with_connection.sql` — replace UNIQUE 3-cột (053) → 4-cột `(source_connection_id, source_db, source_table, target_table)`.

**Go code change (4 file CMS + 1 file worker)**:
```go
// internal/model/table_registry.go
SourceConnectionID *int64 `gorm:"column:source_connection_id" json:"source_connection_id,omitempty"`

// internal/infra/persistence/source_object_v2_sync.go
sourceConnectionID, connectionCode, err := s.resolveSourceConnection(ctx, tx, entry, sourceEngine, sourceDB)
normalizedSourceKey := lower(engine + ":" + connectionCode + ":" + sourceDB + ":" + sourceTable)
objectCode    := "src_" + engine + "_" + connectionCode + "_" + sourceDB + "_" + sourceTable
shadowSchema  := "shadow_" + connectionCode + "_" + sourceDB

// resolveSourceConnection priority:
//   (a) entry.SourceConnectionID nếu set → SELECT connection_code by id
//   (b) Fallback first-wins (legacy backwards compat, log WARN)
```

Files touched:
- `cdc-cms-service/internal/model/table_registry.go` — add field.
- `cdc-cms-service/internal/infra/persistence/source_object_v2_sync.go` — rebuild key/code/schema.
- `cdc-cms-service/internal/bootstrap/registry_mirror.go` — bootstrap mirror cùng pattern.
- `cdc-cms-service/internal/api/registry_handler_register.go` — payload thêm `source_connection_id`.
- `cdc-cms-web` (FE) — dropdown Source Connector. **Out of scope phase này** (user fix sau hoặc phase riêng).
- `centralized-data-service/internal/service/metadata_registry_service.go` — `buildSourceLookupKeys` thêm variant include `connection_code` cho precision.

**Trade-off**: Semantic chuẩn ✓ — scope to (3 migration + 5 file Go + FE form) ✗.

---

### Option B — Shadow-Schema-Only Composite (partial)

Giữ identity merge ở `source_object_registry`, chỉ tách `shadow_schema = "shadow_" + connectionCode + "_" + sourceDB` ở binding insertion.

**Trade-off**: Scope nhỏ ✓ — KHÔNG fix root: 2 connector vẫn share 1 source_object_registry row → metadata (primary_key, sync_engine, profile_status) bị share giữa 2 cluster physical khác config → corruption risk ✗.

**Không recommend** trừ khi user accept compromise (2 connector cùng cấu hình hoàn toàn).

---

### Option C — V1-Level Identity Only

Add `source_connection_id` vào V1, relax V1 UNIQUE, NHƯNG V2 sync vẫn first-wins → V2 merge như cũ.

**Trade-off**: Half-measure — user-visible bug KHÔNG fix ✗.

**Không recommend**.

---

## Recommendation: Option A

Lý do:
1. User report rõ "schemas postgres riêng" → cần tách identity tier-1.
2. V2 model đã có `source_connection_id` FK — completing contract.
3. Worker `connectionOverrides` (phase trước) đã dùng `connection_code` — naming consistent.

## Implementation order (sau khi user approve)

| # | Step | Layer | Risk |
|---|---|---|---|
| 1 | Migration 054 ADD COLUMN (nullable) | DB | Low |
| 2 | Migration 055 backfill first-wins + audit log | DB | Medium |
| 3 | Migration 056 relax UNIQUE thành 4-cột | DB | Low |
| 4 | Model `TableRegistry.SourceConnectionID *int64` | Go CMS | Low |
| 5 | V2 sync rebuild key/code/schema | Go CMS | Medium |
| 6 | Bootstrap mirror cùng pattern | Go CMS | Low |
| 7 | API Register payload + validator nullable | Go CMS | Low |
| 8 | Worker `metadata_registry_service` cache key | Go worker | Medium |
| 9 | Workspace docs + report + global lesson | governance | Low |
| 10 | User: apply 3 migrations + retry register `goopay1` | User | — |

## Verification gates

- [ ] `go build ./...` + `go vet ./...` EXIT=0 cả 2 service.
- [ ] `go test -count=1 ./internal/infra/persistence/... ./internal/api/...` PASS (CMS).
- [ ] `go test -count=1 ./internal/handler/... ./internal/service/...` PASS (worker).
- [ ] User retry register `goopay1.centralized-export-service.export-jobs` → expected:
  - 2 `source_object_registry` rows (object_code khác nhau).
  - 2 shadow schema trong Postgres (`shadow_goopay_centralized_export_service` + `shadow_goopay1_centralized_export_service`).
  - 2 shadow_binding với `source_object_id` khác nhau.

## 4 câu hỏi cần user trả lời trước implement

1. **Option** — A (recommend), B, hay C?
2. **Backfill strategy** cho `source_object_registry` rows ambiguous hiện tại:
   - (a) First-wins → connection_id=1 (`goopay`); `goopay1` sẽ tạo row mới khi register lại.
   - (b) NULL → admin review qua CMS UI.
   - (c) Delete + force re-register.
3. **FE update**: trong scope phase này hay phase riêng?
4. **Shadow schema legacy**: giữ `shadow_centralized_export_service` cho row id=1 (legacy) + tạo `shadow_goopay1_centralized_export_service` mới? Hay rename cả 2 thành pattern `shadow_<connection>_<db>` ngay từ migration?

## Files đã tạo (PLAN ONLY, KHÔNG implement)

- `agent/memory/workspaces/fe-api-worker-action-tracer-2026-05-18/01_requirements_multi_connection_same_collection.md` — evidence + root cause chi tiết.
- `agent/memory/workspaces/fe-api-worker-action-tracer-2026-05-18/02_plan_multi_connection_same_collection.md` — 3 options + code demo Option A + 11-step plan.
- `agent/memory/workspaces/fe-api-worker-action-tracer-2026-05-18/09_tasks_solution_multi_connection_same_collection.md` (file này — summary gọn).
- `05_progress.md` APPENDED.
- KHÔNG có code change. KHÔNG có migration file. Chờ user pick option.

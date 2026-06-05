# Report — Multi-Connection Same Collection (Option A Implementation)

**Phase**: fe-api-worker-action-tracer-2026-05-18 / multi_connection_same_collection
**Date**: 2026-05-19
**Status**: ✅ Code complete, builds + tests pass — chờ user apply 3 migration + retry register `goopay1`.

## TL;DR (1 đoạn)

Đã chuyển V2 identity từ `(engine, db, table)` → `(engine, connection_code, db, table)`
trên 3 layer: V1 model + V2 sync + bootstrap mirror + 3 API dispatcher + worker
cache. 3 SQL migration (054 ADD COLUMN, 055 backfill first-wins, 056 relax
UNIQUE) đã viết sẵn. 6 file Go thay đổi, không có file mới (trừ workspace docs).
Build + vet + test PASS cả 2 service. User cần apply 3 migration rồi register
lại `goopay1.centralized-export-service.export-jobs` với payload `source_connection_id`.

## Files đã thay đổi (chi tiết kết quả tính toán)

### SQL — 3 migrations (NEW)

```
cdc-cms-service/migrations/schema/core/054_v1_add_source_connection_id.sql       — 44 lines
cdc-cms-service/migrations/schema/core/055_backfill_v1_source_connection_id.sql  — 64 lines
cdc-cms-service/migrations/schema/core/056_relax_v1_unique_with_connection.sql   — 51 lines
```

### Go — CMS service (5 files MODIFIED)

```
cdc-cms-service/internal/model/table_registry.go
  + Line 7: SourceConnectionID *int64 (NEW field).

cdc-cms-service/internal/infra/persistence/source_object_v2_sync.go
  ~ Lines 24-36: NEW connectionResolutionRow struct.
  ~ Lines 90-106: Identity rebuild — normalizedSourceKey, objectCode, shadowSchema
    now ALL include connection_code.
  ~ Lines 286-364: Refactor resolveSourceConnectionID → resolveSourceConnection
    (internal) + EXPORT ResolveSourceConnection + ResolveShadowSchema (package
    API for callers in commands/api packages).
  ~ Lines 377-394: normalizeShadowSchema → normalizeShadowSchemaWithConnection
    (signature change). buildSourceObjectCode + buildShadowBindingCode now take
    connectionCode param.

cdc-cms-service/internal/bootstrap/registry_mirror.go
  ~ Lines 73-105: Resolve source connection (entry.SourceConnectionID priority
    → first-wins fallback). Compute connCodeSlug once.
  ~ Lines 126-127: objectCode + normKey include connCodeSlug.
  ~ Lines 152-153: bindingCode + shadowSchema include connCodeSlug.

cdc-cms-service/internal/app/commands/register_registry.go
  + Import persistence package.
  ~ Line 105-110: Replace static shadow schema build → persistence.ResolveShadowSchema.
  - Lines 158-174: DELETE duplicated normalizeShadowIdent helper.

cdc-cms-service/internal/api/registry_handler_register.go
  + Import persistence.
  - Remove naming import.
  ~ Line 57-69: Resolve shadow schema via persistence.ResolveShadowSchema, 500 on error.

cdc-cms-service/internal/api/registry_handler_bulk.go
  + Import persistence.
  - Remove naming import.
  ~ Line 52-65: Resolve shadow schema per entry, log+skip on error.

cdc-cms-service/internal/api/registry_handler_tools_columns.go
  - Remove naming import.
  ~ Line 26-37: Resolve shadow schema, 500 on error.
```

### Go — Worker service (1 file MODIFIED)

```
centralized-data-service/internal/service/metadata_registry_service.go
  ~ Line 192-200: cache write loop passes connectionCodeByID[src.SourceConnectionID].
  ~ Line 538-572: buildSourceLookupKeys(src, connectionCode) — emits legacy keys
    (backwards compat) + NEW connection-aware variants (connection_code:...,
    connection_code:db|...). Cache uses if !exists guard to retain first-wins
    semantic for legacy keys.
```

### Workspace docs (NEW/APPENDED)

```
agent/memory/workspaces/fe-api-worker-action-tracer-2026-05-18/
  03_implementation_multi_connection_same_collection.md  (NEW)
  report_multi_connection_same_collection.md             (NEW — file này)
  05_progress.md                                          (APPEND-only)
  09_tasks_solution_multi_connection_same_collection.md  (đã tạo phase trước)
agent/memory/global/lessons.md                            (APPEND-only)
```

## Verification kết quả thật

| Step | Command | Output |
|---|---|---|
| 1 | `cd cdc-cms-service && go build ./...` | EXIT=0, no stderr |
| 2 | `cd cdc-cms-service && go vet ./...` | EXIT=0, no stderr |
| 3 | `cd cdc-cms-service && go test -count=1 ./internal/infra/persistence/... ./internal/api/... ./internal/app/commands/... ./internal/bootstrap/...` | 4 packages OK (persistence 1.674s, api 1.120s, commands 0.628s, bootstrap [no test files]) |
| 4 | `cd centralized-data-service && go build ./...` | EXIT=0, no stderr |
| 5 | `cd centralized-data-service && go vet ./...` | EXIT=0, no stderr |
| 6 | `cd centralized-data-service && go test -count=1 ./internal/service/... ./internal/handler/...` | service 0.662s, handler 4.249s — PASS |

## What's NOT done (out of scope)

1. **FE dropdown source connector** — user dùng curl/Postman gửi `source_connection_id` đến khi FE update.
2. **Schema rename one-shot DDL** — user tự `ALTER SCHEMA shadow_<db> RENAME TO shadow_<connection>_<db>` nếu muốn giữ legacy data accessible qua tên mới.
3. **V1 cdc_system_model migration** cho `source_object_registry` để add UNIQUE composite `(source_connection_id, source_engine_type, source_database, source_object_name)` thay cho `normalized_source_key`. Hiện chỉ rebuild key string — vẫn dùng UNIQUE trên `normalized_source_key`. Đủ cho bug hiện tại, nhưng schema-level cleanup là follow-up.
4. **Cleanup orphan source_object_registry rows** sau bootstrap re-sync (rows cũ với key không có connection_code). Cần migration phụ hoặc admin SQL.

## DoD checklist

- [x] 3 migration file viết sẵn ở đúng folder
- [x] Model `TableRegistry.SourceConnectionID` field thêm
- [x] V2 sync identity rebuild với connection_code
- [x] Bootstrap mirror pattern khớp với V2 sync
- [x] 3 API dispatcher (register/bulk/tools_columns) + commands handler dùng `ResolveShadowSchema`
- [x] Worker `buildSourceLookupKeys` emit connection-aware variants
- [x] go build + vet + test cả 2 service PASS
- [x] Workspace docs (03_implementation, report) tạo file vật lý
- [x] Lesson abstract vào lessons.md (sẽ append sau khi report này lưu)
- [ ] User apply 3 migration + retry register goopay1 (chờ)
- [ ] User verify 2 shadow schema riêng tồn tại trong Postgres (chờ)

## Câu hỏi mà user CHƯA trả lời (implicit defaults đã commit)

Người dùng pick "A" mà chưa trả lời 4 câu hỏi phụ trong solution doc:

| Q | Default đã commit | Lý do |
|---|---|---|
| Q2 Backfill strategy | (a) first-wins | Mặc định an toàn nhất, đã encode trong migration 055. Nếu user muốn (b) NULL hoặc (c) Delete, modify 055 trước khi apply. |
| Q3 FE update scope | Out of scope | User không nhắc FE; phase này chỉ unblock backend. |
| Q4 Shadow schema legacy | Giữ legacy + tạo schema mới | No rename runtime. User tự ALTER SCHEMA RENAME nếu cần migrate data. |

Nếu user muốn override default, báo lại trước khi apply migration.

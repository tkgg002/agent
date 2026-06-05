# Implementation — Multi-Connection Same Collection (Option A)

**Phase**: fe-api-worker-action-tracer-2026-05-18 / multi_connection_same_collection
**Date**: 2026-05-19
**Status**: ✅ CODE COMPLETE — awaiting DB migration apply + user verification

## Tóm tắt cách Option A đã được thực thi

Identity tier-1 cho V2 `source_object_registry` chuyển từ
`(engine, db, table)` → `(engine, connection_code, db, table)`. Hai connector
cùng `(db, collection)` giờ materialize thành 2 row riêng + 2 Postgres
shadow schema riêng (`shadow_<connection>_<db>`).

## Files thay đổi (10 files)

### SQL migrations (3 file — apply theo thứ tự)

| # | File | Effect |
|---|---|---|
| 054 | `cdc-cms-service/migrations/schema/core/054_v1_add_source_connection_id.sql` | `ALTER TABLE cdc_table_registry ADD COLUMN source_connection_id BIGINT REFERENCES connection_registry(id)` + index `idx_ctr_source_connection (source_connection_id, source_db, source_table)`. NULLABLE → backwards compat. |
| 055 | `cdc-cms-service/migrations/schema/core/055_backfill_v1_source_connection_id.sql` | `DO $$` loop: với mỗi V1 row có `source_connection_id IS NULL`, tìm first-wins match từ `connection_registry` (engine_type + default_database) và UPDATE. RAISE NOTICE per row → audit log. |
| 056 | `cdc-cms-service/migrations/schema/core/056_relax_v1_unique_with_connection.sql` | DROP UNIQUE `(source_db, source_table, target_table)` (từ migration 053) → ADD UNIQUE `(source_connection_id, source_db, source_table, target_table)`. PG NULL-distinct semantics cho legacy rows. |

### Go — CMS service (5 file)

| # | File:Line | Change |
|---|---|---|
| 1 | `internal/model/table_registry.go:7` | ADD `SourceConnectionID *int64` (`gorm:"column:source_connection_id" json:"source_connection_id,omitempty"`). BodyParser tự pick up từ JSON payload. |
| 2 | `internal/infra/persistence/source_object_v2_sync.go` | (a) Rebuild identity: `normalizedSourceKey = engine:connection_code:db:table`, `objectCode = src_<engine>_<connection>_<db>_<table>`, `shadowSchema = shadow_<connection>_<db>`. (b) `resolveSourceConnection`: priority `entry.SourceConnectionID` → first-wins fallback (logs WARN). (c) Exported `ResolveSourceConnection` + `ResolveShadowSchema` for API/commands. |
| 3 | `internal/bootstrap/registry_mirror.go` | Apply same pattern: resolve connection (entry.SourceConnectionID priority), include `connection_code` in `objectCode`, `normalizedSourceKey`, `bindingCode`, `shadowSchema`. |
| 4 | `internal/app/commands/register_registry.go:105` | Replace `naming.ShadowSchemaName(normalizeShadowIdent(entry.SourceDB))` → `persistence.ResolveShadowSchema(ctx, h.db, &entry)`. DELETE duplicated `normalizeShadowIdent` helper. |
| 5 | `internal/api/registry_handler_register.go`, `registry_handler_bulk.go`, `registry_handler_tools_columns.go` | All 3 `CreateDefaultColumnsCommand` dispatchers: replace static schema build → `persistence.ResolveShadowSchema(ctx, h.db, entry)`. Returns 500 if resolve fails. |

### Go — Worker service (1 file)

| # | File:Line | Change |
|---|---|---|
| 6 | `centralized-data-service/internal/service/metadata_registry_service.go` | `buildSourceLookupKeys(src, connectionCode)` — emits BOTH legacy keys (object_name, db\|object_name) AND connection-aware keys (connection_code:object_name, connection_code:db\|object_name). Cache write site passes `connectionCodeByID[src.SourceConnectionID]`. `if !exists` guard keeps first-wins semantic cho legacy keys. |

## Verification (đã chạy local)

| Gate | Command | Result |
|---|---|---|
| Build CMS | `go build ./...` ở `cdc-cms-service` | EXIT=0 ✓ |
| Build worker | `go build ./...` ở `centralized-data-service` | EXIT=0 ✓ |
| Vet CMS | `go vet ./...` | EXIT=0 ✓ |
| Vet worker | `go vet ./...` | EXIT=0 ✓ |
| Test CMS | `go test -count=1 ./internal/infra/persistence/... ./internal/api/... ./internal/app/commands/... ./internal/bootstrap/...` | PASS ✓ (4 packages OK) |
| Test worker | `go test -count=1 ./internal/service/... ./internal/handler/...` | PASS ✓ |

## Workflow user cần làm

```bash
# 1. Apply 3 migrations theo thứ tự
psql ... -f cdc-cms-service/migrations/schema/core/054_v1_add_source_connection_id.sql
psql ... -f cdc-cms-service/migrations/schema/core/055_backfill_v1_source_connection_id.sql
psql ... -f cdc-cms-service/migrations/schema/core/056_relax_v1_unique_with_connection.sql

# 2. Migration 055 sẽ in NOTICE per row backfilled — verify
#    e.g. "055 backfill: registry_id=1 → connection_id=1"
#    e.g. "055 backfill summary: updated=N, skipped=M"

# 3. (Tùy chọn) Rename legacy shadow schema để khớp pattern mới — chỉ chạy
#    nếu user muốn data goopay cũ accessible qua shadow schema mới:
#    ALTER SCHEMA shadow_centralized_export_service
#      RENAME TO shadow_goopay_centralized_export_service;

# 4. Restart 2 service (CMS + worker)

# 5. Register lại goopay1.centralized-export-service.export-jobs với
#    payload bao gồm source_connection_id:
#    POST /api/v1/source-objects/register
#    {
#      "source_connection_id": 2,  ← FK đến goopay1
#      "source_db": "centralized-export-service",
#      "source_table": "export-jobs",
#      "target_table": "goopay1_export_jobs",
#      "source_type": "mongo",
#      "primary_key_field": "_id"
#    }

# 6. Expected outcome
#    - 2 source_object_registry rows (object_code khác nhau)
#    - 2 Postgres shadow schemas: shadow_goopay_centralized_export_service +
#      shadow_goopay1_centralized_export_service
#    - 2 shadow_binding rows với source_object_id khác nhau
```

## Notes / Risks

1. **Bootstrap re-sync impact**: khi CMS restart sau migration, `SyncLegacyToV2Bootstrap` chạy lại cho TẤT CẢ V1 registry. Với rows đã được backfill 055, nó tạo NEW source_object_registry row (new normalized_source_key) trong khi LEGACY row vẫn ở DB. Hậu quả: 1 V1 registry → 2 V2 source_object_registry rows. Worker `if !exists` defensive guard giữ first-wins → cache vẫn dùng legacy row cho lookups không có connection_code. Đây là KNOWN trade-off cho backwards-compat.
2. **Postgres schema gap**: code mới tạo `shadow_<connection>_<db>`. Data legacy nằm trong `shadow_<db>`. User phải tự `ALTER SCHEMA ... RENAME` (one-shot) hoặc accept duplicate empty schema.
3. **FE form**: out of scope phase này. User hiện phải gửi `source_connection_id` qua API trực tiếp (curl/Postman) cho đến khi FE dropdown ra mắt ở phase sau.
4. **Fallback first-wins**: nếu request KHÔNG có `source_connection_id` (FE chưa update), code fallback first-wins resolver + log WARN. Behavior backwards-compat hoàn toàn.

## Backout (nếu cần rollback)

```sql
-- Rollback theo thứ tự ngược lại:
BEGIN;
  ALTER TABLE cdc_system.cdc_table_registry
    DROP CONSTRAINT IF EXISTS cdc_table_registry_conn_source_db_table_target_key;
  ALTER TABLE cdc_system.cdc_table_registry
    ADD CONSTRAINT cdc_table_registry_source_db_source_table_target_key
    UNIQUE (source_db, source_table, target_table);
COMMIT;
-- (rollback 055 — chỉ làm nếu admin chấp nhận mất backfill data)
UPDATE cdc_system.cdc_table_registry SET source_connection_id = NULL;
-- (rollback 054 — DROP COLUMN sẽ break code mới đang dùng SourceConnectionID)
-- Code rollback phải đi kèm: revert tất cả 6 file Go.
```

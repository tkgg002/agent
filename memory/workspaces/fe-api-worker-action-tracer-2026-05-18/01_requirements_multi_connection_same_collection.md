# Requirements — Multi Mongo connector cùng (db, collection) → tách shadow schema riêng

**Phase**: fe-api-worker-action-tracer-2026-05-18 / multi_connection_same_collection
**Author**: Claude Code (Muscle, claude-opus-4-7)
**Date**: 2026-05-19

## Evidence (user APIs)

`GET /api/v1/sources` (CMS) trả về 3 connectors:

| id | connector_name | server_address | database | collection |
|---|---|---|---|---|
| 1 | `goopay`  | `mongodb://gpay-mongo:27017/?replicaSet=rs0` | `centralized-export-service` | `export-jobs` |
| 2 | `goopay1` | `mongodb://root:***@10.200.187.11:27017,...` (replica set 3 nodes) | `centralized-export-service` | `export-jobs` |
| 3 | `goopay2` | `mongodb://root:***@10.200.187.11:27017,...` | `payment-bill-service` | `payment-bills` |

`GET /api/v1/source-objects?page=1` trả về `total: 2`:

| id | object_code | source_db | source_table | shadow_schema |
|---|---|---|---|---|
| 1 | `src_mongodb_centralized_export_service_export_jobs` | `centralized-export-service` | `export-jobs` | `shadow_centralized_export_service` |
| 5 | `src_mongodb_payment_bill_service_payment_bills`     | `payment-bill-service`        | `payment-bills` | `shadow_payment_bill_service` |

→ `goopay` (id=1) và `goopay1` (id=2) physical mongo KHÁC nhau **bị merge** vào CÙNG MỘT `source_object_registry` row (id=1). KHÔNG có row riêng cho `goopay1`. KHÔNG có schema riêng `shadow_<connection_code>_centralized_export_service` cho `goopay1`.

## User report (verbatim)

> 2 cái collection trùng tên, nó ko chịu tạo schemas ở postgres riêng. chắc do trùng tên. đưa ra giải pháp trước.

## Mục tiêu

2 connector physical khác nhau (e.g. `goopay` local docker vs `goopay1` remote replica) cùng `(database, collection)` → tạo 2 `source_object_registry` rows DIFFERENT + 2 shadow_schema DIFFERENT trong Postgres.

Concretely sau fix:
```
source_object_registry:
  id=1 object_code=src_mongodb_goopay_centralized_export_service_export_jobs   shadow_schema=shadow_goopay_centralized_export_service
  id=N object_code=src_mongodb_goopay1_centralized_export_service_export_jobs  shadow_schema=shadow_goopay1_centralized_export_service
```

## Root cause

3 layer cùng lúc đẩy về `(db, table)` identity, drop connection_id:

### Layer 1 — Identity ở `source_object_registry`

`internal/infra/persistence/source_object_v2_sync.go:80`:
```go
normalizedSourceKey := strings.ToLower(fmt.Sprintf("%s:%s:%s", sourceEngine, sourceDB, sourceTable))
```
→ key KHÔNG include connection_id/connection_code. UNIQUE constraint `source_object_registry.normalized_source_key` chặn row thứ 2.

`internal/infra/persistence/source_object_v2_sync.go:91`:
```go
objectCode := buildSourceObjectCode(sourceEngine, sourceDB, sourceTable)
// = "src_mongodb_centralized_export_service_export_jobs"
```
→ `object_code` UNIQUE cũng collision.

### Layer 2 — Resolver first-wins

`internal/infra/persistence/source_object_v2_sync.go:271-291` `resolveSourceConnectionID`:
```sql
SELECT id FROM cdc_system.connection_registry
WHERE role_type IN ('source', 'mixed')
  AND engine_type = ?
  AND status = 'active'
ORDER BY CASE WHEN COALESCE(default_database, '') = ? THEN 0 ELSE 1 END, id ASC
LIMIT 1
```
→ Với `(mongodb, centralized-export-service)` luôn return id=1 (`goopay`). Connector id=2 (`goopay1`) **không bao giờ** được chọn.

### Layer 3 — Shadow schema từ source_db only

`internal/infra/persistence/source_object_v2_sync.go:78`:
```go
shadowSchema := normalizeShadowSchema(sourceDB)
// = "shadow_centralized_export_service"
```
→ `naming.ShadowSchemaName` chỉ nhận `sourceDB` làm suffix. Cùng db → cùng schema.

### Layer 0 — V1 input thiếu connection_id

`internal/model/table_registry.go` (V1 `TableRegistry` Go struct) **KHÔNG có** cột `source_connection_id`. FE Register form cũng không chọn connector cụ thể. Khi insert V1, system hoàn toàn không biết user đang reference connector nào.

## Constraint

- KHÔNG cheat DB — không hack normalized_source_key manually.
- KHÔNG đổi V2 schema unique constraint (`UNIQUE (normalized_source_key)` giữ nguyên — chỉ đổi cách build key).
- Tuân Core direction: identity của source object phải bao gồm connection (semantic chuẩn cross-environment).
- Backwards compatible: existing `source_object_registry` rows phải tiếp tục resolve được.

## Out of scope

- Đổi V2 `shadow_binding` UNIQUE (đã đúng — đã include `source_object_id`).
- Drop V1 `cdc_table_registry` (legacy bridge phase riêng).
- Đa-cluster reconciliation (chỉ tách identity, không reroute traffic).

## Definition of Done

- [ ] V2 sync tạo distinct `source_object_registry` row cho mỗi `(connection_id, db, table)` tuple.
- [ ] `object_code` + `normalized_source_key` include connection_code.
- [ ] `shadow_schema` include connection_code prefix (e.g. `shadow_goopay_centralized_export_service`).
- [ ] V1 `cdc_table_registry` có cột `source_connection_id` (BIGINT, nullable cho backwards compat).
- [ ] FE Register form cho user chọn source connector (dropdown từ `connection_registry`).
- [ ] Existing rows backfill `source_connection_id` từ first-wins lookup (gracefully — không break legacy).
- [ ] Audit worker cache (sourceCache, targetCache, routeBySourceID) — confirm key gồm connection_id hoặc tolerant với multi-row per (db, table).
- [ ] User retry register `goopay1.centralized-export-service.export-jobs` → expected: 2 source_object_registry rows + 2 shadow schemas.

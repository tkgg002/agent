# 02_plan — Audit Shadow Create Bugs (Audit-only Phase)

## Phase A — Trace (DONE)
1. FE: `cdc-cms-web/src/pages/TableRegistry.tsx` — không có FE-side field copy. Form submit `POST /api/v1/source-objects/register` với raw values từ antd Form. → FE clean.
2. cms-service: `cdc-cms-service/internal/api/registry_handler_register.go` line 17 → `Register()` → bus exec `RegisterRegistryCommand` → `persistence.ResolveShadowSchema` (line 55) → dispatch NATS `cdc.cmd.create-default-columns` với payload `{registry_id, source_object_id, shadow_schema, target_table, source_table, primary_key_field, primary_key_type}`.
3. worker: `centralized-data-service/internal/handler/command_handler.go` `HandleCreateDefaultColumns` (line ~480) consume subject `cdc.cmd.create-default-columns`:
   - CREATE TABLE physical at line 586-602.
   - `ensureCDCColumnsInSchema` at line 149-179.
   - Auto-discovery via `scanFieldsDebezium` line 630.
   - `mappingV2Repo.GetActiveRulesBySourceTable(payload.SourceTable)` line 649 → ALTER ADD COLUMN per rule line 690.

## Phase B — Root Cause (DONE)

### Bug 1
File:line: `centralized-data-service/internal/repository/mapping_rule_v2_repo.go:54-61`
```go
func (r *MappingRuleV2Repo) GetActiveRulesBySourceTable(ctx context.Context, sourceTable string) ([]model.MappingRuleV2, error) {
    var items []model.MappingRuleV2
    err := r.db.WithContext(ctx).
        Joins("JOIN cdc_system.source_object_registry so ON cdc_system.mapping_rule_v2.source_object_id = so.id").
        Where("so.source_object_name = ? AND cdc_system.mapping_rule_v2.is_active = ? AND cdc_system.mapping_rule_v2.status = ?", sourceTable, true, "approved").
        Find(&items).Error
    return items, err
}
```
Defect: JOIN filter theo `source_object_name` (chuỗi nhiều khả năng trùng) thay vì identity `source_object_id`. Nếu cũ đã có `source_object_registry { id=42, source_object_name="export_jobs" }` với 12 mapping_rule_v2 approved, và user tạo registry mới `{ id=99, source_object_name="export_jobs", target_table="sd_export_jobs_1" }` → query này trả về cả 12 rules của id=42 → loop ALTER ADD COLUMN vào shadow `sd_export_jobs_1`. Đây là cross-entity bleed.

Caller: `command_handler.go:649` — truyền `payload.SourceTable` (string) thay vì `payload.SourceObjectID` (int64). Đã có sẵn API thay thế cùng repo: `ListActiveBySourceObject(ctx, sourceObjectID)` (line 37-44) filter bằng ID — không bị bleed.

### Bug 2
File:line A: `centralized-data-service/internal/handler/command_handler.go:586-602` (CREATE TABLE)
```go
createSQL := fmt.Sprintf(
    `CREATE TABLE IF NOT EXISTS %s.%s (
        %s %s PRIMARY KEY,
        _raw_data JSONB NOT NULL DEFAULT '{}'::jsonb,
        _source VARCHAR(20) NOT NULL DEFAULT 'debezium',
        _synced_at TIMESTAMP NOT NULL DEFAULT NOW(),
        _version BIGINT NOT NULL DEFAULT 1,
        _hash VARCHAR(64),
        _deleted BOOLEAN DEFAULT FALSE,
        _created_at TIMESTAMP DEFAULT NOW(),
        _updated_at TIMESTAMP DEFAULT NOW()
    )`, ...)
```
Defect: thiếu 3 cột system bắt buộc của Shadow Layer (per `project_context.md §Shadow Layer required cols`):
- `_source_ts BIGINT` (OCC older-wins anchor) — **gốc của toàn bộ guard cơ chế concurrency**.
- `_gpay_source_id TEXT UNIQUE` (V2 anchor, ON CONFLICT key cho master upsert).
- `_gpay_deleted BOOLEAN DEFAULT FALSE` (tombstone).

File:line B: `centralized-data-service/internal/handler/command_handler.go:149-179` (`ensureCDCColumnsInSchema`)
```go
cdcColumns := []struct{ name, def string }{
    {"_raw_data", "JSONB"},
    {"_source", "VARCHAR(20) DEFAULT 'debezium'"},
    {"_synced_at", "TIMESTAMP DEFAULT NOW()"},
    {"_version", "BIGINT DEFAULT 1"},
    {"_hash", "VARCHAR(64)"},
    {"_deleted", "BOOLEAN DEFAULT FALSE"},
    {"_created_at", "TIMESTAMP DEFAULT NOW()"},
    {"_updated_at", "TIMESTAMP DEFAULT NOW()"},
}
```
Defect: identical — cùng thiếu 3 cột.

Cross-check (chứng cứ rằng các path khác vẫn dùng `_source_ts`, tức là bug regression chỉ ở 2 chỗ trên):
- `sinkworker/schema_manager.go:231` — `"_source_ts" BIGINT` trong DDL builder của sinkworker.
- `sinkworker/upsert.go:69-122` — `EXCLUDED._source_ts > shadow._source_ts` OCC guard.
- `service/master_ddl_generator.go:92` — master DDL có `"_source_ts" BIGINT`.
- `service/transmuter.go:89` — `SourceTs int64 gorm:"column:_source_ts"`.

→ Shadow tạo qua FE `/shadow` route đi via `HandleCreateDefaultColumns` chứ KHÔNG đi qua `sinkworker.schema_manager`. Đây là path thứ 2 build DDL nhưng đã out-of-sync với required columns spec.

## Phase C — Đề xuất giải pháp (audit-only, không sửa code)

### Fix Bug 1 (minimal, root-cause)
Thay `GetActiveRulesBySourceTable(ctx, payload.SourceTable)` bằng `ListActiveBySourceObject(ctx, effectiveID)` tại `command_handler.go:649`. `effectiveID` đã được resolve ở line 620-623 = `payload.SourceObjectID` (V2) hoặc `payload.RegistryID` (legacy fallback).

Alternative (giữ tên method): thêm filter `source_object_id` vào WHERE — nhưng đã có sẵn API ID-based, swap caller là minimal.

### Fix Bug 2 (minimal, root-cause)
Bổ sung 3 cột vào CẢ HAI nơi build DDL trong `command_handler.go`:
- `_source_ts BIGINT`
- `_gpay_source_id TEXT UNIQUE` (UNIQUE constraint để ON CONFLICT key)
- `_gpay_deleted BOOLEAN DEFAULT FALSE`

Đồng thời thêm 2 index như sinkworker/schema_manager (line 231-260):
- `CREATE INDEX IF NOT EXISTS idx_<table>_source_ts ON shadow.<table>(_source_ts)` — phục vụ OCC sort.
- `_gpay_source_id` đã có UNIQUE constraint thì PG tự tạo index.

Chi tiết code demo: xem `09_tasks_solution_audit.md`.

## Non-goals của phase này
- KHÔNG migrate dữ liệu shadow đã tạo lỗi (sẽ là phase kế tiếp).
- KHÔNG refactor `HandleCreateDefaultColumns` thành dùng `sinkworker.schema_manager` (out-of-scope, là refactor lớn).
- KHÔNG sửa FE.

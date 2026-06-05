# 09 — Solution tasks (chờ user approve)

## Pre-flight (em sẽ verify nếu user approve)
- [ ] `grep "routeBySourceID" centralized-data-service/internal/service/*.go centralized-data-service/internal/handler/*.go` — confirm 3 usage đã liệt kê là toàn bộ.
- [ ] `grep "binding_id\|shadow_binding_id" cdc-cms-service/internal/api/registry_handler.go` (hoặc handler tương đương) — confirm `/api/v1/source-objects/registry/:id` đã hỗ trợ `?binding_id`.
- [ ] `grep "in_shadow\|InShadow\|is_in_shadow" cdc-cms-service/internal/api/dto/mapping_rule_dto.go cdc-cms-web/src/types/*` — confirm field "In Shadow" tồn tại trên DTO.
- [ ] `psql -c "SELECT COUNT(*) FROM cdc_system.mapping_rule_v2 WHERE shadow_binding_id IS NULL"` — đếm legacy rule chưa scoped (user run).

## Code tasks (sau khi user approve)

### T1 — FE Bug 1 (binding-aware routing)
- `cdc-cms-web/src/pages/TableRegistry.tsx:943`:
  - Đổi navigate sang `\`/shadow/${record.registry_id}/mappings${record.shadow_binding_id ? \`?binding_id=${record.shadow_binding_id}\` : ''}\``.
- `cdc-cms-web/src/pages/MappingFieldsPage.tsx`:
  - Import `useSearchParams`.
  - `const [searchParams] = useSearchParams(); const bindingId = searchParams.get('binding_id');`.
  - `fetchRegistry`: nếu `bindingId` → append `?binding_id=${bindingId}` vào URL.
  - `fetchRules`: dùng `bindingId ?? registry.shadow_binding_id`.
- (Tuỳ chọn) Thêm Tag UI hiển thị `binding_code` đang active.

### T2 — BE Bug 1 (registry endpoint + repo SQL)
- `cdc-cms-service/internal/api/.../source_object_registry get handler` (verify path):
  - Parse `?binding_id` query → resolve via `bridgeReader.ResolveDispatchScopeByBindingID` → trả về binding-scoped context (shadow_schema, target_table…).
- `cdc-cms-service/internal/infra/persistence/mapping_rule_repo_gorm.go`:
  - Khi `f.ShadowBindingID != nil && *f.ShadowBindingID > 0` → thêm `AND mr.shadow_binding_id IS NOT NULL` để loại legacy `NULL`-row leak.
  - Hoặc đổi JOIN clause loại bỏ branch OR-NULL khi filter binding cụ thể.

### T3 — FE Bug 2 (column source_data_type + Status/InShadow split)
- `cdc-cms-web/src/pages/MappingFieldsPage.tsx` columns array:
  - Thêm `{ title: 'Data Type source', dataIndex: 'source_data_type', render: v => v ?? '—' }`.
  - Verify cột "Status" render `record.status` (rule status), cột "In Shadow" render `record.is_in_shadow` (audit) — nếu DTO chưa có `is_in_shadow` → log gap, không tự generate.

### T4 — FE Bug 3 (hide Preview + Backfill)
- `cdc-cms-web/src/pages/MappingFieldsPage.tsx` action column:
  - Thêm const `const SHOW_LEGACY_ACTIONS = false;`.
  - Wrap 2 button: `{SHOW_LEGACY_ACTIONS && <Button ...>Preview</Button>}` + tương tự Backfill.
  - GIỮ NGUYÊN `handlePreview` + `handleBackfill` functions để có thể bật lại.

### T5 — Worker Bug 4.a (routeBySourceID slice-ify)
- `centralized-data-service/internal/service/metadata_registry_service.go`:
  - Line 163: `routeBySourceID := make(map[int64][]*ResolvedSourceRoute, len(sources))`.
  - Line 223: `routeBySourceID[src.ID] = append(routeBySourceID[src.ID], route)`.
  - Line 245 (B3 clone): cập nhật lookup loop để xử lý slice (clone source thường chỉ có 1 binding; loop slice là no-op overhead).
  - Line 262 (mapping_cache attach):
    ```go
    routesForSource := routeBySourceID[sourceID]
    if len(routesForSource) == 0 {
        rs.logger.Warn("V2 mapping rules have no shadow route", ...)
        continue
    }
    for _, route := range routesForSource {
        for _, v2 := range v2Rules {
            rs.mappingCache[route.TableConfig.TargetTable] = append(rs.mappingCache[route.TableConfig.TargetTable], convertV2ToLegacyRule(v2, src.SourceObjectName))
        }
    }
    ```
- Verify với grep không có usage ngoài.

### T6 — Worker Bug 4.b (err taxonomy + DB cross-check)
- `centralized-data-service/internal/handler/snapshot_runner_handler.go` đoạn `if scopedBinding != nil && !hit` (vừa thêm phiên trước):
  - Trước khi `markProgressError`, query DB `SELECT is_active FROM cdc_system.shadow_binding WHERE id=?` → phân biệt:
    - Active nhưng cache miss → `err_type=registry_reload_silent_drop`.
    - Inactive/missing → `err_type=binding_inactive`.
  - Log tech depth `component=snapshot_runner op=run_snapshot phase=route_resolve err_type=<X> shadow_binding_id=N`.

### T7 — Verify
- `cd cdc-cms-service && go build ./... && go test -count=1 -short ./...`
- `cd centralized-data-service && go build ./... && go test -count=1 -short ./...`
- `cd cdc-cms-web && npm run build`
- Manual smoke trên dev (user chạy): visit `/shadow/<so>/mappings?binding_id=<b1>` rồi `?binding_id=<b2>` → 2 set rule độc lập; trigger snapshot binding 4 → log SigNoz `phase=route_resolve err_type=ok` + ghi đúng shadow table.

## Documentation (sau khi apply)
- Append `agent/memory/global/lessons.md` — Lesson: "Multi-binding entity context: FE route + BE registry cache + repo SQL filter PHẢI đồng nhất key (sourceID, bindingID); scalar map keyed source_id OVERWRITES per binding gây silent corruption".
- Append `bug-shadow-mapping-rules-2026-05-29/05_progress.md` với delta code + verify result.

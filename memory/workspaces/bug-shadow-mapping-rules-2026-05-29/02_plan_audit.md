# 02 — Plan (Fix 4 bug)

## Decision matrix

| Option | Mô tả | Risk | Khuyến nghị |
|--------|-------|------|-------------|
| 1. Route `/shadow/:registryId/:bindingId/mappings` | URL path mang bindingId | Trung — đổi nav + route param | ✅ Đề xuất |
| 2. Query string `?binding_id=X` trên route hiện tại | Giữ path cũ, append query | Thấp — không break bookmark legacy | ✅ Đề xuất (đơn giản hơn) |
| 3. Server-side default = ambiguous → 409 | BE trả 409 khi thiếu binding_id mà source >1 binding | Thấp | ✅ Phòng tuyến cuối (đã có pattern ở SnapshotV2) |

**Chọn Option 2 + 3**: ít invasive nhất, bookmark legacy vẫn redirect, BE fail-loud khi thiếu binding_id.

## Phase plan

### Phase A — Bug 1 (FE binding-aware routing)
A1. `cdc-cms-web/src/pages/TableRegistry.tsx:943`:
   - Đổi `navigate(\`/shadow/${record.registry_id}/mappings\`)` → `navigate(\`/shadow/${record.registry_id}/mappings${record.shadow_binding_id ? \`?binding_id=${record.shadow_binding_id}\` : ''}\`)`.
A2. `cdc-cms-web/src/pages/MappingFieldsPage.tsx`:
   - Đọc `binding_id` từ `useSearchParams()` → state `bindingId`.
   - Khi fetch registry: `cmsApi.get(\`/api/v1/source-objects/registry/${id}${bindingId ? \`?binding_id=${bindingId}\` : ''}\`)` → BE trả binding-scoped context.
   - Khi fetch rules: nếu `bindingId` có → dùng `bindingId`; nếu không → dùng `registry.shadow_binding_id` (legacy).
A3. `cdc-cms-service/internal/api/.../source_object_registry...` (registry getter):
   - Verify đã hỗ trợ `?binding_id`. Nếu chưa → thêm parse + filter (giống pattern `parseBindingIDQuery`).
A4. BE list mapping rules — verify behavior khi FE gửi `binding_id`: chỉ trả rule có `shadow_binding_id = bindingId`. **KHÔNG trả** rule `shadow_binding_id IS NULL` để tránh leak. Update SQL JOIN: nếu filter `ShadowBindingID != nil` → bỏ branch `OR (mr.shadow_binding_id IS NULL AND sb.source_object_id = ...)` hoặc thêm `AND mr.shadow_binding_id IS NOT NULL` ở WHERE.

### Phase B — Bug 2 (FE display source_data_type + Status split)
B1. `MappingFieldsPage.tsx` table columns:
   - Thêm column `{ title: 'Data Type source', dataIndex: 'source_data_type', render: v => v ?? '—' }`.
   - Tách "Status" (rule status: pending/approved/rejected) và "In Shadow" (audit). Verify field name backend trả về cho "In Shadow" (có thể là `is_in_shadow` hoặc compute từ probe). Nếu chưa có → ghi TODO `10_gap_analysis` thay vì tự thêm endpoint.
B2. Verify scan path persist `source_data_type` cho binding mới — test bằng smoke (user trigger scan, rồi check DB).

### Phase C — Bug 3 (Hide Preview + Backfill)
C1. `MappingFieldsPage.tsx` action column:
   - Wrap 2 button trong `{false && (...)}` hoặc thêm flag `const SHOW_LEGACY_ACTIONS = false` rồi `{SHOW_LEGACY_ACTIONS && <Button>Preview</Button>}`.
   - KHÔNG xoá `handlePreview` + `handleBackfill` functions.
C2. Type check: TSC PASS. Eslint no-unused-vars: ignore với `// eslint-disable-next-line @typescript-eslint/no-unused-vars` nếu cần.

### Phase D — Bug 4 (Worker registry routeBySourceID + reload taxonomy)
D1. `centralized-data-service/internal/service/metadata_registry_service.go`:
   - Đổi `routeBySourceID` từ `map[int64]*ResolvedSourceRoute` → `map[int64][]*ResolvedSourceRoute`.
   - Line 223: `routeBySourceID[src.ID] = append(routeBySourceID[src.ID], route)`.
   - Line 245 (B3 clone): `cloneRoutes, hasClone := routeBySourceID[src.ID]` → giờ là slice → loop append all clones.
   - Line 262 (mapping_cache attach): loop từng route, attach mapping_cache vào MỖI binding's target_table:
     ```go
     routesForSource := routeBySourceID[sourceID]
     for _, route := range routesForSource {
         targetTable := route.TableConfig.TargetTable
         for _, v2 := range v2Rules {
             rs.mappingCache[targetTable] = append(rs.mappingCache[targetTable], convertV2ToLegacyRule(v2, src.SourceObjectName))
         }
     }
     ```
D2. `snapshot_runner_handler.go` post-reload check (nâng cấp err taxonomy):
   - Sau pre-flight reload, nếu `scopedBinding != nil` mà filter `ResolveSourceRoutes` không có binding → query DB `shadow_binding WHERE id=? AND is_active=true` để phân biệt:
     - DB có active → registry stale (race) → `err_type=registry_reload_silent_drop`.
     - DB inactive/missing → `err_type=binding_inactive`.
   - Log tech depth `component=snapshot_runner op=run_snapshot phase=route_resolve err_type=<X>`.
D3. Verify activate workflow: `cdc-cms-service/internal/app/commands/.../create_shadow_binding.go` (hoặc tương đương) — bind insert có set `is_active=true` mặc định không. Nếu workflow user tạo binding thông qua "Create + Activate" 2 step → ghi note vào `01_requirements` rằng FE/UX cần guard trigger snapshot nếu binding inactive.

### Phase E — Verify
E1. Build:
   - `cd cdc-cms-service && go build ./...`
   - `cd centralized-data-service && go build ./...`
   - `cd cdc-cms-web && npm run build` (hoặc `tsc -b`).
E2. Test:
   - `go test -count=1 -short ./...` 2 service → PASS các test tracked.
   - FE: chạy `vite build` + manual smoke screen `/shadow/<so>/mappings?binding_id=<b>`.
E3. Smoke (user deploy):
   - POST `/api/v1/source-objects/<so>/scan-fields?binding_id=<b>` → rule mới có `source_data_type` non-null.
   - Visit FE `/shadow/<so>/mappings?binding_id=<b1>` + `?binding_id=<b2>` → 2 set rule độc lập, không leak.
   - POST snapshot-v2 `?binding_id=<b2>` → route resolve OK, ghi đúng shadow table b2.

## Risks
- A4 — sửa baseSelect SQL có thể break test/fixture cũ. Cần grep test trước khi sửa.
- D1 — `routeBySourceID` overwrite từng được code khác trong worker dựa vào (sourceCache, debeziumTables). Cần grep usage ngoài 3 line đã xác định.
- B2 — Field "In Shadow" có thể chưa có endpoint chuẩn → ghi gap, không tự thêm.

## Pre-flight check trước khi APPLY
1. User approve Plan này.
2. Confirm migration 067 đã apply prod (Bug 2 schema phụ thuộc).
3. Confirm `routeBySourceID` không có usage ngoài 3 dòng đã liệt kê (grep).
4. Confirm `source_object_registry` endpoint có hỗ trợ `?binding_id` hay chưa.

## Out of scope (note nhưng không fix lần này)
- Backfill historical `mapping_rule_v2.shadow_binding_id` cho rule đã tồn tại (nếu NULL nhiều). Sẽ tách workspace riêng nếu user yêu cầu.
- Refactor registry cache key sang `(sourceID, bindingID)` toàn bộ (large refactor).
- "In Shadow" audit endpoint nếu chưa có → ghi gap.

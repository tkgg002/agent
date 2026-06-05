# 03 — Implementation Audit (Evidence + Root Cause per bug)

## Bug 1 — Mapping Rules Leak

### Evidence
- API BE đã đúng:
  - `cdc-cms-service/internal/api/mapping_rule_handler_list.go:32` — parse `c.Query("binding_id", c.Query("shadow_binding_id"))` → `Filter.ShadowBindingID`.
  - `cdc-cms-service/internal/infra/persistence/mapping_rule_repo_gorm.go:139-142` — `if f.ShadowBindingID != nil && *f.ShadowBindingID > 0 { q += " AND mr.shadow_binding_id = ?" }`.
- Migration đã có: `cdc-cms-service/migrations/schema/core/067_add_mapping_rule_v2_binding_and_source_type.sql:5` — `ADD COLUMN IF NOT EXISTS shadow_binding_id BIGINT REFERENCES cdc_system.shadow_binding(id)`.
- **BUG NẰM Ở FE**:
  - `cdc-cms-web/src/pages/TableRegistry.tsx:943` — `navigate(\`/shadow/${record.registry_id}/mappings\`)` truyền **source_object_id** chứ KHÔNG truyền binding_id.
  - `cdc-cms-web/src/App.tsx:198` — route `/shadow/:id/mappings` — `:id` thực chất là source_object_id.
  - `cdc-cms-web/src/pages/MappingFieldsPage.tsx:66` — `const { id } = useParams<{ id: string }>()` → id là source_object_id.
  - `MappingFieldsPage.tsx:157` — `cmsApi.get<SourceObjectMappingContext>(\`/api/v1/source-objects/registry/${id}\`)` → trả về MỘT binding context default (binding active đầu tiên/hoặc binding hiện hành theo logic backend).
  - `MappingFieldsPage.tsx:168` — fetch rules với `shadow_binding_id: registry.shadow_binding_id` → là binding default, KHÔNG phải binding mà user click.

### Root Cause
- Route FE thiếu binding-aware: cả 2 row của source 110 (binding 110 + 112) cùng deep-link tới `/shadow/110/mappings`. MappingFieldsPage không phân biệt được click row nào → resolve binding default → list rules theo binding default.

### Bug surface phụ
- BaseSelect JOIN `LEFT JOIN cdc_system.shadow_binding sb ON (mr.shadow_binding_id IS NOT NULL AND sb.id = mr.shadow_binding_id) OR (mr.shadow_binding_id IS NULL AND sb.source_object_id = mr.source_object_id AND sb.is_active = TRUE)` ở `mapping_rule_repo_gorm.go:125-127`. Khi `mr.shadow_binding_id IS NULL` (legacy/system_default rule) → JOIN match FIRST active binding của source → row legacy có thể fan-out per active binding gây "leak". Cần audit dataset thực để xác định có bao nhiêu rule `shadow_binding_id IS NULL`.

---

## Bug 2 — Source Data Type + Status logic

### Evidence
- DB schema OK: migration 067 `ADD COLUMN IF NOT EXISTS source_data_type VARCHAR(100)`.
- Domain OK: `cdc-cms-service/internal/domain/mapping/rule.go:50` — `SourceDataType *string`.
- Persistence OK: `mapping_rule_repo_gorm.go:51,80,111` — đọc/ghi `source_data_type`.
- DTO OK: `cdc-cms-service/internal/api/dto/mapping_rule_dto.go:25,55,92` — JSON tag `"source_data_type"`.
- Scan worker OK: `centralized-data-service/internal/handler/command_handler.go:1852,1973` — `SourceDataType: &sourceType` trong insert payload.
- Command create OK: `cdc-cms-service/internal/app/commands/create_mapping_rule.go:34,94,151,172,307` — wire `source_data_type` end-to-end.

### Root Cause
- BE 100% ready. **FE chưa render**: cần kiểm tra `MappingFieldsPage.tsx` table columns có cột "Data Type source" chưa.
- Status logic: cần xem table column "Status" hiện đang render gì. Nếu render 1 field cho cả Status + In Shadow → cần tách.

### Audit FE table render (chưa đọc — sẽ verify trong plan)
- Lệnh kiểm: `grep -n "source_data_type\|status\|in_shadow\|inShadow" cdc-cms-web/src/pages/MappingFieldsPage.tsx`.

---

## Bug 3 — Hide Preview + Backfill

### Evidence
- `cdc-cms-web/src/pages/MappingFieldsPage.tsx:98` — `handlePreview` gọi `/api/v1/mapping-rules/preview`.
- `MappingFieldsPage.tsx:271` — `handleBackfill` gọi `/api/mapping-rules/${ruleId}/backfill`.
- JSX render 2 button trong cột Action (chưa locate exact line — sẽ verify trong plan).

### Root Cause
- FE-only. Hide bằng condition flag hoặc comment JSX. Giữ handler functions.

---

## Bug 4 — Snapshot V2 Registry Lookup Fail

### Evidence
- Worker error format `shadow_binding_id=4 not in active registry routes for source_db=wallet-service source_collection=wallet-capsets — registry may be stale or binding inactive` đến từ code MÌNH vừa thêm ở `centralized-data-service/internal/handler/snapshot_runner_handler.go` (workspace `snapshot_v2_multi_binding/05_progress.md` ghi đoạn `markProgressError` với chính format này).
- `centralized-data-service/internal/service/metadata_registry_service.go`:
  - Line 154: `routeCache map[string][]*ResolvedSourceRoute` — **slice per sourceKey, đúng**. Mỗi binding append vào slice (line 229).
  - Line 163: `routeBySourceID map[int64]*ResolvedSourceRoute` — **scalar per sourceID, sai**: line 223 `routeBySourceID[src.ID] = route` overwrite per binding loop → chỉ giữ binding cuối.
  - `routeBySourceID` dùng cho:
    - Line 245: `cloneRoute, hasClone := routeBySourceID[src.ID]` — B3 clone fan-out. Vì clone là source_id KHÁC master, nên overwrite chỉ ảnh hưởng khi 1 source vừa là clone vừa có nhiều binding (hiếm).
    - Line 262: `route := routeBySourceID[sourceID]` — attach mapping rules vào target_table của route. **HERE BUG IMPACT**: chỉ binding cuối nhận mapping_cache, binding đầu mất mapping_cache → dynamic mapper rỗng cho target_table binding đầu → snapshot binding đó silent-skip.
- `ResolveSourceRoutes` (line 558-580) đọc `routeCache[sourceKey]` slice → trả về ĐÚNG cả 2 binding. **Vậy filter binding_id của fix snapshot v2 multi-binding sẽ THẤY binding 4 nếu reload đã chạy**.
- **Nguyên nhân thực tế của error `binding_id=4 not in active registry routes`**:
  1. Cache stale tại thời điểm snapshot trigger: binding 4 vừa insert, `ReloadAll` chưa fire.
  2. Snapshot runner pre-flight reload (`snapshot_runner_handler.go:286-294`) đã tồn tại sau fix trước. Nếu reload PASS mà vẫn miss → binding 4 có `is_active=false` (user chưa activate sau khi tạo) → `ReloadAll` loop filter `if item.IsActive` (line 125) bỏ qua.
  3. Hoặc binding 4 có `is_active=true` nhưng DB read replica lag (nếu `shadowRepo.ListBySourceObject` đi qua replica).
- `routeBySourceID` overwrite KHÔNG ảnh hưởng trực tiếp đến error này (vì `ResolveSourceRoutes` dùng `routeCache` slice). NHƯNG nó gây bug song song: mapping_cache mất cho binding đầu → snapshot binding đầu run thành công về schema nhưng tất cả field mapping NULL → silent corruption.

### Root Cause (đa nguyên nhân)
- **RC4.a**: `routeBySourceID` overwrite per binding → mapping_cache chỉ attach cho 1 binding/source → binding khác mất mapping rules. **Fix**: đổi sang `map[int64][]*ResolvedSourceRoute` hoặc keyed `(sourceID, bindingID)`.
- **RC4.b**: Snapshot trigger trước khi `ReloadAll` thấy binding mới. **Fix**: pre-flight reload đã có; thêm cross-check "binding has is_active=true in DB nhưng không có trong cache" → trả lỗi rõ ràng `err_type=registry_reload_silent_drop` thay vì generic.
- **RC4.c**: Activate workflow có thể chưa set `is_active=true` đúng cho binding mới — cần verify (xem command handler create shadow_binding).

---

## Tổng hợp bug surface (đã verify)

| Bug | Layer | File:Line | Fix scope |
|-----|-------|-----------|-----------|
| 1 | FE | `cdc-cms-web/src/App.tsx:198`, `MappingFieldsPage.tsx:157,168`, `TableRegistry.tsx:943` | Route + nav + fetch — pass binding_id |
| 2 | FE | `MappingFieldsPage.tsx` table columns | Add column "Data Type source" + split Status/InShadow |
| 3 | FE | `MappingFieldsPage.tsx` action column | Hide 2 buttons, keep handlers |
| 4.a | Worker | `centralized-data-service/internal/service/metadata_registry_service.go:163,223,245,262` | `routeBySourceID` → slice/keyed by binding |
| 4.b | Worker | `centralized-data-service/internal/handler/snapshot_runner_handler.go` post-reload check | DB cross-check + err_type taxonomy |
| 4.c | CMS | `cdc-cms-service/internal/app/commands/.../shadow_binding...` | Verify is_active=true at create time |

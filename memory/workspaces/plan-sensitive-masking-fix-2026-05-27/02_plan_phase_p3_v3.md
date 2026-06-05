# Phase P3.3 — V3: Array flatten → child shadow tables

## Context
- User issue 3: array nested fields (`orgs[]` trong MongoDB doc) hiện đang flatten kiểu cũ (JSON encode + đẩy 1 cột) → mất khả năng query con.
- Yêu cầu: flatten array thành **child shadow table riêng**, mỗi element = 1 row.

## Design (elegant — pattern: explode là property của BINDING, không phải rule)

### Schema (migration 071_add_explode_to_shadow_binding.sql)
Bổ sung 2 cột vào `cdc_system.shadow_binding`:
- `parent_binding_id BIGINT REFERENCES shadow_binding(id) ON DELETE CASCADE` — null = root binding; not-null = child explode binding.
- `explode_path TEXT` — JSONPath relative-to-parent event, ví dụ `$.orgs[*]` hoặc `orgs`. Required khi `parent_binding_id IS NOT NULL`.
- CHECK: `(parent_binding_id IS NULL AND explode_path IS NULL) OR (parent_binding_id IS NOT NULL AND explode_path IS NOT NULL)`.
- Index `(parent_binding_id)` cho lookup.

Lợi điểm pattern này:
- Mapping rule không cần cột mới — rule của child binding chỉ trỏ `shadow_binding_id` về child; `source_field` đã sẵn có để pick column từ array element.
- Re-use toàn bộ ports/handlers/repos hiện tại cho mapping_rule.
- Provisioner sống cùng cấu trúc cũ — chỉ thêm 2 system column (`_parent_source_id`, `_array_index`) khi binding là child.

### Provisioner (cdc-cms-service)
- `ProvisionShadowTable`: detect `binding.ParentBindingID != nil` → schema kèm system column:
  - `_parent_source_id TEXT NOT NULL`
  - `_array_index INTEGER NOT NULL`
  - PK = `(_parent_source_id, _array_index)`
  - Khác bỏ `_source_id` UNIQUE (child không cần OCC anchor riêng — parent xoá → child cascade DELETE).
- Phần còn lại (columns từ mapping rules) giữ nguyên.

### CDS pipeline
- `MetadataRegistryService` cache thêm: `childBindings[parentBindingID][]ChildBindingMeta{ID, ExplodePath, ShadowTable, Rules}`.
- `DynamicMapper`:
  - New `ExplodeChildren(event map[string]interface{}, parentSourceID string, parentBindingID int64) []ChildRows` — với mỗi child binding, dùng JSONPath đọc array từ event, iterate, apply rules để build N rows. Mỗi row có sẵn `_parent_source_id` + `_array_index`.
- Event handler (chỗ insert vào shadow): sau khi upsert parent thành công, gọi `ExplodeChildren`, emit child rows vào batch buffer riêng (key = child shadow table).

### Frontend
- Trang Mapping Fields (cha): thêm panel "Child bindings (array explode)" — list child + button "Create child binding" mở modal (shadow_table + explode_path + connection_id).
- Trong child binding → click vô để vào MappingFieldsPage của child (recursive UI), thêm mapping rule như bình thường.

## Scope cắt cho MVP V3 (theo nguyên tắc Simplicity First nhưng KHÔNG drop chức năng):
- KHÔNG hỗ trợ nested explode đệ quy (child có child) — schema cho phép (parent_binding_id self-ref), runtime chưa loop; thêm sau khi có usecase 2.
- JSONPath lib: dùng `github.com/PaesslerAG/jsonpath` (đã có trong go.sum của CDS? Check trước khi import; nếu không có dùng tiny inline `$.field[*]` parser).
- Provisioner ALTER online: chưa cần — Phase P3.3 chỉ tạo child mới, không refactor parent column hiện hữu.

## Impact files (12 file Go + 2 file React)
**cms-service:**
1. `migrations/schema/core/071_add_explode_to_shadow_binding.sql` (NEW)
2. `internal/domain/source/binding.go` hoặc `shadow_binding.go` — domain (nếu có)
3. `internal/model/shadow_binding.go` (nếu có)
4. Wherever shadow_binding gorm row is defined — add ParentBindingID + ExplodePath
5. Wherever ProvisionShadowTable lives — child-table branch
6. Handler/route create-child-binding (nếu chưa có endpoint create binding)

**cds:**
7. `internal/model/shadow_binding.go` — add ParentBindingID + ExplodePath
8. `internal/service/metadata_registry_service.go` — cache child bindings under parent
9. `internal/service/dynamic_mapper.go` — `ExplodeChildren` method
10. Pipeline insert: locate điểm sau parent upsert; emit child rows
11. Possibly `internal/service/schema_inspector.go` if it has provisioner logic

**web:**
12. `src/types/index.ts` — ShadowBinding type + parent_binding_id/explode_path
13. `src/pages/MappingFieldsPage.tsx` — panel + modal

## DoD
- Migration 071 chạy clean.
- `go build ./...` PASS cả 2 service.
- `npm run build` PASS.
- Document hành vi mới trong workspace.
- Smoke test bằng cách insert 1 binding child manually + 1 mapping rule child → verify CDS không panic.

## Ghi chú
- Default: V3 không tự suggest explode. Operator chủ động tạo child binding khi muốn flatten.
- Backward-compat 100%: parent_binding_id NULL = behavior cũ.

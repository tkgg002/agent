# 00_context — Fix FE shadow_schema (Architectural)

## Triệu chứng
- URL `/shadow/15/mappings?binding_id=59` gọi `GET /api/introspection/shadow-columns/tokens?schema=shadow_auth_service` → `{columns:null, schema:"shadow_auth_service", table:"tokens"}`.
- Schema `shadow_auth_service` **không tồn tại** trong Postgres.
- Schema đúng (theo DB): `shadow_goopay_test_local_as_auth_service` (lấy từ `shadow_binding.id=59.shadow_schema`).

## Evidence (đã verify qua SQL Q1+Q2+Q3 do user cung cấp)
- `shadow_binding.id=59` → `shadow_schema = "shadow_goopay_test_local_as_auth_service"`, `source_object_id = 59`, `shadow_table = "tokens"`.
- `cdc_table_registry.id=15` → `source_db = "auth-service"`, `source_table = "tokens"`.
- `source_object_registry.id=59` → `source_database = "auth-service"`, `source_object_name = "tokens"`.
- JOIN BE phải MATCH cả 2 LEFT JOIN (registry-source_object) và LATERAL (shadow_binding).

## Root cause (giả thuyết)
1. **FE compose schema sai**: `MappingFieldsPage.tsx:213` có fallback `registry.shadow_schema || normalizeShadowSchema(registry.source_db)`.
   - `normalizeShadowSchema("auth-service")` → `"shadow_auth_service"` (xoá dấu `-` thành `_`).
   - DB thực tế: `"shadow_goopay_test_local_as_auth_service"` (BE V2 register dùng `shadow_<connection_code>_<db>`).
   - 2 thuật toán KHÁC NHAU → FE compose ra schema không tồn tại.
2. **Tại sao fallback bị trigger?** → `registry.shadow_schema` đang `undefined` trong response BE.
   - BE struct `ShadowSchema *string` + tag `json:"shadow_schema,omitempty"`.
   - Nếu LATERAL `sb` không match → `sb.shadow_schema = NULL` → `*string = nil` → JSON **omit field**.
   - Hoặc trường có giá trị nhưng GORM scan không bind đúng (low prob).

## Architectural directive (user)
> "sửa mẹ gì 10 chỗ. sao mang logic ghép name shadow vào fe. bị ngu à. phải lấy từ api chứ"

- FE TUYỆT ĐỐI KHÔNG compose `shadow_schema`.
- BE là **single source of truth**.
- Fix = xoá `normalizeShadowSchema` toàn bộ FE + đảm bảo BE luôn trả `shadow_schema` non-null.

## Inventory `normalizeShadowSchema` trong FE
- **4 file định nghĩa**:
  - `src/pages/TableRegistry.tsx:75-81`
  - `src/pages/MappingFieldsPage.tsx:14-17`
  - `src/pages/DataIntegrity.tsx:56-62`
  - `src/pages/ActivityManager.tsx:65` (inline arrow)
- **13 callsite fallback `|| normalizeShadowSchema(...)`**:
  - TableRegistry.tsx: 85, 780, 884
  - MappingFieldsPage.tsx: 24, 112, 198, 213, 357, 372, 572
  - DataIntegrity.tsx: 65, 81
  - ActivityManager.tsx: 135, 138, 176

## Anchors BE liên quan
- Query: `internal/infra/persistence/source_object_read_repo_gorm.go:189-284` (`GetMappingContextByRegistryID`).
- Model: `internal/app/queries/source_objects_read_models.go:61-89` (`SourceObjectMappingContextReadModel`).
- Handler: `internal/api/source_objects_handler.go:371-396` (`GetMappingContext`).
- BE V2 register (đúng pattern): `internal/infra/persistence/source_object_v2_sync.go:453` (`normalizeShadowSchemaWithConnection`).

## Scope
- IN: cleanup FE `normalizeShadowSchema`, BE đảm bảo `shadow_schema` luôn present.
- OUT: KHÔNG đổi naming convention BE; KHÔNG migrate dữ liệu; KHÔNG đụng `shadow_binding` migration.

## Plan-only
Theo CLAUDE.md §12, Brain chỉ tạo doc set + chờ user duyệt → Muscle thực thi.

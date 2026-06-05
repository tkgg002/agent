# 01_requirements — Fix FE shadow_schema (Architectural)

## R1 — FE tuyệt đối không compose `shadow_schema`
- Xoá function `normalizeShadowSchema` ở **mọi nơi** trong `cdc-cms-web/src/`.
- Xoá toàn bộ pattern `|| normalizeShadowSchema(...)` ở các callsite.
- FE đọc `registry.shadow_schema` / `record.shadow_schema` / `row.shadow_schema` trực tiếp từ API.

## R2 — BE luôn trả `shadow_schema` non-null khi có `shadow_binding`
- Endpoint `GET /api/v1/source-objects/registry/:registry_id?binding_id=...` phải:
  - Trả `shadow_schema = sb.shadow_schema` (giá trị thực từ DB).
  - Nếu binding tồn tại nhưng schema null trong DB → vẫn trả string rỗng `""` (không omit).
  - Nếu binding không match → trả 404 thay vì 200 với `shadow_schema` null.

## R3 — Types FE phản ánh contract đúng
- `SourceObjectMappingContext.shadow_schema` đổi từ `string | null | undefined` → `string` (required, có thể `""`).
- `TRegistry.shadow_schema` đồng bộ.
- Các type khác (`ReconReport`, etc.) giữ `string | null` vì có thể null khi chưa register binding.

## R4 — UI fallback khi `shadow_schema` thực sự rỗng
- Khi BE trả `shadow_schema = ""`:
  - Hiển thị placeholder `"(chưa có)"` thay vì compose sai.
  - Disable nút "Mapping Fields" / "Recon" cho row đó.
  - Log warn ở console: `shadow_schema empty for registry_id=X` (debug only).

## A1-A5 — Acceptance criteria

| ID | Criterion | Verify |
|----|-----------|--------|
| A1 | Code `cdc-cms-web/src/` không còn `normalizeShadowSchema` | `grep -r normalizeShadowSchema src/` empty |
| A2 | `npx tsc --noEmit -p tsconfig.app.json` EXIT=0 | Type check pass |
| A3 | Reload `/shadow/15/mappings?binding_id=59` → URL `shadow-columns/tokens?schema=shadow_goopay_test_local_as_auth_service` | DevTools Network |
| A4 | Response `/api/v1/source-objects/registry/15?binding_id=59` chứa `"shadow_schema":"shadow_goopay_test_local_as_auth_service"` | curl/jq |
| A5 | Row /shadow chưa có binding → button mapping disabled, không crash | Manual smoke |

## N1 — Non-functional
- Zero breaking change cho consumer khác của API.
- Không thay đổi hành vi register flow.

## N2 — Backward compat
- Nếu BE chưa deploy version mới → FE vẫn không crash (đọc `string | undefined` → guard bằng `if (!shadow_schema) return`).
- Type field `shadow_schema?: string` (optional `?:`) — không bắt buộc trong tất cả model FE cũ.

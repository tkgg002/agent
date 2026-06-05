# Report — Shadow Page Connector Display

**Phase**: fe-api-worker-action-tracer-2026-05-18 / shadow_connector_display
**Date**: 2026-05-19
**Status**: ✅ CODE COMPLETE — builds + tests PASS — chờ user restart 2 service và mở `/shadow`.

## TL;DR (1 đoạn)

User báo `/shadow` panel header chỉ hiện `Source Database: centralized-export-service · 1 objects` — thiếu connector. Đã mở rộng read-side projection (V2 `source_object_registry` list + `shadow_binding` list) JOIN `connection_registry` để trả về `source_connection_id` + `source_connection_code` per row. FE `TableRegistry.tsx` đổi grouping key từ `source_db` sang `${connection_code}::${source_db}`, panel header hiển thị 2 chunk "Connector" + "Source DB", thêm column "Connector" vào cả Shadow Objects và Shadow Bindings table. 5 file thay đổi. CMS test toàn bộ 9 package PASS, worker build PASS, FE typecheck/lint không có error mới ngoài pre-existing.

## Files đã thay đổi (5 file)

### Backend Go (3 file)

```
cdc-cms-service/internal/app/queries/source_objects_read_models.go
  + Line 18-25: Thêm 2 field SourceConnectionID (*int64) + SourceConnectionCode (string) vào struct SourceObjectListItem; reformat indentation cho 4 field đầu để align.

cdc-cms-service/internal/infra/persistence/source_object_read_repo_gorm.go
  + Line 51-52: Thêm LEFT JOIN cdc_system.connection_registry cn ON cn.id = so.source_connection_id
    (giữa JOIN shadow_binding và JOIN cdc_table_registry).
  + Line 114-115: Projection thêm so.source_connection_id, COALESCE(cn.connection_code, '') AS source_connection_code.
  ~ Line 154: ORDER BY đổi thành COALESCE(cn.connection_code, ''), so.source_database, so.source_object_name (stable ordering per connector).

cdc-cms-service/internal/api/source_objects_handler.go
  + Line 59-68: Mở rộng struct ShadowBindingRow thêm SourceConnectionID (*int64) + SourceConnectionCode (string) + reformat alignment.
  + Line 273-274: SQL baseWhere thêm LEFT JOIN cdc_system.connection_registry cn ON cn.id = so.source_connection_id.
  + Line 316-317: Projection thêm so.source_connection_id, COALESCE(cn.connection_code, '') AS source_connection_code.
  ~ Line 333: ORDER BY thêm COALESCE(cn.connection_code, '') vào đầu.
```

### FE TS (2 file)

```
cdc-cms-web/src/types/index.ts
  + Line 58-61: SourceObjectRow thêm source_connection_id?: number|null, source_connection_code?: string|null.
  + Line 98-99: ShadowBindingRow thêm source_connection_id?: number|null, source_connection_code?: string|null.

cdc-cms-web/src/pages/TableRegistry.tsx
  ~ Line 297-326: groupedData + groupedBindings dùng key = `${conn}::${db}` thay vì chỉ db. Thêm helper splitGroupKey(key) → [conn, db].
  + Line 613-619: columns thêm cột "Connector" (dataIndex source_connection_code) trước Source DB.
  + Line 766-772: bindingColumns thêm cột "Connector" tương tự.
  ~ Line 789-820: Shadow Objects panel header refactor render — hiển thị tag "Connector: <code>" + tag "Source DB: <db>" + tag "<n> objects".
  ~ Line 826-857: Shadow Bindings panel header refactor cùng pattern + tag "<n> bindings".
```

### Workspace docs (2 file — đã tạo file vật lý)

```
agent/memory/workspaces/fe-api-worker-action-tracer-2026-05-18/
  02_plan_shadow_connector_display.md  (NEW — plan + code demo)
  report_shadow_connector_display.md   (NEW — file này)
  05_progress.md                       (APPEND-only, 5 dòng mới)
```

## Verification kết quả thật

| # | Command | Output |
|---|---|---|
| 1 | `cd cdc-cms-service && go build ./...` | EXIT=0, no stderr |
| 2 | `cd cdc-cms-service && go vet ./...` | EXIT=0, no stderr |
| 3 | `cd cdc-cms-service && go test -count=1 ./...` | PASS toàn bộ — api 0.921s, commands 1.588s, queries 0.670s, infra/http 0.416s, infra/messaging 1.128s, infra/observability 1.359s, infra/observability/probes 1.815s, infra/persistence 1.852s, middleware 1.652s. Không có FAIL. |
| 4 | `cd centralized-data-service && go build ./...` | EXIT=0 (sanity — no worker changes) |
| 5 | `cd centralized-data-service && go vet ./...` | EXIT=0 |
| 6 | `cd cdc-cms-web && npx tsc --noEmit -p tsconfig.app.json` | 3 errors pre-existing: `Upload`/`UploadOutlined` unused imports (lines 2-3), `handleBulkImport` unused var (line 549). KHÔNG có error mới ở vùng edit. |
| 7 | `cd cdc-cms-web && npx eslint src/pages/TableRegistry.tsx src/types/index.ts` | 6 errors pre-existing (react-hooks/set-state-in-effect line 100+135, no-empty line 200, unused handleBulkImport line 549). 0 errors mới. |

## What's NOT done (out of scope)

1. **Backfill `source_connection_id` cho legacy V2 rows**: Nếu V2 `source_object_registry` rows trước migration 054 có `source_connection_id IS NULL`, panel sẽ group dưới `(unassigned)`. User cần re-register hoặc backfill bằng SQL `UPDATE` theo first-wins (giống migration 055 nhưng cho V2). Out of scope vì user chỉ yêu cầu fix display.
2. **FE Mapping context page**: `MappingFieldsPage` cũng cần expose `connection_code` nếu UX muốn show connector ở trang chi tiết — chưa đụng vì user chỉ yêu cầu `/shadow`.
3. **Worker cache**: Không liên quan; worker chỉ subscribe; không touch.

## DoD checklist

- [x] 02_plan_shadow_connector_display.md tạo thật, có code demo chi tiết cho từng file.
- [x] Backend Go: 3 file thay đổi đúng theo plan, JOIN thêm `connection_registry`, projection bổ sung 2 trường.
- [x] FE TS: `SourceObjectRow` + `ShadowBindingRow` thêm field.
- [x] FE TSX: grouping key 2 chiều, panel header 2 chunk, column "Connector" cả 2 bảng.
- [x] CMS go build + go vet + go test ALL packages PASS.
- [x] Worker go build + go vet PASS (sanity).
- [x] FE tsc + eslint: không có error mới ngoài pre-existing.
- [x] 05_progress.md APPEND-only, 5 dòng mới timestamped + agent + model.
- [x] report_shadow_connector_display.md tạo file vật lý.
- [ ] User restart 2 service (CMS + worker không cần vì worker không sửa), reload `/shadow`, verify panel có 2 tag (Connector + Source DB) + column Connector (chờ).
- [ ] User confirm 2 panel riêng cho 2 connector cùng source_db (chờ data thật).

## Test thủ công user cần làm

```bash
# 1. Restart cdc-cms-service (FE đã hot-reload nếu vite dev đang chạy)
#    Nếu không reload tự động:
cd cdc-cms-web && npm run dev

# 2. Mở http://localhost:5173/shadow
# 3. Xác nhận panel header có dạng:
#    [DB icon] Connector: [purple tag: <connector_name>] Source DB: [geekblue tag: <db>] [blue tag: N objects]
# 4. Trong table cột đầu tiên là "Connector" hiển thị tag purple với connector_name.
# 5. Nếu có 2 connector khác nhau cùng source_db: thấy 2 panel riêng.
# 6. Tab "Shadow Bindings" cũng cùng pattern (column Connector + 2 tag header).
```

## Note kỹ thuật

- `source_connection_code` được lấy từ `connection_registry.connection_code` (V2). Nếu row V2 chưa có FK (legacy null), backend trả empty string `""`; FE map sang `"(unassigned)"`.
- Backend dùng `LEFT JOIN` để không loại bỏ row legacy null.
- `ORDER BY COALESCE(cn.connection_code, '')` — null/empty xuống đầu (alphabetical empty < bất kỳ chuỗi nào) — stable.
- Backwards compat: clients cũ ignore unknown JSON field nhờ `omitempty`.

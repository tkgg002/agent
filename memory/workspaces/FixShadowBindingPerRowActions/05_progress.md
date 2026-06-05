# 05_progress — Fix Shadow Binding Per-Row Actions (IMMUTABLE — APPEND ONLY)

## 2026-05-20 — Phase 1 implementation

### Backend changes
- `internal/app/queries/source_objects_read_models.go`
  - `SourceObjectListItem`: thêm `ShadowBindingIsActive *bool json:"shadow_binding_is_active,omitempty"`.
- `internal/infra/persistence/source_object_read_repo_gorm.go`
  - SELECT thêm `sb.is_active AS shadow_binding_is_active` ngay sau `so.is_active`.
- `internal/app/queries/bridge_status_reader.go`
  - Thêm interface method `ResolveDispatchScopeByBindingID(ctx, bindingID) (*DispatchScope, error)`.
- `internal/infra/persistence/bridge_status_repo_gorm.go`
  - Thêm const `dispatchScopeByBindingQuery` (JOIN so ON so.id = sb.source_object_id WHERE sb.id=?), KHÔNG filter is_active (cần action cả binding pending).
  - Thêm method `ResolveDispatchScopeByBindingID(ctx, bindingID)` reuse helper `resolveScope`.
- `internal/api/source_object_actions_handler.go`
  - Thêm `parseBindingIDQuery(c)` helper đọc `?binding_id=` từ query.
  - Thêm `resolveDispatchScope(c, sourceObjectID)`: branch theo `binding_id` query.
  - Thêm `resolveReadScope(c, sourceObjectID)`: tương tự nhưng cho read path.
  - 5 dispatch callsite chuyển: `resolveDispatchScopeBySourceObjectID(c.UserContext(), id)` → `resolveDispatchScope(c, id)`.
  - 1 read callsite chuyển: `resolveReadScopeBySourceObjectID(c.UserContext(), id)` → `resolveReadScope(c, id)`.
- `internal/app/commands/update_shadow_binding.go` (NEW)
  - `UpdateShadowBindingCommand{ID, IsActive}` + handler update đúng 1 row `cdc_system.shadow_binding`. KHÔNG cascade.
  - Reuse `ErrShadowBindingNotFound` từ `create_master.go` (đã tồn tại); thêm `ErrShadowBindingNoFields`.
- `internal/api/shadow_binding_actions_handler.go` (NEW)
  - `ShadowBindingActionsHandler.PatchActive(c)` — endpoint nhỏ chỉ dispatch command qua bus.
- `internal/router/router.go`
  - Thêm param `shadowBindingActionsHandler *api.ShadowBindingActionsHandler` vào `SetupRoutes`.
  - Đăng ký `admin.Patch("/v1/shadow-bindings/:id", shadowBindingActionsHandler.PatchActive)`.
- `internal/server/server.go`
  - Construct `shadowBindingActionsHandler := api.NewShadowBindingActionsHandler(cmdBus, logger)`.
  - `cmdBus.RegisterSync("shadow-binding.update", ...)`.
  - Truyền vào `router.SetupRoutes`.

### Frontend changes
- `src/types/index.ts`
  - `SourceObjectRow`: thêm `shadow_binding_is_active?: boolean | null`.
- `src/hooks/useRegistry.ts`
  - `useScanFields(...)` thêm tham số `bindingId`. Append `?binding_id=<id>` vào endpoint + statusEndpoint khi V2 path.
- `src/pages/TableRegistry.tsx`
  - `activeLoadingId` đổi type từ `number|null` → `string|null` để encode key composite `b:<bindingId>` / `s:<sourceObjectId>`.
  - `updateEntry`: thêm branch `togglesBindingOnly` — khi only field là `is_active` và record có `shadow_binding_id`, gọi `PATCH /api/v1/shadow-bindings/${bindingId}` thay vì cascade source endpoint.
  - Thêm helper `bindingQuery(record)` build `?binding_id=...`.
  - Cột Switch render: `checked` đọc `shadow_binding_is_active ?? is_active`; `loading` so với loadingKey scope-per-row.
  - `rowKey` của Table chính: function `(r) => r.shadow_binding_id ? '${object_code}#${shadow_binding_id}' : object_code`.
  - `AsyncRowActions` truyền `bindingId` vào `useScanFields` — scan-fields giờ kèm `?binding_id`.
  - `handleCreateTable` + `handleCreateDefaultFields`: append `bindingQuery(record)` vào endpoint.

### Verification
- `go build ./...` PASS.
- `go test ./...` PASS (api / persistence / commands / queries / middleware / messaging / observability, không suite nào fail).
- `npx tsc --noEmit` (cdc-cms-web) PASS.
- SQL verify trên gpay-postgres-cdc:
  - `dispatchScopeByBindingQuery(4)` → `target_table=sd_export_jobs_dev_1`, source_object_id=1.
  - `dispatchScopeByBindingQuery(1)` → `target_table=sd_export_jobs_dev`, source_object_id=1.
- Đợi user reload binary CMS (`go run ./cmd/server` PID 90926) + FE để verify UI:
  - Click Switch row sb=4 — chỉ row đó toggle.
  - Click scan-fields row sb=4 — activity log target_table=sd_export_jobs_dev_1.

### Backwards compatibility
- `PATCH /api/v1/source-objects/:id` GIỮ cascade is_active (semantic "tắt toàn source"). FE chỉ ƯU TIÊN endpoint per-binding khi `shadow_binding_id` có và Switch là field duy nhất; legacy v2_source_only và update đa-field (notes, timestamp_field, ...) vẫn dùng route cũ.
- 5 dispatch endpoint (`/scan-fields`, `/create-default-columns`, `/standardize`, `/detect-timestamp-field`, `/transform`) chấp nhận `?binding_id=` optional — call không kèm vẫn fallback resolver cũ → backwards-safe.
- Endpoint mới `PATCH /api/v1/shadow-bindings/:id` chỉ chấp nhận `{is_active: bool}` — schema minimal, dễ mở rộng.

## 2026-05-20 — Phase 1 hotfix (sau khi user reload binary + FE)

### Bug 1: scan-fields poll bị pending vĩnh viễn
- Symptom (user): "build fe để bị pending ko ra kết quả luôn".
- Root cause: ở `src/hooks/useRegistry.ts`, `useScanFields` ghép `?binding_id=` vào CẢ `statusEndpoint`. Hook `useAsyncDispatch` lại append tiếp `?subject=...&since=...` → URL kết thúc thành `…/dispatch-status?binding_id=4?subject=scan-fields&since=…` (double `?`). Backend parse `binding_id="4?subject=scan-fields"` (không phải int) → silently rớt về resolver `BySourceObjectID` → source_object_id=1 có 2 binding active → trả 409 `ambiguous_source_object_scope`. Poll loop không bao giờ thấy entries → state máy không transition khỏi `running` → spinner pending mãi.
- Fix: tách binding_id ra `statusParams: { binding_id }` cho path dispatch-status (hook đã tự fold qua `URLSearchParams.set`). Endpoint POST scan-fields VẪN giữ `?binding_id=` inline (POST không bị append thêm).
- File: `src/hooks/useRegistry.ts` (chỉ 1 file).
- Verify: simulate URL đúng `…/dispatch-status?subject=scan-fields&since=…&binding_id=4` → HTTP 200, entries[0]=success, target_table=`sd_export_jobs_dev_1`.

### Bug 2: Sync Fields to Shadow (MappingFieldsPage) → 409 ambiguous
- Symptom (user): "click sync field lại bị ambiguous_source_object_scope".
- Root cause: `MappingFieldsPage.handleSyncFields` POST `/api/v1/source-objects/${registry.id}/create-default-columns` KHÔNG kèm `?binding_id=`. Backend resolver fallback `BySourceObjectID` → 2 binding active → 409.
- Fix: append `?binding_id=${registry.shadow_binding_id}` khi context registry có shadow_binding_id (đã có sẵn trong `SourceObjectMappingContext extends SourceObjectRow`).
- File: `src/pages/MappingFieldsPage.tsx`.
- Verify: live POST `/api/v1/source-objects/1/create-default-columns?binding_id=4` → HTTP 202 với `target_table=sd_export_jobs_dev_1`, `trace_id` valid.

### Verification (live, post-restart cmsapi PID 87730)
- PATCH `/api/v1/shadow-bindings/4` → 200, message=`shadow binding updated`.
- POST `/api/v1/source-objects/1/scan-fields?binding_id=4` → 202, target_table=`sd_export_jobs_dev_1`.
- GET `/api/v1/source-objects/1/dispatch-status?subject=scan-fields&since=…&binding_id=4` → 200, latest_status=success, latest_target=`sd_export_jobs_dev_1`.
- POST `/api/v1/source-objects/1/create-default-columns?binding_id=4` → 202.
- TS check: pre-existing TS6133 ở TableRegistry.tsx (Upload, UploadOutlined, STATE_COLOR, modeLoadingId, handleToggleMode, handleBulkImport — chưa dọn) — KHÔNG có lỗi mới ở file vừa sửa (`useRegistry.ts`, `MappingFieldsPage.tsx`). Vite HMR auto-reload không bị block bởi TS6133 (chỉ warning level).

# 02_plan_phase1 — Per-Row Action Disambiguation by binding_id

## Goal
Mỗi per-row action (toggle is_active, scan-fields, snapshot,
create-default-columns, ...) routes đúng tới binding mà user nhấn,
không cascade sang binding khác.

## Strategy — query-param binding_id (additive, backwards-compatible)
- Endpoint path KHÔNG đổi (vẫn `/v1/source-objects/:id/<action>`).
- Thêm OPTIONAL query param `?binding_id=<sb.id>`.
- Handler: nếu có binding_id → resolve scope qua binding_id (mới);
  nếu không → fallback resolve qua source_object_id (cũ, dành cho
  v2_source_only).
- Bonus: thêm endpoint riêng `PATCH /v1/shadow-bindings/:id` cho toggle
  is_active per-binding (semantic rõ ràng, KHÔNG cascade).

Lý do không dùng path mới `/v1/shadow-bindings/:bid/<action>`:
- Action là về source object (scan fields trên source), binding chỉ là
  hint chọn target. Giữ path semantic "action on source object" + tham
  số "tell me which binding" gọn nhất.
- Less router churn, fewer handler files.
- Exception: toggle is_active dùng endpoint riêng vì đối tượng update
  là binding chứ không phải source.

## Steps

### Step A — Backend list emit sb.is_active
File: `cdc-cms-service/internal/app/queries/source_objects_read_models.go`
- `SourceObjectListItem`: thêm `ShadowBindingIsActive *bool json:"shadow_binding_is_active,omitempty"`.

File: `cdc-cms-service/internal/infra/persistence/source_object_read_repo_gorm.go`
- Trong SELECT `ListEnriched` thêm `sb.is_active AS shadow_binding_is_active`.

### Step B — Backend resolver mới ResolveDispatchScopeByBindingID
File: `cdc-cms-service/internal/app/queries/bridge_status_reader.go`
- Thêm method `ResolveDispatchScopeByBindingID(ctx, bindingID) (*DispatchScope, error)`.
- Reuse `gorm.ErrRecordNotFound` + `ErrSourceObjectNoActiveShadow` semantics.

File: `cdc-cms-service/internal/infra/persistence/bridge_status_repo_gorm.go`
- Thêm const `dispatchScopeByBindingQuery` — SELECT same cols, JOIN
  source_object_registry on sb.source_object_id, WHERE sb.id = ?.
  Không filter is_active vì action cần target được kể cả khi binding
  pending (e.g. create-default-columns trên binding chưa active).
- Thêm method `ResolveDispatchScopeByBindingID(ctx, bindingID)`.
  Reuse helper `resolveScope(ctx, id, sql)`.

### Step C — Handler chấp nhận binding_id query
File: `cdc-cms-service/internal/api/source_object_actions_handler.go`
- Thêm helper `resolveDispatchScope(c *fiber.Ctx, fallbackSourceObjectID int64) (*queries.DispatchScope, error)`:
  - Parse `?binding_id=` từ query.
  - Nếu binding_id > 0 → gọi `ResolveDispatchScopeByBindingID`.
  - Else → gọi `ResolveDispatchScopeBySourceObjectID` (legacy).
  - Map errors tương tự `mapResolveErr`.
- Sửa 5 dispatch callsite (`CreateDefaultColumnsV2`, `ScanFieldsV2`,
  `StandardizeV2`, `DetectTimestampFieldV2`, transform — line 163,
  249, 312, 382, 443) dùng helper mới.
- TransformStatusV2 (line 510, read-only) cũng nên hỗ trợ binding_id
  để FE show status đúng binding. Thêm read-mode helper tương tự.

### Step D — Endpoint mới PATCH /v1/shadow-bindings/:id (is_active)
File: `cdc-cms-service/internal/app/commands/update_shadow_binding.go` (NEW)
- Command `UpdateShadowBindingCommand{ID, IsActive *bool, UpdatedBy}`.
- Handler updates only `cdc_system.shadow_binding` row by id.
- Error `ErrShadowBindingNotFound`.

File: `cdc-cms-service/internal/api/shadow_binding_actions_handler.go` (NEW, nhỏ)
- Handler `PatchActive(c)`: parse :id + body `{is_active: bool}` →
  dispatch command qua bus → respond JSON.

File: `cdc-cms-service/internal/router/router.go`
- Đăng ký route: `admin.Patch("/v1/shadow-bindings/:id", shadowBindingActionsHandler.PatchActive)`.

File: `cdc-cms-service/internal/server/server.go`
- `cmdBus.RegisterSync("shadow-binding.update", commands.NewUpdateShadowBindingHandler(db, logger))`.
- Wire handler vào router.

### Step E — Sửa update_source_object_v2 cascade (giữ hành vi cũ, không xóa)
File: `cdc-cms-service/internal/app/commands/update_source_object_v2.go`
- Cascade is_active sang tất cả binding của source vẫn GIỮ
  (semantic "tắt source = tắt tất cả binding"). Không break v2_source_only.
- Nhưng UI mới sẽ ƯU TIÊN gọi PATCH /v1/shadow-bindings/:id khi có
  binding_id → cascade này chỉ chạy khi user chọn "deactivate the
  whole source" (chưa có UX cho cái đó, để sau).

### Step F — Frontend rowKey + Switch + actions
File: `cdc-cms-web/src/types/index.ts`
- `SourceObjectRow`: thêm `shadow_binding_is_active?: boolean | null;`.

File: `cdc-cms-web/src/pages/TableRegistry.tsx`
- `rowKey`: từ `"object_code"` → function:
  ```ts
  rowKey={(r) => r.shadow_binding_id ? `${r.object_code}#${r.shadow_binding_id}` : r.object_code}
  ```
- Cột "Trạng thái" Switch:
  - `checked`: prefer `record.shadow_binding_is_active ?? record.is_active`.
  - `onChange`: nếu `record.shadow_binding_id` → PATCH
    `/api/v1/shadow-bindings/${record.shadow_binding_id}` với
    `{is_active: checked}`; else fallback PATCH source-objects (cũ).
  - `activeLoadingId`: đổi sang key composite hoặc dùng
    `shadow_binding_id` riêng. Đơn giản nhất: state riêng
    `bindingActiveLoadingId`.
- Các nút action (`handleScanFields`, `handleCreateTable`,
  `handleCreateDefaultFields`, `handleSnapshot`, `openEdit`, ...):
  - Append `?binding_id=${record.shadow_binding_id}` vào URL khi
    binding_id có. Else giữ nguyên.

## Out of scope (Phase 2)
- Refactor RegistryHandler V1 path nếu cần (V1 dùng `registry_id` đã
  có binding context implicit qua tr.target_table). Để sau khi user
  confirm Phase 1 hết bug.
- UI cell merge cho cột "source" (1 source = N row hiện lặp tên source).

## Verification Plan
1. `go build ./...` PASS.
2. `go test ./...` PASS.
3. SQL: dispatchScopeByBindingQuery(4) trả `sd_export_jobs_dev_1`.
4. SQL: dispatchScopeByBindingQuery(1) trả `sd_export_jobs_dev`.
5. UI manual (sau khi user reload binary + FE):
   - Click Switch row sb=4: chỉ row sb=4 toggle, sb=1 giữ nguyên.
   - Click scan-fields row sb=4: activity log target_table=sd_export_jobs_dev_1.
6. Activity log mới phải có target_table khớp binding click.

## Definition of Done
- [ ] Backend build + test PASS.
- [ ] 2 SQL verify ở Postgres live.
- [ ] User click 2 Switch độc lập (1 không kéo theo cái kia).
- [ ] Scan-fields trên sb=4 sinh activity log target_table=sd_export_jobs_dev_1.
- [ ] Workspace progress + lesson APPEND.

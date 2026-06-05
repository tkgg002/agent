# 01 — Requirements (Audit + Fix)

## Goal
Snapshot v2 phải dispatch đúng vào **shadow_binding mà user chọn** khi source object có >1 binding. Hai endpoint cần fix:

### R1 — `POST /api/v1/source-objects/:id/snapshot-v2?binding_id=<sb.id>`
- Handler BẮT BUỘC đọc `binding_id` từ query.
- `binding_id > 0` → command kèm `BindingID`, worker chỉ ghi vào target_table của binding đó.
- `binding_id` thiếu nhưng source_object có >1 active binding → 409 `ambiguous_source_object_scope` (giống `SnapshotV2 ` UpdateBridge / CreateDefaultColumns).
- `binding_id` thiếu nhưng source_object có 1 binding duy nhất → fallback theo source_object_id (giữ legacy behavior).

### R2 — `GET /api/v1/source-objects/:id/transform-status[?binding_id=<sb.id>]`
- Handler đã có `resolveReadScope` đọc binding_id (đã đúng).
- Khi FE không truyền binding_id và source có >1 binding → 409 đúng spec hiện tại.
- **Action: KHÔNG cần fix BE**. Document để FE biết phải truyền binding_id khi list trả >1 row cùng source_object_id (UI render N rows phải attach binding_id vào link).

## Acceptance Criteria
1. `SnapshotV2Command` có field `ShadowBindingID int64`.
2. `snapshotV2Payload` (worker) có field `shadow_binding_id`.
3. `SnapshotV2` handler:
   - Parse `binding_id` query → set vào command.
   - Nếu thiếu binding_id → gọi `resolveDispatchScope` (đã có logic 409 ambiguous).
   - Nếu có binding_id → validate qua `ResolveDispatchScopeByBindingID`.
4. Worker `runSnapshot`:
   - Đọc `p.ShadowBindingID`.
   - Nếu > 0: filter `ResolveSourceRoutes` chỉ giữ route có `ShadowBindingID` khớp.
   - Nếu = 0: giữ behavior cũ (master + clone routes).
5. Build PASS + test PASS (`go build`, `go test -count=1 -short ./...`).
6. Smoke (manual nếu user yêu cầu): POST với `binding_id=112` → snapshot chỉ ghi vào `wallet_capsets_1`. POST không có binding_id (source có 2 binding) → 409.

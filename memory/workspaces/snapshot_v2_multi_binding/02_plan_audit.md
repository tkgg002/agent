# 02 — Plan (Fix snapshot.v2 multi-binding routing)

## Decision matrix

| Option | Mô tả | Risk | Khuyến nghị |
|--------|-------|------|-------------|
| **1. End-to-end `binding_id` plumbing** | Add field vào Command + Payload, handler đọc query, worker filter route theo binding | Thấp — pattern đã có trong CreateDefaultColumnsV2, UpdateBridge | ✅ **Đề xuất apply** |
| 2. Để FE giải quyết — UI ép user chọn 1 binding rồi gọi delete binding khác trước khi snapshot | Workaround, không scale | Cao | ❌ Bỏ |
| 3. Force ambiguous error 409 khi multi-binding, không cho fan-out | Block snapshot hợp lệ với source 1 binding | Trung bình | Phần của Option 1 |

## Phase plan (Option 1)

### Phase A — Wire `BindingID` từ HTTP → Command (CMS service)
A1. `cdc-cms-service/internal/app/commands/recon_async.go`:
   - Add field `ShadowBindingID int64 \`json:"shadow_binding_id,omitempty"\`` vào `SnapshotV2Command`.
   - KHÔNG add vào `Validate()` (binding_id optional — default behavior khi source-only).
A2. `cdc-cms-service/internal/api/source_object_actions_handler.go` — `SnapshotV2`:
   - Đầu hàm sau khi parse `id`, gọi `bid := parseBindingIDQuery(c)`.
   - Nếu `bid > 0` → validate qua `h.bridgeReader.ResolveDispatchScopeByBindingID(ctx, bid)`. Nếu binding's source_object_id ≠ `id` → 400 `binding_id_mismatch`. Trả lỗi nếu not found.
   - Nếu `bid == 0` → gọi `h.resolveDispatchScopeBySourceObjectID(ctx, id)`. Nếu trả `ErrAmbiguousDispatchScope` → 409 đã có (mapResolveErr).
   - Set `cmd.ShadowBindingID = bid` (0 khi không có).

### Phase B — Wire `BindingID` từ Command → Worker payload (cdc-cms-service publish layer)
B1. Confirm `commands.SnapshotV2Command` serialize JSON kèm `shadow_binding_id` (đã add tag).
B2. NATS publisher hiện tại sẽ tự forward toàn bộ field — verify.

### Phase C — Worker tiêu thụ binding (centralized-data-service)
C1. `centralized-data-service/internal/handler/snapshot_runner_handler.go`:
   - Thêm `ShadowBindingID int64 \`json:"shadow_binding_id"\`` vào `snapshotV2Payload`.
   - Trong `runSnapshot`:
     - Sau khi load `so`, nếu `p.ShadowBindingID > 0` → load shadow_binding qua repo, override `targetTable = sb.TargetTable` (thay cho `so.ObjectCode`).
     - Sau `ResolveSourceRoutes(srcDB, srcColl)` → nếu binding_id > 0, filter routes giữ duy nhất route có `ShadowBindingID` khớp. Nếu rỗng → `markProgressError` + return.
   - `claimProgress`: thêm `shadow_binding_id` vào unique key của `snapshot_progress` row để 2 binding không block nhau (yêu cầu migration nếu schema chưa có cột).

### Phase D — Schema check
D1. Verify `cdc_system.snapshot_progress` table có cột `shadow_binding_id`. Nếu không → migration thêm cột + unique constraint `(source_object_id, shadow_binding_id, status)` để cho phép 2 progress songsong.
D2. Nếu chưa có migration, **DỪNG và hỏi user** trước khi tạo migration mới.

### Phase E — Verify
- `go build ./...` cả 2 service (`cdc-cms-service`, `centralized-data-service`).
- `go test -count=1 -short ./...` không break test cũ.
- Add test mới: parseBindingIDQuery + worker filter logic.
- Manual smoke (nếu user yêu cầu deploy): POST `?binding_id=112` → check `snapshot_progress` chỉ tạo row binding 112; UPDATE chỉ chạy `wallet_capsets_1`.

## Risks
- Migration `snapshot_progress` (D1) đụng prod data — phải review/CR trước.
- `MetadataRegistryService.routeCache` keyed bằng (srcDB, srcTable) → filter theo binding cần `ShadowBindingID` field trong `ResolvedSourceRoute` (cần check).
- `claimProgress` đang dedupe theo source_object → 2 binding song song có thể race trên cùng connection. Có thể chấp nhận serialize (chạy lần lượt) nhưng phải cho phép user pick binding cụ thể.

## Out of scope (note nhưng không fix lần này)
- Refactor `MetadataRegistryService` để route lookup keyed theo `(source, shadow_binding_id)` (large refactor).
- Migration `snapshot_progress` schema thay đổi (cần user approve riêng).
- FE update — báo FE phải truyền binding_id ở 2 endpoint nếu source >1 binding.

## Pre-flight check trước khi APPLY
1. User approve Option 1.
2. Confirm có/không `snapshot_progress.shadow_binding_id` column.
3. Confirm có/không `ResolvedSourceRoute.ShadowBindingID` field.

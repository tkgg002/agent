# 09 — Solution tasks (chờ user approve)

## Pre-flight (em sẽ verify nếu user approve)
- [ ] `grep "shadow_binding_id" centralized-data-service/internal/model/snapshot_progress*.go` — confirm column tồn tại trong `snapshot_progress` schema.
- [ ] `grep "ShadowBindingID" centralized-data-service/internal/service/metadata_registry_service.go` — confirm route có field binding hay không.
- [ ] Nếu schema thiếu — escalate user, KHÔNG tự thêm migration.

## Code tasks (sau khi user approve)

### T1 — CMS service (cdc-cms-service)
- `internal/app/commands/recon_async.go`:
  - Add `ShadowBindingID int64 \`json:"shadow_binding_id,omitempty"\`` vào `SnapshotV2Command`.
- `internal/api/source_object_actions_handler.go` `SnapshotV2`:
  - Parse `binding_id` query qua `parseBindingIDQuery(c)`.
  - Nếu > 0 → `h.bridgeReader.ResolveDispatchScopeByBindingID(ctx, bid)` validate. Nếu source_object_id mismatch → 400.
  - Nếu = 0 → `h.resolveDispatchScopeBySourceObjectID(ctx, id)`. Match 2 binding → 409 (mapResolveErr đã sẵn).
  - Set `cmd.ShadowBindingID = bid`.
  - Activity log thêm `shadow_binding_id` field.

### T2 — Worker (centralized-data-service)
- `internal/handler/snapshot_runner_handler.go`:
  - Add `ShadowBindingID int64 \`json:"shadow_binding_id"\`` vào `snapshotV2Payload`.
  - `runSnapshot`:
    - Sau `soRepo.GetByID`, nếu `p.ShadowBindingID > 0`:
      - Load `shadow_binding` qua repo (cần ShadowBindingRepo) → lấy `target_table`, `shadow_schema` chính xác.
      - Override `targetTable = sb.TargetTable`.
    - Sau `ResolveSourceRoutes(srcDB, srcColl)` → nếu binding_id > 0, filter routes giữ `route.ShadowBinding.ID == p.ShadowBindingID`. Rỗng → `markProgressError(...,"binding %d not in active registry routes",...)` + return.
  - `claimProgress`: include `p.ShadowBindingID` vào lookup. Yêu cầu schema có cột `shadow_binding_id`.
  - Tech depth log (theo lesson 2026-05-29): `component=snapshot_runner op=run_snapshot shadow_binding_id=<id> target_table=<X> ...`.

### T3 — Tests
- `cdc-cms-service/test/api/source_object_actions_handler_test.go`:
  - Case: POST snapshot-v2 với binding_id=112 → command kèm ShadowBindingID=112.
  - Case: POST không có binding_id, source có >1 binding → 409 ambiguous.
  - Case: POST không có binding_id, source có 1 binding → 202 + ShadowBindingID=0 fallback.
- `centralized-data-service/test/internal/handler/snapshot_runner_test.go`:
  - Case: payload với ShadowBindingID → route filter giữ đúng.
  - Case: payload không có ShadowBindingID → fan-out master+clones như cũ.

### T4 — Verify
- `cd cdc-cms-service && go build ./... && go test -count=1 -short ./...`
- `cd centralized-data-service && go build ./... && go test -count=1 -short ./...`
- Optional smoke trên dev (user chạy): POST với binding_id=112 → log SigNoz hiển thị `shadow_binding_id=112 target_table=wallet_capsets_1` + bảng `shadow_goopay_local_ws_wallet_service.wallet_capsets_1` được ghi.

## FE follow-up (báo team FE)
- `/transform-status` và `/snapshot-v2` BẮT BUỘC truyền `?binding_id=<sb.id>` khi list source-objects trả >1 row cùng `id`. Lấy `binding_id` từ `shadow_binding_id` của row đang hiển thị.
- (Note: BE giữ fallback ambiguous → 409, không silent route.)

## Documentation (sau khi apply)
- Update `agent/memory/global/lessons.md` — Lesson: "Multi-shadow_binding source: HTTP scope param phải đi end-to-end (HTTP query → Command field → Worker payload → registry route filter), KHÔNG dừng ở handler".
- Update `data-hub/report_snapshot_v2_multi_binding.md` với delta code + verify result.

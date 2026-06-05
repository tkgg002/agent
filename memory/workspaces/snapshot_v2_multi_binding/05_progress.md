# 05 — Progress (APPEND ONLY)

## 2026-05-29T17:40+07 [Muscle:claude-opus-4-7] Init workspace
- Trigger: user báo 2 bug:
  - GET `/api/v1/source-objects/110/transform-status` → 409 `ambiguous_source_object_scope`.
  - POST `/api/v1/source-objects/110/snapshot-v2?binding_id=112` → 202 nhưng không snapshot binding nào.
- Doc tạo: 00_context, 01_requirements, 02_plan, 03_implementation.

## 2026-05-29T17:48+07 [Muscle:claude-opus-4-7] Root cause confirmed (audit only — chưa code)
- `SnapshotV2Command` thiếu `ShadowBindingID` field (`cdc-cms-service/internal/app/commands/recon_async.go:111-127`).
- `SnapshotV2` handler không gọi `parseBindingIDQuery` (`cdc-cms-service/internal/api/source_object_actions_handler.go:552-615`).
- `snapshotV2Payload` worker thiếu `shadow_binding_id` (`centralized-data-service/internal/handler/snapshot_runner_handler.go:72-79`).
- Worker `runSnapshot` lấy `targetTable = so.ObjectCode` (line 201) thay vì `shadow_binding.target_table` → activity_log lệch binding thực.
- `ResolveSourceRoutes(srcDB, srcColl)` fan-out master + clone routes (`metadata_registry_service.go:560-583`) — không filter theo binding.
- `claimProgress` dedupe theo source_object_id → 2 binding chặn nhau.
- `transform-status` handler đã có logic đọc binding_id (`resolveReadScope`) → KHÔNG bug BE, gap FE.

## Quyết định
- Theo CLAUDE.md §12 (Brain Code Prohibition) + audit-first discipline: dừng tại bước document.
- Đợi user approve Option 1 (end-to-end binding_id plumbing) trước khi code.

## 2026-05-29T18:35+07 [Muscle:claude-opus-4-7] Option 1 applied — code edits
- User approve "làm đi". Apply T1–T4 trong `09_tasks_solution_audit.md`.

### Changes (CMS — cdc-cms-service)
- `internal/app/commands/recon_async.go` — `SnapshotV2Command` thêm `ShadowBindingID int64 \`json:"shadow_binding_id,omitempty"\``.
- `internal/api/source_object_actions_handler.go` `SnapshotV2`:
  - Parse `binding_id` query qua `parseBindingIDQuery(c)`.
  - Gọi `h.resolveDispatchScope(c, id)` → resolve scope (binding-aware nhờ helper sẵn có).
  - Nếu `bid > 0` và `scope.SourceObjectID != id` → 400 `binding_id_mismatch`.
  - Nếu thiếu binding_id và source có >1 binding active → 409 `ambiguous_source_object_scope` (mapResolveErr xử lý sẵn).
  - Set `cmd.ShadowBindingID = bid`; activity log + JSON response thêm `shadow_binding_id`, `target_table`, `shadow_schema`.
- Build PASS.

### Changes (Worker — centralized-data-service)
- `internal/handler/snapshot_runner_handler.go`:
  - `snapshotV2Payload` thêm `ShadowBindingID int64 \`json:"shadow_binding_id"\``.
  - `SnapshotRunner` + constructor thêm `shadowRepo *repository.ShadowBindingRepo`.
  - `runSnapshot`:
    - Sau `soRepo.GetByID`, nếu `p.ShadowBindingID > 0`:
      - Load `shadow_binding` qua `shadowRepo.GetByID`. Validate `sb.SourceObjectID == so.ID` (else error).
      - Override `targetTable = sb.ShadowTable` (cho activity log + writeActivity).
    - Sau `ResolveSourceRoutes(srcDB, srcColl)`: nếu scoped, check tồn tại route với `ShadowBinding.ID == sb.ID`; nếu rỗng → `markProgressError("shadow_binding_id=N not in active registry routes…")` + return.
    - Pin ctx scope: `ctx = WithBindingScope(ctx, sb.ID)` để `eventHandler.HandleRaw` filter.
  - Log "snapshot.v2 started" + writeActivity details: thêm `component/op/phase` + `shadow_binding_id` + `target_table`.
  - `claimProgress`: INSERT kèm `shadow_binding_id`; resume SELECT dùng `IS NOT DISTINCT FROM` để NULL và id cụ thể là 2 group dedup riêng → 2 binding chạy song song.
- `internal/handler/event_handler.go`:
  - Thêm `WithBindingScope(ctx, id)` + `bindingScopeFromCtx(ctx)` helpers (context-key pattern, không đổi API public).
  - `processEvent`: sau `ResolveSourceRoutes`, nếu ctx có scope → filter routes giữ duy nhất binding khớp; rỗng → warn + return 0 (giữ behavior "0 → snapshot CB trip" sẵn có).
  - CDC consumer (Kafka/NATS) KHÔNG set scope → giữ fan-out master+clone như cũ.
- `internal/server/worker_server.go`: `NewSnapshotRunner` nhận `shadowBindingRepo` (đã có sẵn ở dòng 186).
- Build PASS.

### Migration
- `cdc-cms-service/migrations/schema/core/066_add_shadow_binding_id_to_snapshot_progress.sql`:
  - `ADD COLUMN IF NOT EXISTS shadow_binding_id BIGINT` (nullable — legacy NULL flow vẫn chạy).
  - `CREATE INDEX IF NOT EXISTS idx_snapshot_progress_binding_status (source_object_id, shadow_binding_id, status, started_at DESC)`.
  - Comment giải thích NULL vs concrete id semantics.

### Verify
- `cd centralized-data-service && go build ./...` → PASS.
- `cd cdc-cms-service && go build ./...` → PASS.
- `cd centralized-data-service && go test -count=1 -short ./...` →
  - `internal/handler`, `internal/service`, các package khác PASS.
  - `test/internal/handler/event_handler_test.go` (untracked, chưa từng commit) fail `TestHandleDelete_FirstTouch_TombstoneInsert` — sqlmock fixture mismatch ("expected 2, but got 1 arguments"), production code chỉ pass 1 arg `pkValue`. **Pre-existing fail, KHÔNG liên quan multi-binding fix** (handleDelete path không có thay đổi).
- `cd cdc-cms-service && go test -count=1 -short ./...` →
  - Hầu hết PASS.
  - `test/internal/api/mapping_rule_handler_test.go` `TestUpdateStatus_MissingStatus` + `test/internal/app/commands/sync_metadata_test.go` `TestUpdateMappingRule_TypeAndValidate` fail — error-message fixture stale ("status is required" vs "status or data_type is required"). Cả 2 file untracked, KHÔNG liên quan SnapshotV2 / multi-binding.

### Smoke (chờ user deploy)
- POST `/api/v1/source-objects/110/snapshot-v2?binding_id=112` → 202.
  - SigNoz: `component=snapshot_runner op=run_snapshot phase=scope_resolved shadow_binding_id=112 target_table=wallet_capsets_1`.
  - DB: `cdc_system.snapshot_progress` có row mới với `shadow_binding_id=112`, không đụng row binding 110.
  - Chỉ bảng `shadow_goopay_local_ws_wallet_service.wallet_capsets_1` được ghi (binding 110 = `wallet_capsets` không touch).
- POST `/api/v1/source-objects/110/snapshot-v2` (không binding_id, source vẫn 2 binding) → 409 `ambiguous_source_object_scope`.

### FE follow-up
- `/transform-status` và `/snapshot-v2`: khi list source-objects trả >1 row cùng `id`, BẮT BUỘC truyền `?binding_id=<sb.id>` lấy từ `shadow_binding_id` của row đang hiển thị. Nếu thiếu → BE trả 409 (giữ fail-loud, không silent route).

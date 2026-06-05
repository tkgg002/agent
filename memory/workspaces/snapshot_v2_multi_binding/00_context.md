# 00 — Context

## Scope
Source object `id=110` (`src_mongodb_goopay_local_ws_wallet_service_wallet_capsets`) có 2 shadow_binding:
- `shadow_binding_id=110` → `target_table=wallet_capsets`, `ddl_status=pending`, `is_table_created=false`.
- `shadow_binding_id=112` → `target_table=wallet_capsets_1`, `ddl_status=created`, `is_table_created=true`.

## Triệu chứng user báo
1. `GET /api/v1/source-objects/110/transform-status` → 409 `{"error":"ambiguous_source_object_scope"}`.
2. `POST /api/v1/source-objects/110/snapshot-v2?binding_id=112` (payload `{action:"snapshot", overwrite:true}`) → BE trả 202 nhưng **snapshot không chạy vào target 110 cũng không chạy vào 112**.

## Câu hỏi cần audit
- Endpoint `SnapshotV2` có đọc `binding_id` query param không?
- `SnapshotV2Command` (CMS) + `snapshotV2Payload` (worker) có field `BindingID`/`ShadowBindingID` không?
- Worker `runSnapshot` có filter routes theo binding_id không, hay fan-out qua mọi shadow của source?
- `claimProgress` theo source_object_id hay shadow_binding_id?

## Files trong scope
- CMS: `cdc-cms-service/internal/api/source_object_actions_handler.go` (handler `SnapshotV2` + `TransformStatusV2`).
- CMS: `cdc-cms-service/internal/app/commands/recon_async.go` (`SnapshotV2Command` struct).
- CMS: `cdc-cms-service/internal/app/queries/bridge_status_reader.go` (`DispatchScope`, `ResolveDispatchScopeByBindingID`).
- Worker: `centralized-data-service/internal/handler/snapshot_runner_handler.go` (NATS subscriber `cdc.cmd.snapshot.v2`).
- Worker: `centralized-data-service/internal/service/metadata_registry_service.go` (`ResolveSourceRoutes`).

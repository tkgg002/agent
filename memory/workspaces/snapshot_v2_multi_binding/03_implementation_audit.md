# 03 — Implementation Audit (evidence + root cause)

## A. Endpoint `transform-status` (KHÔNG bug BE — gap FE)
- `source_object_actions_handler.go:633-639` → `TransformStatusV2` gọi `resolveReadScope(c, id)`.
- `source_object_actions_handler.go:87-92` → `resolveReadScope` ưu tiên `parseBindingIDQuery(c)`.
- `source_object_actions_handler.go:65-75` → `parseBindingIDQuery` đọc `c.Query("binding_id")`.
- `bridge_status_reader.go:84` → `ResolveDispatchScopeByBindingID` resolve duy nhất binding đó.

→ **BE đã đúng**. URL user gửi `/transform-status` (không có `?binding_id=`) → handler fallback `ResolveReadScopeBySourceObjectID` → match 2 binding → `ErrAmbiguousDispatchScope` → 409. **Spec ok**. FE phải append `?binding_id=<sb.id>` khi list trả >1 row.

## B. Endpoint `snapshot-v2` (BUG BE — binding_id bị drop)

### Evidence
- `source_object_actions_handler.go:552-615` — handler `SnapshotV2`:
  ```go
  id, _ := strconv.ParseInt(c.Params("id"), 10, 64)
  var body struct { TraceID, Action, Origin string; BatchSize int; Overwrite bool }
  _ = c.BodyParser(&body)
  cmd := commands.SnapshotV2Command{
      SourceObjectID: id, TraceID: traceID, Action: body.Action,
      Origin: body.Origin, BatchSize: body.BatchSize, Overwrite: body.Overwrite,
  }
  res, derr := h.bus.Dispatch(ctx, cmd)
  ```
  → **KHÔNG đọc `binding_id` query**. KHÔNG validate ambiguous trước khi dispatch.

- `commands/recon_async.go:111-127` — `SnapshotV2Command`:
  ```go
  type SnapshotV2Command struct {
      ports.AsyncCommandMixin
      SourceObjectID int64  `json:"source_object_id"`
      TraceID        string `json:"trace_id,omitempty"`
      Action         string `json:"action,omitempty"`
      Origin         string `json:"origin,omitempty"`
      BatchSize      int    `json:"batch_size,omitempty"`
      Overwrite      bool   `json:"overwrite"`
  }
  ```
  → **KHÔNG có field `BindingID`/`ShadowBindingID`**.

- Worker `snapshot_runner_handler.go:72-79` — `snapshotV2Payload`:
  ```go
  type snapshotV2Payload struct {
      SourceObjectID int64 `json:"source_object_id"`
      TraceID, Action, Origin string
      BatchSize int; Overwrite bool
  }
  ```
  → Worker không có khái niệm binding.

- Worker `runSnapshot` (`snapshot_runner_handler.go:182-303`):
  - `r.soRepo.GetByID(ctx, p.SourceObjectID)` → trả 1 row `so`.
  - `targetTable = so.ObjectCode` (line 201) → **chính đây là gốc**: target_table được lấy từ `source_object.object_code`, KHÔNG phải `shadow_binding.target_table`. Trong UI list response, `target_table` thực sự khác nhau giữa 2 binding (wallet_capsets vs wallet_capsets_1) — nhưng worker dùng `ObjectCode` (constant của source_object).
  - `subject := fmt.Sprintf("cdc.snapshot.%s.%s", srcDB, srcColl)` (line 324) — không phụ thuộc binding.
  - `eventHandler.HandleRaw(ctx, subject, envelope)` (line 480) — fan-out theo `ResolveSourceRoutes(srcDB, srcColl)` → trả master + clone routes (`metadata_registry_service.go:560-583`). Khi source có nhiều shadow_binding active, route cache map source → multiple routes → **fan-out cả 2**.

### Tại sao user thấy "ko chạy cả 2"?
Khả năng cao một trong các nhánh sau:
1. Binding 110 có `is_table_created=false`, `ddl_status=pending` → bảng `wallet_capsets` chưa tồn tại. `HandleRaw` cố apply vào bảng chưa tồn tại → fail. Snapshot circuit-breaker (`snapshotV2MaxConsecutiveErrors`) trip rất sớm → `snapshot_progress` status = error, không doc nào được commit.
2. Hoặc `claimProgress` (line 605) theo `source_object_id` → row hot trước đó của binding 110 đang block, lần dispatch cho binding 112 bị skip "another run is still active".
3. Hoặc `processEvent` silent-skip do registry cache chưa có route active cho binding 112 (DDL vừa create xong, registry chưa reload).

→ Tóm lại: dù lý do A/B/C, **gốc rễ là worker không phân biệt được binding nào**. Fix bằng cách truyền `binding_id` end-to-end và filter route theo binding.

## C. Bug surface phụ phát hiện
| Mã | File:line | Mô tả | Severity |
|----|-----------|-------|----------|
| B1 | `snapshot_runner_handler.go:201` | `targetTable = so.ObjectCode` thay vì `shadow_binding.target_table` → activity_log + writeActivity ghi sai target | Medium |
| B2 | `snapshot_runner_handler.go:605+` `claimProgress` theo source_object_id | Hai binding khác nhau dùng chung 1 progress row → 1 dispatch chặn dispatch còn lại | Medium |
| B3 | `metadata_registry_service.go:560-583` `ResolveSourceRoutes` master+clones | Khi shadow_binding scope, fan-out qua mọi binding → ghi vào bảng không liên quan | Medium |
| B4 | `source_object_actions_handler.go:552` `SnapshotV2` không log warn khi multi-binding nhưng thiếu binding_id | Operator không có evidence | Low |

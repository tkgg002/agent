# Phase D — Wire Worker Step Handlers + Auto-Loop (Plan)

**Workspace**: feature-cdc-integration
**Phase**: provisioning_mode / D
**Status**: PLAN — chờ Architect duyệt 5 Open Questions ở `01_requirements_provisioning_phase_d.md`.
**Prereq**: Phase A/B/C DONE (state machine + CAS orchestrator + REST API behind RequireOpsAdmin).

## Triết lý thiết kế

- **Single emit point per handler**: mỗi worker handler kết thúc với `defer`-style emit `step_completed` để không quên path lỗi.
- **Reuse existing handlers khi có thể**: `discover` đã ship → chỉ wire emit. `master.bind` thử reuse `master-create` (chờ Q2).
- **Symmetric với D8 trace propagation**: handler nhận `trace_id`/`span_id` từ inbound payload → echo lại vào outbound event để CMS Phase B nối được span.
- **Safe failure**: handler crash → emit `success=false` trước khi return error. Không silent drop.

## Critical files

### CMS (chỉ read, không sửa Phase D)
- `cdc-cms-service/internal/service/provisioning_orchestrator.go:308-360` — `HandleStepCompleted` đã CAS pending→active. **Cần extend** để xử lý `success=false`.

### Worker (sửa + tạo mới)
- `centralized-data-service/internal/handler/command_handler.go:341` — `HandleDiscover` (đã có, cần wire emit).
- `centralized-data-service/internal/handler/master_ddl_handler.go` — `HandleMasterCreate` (cần đọc để quyết Q2).
- `centralized-data-service/internal/handler/provisioning_handler.go` — đã có inbound subscriber. Phase D bổ sung **outbound emit helper** ở đây để tập trung 1 chỗ.
- `centralized-data-service/internal/server/worker_server.go:248-315` — block subscribe; sẽ thêm 3 subscribe mới (shadow.bind, master.bind, schedule.enable) tùy ruling.

## Implementation steps (sequential, có gate ruling)

### D-1. Tạo `EmitStepCompleted` helper (worker)
File: `centralized-data-service/internal/handler/provisioning_emit.go` (NEW)

```go
package handler

// Helper để 4 step handler emit `cdc.evt.provisioning.step_completed`
// đồng nhất. Chấp nhận success path (err=nil) và failure path.
//
// Payload theo schema Phase B HandleStepCompleted yêu cầu:
//   { source_id, step, success, correlation_id, error?, trace_id?, span_id? }
//
// Trace propagation theo D8: nếu inbound payload có trace_id/span_id,
// caller pass-through vào opts.TraceID/SpanID. KHÔNG tự generate.

func (h *ProvisioningHandler) EmitStepCompleted(
    sourceID int64,
    step string,
    err error,                  // nil → success
    correlationID string,
    traceID, spanID string,
) error {
    payload := map[string]any{
        "source_id":      sourceID,
        "step":           step,
        "success":        err == nil,
        "correlation_id": correlationID,
        "completed_at":   time.Now().UTC().Format(time.RFC3339Nano),
    }
    if err != nil {
        payload["error"] = SanitizeFreeformText(err.Error(), 2000)
    }
    if traceID != "" {
        payload["trace_id"] = traceID
        payload["span_id"] = spanID
    }
    body, _ := json.Marshal(payload)
    return h.natsConn.Publish(SubjectProvisioningStepCompleted, body)
}
```

### D-2. Wire `HandleDiscover` emit (low-risk)
File: `command_handler.go:341` `HandleDiscover`

Pattern (defer + named return):
```go
func (h *CommandHandler) HandleDiscover(msg *nats.Msg) {
    var req DiscoverRequest
    json.Unmarshal(msg.Data, &req)
    var emitErr error
    defer func() {
        // best-effort emit — không block path chính nếu publish lỗi
        _ = h.provHandler.EmitStepCompleted(
            req.SourceID, "discover", emitErr,
            req.CorrelationID, req.TraceID, req.SpanID)
    }()
    // ... existing logic ...
    if err := /* discover work */; err != nil {
        emitErr = err
        return
    }
}
```

**Rủi ro**: nếu request không có `source_id` (ad-hoc discover từ SetupSwagger gọi), `EmitStepCompleted` skip với guard `if sourceID == 0`. Phase B `HandleStepCompleted` đã guard tương tự (line 337).

### D-3. Quyết Q2 — master.bind handler
**Hai phương án**:

**Phương án A (em ưu tiên nếu Architect duyệt)**: `master-create` đã làm CREATE TABLE master + binding insert → alias `cdc.cmd.master.bind` về cùng `HandleMasterCreate`. State machine doc cập nhật ghi rõ "master.bind = master-create + binding insert atomic". Thi công 1 dòng:
```go
natsClient.Conn.Subscribe("cdc.cmd.master.bind", masterDDLHandler.HandleMasterCreate)
```
+ wire `EmitStepCompleted` trong `HandleMasterCreate`.

**Phương án B**: Tách handler riêng `HandleMasterBind` chỉ insert binding (master table assumed exists). Thi công thêm ~80 LOC, nhưng tách responsibility sạch hơn cho schema migration scenarios.

→ Cần ruling Q2.

### D-4. Tạo handler `HandleShadowBind` (Q1)
Logic dựa trên: đọc `source_object_id` từ payload → tạo entry vào `cdc_system.shadow_binding` (table đã có Phase A migration) → emit step_completed.

```go
func (h *CommandHandler) HandleShadowBind(msg *nats.Msg) {
    // 1. parse payload
    // 2. INSERT INTO cdc_system.shadow_binding (source_object_id, shadow_table, ...) ON CONFLICT DO NOTHING
    // 3. Emit step_completed
}
```

Câu hỏi mở (Q1): hiện tại có cron/handler nào TẠO shadow_binding row không? Nếu có → reuse. Nếu chưa → handler mới này là single source.

### D-5. Tạo handler `HandleScheduleEnable` (Q3)
Logic: UPDATE `cdc_system.transmute_schedule SET is_enabled=true WHERE master_table=...` → cron tick tiếp theo (60s) sẽ fire.

```go
func (h *CommandHandler) HandleScheduleEnable(msg *nats.Msg) {
    // 1. parse payload (source_id → resolve master_table)
    // 2. UPDATE transmute_schedule SET is_enabled=true
    // 3. Emit step_completed (note: state vẫn là schedule_pending cho tới khi cron tick chạy thật → step_completed báo "schedule armed", chưa phải "running")
}
```

**Trade-off Q3**: state machine định nghĩa schedule_pending → running ngay khi step_completed về. Nhưng thực tế cron chưa tick → có thể source state=running nhưng chưa có data flow. Ruling: chấp nhận state=running là "schedule armed", hay đợi tick đầu tiên success rồi mới emit?

### D-6. Extend CMS Phase B `HandleStepCompleted` cho failure path
File: `cdc-cms-service/internal/service/provisioning_orchestrator.go` HandleStepCompleted (line 308-360)

**Hiện tại**: chỉ CAS pending→active khi success.

**Phase D thêm**: nếu `success=false` trong payload → CAS pending→failed + stamp `last_step_error` từ field `error`. Reuse `casUpdateState(ctx, id, pending, StateFailed, entry, &errMsg)`.

```go
// Trong HandleStepCompleted, sau khi parse payload:
if !ev.Success {
    errMsg := ev.Error
    entry := provisioningEntryWithSpan(ctx, provisioningStepLogEntry{
        Seq: o.nextLogSeq(ctx, id), Step: ev.Step,
        FromState: string(cur), ToState: string(StateFailed),
        StartedAt: now, CompletedAt: now,
        Success: false, Message: "step failed: " + errMsg,
    })
    return o.casUpdateState(ctx, id, cur, StateFailed, entry, &errMsg)
}
// existing success path unchanged
```

### D-7. Subscribe block trong `worker_server.go`
Sau line 261 (cdc.cmd block):
```go
// Phase D — Provisioning step handlers (auto-loop publishers)
natsClient.Conn.Subscribe("cdc.cmd.shadow.bind", cmdHandler.HandleShadowBind)
natsClient.Conn.Subscribe("cdc.cmd.master.bind", masterDDLHandler.HandleMasterCreate) // alias if Q2=A
natsClient.Conn.Subscribe("cdc.cmd.schedule.enable", cmdHandler.HandleScheduleEnable)
// cdc.cmd.discover đã subscribe line 249 — chỉ wire emit bên trong handler
```

## Verification

### Unit test
- `provisioning_emit_test.go`: stub NATS conn, gọi `EmitStepCompleted(success/failure)`, assert payload đúng schema, trace_id pass-through đúng.

### Integration test (build tag `integration`)
File: `worker/internal/handler/provisioning_e2e_test.go` (NEW)

```go
func TestProvisioning_AutoLoop_E2E(t *testing.T) {
    // 1. Seed: source state=draft, mode=auto
    // 2. Trực tiếp gọi orchestrator.Advance() (skip REST surface)
    // 3. Loop poll DB 30s, expect state=running
    // 4. Assert step_log có 8 entries: 4 advance dispatch + 4 step_completed finalize
    // 5. Failure variant: stub HandleDiscover trả error → expect state=failed,
    //    last_step_error chứa msg
}
```

### Smoke test live
1. Build worker với `PROVISIONING_ORCHESTRATOR_ENABLED=1`.
2. Build CMS bình thường.
3. POST `/api/v1/cms/sources/N/provisioning/mode` `{"mode":"auto"}`.
4. POST `/advance` (chỉ 1 lần, kick từ draft).
5. Tail `cdc_system.source_object_registry WHERE id=N` mỗi 5s — quan sát state nhảy sạch:
   `draft → shadow_pending → shadow_active → master_pending → master_active → mapping_pending → mapping_ready → schedule_pending → running`.
6. Pause path: POST `/pause` mid-loop → state=paused, không nhận step_completed nữa (Q5 phụ: HandleStepCompleted có guard state=paused không? Phase B chỉ check `IsPending(cur)` — nếu state=paused mid-flow, event đến muộn sẽ bị reject ErrInvalidTransition).

## Risk register

| Risk | Likelihood | Mitigation |
|---|---|---|
| Q1/Q2/Q3 ruling không kịp → Phase D block | Med | Document gap rõ trong file requirements; nhờ Architect chốt 3 ruling đồng thời. |
| Mid-flow pause race với step_completed inbound | Low | RecoveryLoop sẽ flush stale state sau 10min. Test case explicit. |
| Worker crash giữa "do work" và "emit step_completed" → state stuck pending | Low | RecoveryLoop Phase B (10min TTL) flip pending→failed. Acceptable per D3. |
| Trace_id/span_id rỗng nếu inbound publish không có (CMS Advance chưa nhúng OTel) | Med | Chấp nhận no-op; ghi vào step_log entry với field rỗng. Phase F sẽ hardening OTel SDK. |

## Execution order

1. **GATE**: Architect ruling Q1, Q2, Q3, Q4 (D-3 đến D-6 phụ thuộc).
2. D-1 (helper) → D-2 (discover wire) → D-6 (CMS extend failure) → unit tests.
3. D-3 hoặc D-4/D-5 song song tùy ruling Q2.
4. D-7 (subscribe block) sau khi 3 handler ready.
5. Integration test E2E.
6. Append `05_progress.md` (KHÔNG ghi đè) audit Phase D.

## Files modified/created

| Path | Action |
|---|---|
| `centralized-data-service/internal/handler/provisioning_emit.go` | NEW (helper) |
| `centralized-data-service/internal/handler/command_handler.go` | EDIT (HandleDiscover wire emit; thêm HandleShadowBind, HandleScheduleEnable) |
| `centralized-data-service/internal/handler/master_ddl_handler.go` | EDIT (wire emit nếu Q2=A) |
| `centralized-data-service/internal/server/worker_server.go` | EDIT (3 subscribe mới) |
| `cdc-cms-service/internal/service/provisioning_orchestrator.go` | (worker copy, không phải CMS port) EDIT extend HandleStepCompleted xử lý success=false |
| `centralized-data-service/internal/handler/provisioning_e2e_test.go` | NEW (integration test) |
| `agent/memory/workspaces/feature-cdc-integration/05_progress.md` | APPEND (sau Phase D done) |

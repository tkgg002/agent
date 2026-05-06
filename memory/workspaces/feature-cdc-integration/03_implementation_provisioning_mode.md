# 03_implementation — Provisioning Mode (Auto/Manual)

> Phase trong workspace `feature-cdc-integration`. Đọc kèm:
> - `00_context_provisioning_mode.md` — bối cảnh
> - `01_requirements_provisioning_mode.md` — R1..R5
> - `02_plan_provisioning_mode.md` — kiến trúc P1..P8

## I1. Code skeleton — `internal/service/provisioning_state_machine.go` (NEW, pure)

```go
package service

// State machine pure (no I/O) — dễ unit test, chia tách khỏi orchestrator.
// Bảng transitions là single source of truth; thêm step mới chỉ chỉnh table.

type ProvisioningState string

const (
    StateDraft           ProvisioningState = "draft"
    StateShadowPending   ProvisioningState = "shadow_pending"
    StateShadowActive    ProvisioningState = "shadow_active"
    StateMasterPending   ProvisioningState = "master_pending"
    StateMasterActive    ProvisioningState = "master_active"
    StateMappingPending  ProvisioningState = "mapping_pending"
    StateMappingReady    ProvisioningState = "mapping_ready"
    StateSchedulePending ProvisioningState = "schedule_pending"
    StateRunning         ProvisioningState = "running"
    StatePaused          ProvisioningState = "paused"
    StateFailed          ProvisioningState = "failed"
    StateArchived        ProvisioningState = "archived"
)

type StepDescriptor struct {
    Step          string            // canonical step name in audit log
    CmdSubject    string            // NATS subject to publish
    NextPending   ProvisioningState // intermediate state after dispatch
    NextOnSuccess ProvisioningState // final state after step_completed{success=true}
}

// Transitions defines the auto-flow chain.
var Transitions = map[ProvisioningState]StepDescriptor{
    StateDraft:           {"shadow_bind",     "cdc.cmd.shadow.bind",     StateShadowPending,   StateShadowActive},
    StateShadowActive:    {"master_bind",     "cdc.cmd.master.bind",     StateMasterPending,   StateMasterActive},
    StateMasterActive:    {"discover",        "cdc.cmd.discover",        StateMappingPending,  StateMappingReady},
    StateMappingReady:    {"schedule_enable", "cdc.cmd.schedule.enable", StateSchedulePending, StateRunning},
}

// CanAdvance: chỉ accept nếu state hiện tại nằm trong Transitions key.
func CanAdvance(s ProvisioningState) bool {
    _, ok := Transitions[s]
    return ok
}

// PendingToFinalize maps *_pending → caller's expected final state for
// idempotency check sau khi nhận event.
var PendingToFinalize = map[ProvisioningState]ProvisioningState{
    StateShadowPending:   StateShadowActive,
    StateMasterPending:   StateMasterActive,
    StateMappingPending:  StateMappingReady,
    StateSchedulePending: StateRunning,
}
```

### Test file `provisioning_state_machine_test.go`
- `TestTransitions_FullChain` — đi từ draft → running, mỗi state CanAdvance() đúng.
- `TestTransitions_TerminalStates` — paused/failed/archived NOT advanceable.
- `TestPendingToFinalize_All4` — đủ 4 *_pending có ánh xạ.

## I2. `internal/service/provisioning_orchestrator.go` (NEW, I/O layer)

```go
package service

import (
    "context"
    "encoding/json"
    "fmt"
    "time"

    "github.com/nats-io/nats.go"
    "go.uber.org/zap"
    "gorm.io/gorm"
)

type ProvisioningOrchestrator struct {
    db     *gorm.DB
    nats   *nats.Conn
    logger *zap.Logger
}

type StepCompletedEvent struct {
    SourceID      int64           `json:"source_id"`
    Step          string          `json:"step"`
    CorrelationID string          `json:"correlation_id"`
    Success       bool            `json:"success"`
    Error         string          `json:"error,omitempty"`
    StatsJSON     json.RawMessage `json:"stats,omitempty"`
    CompletedAt   time.Time       `json:"completed_at"`
}

// Advance: dispatched từ API manual click HOẶC từ HandleStepCompleted (nếu mode=auto).
// CAS UPDATE: chỉ chuyển nếu state hiện tại = expected `from`.
func (o *ProvisioningOrchestrator) Advance(ctx context.Context, sourceID int64, actor string) error {
    var row struct {
        ID                 int64
        ProvisioningMode   string
        ProvisioningState  string
    }
    if err := o.db.WithContext(ctx).Raw(
        `SELECT id, provisioning_mode, provisioning_state
           FROM cdc_system.source_object_registry
          WHERE id = ?`, sourceID).Scan(&row).Error; err != nil {
        return err
    }
    cur := ProvisioningState(row.ProvisioningState)
    desc, ok := Transitions[cur]
    if !ok {
        return fmt.Errorf("source %d in state %q is not advanceable", sourceID, cur)
    }
    correlationID := fmt.Sprintf("prov-%d-%d", sourceID, time.Now().UnixNano())

    // CAS UPDATE state -> *_pending + append step_log entry.
    res := o.db.WithContext(ctx).Exec(`
        UPDATE cdc_system.source_object_registry
           SET provisioning_state = ?,
               provisioning_step_log = provisioning_step_log
                   || jsonb_build_array(jsonb_build_object(
                       'seq', COALESCE(jsonb_array_length(provisioning_step_log),0)+1,
                       'step', ?,
                       'from_state', ?,
                       'to_state', ?,
                       'actor', ?,
                       'correlation_id', ?,
                       'started_at', NOW(),
                       'success', null,
                       'error', null
                   )),
               updated_at = NOW()
         WHERE id = ?
           AND provisioning_state = ?`,
        desc.NextPending, desc.Step, cur, desc.NextPending,
        actor, correlationID, sourceID, cur)
    if res.Error != nil {
        return res.Error
    }
    if res.RowsAffected == 0 {
        return fmt.Errorf("source %d state changed concurrently — abort", sourceID)
    }

    // Publish command.
    payload, _ := json.Marshal(map[string]any{
        "source_id":      sourceID,
        "correlation_id": correlationID,
        "triggered_by":   "provisioning",
    })
    if err := o.nats.Publish(desc.CmdSubject, payload); err != nil {
        // Best-effort: log + leave row in *_pending; recovery tick sẽ re-fire.
        o.logger.Warn("provisioning publish failed",
            zap.Int64("source_id", sourceID),
            zap.String("subject", desc.CmdSubject), zap.Error(err))
    }
    return nil
}

// HandleStepCompleted: NATS subscriber for cdc.evt.provisioning.step_completed.
// Idempotent: WHERE provisioning_state IN PendingToFinalize keys; lần 2 = no-op.
func (o *ProvisioningOrchestrator) HandleStepCompleted(msg *nats.Msg) {
    var ev StepCompletedEvent
    if err := json.Unmarshal(msg.Data, &ev); err != nil {
        o.logger.Warn("provisioning evt: bad payload", zap.Error(err))
        return
    }
    var row struct {
        ID                int64
        ProvisioningMode  string
        ProvisioningState string
    }
    if err := o.db.Raw(
        `SELECT id, provisioning_mode, provisioning_state
           FROM cdc_system.source_object_registry
          WHERE id = ?`, ev.SourceID).Scan(&row).Error; err != nil {
        o.logger.Warn("provisioning evt: lookup failed", zap.Error(err))
        return
    }
    cur := ProvisioningState(row.ProvisioningState)

    // Determine target final state.
    var target ProvisioningState
    if ev.Success {
        var ok bool
        target, ok = PendingToFinalize[cur]
        if !ok {
            // Not in pending — duplicate event after we already advanced. No-op.
            return
        }
    } else {
        target = StateFailed
    }

    // CAS UPDATE *_pending -> final + close step_log entry.
    res := o.db.Exec(`
        UPDATE cdc_system.source_object_registry
           SET provisioning_state = ?,
               last_step_error = NULLIF(?, ''),
               provisioning_step_log = jsonb_set(
                   provisioning_step_log,
                   array[(jsonb_array_length(provisioning_step_log)-1)::text],
                   (provisioning_step_log->-1)
                       || jsonb_build_object(
                           'completed_at', NOW(),
                           'success', ?::boolean,
                           'error', NULLIF(?, '')::text)),
               updated_at = NOW()
         WHERE id = ?
           AND provisioning_state = ?`,
        target, ev.Error, ev.Success, ev.Error, ev.SourceID, cur)
    if res.Error != nil {
        o.logger.Warn("provisioning evt: update failed", zap.Error(res.Error))
        return
    }
    if res.RowsAffected == 0 {
        return // raced — another instance already handled.
    }

    // Auto-fan if mode=auto AND step succeeded AND target is advanceable.
    if ev.Success && row.ProvisioningMode == "auto" && CanAdvance(target) {
        if err := o.Advance(context.Background(), ev.SourceID, "orchestrator"); err != nil {
            o.logger.Warn("provisioning auto-advance failed",
                zap.Int64("source_id", ev.SourceID), zap.Error(err))
        }
    }
}

// Pause/Resume/Retry/Archive: tương tự, mỗi cái CAS update state + log entry.
// Implementation chi tiết ở phase B2.
```

## I3. `internal/handler/provisioning_handler.go` (NEW, NATS glue)

Subscribe + dispatch cho 4 cmd subjects + 1 evt subject. Body chỉ là façade gọi orchestrator method (vì state logic đã ở service layer).

## I4. `internal/api/provisioning_api.go` (NEW, REST)

Route group `/api/cms/sources/:id/provisioning/*`:
- `GET /` → SELECT row + decode JSONB → response.
- `POST /advance` → orchestrator.Advance(id, actor=authUser).
- `POST /pause`, `/resume`, `/retry`, `/archive` → tương ứng.
- `POST /mode` body `{"mode":"auto"|"manual"}` → simple UPDATE + log entry.

Auth middleware: cần xác minh đường path hiện có (web layer trong centralized-data-service); nếu chưa có thì wire giống các endpoint admin hiện có.

## I5. Integration điểm với handler hiện có

**KHÔNG sửa logic** của các handler hiện có. CHỈ thêm 1 publish call ở cuối mỗi handler khi `correlation_id` có prefix `prov-`:

```go
// cuối HandleScanFields / HandleDiscover / ... / TransmuteHandler
if strings.HasPrefix(req.CorrelationID, "prov-") && h.natsConn != nil {
    evt, _ := json.Marshal(map[string]any{
        "source_id":      req.SourceID,
        "step":           "discover", // hoặc step name của handler đó
        "correlation_id": req.CorrelationID,
        "success":        runErr == nil,
        "error":          errStr,
        "completed_at":   time.Now().UTC(),
    })
    _ = h.natsConn.Publish("cdc.evt.provisioning.step_completed", evt)
}
```

**Handler chưa tồn tại** (cần tạo MOCK hoặc dùng existing):
- `cdc.cmd.shadow.bind` — hiện chưa có. Cần map sang `cdc.cmd.scan-fields` + tạo shadow_binding row HOẶC tạo handler mới (Phase D2).
- `cdc.cmd.master.bind` — chưa có. Map sang flow tạo `master_binding`.
- `cdc.cmd.schedule.enable` — chưa có. Đơn giản UPDATE `transmute_schedule.is_enabled=true`.

→ Phase D quan trọng: connect 4 cmd subject đến code path đã có sẵn.

## I6. Worker boot wiring (`internal/server/worker_server.go`)

```go
// Sau scheduler + jobMonitor đã wire (P4):
provOrch := service.NewProvisioningOrchestrator(db, natsClient.Conn, logger)
provHandler := handler.NewProvisioningHandler(provOrch, logger)

if _, err := natsClient.Conn.Subscribe("cdc.evt.provisioning.step_completed",
    provOrch.HandleStepCompleted); err != nil {
    return nil, fmt.Errorf("subscribe step_completed: %w", err)
}
if _, err := natsClient.Conn.Subscribe("cdc.cmd.provisioning.advance",
    provHandler.HandleAdvance); err != nil {
    return nil, fmt.Errorf("subscribe advance: %w", err)
}
// pause/resume/retry/archive subscribe similarly.

// Recovery tick:
go provOrch.RecoveryLoop(ctx, time.Minute)
```

## I7. Testing strategy
- **Unit**: state machine pure (I1) — bảng transition + boundary.
- **Integration**: spin up testcontainer postgres + NATS, test Advance() CAS guard, HandleStepCompleted idempotency.
- **Smoke E2E**: tạo source mode=auto → 30s sau state=running; tạo source mode=manual → click 4 lần → state=running.
- **Concurrency**: 2 worker instance chạy song song, fire 1 event → chỉ 1 worker UPDATE thành công (RowsAffected=1 vs 0).

## I8. Migration order (chronological in deploy)
1. `047_source_provisioning_state.sql` (DDL + backfill).
2. Deploy worker với feature flag `PROVISIONING_ORCHESTRATOR_ENABLED=false` (subscribe nhưng disable auto-fan).
3. Bật flag = true sau khi soak test.
4. Frontend CMS phát triển song song trong workspace `feature-cms-fe-overhaul/`.

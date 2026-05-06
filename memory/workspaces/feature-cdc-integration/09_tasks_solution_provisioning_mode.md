# 09_tasks_solution — Provisioning Mode (chi tiết solution per task)

> Đối ứng với `08_tasks_provisioning_mode.md`. Mỗi task ở đây có:
> - **Solution**: code/SQL skeleton sẵn sàng dán
> - **Verification**: command/SQL để kiểm
> - **Rollback**: cách quay lui nếu hỏng

---

## A1. Migration 047

### Solution — file `migrations/cdc/047_source_provisioning_state.sql`
```sql
-- ============================================================
-- Provisioning Mode (auto/manual) — state machine on
-- cdc_system.source_object_registry. Adds 4 columns + 1 index +
-- backfill stamp for currently-active sources.
--
-- Idempotent: ADD COLUMN IF NOT EXISTS, CREATE INDEX IF NOT EXISTS,
-- backfill UPDATE only touches rows still at default 'draft'.
-- ============================================================
BEGIN;

DO $cols$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_class c JOIN pg_namespace n
        ON n.oid = c.relnamespace
        WHERE n.nspname='cdc_system' AND c.relname='source_object_registry') THEN
        RAISE NOTICE '[047] source_object_registry not present - skipping';
        RETURN;
    END IF;

    EXECUTE 'ALTER TABLE cdc_system.source_object_registry
             ADD COLUMN IF NOT EXISTS provisioning_mode VARCHAR(20)
               DEFAULT ''manual''
               CHECK (provisioning_mode IN (''auto'',''manual''))';

    EXECUTE 'ALTER TABLE cdc_system.source_object_registry
             ADD COLUMN IF NOT EXISTS provisioning_state VARCHAR(40)
               DEFAULT ''draft''';

    EXECUTE 'ALTER TABLE cdc_system.source_object_registry
             ADD COLUMN IF NOT EXISTS provisioning_step_log JSONB
               DEFAULT ''[]''::jsonb';

    EXECUTE 'ALTER TABLE cdc_system.source_object_registry
             ADD COLUMN IF NOT EXISTS last_step_error TEXT';

    RAISE NOTICE '[047] source_object_registry provisioning columns ensured';
END $cols$;

UPDATE cdc_system.source_object_registry
   SET provisioning_state = 'running',
       provisioning_step_log = jsonb_build_array(
           jsonb_build_object(
               'seq', 1,
               'step', 'backfill',
               'from_state', 'draft',
               'to_state', 'running',
               'actor', 'migration-047',
               'correlation_id', null,
               'started_at', NOW(),
               'completed_at', NOW(),
               'success', true,
               'error', null))
 WHERE is_active = true
   AND provisioning_state = 'draft';

CREATE INDEX IF NOT EXISTS idx_sor_provisioning_state
  ON cdc_system.source_object_registry (provisioning_state)
  WHERE provisioning_state IN
    ('shadow_pending','master_pending','mapping_pending','schedule_pending','failed');

COMMIT;
```

### Verification
```bash
docker exec -i gpay-postgres-cdc psql -U gpay_admin -d cdc_dw \
  < migrations/cdc/047_source_provisioning_state.sql
docker exec gpay-postgres-cdc psql -U gpay_admin -d cdc_dw -c \
  "SELECT column_name, column_default FROM information_schema.columns
    WHERE table_schema='cdc_system' AND table_name='source_object_registry'
      AND column_name LIKE 'provisioning_%' OR column_name='last_step_error';"
docker exec gpay-postgres-cdc psql -U gpay_admin -d cdc_dw -c \
  "SELECT provisioning_state, count(*) FROM cdc_system.source_object_registry GROUP BY 1;"
```
Expected: 4 cột default đúng; group có row `running` (= số `is_active=true` cũ) và `draft` (= 0 hoặc số inactive).

### Rollback
```sql
ALTER TABLE cdc_system.source_object_registry DROP COLUMN IF EXISTS provisioning_mode;
ALTER TABLE cdc_system.source_object_registry DROP COLUMN IF EXISTS provisioning_state;
ALTER TABLE cdc_system.source_object_registry DROP COLUMN IF EXISTS provisioning_step_log;
ALTER TABLE cdc_system.source_object_registry DROP COLUMN IF EXISTS last_step_error;
DROP INDEX IF EXISTS cdc_system.idx_sor_provisioning_state;
```

---

## A2. Model field

### Solution — Edit `internal/model/source_object_registry.go`
Thêm vào struct (sau `Notes`):
```go
ProvisioningMode    string          `gorm:"column:provisioning_mode;default:manual" json:"provisioning_mode"`
ProvisioningState   string          `gorm:"column:provisioning_state;default:draft" json:"provisioning_state"`
ProvisioningStepLog json.RawMessage `gorm:"column:provisioning_step_log;type:jsonb;default:'[]'" json:"provisioning_step_log"`
LastStepError       *string         `gorm:"column:last_step_error" json:"last_step_error"`
```

### Verification
```bash
go build ./internal/model/...
go vet ./internal/model/...
```

---

## A3. State machine pure

### Solution
Code đã có ở `03_implementation_provisioning_mode.md` §I1.

### Test file `internal/service/provisioning_state_machine_test.go`
```go
package service

import "testing"

func TestProvisioningStateMachine_FullChain(t *testing.T) {
    states := []ProvisioningState{StateDraft, StateShadowActive, StateMasterActive, StateMappingReady}
    for _, s := range states {
        if !CanAdvance(s) {
            t.Fatalf("expected %s advanceable", s)
        }
    }
}

func TestProvisioningStateMachine_TerminalNotAdvanceable(t *testing.T) {
    for _, s := range []ProvisioningState{StateRunning, StatePaused, StateFailed, StateArchived} {
        if CanAdvance(s) {
            t.Fatalf("%s must not be advanceable", s)
        }
    }
}

func TestProvisioningStateMachine_PendingMappings(t *testing.T) {
    if len(PendingToFinalize) != 4 {
        t.Fatalf("expected 4 pending->final mappings, got %d", len(PendingToFinalize))
    }
}
```

### Verification
```bash
go test ./internal/service -run TestProvisioningStateMachine -count=1 -v
```

---

## B1. Orchestrator service

Skeleton ở `03_implementation_provisioning_mode.md` §I2. Bổ sung 4 method còn lại:

```go
// Pause: chỉ chuyển từ running → paused.
func (o *ProvisioningOrchestrator) Pause(ctx context.Context, sourceID int64, actor string) error {
    return o.casTransition(ctx, sourceID, StateRunning, StatePaused, "pause", actor, "")
}
func (o *ProvisioningOrchestrator) Resume(ctx context.Context, sourceID int64, actor string) error {
    if err := o.casTransition(ctx, sourceID, StatePaused, StateRunning, "resume", actor, ""); err != nil {
        return err
    }
    return nil
}
func (o *ProvisioningOrchestrator) Archive(ctx context.Context, sourceID int64, actor string) error {
    // Archive from any state (no `from` constraint) — but log carefully.
    res := o.db.WithContext(ctx).Exec(`
        UPDATE cdc_system.source_object_registry
           SET provisioning_state = 'archived',
               is_active = false,
               provisioning_step_log = provisioning_step_log
                   || jsonb_build_array(jsonb_build_object(
                       'seq', COALESCE(jsonb_array_length(provisioning_step_log),0)+1,
                       'step','archive',
                       'from_state', provisioning_state,
                       'to_state','archived',
                       'actor', ?,
                       'completed_at', NOW(),
                       'success', true)),
               updated_at = NOW()
         WHERE id = ? AND provisioning_state <> 'archived'`, actor, sourceID)
    return res.Error
}
func (o *ProvisioningOrchestrator) Retry(ctx context.Context, sourceID int64, actor string) error {
    // Reset state to last successful state from log, clear last_step_error,
    // then call Advance() to re-fire the failed step.
    // Implementation skeleton — chi tiết validate + dedup ở phase B.
    ...
}
```

### Verification
- Unit test với postgres testcontainer (sử dụng `github.com/testcontainers/testcontainers-go`).
- Concurrency test: spawn 2 goroutine cùng gọi `Advance(1)` → đúng 1 nhận `RowsAffected=1`.

---

## C1. REST API

### Solution — `internal/api/provisioning_api.go` (skeleton)
```go
package api

import (
    "encoding/json"
    "net/http"
    "strconv"

    "centralized-data-service/internal/service"
    "github.com/go-chi/chi/v5"
)

type ProvisioningAPI struct {
    orch *service.ProvisioningOrchestrator
}

func NewProvisioningAPI(orch *service.ProvisioningOrchestrator) *ProvisioningAPI {
    return &ProvisioningAPI{orch: orch}
}

func (p *ProvisioningAPI) Routes(r chi.Router) {
    r.Get("/{id}/provisioning", p.handleGet)
    r.Post("/{id}/provisioning/advance", p.handleAdvance)
    r.Post("/{id}/provisioning/pause", p.handlePause)
    r.Post("/{id}/provisioning/resume", p.handleResume)
    r.Post("/{id}/provisioning/retry", p.handleRetry)
    r.Post("/{id}/provisioning/archive", p.handleArchive)
    r.Post("/{id}/provisioning/mode", p.handleMode)
}

func (p *ProvisioningAPI) handleAdvance(w http.ResponseWriter, r *http.Request) {
    id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
    actor := authActor(r)  // helper from auth middleware
    if err := p.orch.Advance(r.Context(), id, actor); err != nil {
        http.Error(w, err.Error(), http.StatusConflict)
        return
    }
    w.WriteHeader(http.StatusAccepted)
}
// ... 6 handler còn lại tương tự ...
```

> Router framework hiện tại của centralized-data-service cần xác minh — có thể là `chi`, `gin`, hoặc `echo`. Muscle check trước, áp pattern phù hợp.

### Verification
```bash
curl -s -H "Authorization: Bearer $TOKEN" \
  -X POST http://localhost:8080/api/cms/sources/1/provisioning/advance | jq
docker exec gpay-postgres-cdc psql -U gpay_admin -d cdc_dw \
  -c "SELECT id, provisioning_state, jsonb_array_length(provisioning_step_log)
        FROM cdc_system.source_object_registry WHERE id=1;"
```

---

## D2. shadow.bind / master.bind / schedule.enable handler

### Solution — `internal/handler/provisioning_step_handlers.go` (NEW)

```go
package handler

func (h *ProvisioningStepHandler) HandleShadowBind(msg *nats.Msg) {
    var req struct {
        SourceID      int64  `json:"source_id"`
        CorrelationID string `json:"correlation_id"`
    }
    json.Unmarshal(msg.Data, &req)

    err := h.svc.CreateShadowBinding(context.Background(), req.SourceID)
    h.publishStepCompleted("shadow_bind", req.SourceID, req.CorrelationID, err)
}

// Tương tự cho HandleMasterBind, HandleScheduleEnable.

func (h *ProvisioningStepHandler) publishStepCompleted(step string, sourceID int64, corrID string, runErr error) {
    success := runErr == nil
    errStr := ""
    if runErr != nil {
        errStr = service.SanitizeFreeformText(runErr.Error(), 2000)
    }
    evt, _ := json.Marshal(map[string]any{
        "source_id":      sourceID,
        "step":           step,
        "correlation_id": corrID,
        "success":        success,
        "error":          errStr,
        "completed_at":   time.Now().UTC().Format(time.RFC3339Nano),
    })
    h.natsConn.Publish("cdc.evt.provisioning.step_completed", evt)
}
```

> Logic `CreateShadowBinding`/`CreateMasterBinding` phải reuse code đã có trong `metadata_registry_service.go`/`schema_manager.go`. Không tự viết lại.

---

## E1+E2. Smoke E2E commands

### Auto mode
```bash
SRC_PAYLOAD='{"object_code":"orders_e2e_auto","source_connection_id":1,...,"provisioning_mode":"auto"}'
ID=$(curl -s -X POST http://localhost:8080/api/cms/sources -d "$SRC_PAYLOAD" | jq .id)
sleep 60
curl -s http://localhost:8080/api/cms/sources/$ID/provisioning | jq '{state:.provisioning_state, log_len:(.provisioning_step_log|length)}'
# Expect: {"state":"running","log_len":4}
```

### Manual mode
```bash
SRC_PAYLOAD='{"object_code":"orders_e2e_manual",...,"provisioning_mode":"manual"}'
ID=$(curl -s -X POST http://localhost:8080/api/cms/sources -d "$SRC_PAYLOAD" | jq .id)
for i in 1 2 3 4; do
  curl -s -X POST http://localhost:8080/api/cms/sources/$ID/provisioning/advance
  sleep 5
done
curl -s http://localhost:8080/api/cms/sources/$ID/provisioning | jq '.provisioning_state'
# Expect: "running"
```

---

## Rollback toàn phase
1. Set env `PROVISIONING_ORCHESTRATOR_ENABLED=false` → orchestrator không subscribe.
2. Optional: revert migration 047 (rollback SQL ở §A1).
3. Source `is_active=true` quay về dùng path cũ (transmute_scheduler vẫn chạy độc lập).

## Câu hỏi mở (R5 plan) cần user xác nhận
- Q1..Q5 ở `01_requirements_provisioning_mode.md` §R5 — đề xuất default đã ghi, nhưng cần xác nhận trước khi viết code.

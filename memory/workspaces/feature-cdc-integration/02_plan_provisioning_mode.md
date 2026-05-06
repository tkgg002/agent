# 02_plan — Source Provisioning Mode

## P1. Thiết kế tổng quan (architecture)

```
┌─────────────────────────────────────────────────────────────────────┐
│                      CMS / API Frontend                              │
│                                                                      │
│  POST /sources                              POST /sources/:id/       │
│  {provisioning_mode: "auto"|"manual"}        provisioning/advance    │
│                                                                      │
└────────────────┬───────────────────────────────────┬─────────────────┘
                 │                                   │
                 ▼                                   ▼
       ┌─────────────────────────────────────────────────────┐
       │   ProvisioningAPI  (internal/api/provisioning.go)   │
       │   - Validate role / payload                         │
       │   - Translate to orchestrator op                    │
       └────────────────────────┬────────────────────────────┘
                                │
                                ▼
       ┌─────────────────────────────────────────────────────┐
       │ ProvisioningOrchestrator (service)                  │
       │ - State machine (table-driven)                      │
       │ - DispatchNextStep(source_id) — auto + manual share │
       │ - HandleStepCompleted(evt) — auto-fan if mode=auto  │
       │ - Pause/Resume/Retry/Archive                        │
       └────────┬────────────┬───────────────────┬───────────┘
                │            │                   │
                │ NATS pub   │ DB UPDATE         │ NATS sub
                │ cdc.cmd.X  │ source_object_    │ cdc.evt.provisioning.
                ▼            ▼ registry          ▼ step_completed
       ┌────────────────────────────────────────────────────┐
       │  Existing handlers (CommandHandler, TransmuteH.,   │
       │  SchemaManager, ...) — KHÔNG phải sửa, chỉ thêm    │
       │  emit `cdc.evt.provisioning.step_completed` ở      │
       │  cuối từng step.                                   │
       └────────────────────────────────────────────────────┘
```

**Tách concern (architect P4 ruling apply lại)**:
- Orchestrator KHÔNG biết logic shadow/master/mapping bên trong — chỉ biết "publish cmd, chờ evt".
- Existing handlers không biết về orchestrator — chỉ emit completion event với `correlation_id` mang prefix `prov-`.
- ProvisioningHandler (NATS subscriber) là kẻ trung gian: dịch event domain (`cdc.evt.shadow.bind.completed`) → meta event (`cdc.evt.provisioning.step_completed`).

## P2. Data model changes

### Migration `047_source_provisioning_state.sql`
```sql
BEGIN;

DO $cols$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_class c JOIN pg_namespace n
        ON n.oid=c.relnamespace WHERE n.nspname='cdc_system'
        AND c.relname='source_object_registry') THEN
        RETURN;
    END IF;

    EXECUTE 'ALTER TABLE cdc_system.source_object_registry
             ADD COLUMN IF NOT EXISTS provisioning_mode VARCHAR(20)
               DEFAULT ''manual'' CHECK (provisioning_mode IN (''auto'',''manual''))';

    EXECUTE 'ALTER TABLE cdc_system.source_object_registry
             ADD COLUMN IF NOT EXISTS provisioning_state VARCHAR(40)
               DEFAULT ''draft''';

    EXECUTE 'ALTER TABLE cdc_system.source_object_registry
             ADD COLUMN IF NOT EXISTS provisioning_step_log JSONB
               DEFAULT ''[]''::jsonb';

    EXECUTE 'ALTER TABLE cdc_system.source_object_registry
             ADD COLUMN IF NOT EXISTS last_step_error TEXT';
END $cols$;

-- Backfill rows hiện có (R2.5): set state=running cho is_active=true.
UPDATE cdc_system.source_object_registry
   SET provisioning_state = 'running',
       provisioning_step_log = jsonb_build_array(
           jsonb_build_object(
               'seq', 1,
               'step', 'backfill',
               'from_state', 'draft',
               'to_state', 'running',
               'actor', 'migration-047',
               'completed_at', NOW(),
               'success', true,
               'error', null
           ))
 WHERE is_active = true
   AND provisioning_state = 'draft';  -- chỉ row chưa stamp

CREATE INDEX IF NOT EXISTS idx_sor_provisioning_state
  ON cdc_system.source_object_registry (provisioning_state)
  WHERE provisioning_state IN ('shadow_pending','master_pending','mapping_pending','schedule_pending','failed');

COMMIT;
```

### Model update `internal/model/source_object_registry.go`
```go
ProvisioningMode    string          `gorm:"column:provisioning_mode;default:manual"`
ProvisioningState   string          `gorm:"column:provisioning_state;default:draft"`
ProvisioningStepLog json.RawMessage `gorm:"column:provisioning_step_log;type:jsonb;default:'[]'"`
LastStepError       *string         `gorm:"column:last_step_error"`
```

## P3. NATS subjects

| Subject | Direction | Mô tả |
|---------|-----------|-------|
| `cdc.cmd.provisioning.advance` | Pub→Orch | Manual click advance từ API |
| `cdc.cmd.provisioning.pause` | Pub→Orch | API |
| `cdc.cmd.provisioning.resume` | Pub→Orch | API |
| `cdc.cmd.provisioning.retry` | Pub→Orch | API |
| `cdc.evt.provisioning.step_completed` | Pub←Handler / Sub→Orch | Step (shadow/master/mapping/schedule) đã xong |
| `cdc.evt.provisioning.state_changed` | Pub←Orch | Broadcast state change cho FE realtime |

## P4. State transition table (sẽ implement table-driven)

| Current state | Action | Cmd publish | Next pending state | Next final state (on success) |
|---------------|--------|-------------|--------------------|-------------------------------|
| draft | advance | `cdc.cmd.shadow.bind` | shadow_pending | shadow_active |
| shadow_active | advance | `cdc.cmd.master.bind` | master_pending | master_active |
| master_active | advance | `cdc.cmd.discover` | mapping_pending | mapping_ready |
| mapping_ready | advance | `cdc.cmd.schedule.enable` | schedule_pending | running |
| running | pause | (no cmd) | running → paused | paused |
| paused | resume | (re-fire last cmd) | (last pending) | running |
| failed | retry | (re-fire last cmd) | (last pending) | (resume flow) |

## P5. Critical files (mục tiêu sửa)

### NEW (Muscle tạo)
| Path | Vai trò |
|------|---------|
| `migrations/cdc/047_source_provisioning_state.sql` | DDL |
| `internal/service/provisioning_orchestrator.go` | State machine core |
| `internal/service/provisioning_state_machine.go` | Pure transition table (test dễ) |
| `internal/handler/provisioning_handler.go` | NATS sub: `cdc.evt.provisioning.step_completed` + `cdc.cmd.provisioning.*` |
| `internal/api/provisioning_api.go` | REST endpoints (R1.5) |
| `internal/service/provisioning_orchestrator_test.go` | Unit test state machine + boundary cases |

### EDIT (Muscle sửa, tối thiểu impact)
| Path | Vai trò |
|------|---------|
| `internal/model/source_object_registry.go` | Thêm 4 field |
| `internal/server/worker_server.go` | Wire orchestrator + handler subscribe |
| `internal/handler/command_handler.go` | Sau mỗi step (scan-fields, discover, ...) emit `cdc.evt.provisioning.step_completed` với `correlation_id` (chỉ thêm 1 publish call/step, không sửa logic) |
| `internal/handler/transmute_handler.go` | `running` state requires `transmute_schedule.is_enabled=true` — emit step_completed sau enable (best-effort flag) |

### KHÔNG động (vẫn hoạt động cũ)
- `internal/service/transmuter.go` — không liên quan provisioning, chỉ chạy khi state=`running`.
- `internal/service/transmute_scheduler.go` — đã có cron, không phải sửa.
- Existing migrations 001..046 — append-only.

## P6. Phân kỳ thực thi (rolling)

### Phase A — Foundation (Muscle 1 ngày)
- A1. Migration 047 + verify idempotent.
- A2. Model field + repo mở rộng.
- A3. State machine pure (file `provisioning_state_machine.go`) + unit test bảng transition.
- **Gate**: A1 áp DB ok, `go test ./internal/service -run TestProvisioningStateMachine` PASS.

### Phase B — Orchestrator (Muscle 1 ngày)
- B1. `ProvisioningOrchestrator` service: dispatch + handle event.
- B2. ProvisioningHandler NATS subscriber.
- B3. Wire worker boot.
- **Gate**: Local smoke — publish `cdc.evt.provisioning.step_completed` giả → orchestrator UPDATE state, emit cmd kế.

### Phase C — REST API (Muscle 0.5 ngày)
- C1. ProvisioningAPI handlers.
- C2. Auth middleware integration.
- C3. Wire vào server router (web layer — cần xác minh location).
- **Gate**: `curl POST /sources/:id/provisioning/advance` qua manual mode → state advance.

### Phase D — Existing handlers emit step_completed (Muscle 0.5 ngày)
- D1. Sửa CommandHandler.HandleScanFields/HandleDiscover/... để emit event sau khi xong (no-op nếu không có `correlation_id` prefix `prov-`).
- D2. TransmuteHandler đã emit completion (P4) — chỉ thêm relay sang provisioning subject nếu cần.
- **Gate**: E2E auto mode demo — POST /sources với mode=auto → 30s sau state=running.

### Phase E — CMS click flow + audit verify (Muscle 0.5 ngày)
- E1. POST /sources với mode=manual → state=draft.
- E2. Click advance 4 lần → state qua shadow_active → master_active → mapping_ready → running.
- E3. Đọc `provisioning_step_log` — đủ 4 entry với `actor=cms:<user>`.
- **Gate**: Smoke pass, log đầy đủ, idempotent (click 2 lần cùng advance = 1 entry).

## P7. Rủi ro & mitigation

| Rủi ro | Mức | Mitigation |
|--------|-----|-----------|
| Orchestrator crash giữa chừng → state stuck `*_pending` | Cao | Boot recovery tick (R2.2) re-publish cmd |
| Race khi 2 instance worker cùng nhận 1 step_completed event | Cao | UPDATE WHERE state=expected (CAS); WIN lấy 0 row = đã xử lý |
| User đổi mode `auto→manual` ngay khi advance đang chạy | TB | Mode đọc lúc nhận event (lazy) — nếu đổi sang manual, orchestrator skip dispatch kế |
| Existing source đang `is_active` không có log → CMS hiển thị rỗng | Thấp | Backfill stamp `migration-047` entry (P2 SQL) |
| Migration block UPDATE 47k row hiện có | Thấp | Backfill chỉ UPDATE row `is_active=true AND state='draft'` — đảm bảo subset nhỏ; chạy outside peak |

## P8. Strategy rollout
1. Migration trước (đơn lẻ, idempotent — chạy được trên prod live).
2. Code deploy backwards-compat: source mới mặc định `manual`, source cũ stamp `running`.
3. Feature flag (env): `PROVISIONING_ORCHESTRATOR_ENABLED=true|false` — disable = orchestrator boot nhưng không subscribe (rollback nhanh nếu lỗi).
4. Sau khi stable 1 tuần → bật feature flag default trong config.

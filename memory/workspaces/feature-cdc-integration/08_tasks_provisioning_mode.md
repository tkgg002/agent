# 08_tasks — Provisioning Mode (Auto/Manual)

> Bộ task của phase, breakdown theo Phase A..E ở `02_plan_provisioning_mode.md` §P6.
> Mỗi task có Definition of Done (DoD) riêng để Muscle thực thi không cần hỏi lại.

## Phase A — Foundation

### A1. Migration `047_source_provisioning_state.sql`
- **Owner**: Muscle
- **Action**: Tạo file migration idempotent thêm 4 cột (`provisioning_mode`, `provisioning_state`, `provisioning_step_log`, `last_step_error`) + index `idx_sor_provisioning_state` + backfill stamp.
- **DoD**:
  - Apply lần 1 — ok, 4 cột hiện diện đúng default.
  - Apply lần 2 — `column ... already exists, skipping`.
  - `SELECT COUNT(*) FROM cdc_system.source_object_registry WHERE provisioning_state='running'` ≥ số source `is_active=true` cũ.
  - `idx_sor_provisioning_state` xuất hiện trong `\di cdc_system.*`.

### A2. Model field mở rộng `internal/model/source_object_registry.go`
- **Owner**: Muscle
- **Action**: Thêm 4 field GORM với tag matching A1.
- **DoD**:
  - `go build ./...` PASS.
  - `go vet ./...` PASS.

### A3. State machine pure `internal/service/provisioning_state_machine.go` + test
- **Owner**: Muscle
- **Action**: Implement bảng `Transitions` + `PendingToFinalize` + helper `CanAdvance`.
- **DoD**:
  - `go test ./internal/service -run TestProvisioningStateMachine -count=1 -v` PASS với ≥3 case (full chain, terminal block, all 4 pending mappings).

## Phase B — Orchestrator

### B1. `internal/service/provisioning_orchestrator.go`
- **Owner**: Muscle
- **Action**: Implement `NewProvisioningOrchestrator`, `Advance`, `HandleStepCompleted`, `Pause`, `Resume`, `Retry`, `Archive`, `SetMode`, `RecoveryLoop`. Toàn bộ CAS UPDATE.
- **DoD**:
  - Build pass.
  - Unit test `provisioning_orchestrator_test.go`: testcontainer postgres + nats embedded, test cả happy path + concurrency CAS (2 goroutine chạy Advance cùng row → 1 success, 1 returns `state changed concurrently`).

### B2. `internal/handler/provisioning_handler.go`
- **Owner**: Muscle
- **Action**: NATS subscriber thin layer cho 4 cmd subject + relay sang orchestrator method.
- **DoD**:
  - Build pass.
  - Smoke: publish `cdc.cmd.provisioning.advance` payload `{source_id:1}` → log "advance dispatched".

### B3. Wire vào `internal/server/worker_server.go`
- **Owner**: Muscle
- **Action**: Khởi tạo orchestrator + handler, subscribe 5 subject, start RecoveryLoop goroutine.
- **DoD**:
  - Worker boot xanh, log "provisioning orchestrator registered".
  - Không regression P1..P4 (jobMonitor vẫn close schedule_id).
  - Feature flag env `PROVISIONING_ORCHESTRATOR_ENABLED` respected (nếu false → bỏ qua subscribe).

## Phase C — REST API

### C1. `internal/api/provisioning_api.go`
- **Owner**: Muscle
- **Action**: Route group + 7 endpoint (R1.5).
- **DoD**:
  - Build pass.
  - `curl GET /api/cms/sources/1/provisioning` → trả JSON đầy đủ field.
  - `curl POST .../advance` → 200 + state advance trong DB.

### C2. Auth middleware integration
- **Owner**: Muscle
- **Action**: Xác minh path auth hiện có; nếu chưa có ở route group này thì add middleware từ JWT/session.
- **DoD**:
  - Request không token → 401.
  - Request token role thường + body `{mode:"auto"}` → 403.
  - Request token admin → 200.

### C3. Wire vào router
- **Owner**: Muscle
- **Action**: Mount route group ở `cmd/server/main.go` hoặc `cmd/worker/main.go` (xác định kiến trúc — phía nào host CMS REST).
- **DoD**:
  - `make run` không lỗi route conflict.
  - Smoke list 7 endpoint qua `curl -i`.

## Phase D — Existing handler emit step_completed

### D1. CommandHandler emit (scan-fields, discover, sync-register)
- **Owner**: Muscle
- **Action**: Sau khi mỗi handler hoàn tất, nếu `correlation_id` có prefix `prov-` thì publish `cdc.evt.provisioning.step_completed`.
- **DoD**:
  - Trigger qua orchestrator → orchestrator nhận event → state advance.

### D2. Tạo shadow.bind / master.bind / schedule.enable command handler
- **Owner**: Muscle
- **Action**: 3 handler mới subscribe 3 subject mới — wrap quanh code đã có (createShadowBinding, createMasterBinding, enableSchedule). Mỗi handler emit step_completed cuối cùng.
- **DoD**:
  - 3 subject publish riêng biệt → DB row tương ứng tạo/UPDATE đúng.

### D3. TransmuteHandler relay (nếu cần)
- **Owner**: Muscle
- **Action**: TransmuteHandler đã emit `cdc.evt.transmute.completed` (P4). Thêm relay: nếu `correlation_id` prefix `prov-` thì cũng publish `cdc.evt.provisioning.step_completed`.
- **DoD**:
  - Trigger transmute qua orchestrator → step_completed cũng phát.

## Phase E — E2E smoke + audit

### E1. Auto mode E2E
- **Owner**: Muscle
- **Action**:
  ```bash
  curl -X POST /api/cms/sources -d '{"object_code":"orders_e2e","provisioning_mode":"auto",...}'
  sleep 60
  curl GET /api/cms/sources/<id>/provisioning
  ```
- **DoD**: state=`running`, step_log có 4 entry success, mode=auto, mỗi entry có actor=`orchestrator`.

### E2. Manual mode E2E
- **Owner**: Muscle
- **Action**:
  ```bash
  curl -X POST /api/cms/sources -d '{"object_code":"orders_manual","provisioning_mode":"manual",...}'
  curl -X POST .../advance  # ×4
  ```
- **DoD**: state=`running` sau 4 click; mỗi entry actor=`cms:<user>`.

### E3. Pause/Resume/Retry/Archive
- **Owner**: Muscle
- **Action**: Test 4 action endpoint trên source ở trạng thái phù hợp.
- **DoD**: State chuyển đúng theo bảng P4 plan; log entry đầy đủ; idempotent.

### E4. Idempotency double-fire
- **Owner**: Muscle
- **Action**: Publish `cdc.evt.provisioning.step_completed` 2 lần cùng correlation_id.
- **DoD**: Lần 2 = no-op (RowsAffected=0), state không nhảy 2 bước.

### E5. Recovery tick
- **Owner**: Muscle
- **Action**: Set state=`shadow_pending` thủ công, kill orchestrator, restart sau 2 phút (TTL 1 phút).
- **DoD**: RecoveryLoop re-publish `cdc.cmd.shadow.bind` cùng correlation_id; flow tiếp tục.

## Cross-cutting

### X1. APPEND `05_progress.md`
- **Owner**: Muscle (sau khi từng task DoD pass)
- **Action**: Append entry per task — KHÔNG overwrite.

### X2. Lesson global
- **Owner**: Muscle
- **Action**: Sau Phase E pass, viết lesson Global Pattern: "Multi-step domain workflow cần lớp orchestrator riêng tách logic dispatch khỏi domain handler" vào `agent/memory/global/lessons.md`.

### X3. Security gate
- **Owner**: Muscle
- **Action**: CLAUDE.md §8 — chạy `/security-agent` review.
- **DoD**: Không finding High/Critical về authz bypass / state forge.

## Estimation tổng
| Phase | Dev-day |
|-------|---------|
| A | 1 |
| B | 1 |
| C | 0.5 |
| D | 0.5 |
| E | 0.5 |
| **Total** | **3.5** |

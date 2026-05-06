# Phase D — Worker step_completed Emit (Requirements)

**Workspace**: feature-cdc-integration
**Phase**: provisioning_mode / D (close auto-loop)
**Status**: PROPOSED — chờ Architect duyệt scope vì có gap so với assumption ban đầu.
**Depends on**: Phase A/B/C ĐÃ DONE (state machine + orchestrator + REST API + RequireOpsAdmin).

---

## 1. Mục tiêu

Đóng vòng lặp Auto-Provisioning: khi `provisioning_mode='auto'`, sau mỗi step thành công, worker phát `cdc.evt.provisioning.step_completed` → CMS Orchestrator (Phase B đã ship `HandleStepCompleted`) finalize state PENDING → ACTIVE → fanout Advance kế tiếp mà không cần Manager bấm `/advance`.

State machine flow target (D1, D6 đã ship Phase B):
```
draft ─advance──▶ shadow_pending ─[step_completed]──▶ shadow_active ─advance(auto)──▶
master_pending ─[step_completed]──▶ master_active ─advance(auto)──▶
mapping_pending ─[step_completed]──▶ mapping_ready ─advance(auto)──▶
schedule_pending ─[step_completed]──▶ running
```

## 2. GAP Analysis (live grep evidence)

Subscriber side đã ready:
- `worker_server.go:301` subscribe `cdc.evt.provisioning.step_completed` → `provHandler.HandleStepCompleted` (Phase B).
- `provisioning_orchestrator.go:308` finalize PENDING→ACTIVE bằng CAS.

Publisher side (worker) — **3/4 subject CHƯA có handler**:

| Subject | from_state → pending_state | Worker handler hiện tại | Phase D action |
|---|---|---|---|
| `cdc.cmd.shadow.bind` | draft → shadow_pending | ❌ Không subscribe | **Q1**: Tạo handler MỚI hay reuse logic `master_binding_repo` + `event_bridge`? |
| `cdc.cmd.master.bind` | shadow_active → master_pending | ❌ Không subscribe (chỉ có `cdc.cmd.master-create` ở `worker_server.go:315`) | **Q2**: `master-create` có cover binding không, hay cần handler riêng? |
| `cdc.cmd.discover` | master_active → mapping_pending | ✅ `command_handler.go:341 HandleDiscover` | Wire emit `step_completed` ở cuối success path |
| `cdc.cmd.schedule.enable` | mapping_ready → schedule_pending | ❌ Không subscribe (CMS có `transmute_schedule_handler` nhưng đó là cron, không phải command-driven) | **Q3**: Tạo handler enable schedule, hay reuse existing schedule activation API? |

## 3. Open Questions cho Architect

- **Q1 (shadow_bind handler)**: Logic shadow binding hiện được trigger ở đâu trong production hiện tại? Manual qua `/api/.../shadow-binding` REST hay đã có cron/event-driven path? Phase D cắm vào path nào để không split-brain?
- **Q2 (master_bind vs master-create)**: `cdc.cmd.master-create` (worker_server.go:315 → masterDDLHandler.HandleMasterCreate) tạo bảng master DDL. Còn `master.bind` trong state machine ám chỉ binding nguồn → master logic. Có phải 2 việc khác nhau? Nếu 1, em alias `cdc.cmd.master.bind` về `HandleMasterCreate` + emit step_completed. Nếu 2, em cần spec riêng cho master_bind.
- **Q3 (schedule_enable handler)**: TransmuteScheduler hiện tại là cron poll 60s (transmute_scheduler.go). State machine "schedule_enable" có nghĩa: bật flag `schedule.is_enabled = true` để cron tick mới fire? Hay tạo job mới? Cần ruling.
- **Q4 (failure path)**: Khi handler lỗi, payload `step_completed` có field `success: false` + `error: "..."` → Phase B `HandleStepCompleted` xử lý ra sao? Em thấy `provisioning_orchestrator.go:337` chỉ check `source_id` + `step` missing → reject. Nếu success=false, có cần CAS pending → failed + stamp last_step_error? Phase B chưa cover, Phase D cần extend orchestrator.
- **Q5 (correlation_id reuse)**: Phase B Advance emit `cdc.cmd.X` payload có `correlation_id`. Phase D handler echo lại `correlation_id` đó vào event `step_completed` để traceable round-trip — bắt buộc theo D8.

## 4. Definition of Done

- 4 worker handler subjects đều subscribe trong `worker_server.go`, mỗi handler kết thúc bằng emit `cdc.evt.provisioning.step_completed` (success hoặc fail) kèm `source_id`, `step`, `success`, `correlation_id`, `error?`, `trace_id?/span_id?`.
- Phase B `HandleStepCompleted` extend xử lý `success=false`: CAS pending → failed + stamp `last_step_error` (yêu cầu Phase D ship thêm code orchestrator worker, không touch CMS port).
- Integration test E2E (build tag `integration`):
  1. Seed source state=draft, mode=auto.
  2. POST `/advance` qua REST CMS.
  3. Wait 30s, assert source state=running, step_log có 4 step success entries (shadow_bind, master_bind, discover, schedule_enable) + 4 entries finalize PENDING→ACTIVE.
- Failure path test: 1 handler trả error → step_log có entry `success:false`, state=failed, last_step_error stamp đúng.
- Recovery loop verify: nếu step_completed event lost → 10min sau RecoveryLoop (Phase B) flip pending→failed với error=`TIMEOUT_EXCEEDED`. (Đã proven Phase B unit test, Phase D chỉ regression check.)

## 5. Out of scope

- UI dashboard (Phase E).
- OTel collector wiring (Phase F gợi ý — trace_id chưa proven E2E ở CMS Phase C).
- MongoDB connector (Track E workspace riêng).

# 04 — Decisions: Flow 1 root cause của "phantom stuck pending" + plan correction

> **Author**: max (Brain) | **Date**: 2026-05-07 ICT post-Phase A discovery
> **Trigger**: Boss directive "bằng mọi giá phải lên đc flow1" + max read full `provisioning_orchestrator.go` + `admin/source_register.go` + grep all publish sites cho `cdc.cmd.shadow.bind`
> **Audience**: x2 (cho `09_tasks_solution_flow1_x2_*`) + Boss approve gate

---

## 1. Tóm tắt evidence (file:line)

| Câu hỏi | Evidence | Kết luận |
|---|---|---|
| Ai publish `cdc.cmd.shadow.bind`? | `provisioning_orchestrator.go:331` (gọi từ `Advance` line 331 trong `o.publishCmd(ctx, desc.CmdSubject, payload)`). State machine row `provisioning_state_machine.go:54` map `StateDraft → step="shadow_bind", subject="cdc.cmd.shadow.bind", NextPending=ShadowPending, NextFinal=ShadowActive`. | **Chỉ 1 publisher** = `ProvisioningOrchestrator.Advance()` |
| `/v2/sources/register` (admin) có call Advance/publish shadow.bind không? | `grep -rn "Advance" internal/admin/` returns **NO MATCH**. `admin/source_register.go` Step 4 publish `cdc.cmd.kafka.refresh-topics` (Debezium connector level — refresh include list, không phải shadow.bind). | **KHÔNG fire shadow.bind ở admin endpoint** |
| Step 5 admin endpoint set state gì? | `source_register.go:94` direct `UPDATE provisioning_state = 'active'` (bypass state machine, không qua CAS). | **State = 'active' (legacy, không phải `shadow_active`)**. State 'active' không có trong `Transitions` map → orchestrator.Advance() return `ErrInvalidTransition` nếu sau này gọi. |
| State `'active'` có advanceable không? | Comment file `provisioning_orchestrator.go:11` ghi: `D4 — provisioned là legacy-only, NOT in Transitions. Don't advance it.` `'active'` rơi vào nhóm legacy tương tự (không có entry trong Transitions). | **'active' = terminal legacy state**. Source bị mark active = không bao giờ advance được nữa. |
| Vậy 4 phantom rows (id 33,34,35,37 `provisioning_state='active'` + table không tồn tại) bị stuck do đâu? | Từ 3 dòng trên: `/v2/sources/register` insert state='pending' → mark state='active' direct sau 4 sub-step, KHÔNG fire shadow.bind. shadow_binding row được insert step1 với `ddl_status='pending'` → không có ai update vì không có handler nào fire. → **`shadow_binding.ddl_status='pending'` mãi mãi + table không tồn tại**. | **Root cause = legacy admin endpoint architecture: bypass orchestrator state machine.** Không phải race condition, không phải NATS publish silent fail. Là **architectural drift**. |

## 2. Plan correction

### 2.1 P1 smoke (cũ trong `02_plan_flow1_e2e_2026-05-07.md`)

```text
P1.3 — POST /v2/sources/register cho flow1_smoke_pg_<TS>
P1.4 — Verify AC-1: HTTP 200, provisioning_state='active', steps_completed chứa 4 step
P1.5–P1.12 — Verify shadow schema/table tồn tại, ddl_status='created', count > 0...
```

**→ Sai expectation**. AC-1 sẽ PASS (admin endpoint trả 200 + state='active' đúng như coded), **nhưng** AC-3..AC-8 sẽ FAIL vì shadow.bind không fire → table không tạo → ingest không chạy.

### 2.2 P1 smoke (corrected — 2 phương án)

#### Phương án Z (KHUYẾN NGHỊ — không đụng worker code, chỉ dùng cms wizard endpoint sẵn có)

x2 đã document trong `10_gap_analysis_flow1_2026-05-07.md` (Author: x2):

| Step | Endpoint | Handler | Path |
|---|---|---|---|
| Z.1 | `POST /api/v1/source-objects/register` | `RegistryHandler.Register` (qua `RegisterRegistryCommand`) | cms-lane (`cdc-cms-service`) |
| Z.2 | `POST /api/v1/cms/sources/:id/provisioning/mode {mode:manual}` | `ProvisioningHandler.SetMode` | cms-lane |
| Z.3 | `POST /api/v1/cms/sources/:id/provisioning/advance` | `ProvisioningHandler.Advance` → publish `cdc.cmd.shadow.bind` | cms-lane → kích worker |
| Z.4 | Verify state machine: poll `GET /api/v1/cms/sources/:id/provisioning` cho đến khi `state='shadow_active'` | `ProvisioningHandler.GetState` | cms-lane |
| Z.5 | Verify shadow table: `\d shadow_<db>.<table>` 8 CDC + business cols | psql cdc-metadata 5433 | max ops |
| Z.6 | Trigger snapshot via `dbz_signal` insert | psql source 5435 | max ops |
| Z.7 | Verify shadow row count ≥ source | psql cdc-metadata 5433 | max ops |

**Điểm mạnh**: 0 code change ở worker. Tất cả endpoint cms đã exist.

**Điểm yếu**: Phải có cms server chạy (PID 52079 đang alive — OK). Phải biết path cms `/api/v1/cms/sources/:id/provisioning/advance`.

#### Phương án Y (Fix admin /v2/sources/register tự fire Advance — worker code change)

Sửa `admin/source_register.go:92` thay Step 5 từ direct UPDATE 'active' sang gọi `orchestrator.Advance()`:

```go
// ── Step 5: orchestrator advance (CAS state=draft → shadow_pending + publish cdc.cmd.shadow.bind)
// Replace direct mark='active' with state-machine call so shadow.bind actually fires.
if err := s.deps.Orchestrator.Advance(c.Request.Context(), sourceID, "v2_register"); err != nil {
    s.deps.Logger.Warn("step5 orchestrator advance failed", zap.Error(err))
    s.markProvisioningFailed(sourceID, "step5_failed", err)
    c.JSON(http.StatusMultiStatus, RegisterSourceResponse{...})
    return
}
stepsCompleted = append(stepsCompleted, "orchestrator_advance")

// State after Advance = shadow_pending. Worker handler sẽ promote → shadow_active.
c.JSON(http.StatusOK, RegisterSourceResponse{
    SourceObjectID:    sourceID,
    ProvisioningState: "shadow_pending",  // not 'active'
    StepsCompleted:    stepsCompleted,
    Warnings:          respWarnings,
})
```

**Pre-req**: Step 1 (`step1InsertRegistry`) phải đổi `provisioning_state='pending'` → `'draft'` (line 147 + 150) để Transitions map match (`StateDraft` → shadow_bind).

**Điểm mạnh**: Backward compat cho clients đang gọi `/v2/sources/register`. Fix root cause cho 4 phantom rows.

**Điểm yếu**: Đụng admin handler + struct response (200 trả `shadow_pending` thay `active` — breaking change cho client cũ). Và state='pending' → 'draft' migration cho legacy data.

#### Phương án X (Hybrid — admin endpoint fire NATS direct, không qua orchestrator)

`source_register.go:97` sau UPDATE 'active', thêm:

```go
// Trigger shadow.bind worker handler — bypass state machine vì admin endpoint
// tự sync state='active' rồi (legacy contract).
payload := map[string]any{
    "source_id": sourceID,
    "correlation_id": fmt.Sprintf("v2-reg-%d-%d", sourceID, time.Now().UnixNano()),
    "triggered_by": "v2_register",
}
body, _ := json.Marshal(payload)
if perr := s.deps.NATS.Publish("cdc.cmd.shadow.bind", body); perr != nil {
    s.deps.Logger.Warn("step5b shadow.bind publish failed", zap.Error(perr))
}
```

**Điểm mạnh**: Min change, no contract break, fix flow.
**Điểm yếu**: Persist drift giữa legacy admin path vs orchestrator path. State='active' không qua state machine → tiếp tục có 2 nguồn truth state. Không clean về kiến trúc.

### 2.3 Decision (max recommend, chờ Boss approve)

**Khuyến nghị**: **Phương án Z** cho P1 smoke ngay (không touch code, chỉ dùng cms endpoint sẵn có). Nếu PASS → root cause confirmed.

**Phase 2 (sau P1 smoke PASS)**: thi công **Phương án Y** (orchestrator integration) — proper architectural fix. Đụng admin handler nhưng đó là worker-lane (max). Migration backfill state legacy 'active' → 'archived' (terminal) hoặc 'failed' (cho retry path).

**Phương án X**: bỏ — không clean.

## 3. Tác động lên `01_requirements_flow1_e2e_2026-05-07.md` AC

| AC cũ | AC mới (Phương án Z) |
|---|---|
| AC-1: `curl POST /v2/sources/register` HTTP 200 + `state='active'` | AC-1: `curl POST /api/v1/source-objects/register` HTTP 200 + `state='draft'` (or alias). Sau đó `curl POST /api/v1/cms/sources/:id/provisioning/advance` HTTP 200, poll `GET /api/v1/cms/sources/:id/provisioning` đến `state='shadow_active'`. |
| AC-5: `shadow_binding.ddl_status='created'` | (giữ nguyên — đây là output của HandleShadowBind worker handler) |
| AC-6: `provisioning_state='shadow_active'` | (giữ nguyên — chính là expected sau Advance + step_completed flow) |
| AC-7: Kafka topic ≥1 msg | (giữ nguyên) |
| AC-8: shadow row count ≥ source | (giữ nguyên) |

AC-2,3,4 không đổi.

## 4. Tác động lên `08_tasks_flow1_e2e_2026-05-07.md`

- P1.3 đổi `POST /v2/sources/register` → 2-step `POST /api/v1/source-objects/register` (Z.1) + `POST /api/v1/cms/sources/:id/provisioning/advance` (Z.3).
- P1.4 đổi expected state thành `shadow_active`.
- P2 fix candidates A/B/C cập nhật:
  - **Fix Y** (mới — replace fix A): Refactor `admin/source_register.go:92` Step 5 → call `orchestrator.Advance()`.
  - **Fix BackfillSQL** (mới — backfill 4 phantom row): UPDATE state='active' → 'draft' cho id 33,34,35,37 (clear `last_step_error`) → cms-server gọi `/advance` để fire shadow.bind. Hoặc DELETE 4 row nếu không cần migrate.

## 5. Decision matrix cho Boss / x2

| Action | Phụ trách | Effort | Risk | Approval needed? |
|---|---|---|---|---|
| Phương án Z P1 smoke (cms 2-step) | max ops + x2 (cms-lane) | 30 min | Low (no code change) | Boss confirm OK |
| Phương án Y Phase 2 fix admin endpoint | max (worker-lane) | 2h (code + test + migration) | Medium (breaking response change) | Boss approve |
| Backfill 4 phantom row state='draft' | max (SQL) | 15 min | Low | Boss approve |

## 6. Risk

- Nếu Boss reject Phương án Y (vì breaking change), default Phương án X làm transient fix cho 4 phantom rows.
- Nếu cms `/api/v1/cms/sources/:id/provisioning/advance` (Z.3) yêu cầu `provisioning_mode='manual'` mới chạy được (Z.2 SetMode), x2 phải confirm path trong `RegistryHandler.Register` default mode.

## 7. Reference cross-link

- Lessons cross-ref: lesson L-1629 (Schema Registry preempt), L-1688 (Cascade Liability gate Mongo-only) đã document. **Lesson mới cần APPEND** sau khi fix: `L-FLOW1-LEGACY-ADMIN-BYPASS` — Global Pattern: `[A direct-update-to-terminal-state] does [skip B state-machine] to [X downstream subscriber Y] → [Y never fires] → [Result: silent stuck pending forever]. Đúng: [A always-via-orchestrator-Advance to fire B publishCmd to X subscriber Y].`

— max

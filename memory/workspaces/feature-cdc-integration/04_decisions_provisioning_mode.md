# 04_decisions — Provisioning Mode (Architect Rulings)

> Phán quyết kiến trúc đã chốt từ user (Architect) ngày 2026-04-29 cho phase
> Source Provisioning Mode. Đây là **single source of truth** — mọi
> implementation phải bám theo. Vi phạm = re-plan.

## D1. Chuyển đổi Mode (Manual → Auto): Reconciliation tick ngay
**Phán quyết**: Khi Manager chuyển source từ `manual` → `auto`, Orchestrator phải:
1. UPDATE `provisioning_mode='auto'` (CAS với mode='manual' để chống race).
2. Đọc state hiện tại.
3. Nếu state advanceable (không phải terminal/paused/failed) → kick `Advance()` ngay.
4. Auto-fan tự chạy đến hết flow qua `HandleStepCompleted`.

**Implementation impact**:
- `SetMode(sourceID, "auto", actor)` cuối hàm gọi `Advance()` nếu `CanAdvance(currentState)`.
- `SetMode(sourceID, "manual", ...)` chỉ UPDATE field, không kick gì.

## D2. Retry từ Failed: Re-fire current step
**Phán quyết**: Bấm Retry → re-fire CHÍNH bước vừa fail (vd: nếu fail tại `shadow_pending` thì re-publish `cdc.cmd.shadow.bind` cùng correlation_id mới). KHÔNG quay về state trước. Muốn quay về bước trước → dùng lệnh riêng `POST .../rollback?to=<state>` (manual, tường minh).

**Implementation impact**:
- `Retry(sourceID, actor)`:
  1. Đọc state — phải là `failed`.
  2. Tìm bước fail cuối từ `step_log`: lấy entry mới nhất có `success=false`, đọc `from_state` (= state trước khi fail).
  3. CAS UPDATE: `state=failed → state=<from_state>` + clear `last_step_error`.
  4. Gọi `Advance()` để re-publish cmd của step đó.
- `Rollback` endpoint riêng `POST /sources/:id/provisioning/rollback` body `{to_state}` (phase 2, không làm phase này).

## D3. TTL Pending: 10 phút → TIMEOUT_EXCEEDED
**Phán quyết**: 10 phút cho mọi `*_pending` state. Sau 10 phút không có completion event → Orchestrator's RecoveryLoop set state=`failed`, `last_step_error='TIMEOUT_EXCEEDED'`.

**Implementation impact**:
- Const `ProvisioningPendingTTL = 10 * time.Minute` (override qua env `PROVISIONING_PENDING_TTL_MIN`).
- `RecoveryLoop` chạy mỗi `time.Minute`:
  ```go
  rows := SELECT id, provisioning_state FROM cdc_system.source_object_registry
            WHERE provisioning_state IN ('shadow_pending','master_pending','mapping_pending','schedule_pending')
              AND updated_at < NOW() - INTERVAL '10 minutes'
  for each row:
      CAS UPDATE: state=cur → state='failed', last_step_error='TIMEOUT_EXCEEDED'
                  WHERE provisioning_state = cur AND updated_at < ...
  ```
- Hardening: dùng cả `state` và `updated_at` trong WHERE để tránh re-fire vô hạn nếu auto-advance race.

## D4. Backfill Legacy: 1 stamp duy nhất, state='provisioned'
**Phán quyết**: Source `is_active=true` đã chạy trước phase này → set state=`provisioned` (state mới, terminal, KHÁC `running`). Stamp duy nhất:
```
{step:'backfill', from_state:'draft', to_state:'provisioned', actor:'migration-047',
 message:"[Migration-047]: Backfilled legacy source to state 'provisioned'"}
```
KHÔNG tạo log giả 4 entry shadow→master→mapping→schedule.

**Implementation impact**:
- Thêm state `provisioned` vào enum (state machine pure):
  ```go
  StateProvisioned ProvisioningState = "provisioned"
  ```
- `provisioned` KHÔNG ở trong `Transitions` keys → orchestrator không advance.
- `provisioned` là terminal-active (giống `running`): transmute_scheduler vẫn cron tick row này bình thường.
- FE hiển thị `provisioned` + `running` cùng nhóm "Active" (UI concern, không phải state machine concern).
- Migration 047 backfill query: `WHERE is_active=true AND provisioning_state='draft'` set `state='provisioned'`.

## D5. API path: `/api/v1/cms/sources/...`
**Phán quyết**: Versioning `v1` + scope `cms`. Áp dụng cho TẤT CẢ endpoint provisioning.

**Implementation impact**:
- 7 endpoint (R1.5):
  - `GET    /api/v1/cms/sources/:id/provisioning`
  - `POST   /api/v1/cms/sources/:id/provisioning/advance`
  - `POST   /api/v1/cms/sources/:id/provisioning/pause`
  - `POST   /api/v1/cms/sources/:id/provisioning/resume`
  - `POST   /api/v1/cms/sources/:id/provisioning/retry`
  - `POST   /api/v1/cms/sources/:id/provisioning/archive`
  - `POST   /api/v1/cms/sources/:id/provisioning/mode`
- Mount vào router root chung; auth middleware ở v1 group.

---

## D6 (Hardening). CAS Guard bắt buộc trên mọi state UPDATE
**Phán quyết**: Mỗi câu lệnh UPDATE chuyển state PHẢI có `WHERE provisioning_state = 'expected_state'` (Compare-And-Swap). Đây là chống race condition khi nhiều worker instance chạy song song.

**Áp dụng**:

| Operation | Expected (WHERE) | Target (SET) |
|-----------|------------------|--------------|
| `Advance` | current advanceable state | corresponding `*_pending` |
| `HandleStepCompleted` (success) | matching `*_pending` | corresponding final state |
| `HandleStepCompleted` (failure) | matching `*_pending` | `failed` |
| `Pause` | `running` | `paused` |
| `Resume` | `paused` | `running` |
| `Retry` | `failed` | (state trước fail từ log) |
| `Archive` | (any non-archived) | `archived` |
| `SetMode` | matching opposite mode | new mode |
| `RecoveryLoop timeout` | `*_pending` AND `updated_at < NOW() - INTERVAL` | `failed` |

**Failure mode khi CAS thất bại**:
- `RowsAffected == 0` → operation no-op, log info "state changed concurrently — skipped".
- API endpoint return `409 Conflict` cho client.
- NATS handler return im lặng (event sẽ được instance khác handle).

**Cấm**:
- KHÔNG dùng UPDATE thuần `WHERE id=?` không kèm state guard.
- KHÔNG dùng SELECT-then-UPDATE pattern (race window). Phải single SQL với WHERE state.
- KHÔNG dùng `FOR UPDATE` lock + UPDATE (chậm hơn CAS, không cần thiết với atomic UPDATE).

**Test bắt buộc**:
- `TestOrchestrator_CAS_Concurrent` — 2 goroutine cùng gọi `Advance(1)` → đúng 1 success, 1 returns conflict. Verify final state chỉ có 1 entry `*_pending` trong log (không double-step).

---

## Rollback decision matrix
Nếu phán quyết nào sai/lỗi sau implementation:
- D1, D2, D5: code-only rollback, không touch DB.
- D3: chỉnh ENV `PROVISIONING_PENDING_TTL_MIN=0` để disable timeout.
- D4: migration rollback (DROP COLUMN) trong `09_tasks_solution_provisioning_mode.md` §A1.
- D6: KHÔNG rollback. CAS là invariant, vi phạm = data corruption.

---

## D7 (Hardening). Log Capping — provisioning_step_log ≤ 50 entries
**Phán quyết**: `provisioning_step_log` JSONB array PHẢI giới hạn ở 50 entry mới nhất. Tránh phình vô hạn khi 1 source bị kẹt vòng Retry/Failed loop.

**Implementation impact**:
- Mọi UPDATE append entry vào log PHẢI bọc bằng SQL trim:
  ```sql
  provisioning_step_log = (
    WITH appended AS (SELECT (provisioning_step_log || ?::jsonb) AS arr),
         trimmed  AS (
           SELECT CASE WHEN jsonb_array_length(arr) > 50
                       THEN (SELECT jsonb_agg(e ORDER BY ord) FROM (
                              SELECT e, row_number() OVER () AS ord
                                FROM jsonb_array_elements(arr) e
                                OFFSET (jsonb_array_length(arr) - 50)) sub)
                       ELSE arr END AS arr
             FROM appended)
    SELECT arr FROM trimmed)
  ```
  Hoặc đơn giản hơn — đẩy logic vào helper function PG `cdc_system.append_step_log_capped(jsonb, jsonb, int)`. Quyết định: dùng helper PG function (tạo trong migration 048) để giữ orchestrator query gọn.
- 50 = const `ProvisioningStepLogMaxEntries`, override env `PROVISIONING_STEP_LOG_MAX`.
- Trim FIFO (giữ 50 entry mới nhất, drop oldest).

**Test**:
- `TestOrchestrator_LogCap_Trim50`: Force 60 entry append → DB column `jsonb_array_length` = 50, entry đầu là entry thứ 11 trong batch.

## D8 (Hardening). Trace Propagation — OTel trace_id trong NATS payload
**Phán quyết**: Mỗi NATS payload do orchestrator phát PHẢI mang `trace_id` + `span_id` từ OpenTelemetry context của caller (CMS request hoặc internal goroutine). Handler downstream extract → continue span → end-to-end trace trong SignOz/Jaeger.

**Implementation impact**:
- Payload format chuẩn cho mọi `cdc.cmd.provisioning.*` + `cdc.cmd.shadow.bind` / `master.bind` / `discover` / `schedule.enable`:
  ```json
  {
    "source_id": 123,
    "correlation_id": "prov-123-1745920000000000",
    "trace_id": "abcdef1234567890abcdef1234567890",
    "span_id": "1234567890abcdef",
    "triggered_by": "provisioning",
    ...domain fields
  }
  ```
- Helper `service.InjectTraceContext(ctx, payload map)` — extract `trace_id`/`span_id` từ `trace.SpanFromContext(ctx).SpanContext()`, ghi vào map; no-op nếu context không có active span (fallback cho RecoveryLoop chạy nền).
- Handler subscribe gọi `service.ExtractTraceContext(payload, ctx)` → wrap span con cho operation đó.
- KHÔNG block khi OTel exporter offline (signoz off như hiện tại) — chỉ là metadata propagation, exporter retry là việc của OTel SDK.

**Test**:
- `TestOrchestrator_TracePropagation`: Khởi context có active span → call Advance() → assert NATS msg unmarshal có `trace_id` khớp `span.SpanContext().TraceID().String()`.

**Cấm**:
- KHÔNG dùng W3C `traceparent` header trên NATS msg.Header — codebase hiện chưa thống nhất convention này. Nhúng vào JSON body là least invasive.

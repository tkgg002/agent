# 01_requirements — Source Provisioning Mode

## R1. Functional Requirements

### R1.1 Mode toggle khi tạo source
- API tạo source nhận thêm field `provisioning_mode` ∈ {`auto`, `manual`}, default `manual`.
- Có thể đổi mode sau khi tạo (qua endpoint riêng `POST /sources/:id/provisioning/mode`) — cho phép operator đang manual chuyển sang auto giữa chừng và ngược lại.
- Đổi mode = log audit entry, không reset state hiện tại.

### R1.2 State machine 9 trạng thái
Trạng thái lưu ở `source_object_registry.provisioning_state`:

```
draft
 │ (advance: Bind shadow + DDL)
 ▼
shadow_pending  ──fail──► failed
 │ (evt.shadow.bind.completed)
 ▼
shadow_active
 │ (advance: Bind master + DDL)
 ▼
master_pending  ──fail──► failed
 │ (evt.master.bind.completed + schema_status=approved)
 ▼
master_active
 │ (advance: discover mapping rules)
 ▼
mapping_pending ──fail──► failed
 │ (evt.discover.completed + rules.status=approved)
 ▼
mapping_ready
 │ (advance: enable transmute_schedule)
 ▼
schedule_pending ──fail──► failed
 │ (transmute_schedule.is_enabled=true)
 ▼
running ◄─resume── paused
 │      ──pause─► paused
 │
 (archive any time) ──► archived (terminal)
```

### R1.3 Auto mode behavior
- Orchestrator subscribe `cdc.evt.provisioning.step_completed`.
- Khi event mang `success=true`, orchestrator advance state + dispatch command kế (NATS publish `cdc.cmd.<step>`).
- Auto mode KHÔNG tự dispatch nếu state hiện tại = `paused` hoặc `failed` — phải có `resume`/`retry` từ CMS.

### R1.4 Manual mode behavior
- Khi step completed event đến, orchestrator chỉ UPDATE state, KHÔNG dispatch step kế.
- Operator CMS gọi `POST /sources/:id/provisioning/advance` → orchestrator dispatch command kế.
- Mỗi click manual log entry `actor=<cms_user_id>`, `triggered_by=manual_click`.

### R1.5 Action endpoints (CMS REST)
| Method | Path | Mô tả | Constraint |
|--------|------|-------|------------|
| GET | `/api/cms/sources/:id/provisioning` | Đọc state + log + next valid actions | n/a |
| POST | `/api/cms/sources/:id/provisioning/advance` | Bước tiếp (manual hoặc force-tick auto) | Chỉ chạy nếu `state` không phải terminal |
| POST | `/api/cms/sources/:id/provisioning/pause` | `running` → `paused` | Chỉ ở `running` |
| POST | `/api/cms/sources/:id/provisioning/resume` | `paused` → `running` | Chỉ ở `paused` |
| POST | `/api/cms/sources/:id/provisioning/retry` | `failed` → state trước đó (theo log) | Chỉ ở `failed` |
| POST | `/api/cms/sources/:id/provisioning/mode` | Đổi mode (`auto`↔`manual`) | n/a |
| POST | `/api/cms/sources/:id/provisioning/archive` | Archive (terminal) | Bất kỳ state nào |

### R1.6 Audit log
- `provisioning_step_log` JSONB array, append-only.
- Mỗi entry: `{seq, step, from_state, to_state, actor (cms_user|orchestrator|nats), correlation_id, started_at, completed_at, success (bool), error (string|null)}`.
- Nguồn `actor`:
  - `orchestrator` — auto mode tự fan ra
  - `cms:<user_id>` — manual click
  - `nats-event` — completion event từ handler (intermediate state)

## R2. Non-Functional Requirements

### R2.1 Tính nhất quán (consistency)
- State transition phải atomic: 1 UPDATE SQL trong transaction, kèm WHERE current_state guard (CAS).
- Duplicate event = no-op (WHERE state=expected returns 0 rows).

### R2.2 Khả năng hồi phục (recoverability)
- Nếu orchestrator crash giữa chừng (sau khi UPDATE state nhưng trước khi publish command kế):
  - Boot tick recovery (1 lần khi worker start): SELECT các source ở state `*_pending` quá X phút → re-publish command tương ứng.

### R2.3 Idempotency
- Mọi command kế dispatched đều có `correlation_id` = `prov-<source_id>-<step_seq>`.
- Handler có thể nhận trùng cmd, response event mang cùng `correlation_id`.
- Orchestrator dedup bằng `correlation_id` đã thấy trong `provisioning_step_log`.

### R2.4 Observability
- Mỗi transition log Zap structured: `source_id`, `from`, `to`, `mode`, `actor`, `latency_ms`.
- Prometheus counter (nếu đã có metrics infra): `cdc_provisioning_state_transitions_total{mode,from,to,success}`.

### R2.5 Backwards compatibility
- Source hiện có (đã chạy production) — migration set default `provisioning_mode='manual'`, `provisioning_state='running'` (giả định đang ổn định) khi `is_active=true`. Không phá flow đang chạy.
- Cột mới optional ở payload tạo source — clients cũ không gãy.

## R3. Security
- Tất cả CMS action endpoint phải có auth middleware (JWT/session đã có ở `centralized-data-service`).
- `actor` field phải lấy từ token, không trust client payload.
- `mode='auto'` không cho phép user thường — yêu cầu role `admin` hoặc `data_engineer` (gate ở handler API).

## R4. Phụ thuộc
- Không thay đổi schema `shadow_binding`, `master_binding`, `mapping_rule_v2`, `transmute_schedule` về cơ chế. Chỉ thêm tham chiếu `correlation_id` qua payload + emit event.
- Có sẵn NATS connection (đã wire ở worker).
- Có sẵn DB session (gorm).
- Có sẵn auth middleware (web layer — cần xác minh path).

## R5. Câu hỏi mở (cần user xác nhận trước khi implement)
1. **Q1**: Khi đổi mode `manual → auto` ở giữa flow (vd đang state=`shadow_active`), có muốn orchestrator tự kick advance ngay không? **Đề xuất**: có — UPDATE mode = trigger auto-tick.
2. **Q2**: `retry` từ `failed` quay về state nào? **Đề xuất**: state đứng trước `failed` trong log gần nhất, kèm clear `last_step_error`.
3. **Q3**: Có cần TTL cho `*_pending` (vd quá 10 phút coi như stuck)? **Đề xuất**: có, default 10 phút, configurable qua env `PROVISIONING_PENDING_TTL_MIN`.
4. **Q4**: Source đã active từ trước (R2.5) có cần backfill `provisioning_step_log` không? **Đề xuất**: không — chỉ stamp 1 entry `{step:"backfill", to_state:"running", actor:"migration-047"}` cho audit.
5. **Q5**: API path prefix — `/api/cms/sources/:id/provisioning/...` hay `/api/v1/sources/:id/provisioning/...`? **Đề xuất**: theo pattern hiện có của FE workspace (cần grep ra).

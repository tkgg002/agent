# 09_tasks_solution — Rationale + Alternative Rejected

> Mỗi patch trong `08_tasks.md` đều có lý do chọn + giải pháp đã loại.

---

## B1 — `ReclaimOrphans(ctx, staleAfter)`

### Rationale
- DB query single shot: `UPDATE ... WHERE status='running' AND updated_at < NOW() - staleAfter RETURNING source_object_id` → demote sang `paused`, lấy về list ID cần re-publish.
- Demote-to-paused tận dụng nhánh resume hiện có ở `claimProgress` (line 637-655) → KHÔNG sửa state machine.
- Return `(int, error)` để boot log số orphan reclaimed (observability).

### Alternative rejected
| Alt | Lý do reject |
|---|---|
| Trigger `claimProgress` trực tiếp inline (không qua NATS) | Bypass queue group → mất load distribution khi multi-worker. |
| Mark `status='error'` + manual operator | UX bad, defeats automation purpose. |
| Background tick cron 1 phút | Thêm scheduler infra. Boot-time đủ cho crash recovery scenario chính. |

---

## B2 — `publishResumeMessage(ctx, sourceObjectID)`

### Rationale
- Reuse NATS subject `cdc.snapshot.resume.<sourceObjectID>` đã có sẵn cho UI resume manual.
- Payload tối thiểu: `{"source_object_id": ID, "trigger": "boot-reclaim"}`.
- Marshal JSON + `r.natsConn.Publish` đơn giản, không cần ACK (NATS core fire-and-forget; idempotent guard ở `claimProgress` lock).

### Alternative rejected
| Alt | Lý do reject |
|---|---|
| Tạo subject mới `cdc.snapshot.reclaim.<id>` | Subscriber handler phải có nhánh xử lý riêng → tăng surface. |
| Publish raw payload không có `trigger` field | Mất khả năng debug "ai trigger resume này" trong log. |

---

## B3 — Worker server wire (goroutine async)

### Rationale
- Spawn goroutine SAU `QueueSubscribe` thành công → subscriber sẵn sàng nhận message reclaim publish.
- Đọc env `SNAPSHOT_STALE_AFTER_SECONDS` (default 60s) → cấu hình runtime mà không cần rebuild.
- Lỗi reclaim → `log.Warn` không trả lên main → worker vẫn alive nhận message mới (ADR-003).

### Alternative rejected
| Alt | Lý do reject |
|---|---|
| Block startup chờ reclaim xong | DB hiccup → worker không lên được. |
| Reclaim trước `QueueSubscribe` | Publish lúc subscriber chưa subscribe → message lost. |
| Hardcode 60s không env | Khó tune trong production. |

---

## B4 — Const `snapshotV2DefaultStaleAfter = 60 * time.Second`

### Rationale
- Đặt cạnh const `snapshotV2ZombieAfter` (line 45) để code locality.
- 60s = 12× normal batch interval (~5s) → buffer rộng cho transient hiccup.

### Alternative rejected
| Alt | Lý do reject |
|---|---|
| Magic number `60` rải rác | Anti-pattern. |
| Reuse `snapshotV2ZombieAfter` (10 phút) | Quá lâu, user phải chờ 10 phút sau restart. |

---

## F1 — `isStaleRunning` helper + const `STALE_RUNNING_THRESHOLD_MS = 60_000`

### Rationale
- Pure function: `(row) => row.status === 'running' && Date.now() - new Date(row.updated_at).getTime() > STALE_RUNNING_THRESHOLD_MS`.
- Threshold khớp BE `snapshotV2DefaultStaleAfter` (60s) → operator UI và worker đồng bộ thời gian (ADR-004).
- Đặt ngay trước `default export` component để dễ tìm.

### Alternative rejected
| Alt | Lý do reject |
|---|---|
| Threshold 30s | False-positive cao khi batch nặng. |
| Query backend `/api/v1/config` lấy threshold | Thêm round-trip, defer phase 2 (ADR-004). |
| Render Force Resume cho mọi `running` row | Spam button, operator confuse. |

---

## F2 — Actions column "Force Resume"

### Rationale
- Render `<Button icon={<WarningOutlined/>} onClick={() => setActionPending({id, action:'resume', stale:true})}>Force Resume</Button>` khi `isStaleRunning(row)`.
- Label "Force Resume" + icon warning → operator awareness (ADR-005).
- Reuse setActionPending state đã có cho normal Resume → KHÔNG thêm modal mới.

### Alternative rejected
| Alt | Lý do reject |
|---|---|
| Cùng label "Resume" như paused | Operator không phân biệt orphan vs paused intent. |
| Tự động trigger resume từ FE polling | Risk double-trigger nhiều tab; operator phải confirm rõ. |
| Show dropdown menu Action | Tăng click steps; force-resume là emergency op cần 1-click. |

---

## F3 — Modal warning branch

### Rationale
- Extend type `actionPending` thêm field `stale?: boolean`.
- Modal `description` branch theo `actionPending?.stale`:
  - `false`: "Bạn có chắc resume snapshot này?"
  - `true`: "Snapshot có thể đang orphan (worker đã chết). Resume sẽ demote về paused rồi re-trigger. Tiếp tục?"
- Confirm button text giữ nguyên ("Resume") — backend xử lý đồng nhất.

### Alternative rejected
| Alt | Lý do reject |
|---|---|
| Modal riêng cho Force Resume | Code duplication. |
| Không có warning, click thẳng | Operator có thể bấm nhầm trên running healthy nếu logic isStaleRunning có bug. |

---

## T1 — `TestReclaimOrphans_StaleRowsPublished`

### Rationale
- Setup sqlmock: kỳ vọng query `UPDATE ... RETURNING source_object_id` trả 2 row stale (ID 1, ID 2).
- Mock natsPublisher (interface) → assert `Publish` được call 2 lần với subject + payload đúng.
- Assert return value `(2, nil)`.

### Alternative rejected
| Alt | Lý do reject |
|---|---|
| Integration test với NATS embed server | Slow startup (>500ms); unit test interface đủ. |
| Skip mock NATS, dùng channel | Khó test subject + payload format. |

---

## T2 — `TestReclaimOrphans_NoStale_NoOp`

### Rationale
- sqlmock UPDATE trả empty rowset.
- Assert: natsPublisher.Publish KHÔNG được call (`mockPublisher.Calls == 0`).
- Return `(0, nil)`.

---

## T3 — `TestReclaimOrphans_DBError_Propagates`

### Rationale
- sqlmock UPDATE trả `errors.New("connection refused")`.
- Assert: return value `(0, err)` với err wrap đúng.
- natsPublisher.Publish KHÔNG được call.

### Alternative rejected
| Alt | Lý do reject |
|---|---|
| Test silent swallow err | Vi phạm fail-loud principle. |

---

## Refactor nhỏ — `natsPublisher` interface

### Rationale
- Đổi `r.natsConn *nats.Conn` thành interface `natsPublisher { Publish(subject string, data []byte) error }`.
- Cho phép mock trong test mà không cần NATS embed server.
- Production wiring: `*nats.Conn` đã satisfy interface (zero refactor caller).

### Alternative rejected
| Alt | Lý do reject |
|---|---|
| Giữ `*nats.Conn` + dùng `nats-server/test` embed | Slow CI (~500ms per test); interface đẹp hơn. |
| Function variable `var publishFn = ...` | Khó test multi-call assertion. |

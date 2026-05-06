# 08 — Tasks: Core-Flow Hardening Phase P0+P1

**Phase code**: `core_flow_hardening_p0_p1`
**Created**: 2026-05-04 13:55 (+07)
**Companion**: `01_requirements_core_flow_hardening_p0_p1.md`, `02_plan_core_flow_hardening_p0_p1.md`, `09_tasks_solution_core_flow_hardening_p0_p1.md`.

---

## Task list (execution order theo plan)

### Task P1.1.A — Edit `handleDelete` UPSERT
- **Owner**: Muscle (`/muscle-execute`)
- **File**: `internal/handler/event_handler.go:145-184`
- **Action**: Replace UPDATE statement với INSERT ON CONFLICT pattern. Set `_gpay_source_id` (TEXT cast), `_deleted=TRUE`, timestamps, `_source='debezium'`.
- **DoD**: Build pass; SQL string contains `INSERT INTO ... ON CONFLICT (... ) DO UPDATE SET _deleted = TRUE`.

### Task P1.1.B — Unit test cho handleDelete first-touch
- **Owner**: Muscle
- **File**: `internal/handler/event_handler_test.go` (extend nếu tồn tại, tạo mới nếu chưa)
- **Action**: Mock `*gorm.DB` (sqlmock), gọi `handleDelete` với route + event delete `before.id=999`. Assert exec SQL match regex INSERT ON CONFLICT.
- **DoD**: `go test ./internal/handler/... -run TestHandleDelete_FirstTouch_TombstoneInsert` PASS.

### Task P1.1.C — Smoke E2E P1.1
- **Owner**: Muscle (sau khi A+B land)
- **Action**: Live DELETE id=64 (đã có shadow row từ B11) + DELETE id mới chưa từng INSERT → verify cả 2 case.
- **DoD**: 2 shadow rows có `_deleted=true`, `_gpay_source_id` đầy đủ.

---

### Task P0.1.A — Refactor `KafkaConsumer` struct
- **Owner**: Muscle
- **File**: `internal/handler/kafka_consumer.go`
- **Action**: Add fields `mu sync.Mutex`, `currentTopics []string`. Tách method `buildReader(topics []string) *kafka.Reader`. Add method `RefreshTopics(ctx context.Context) error`. Add helper `topicSetEqual`.
- **DoD**: Build pass; existing tests pass.

### Task P0.1.B — Update `Start` consume loop
- **Owner**: Muscle
- **File**: `internal/handler/kafka_consumer.go::Start`
- **Action**: Replace `reader := kafka.NewReader(...)` direct use bằng snapshot từ `kc.readers[0]` (lock-protected). Add `refreshTicker := time.NewTicker(60*time.Second)` case ở select. Init `kc.currentTopics = topics` sau initial discover.
- **DoD**: Build pass; consume loop không deadlock; logs có "topic set unchanged" khi auto-tick mà không có thay đổi.

### Task P0.1.C — NATS subscribe `cdc.cmd.kafka.refresh-topics`
- **Owner**: Muscle
- **File**: `internal/server/worker_server.go` (sau line 261, trước line 264)
- **Action**: `natsClient.Conn.Subscribe("cdc.cmd.kafka.refresh-topics", func(msg *nats.Msg) { kafkaConsumer.RefreshTopics(ctx) })`. Cần expose `kafkaConsumer` reference ở scope đó (verify line nơi tạo).
- **DoD**: Build pass; restart worker, `nats pub cdc.cmd.kafka.refresh-topics ""` → log "nats-triggered topic refresh ok".

### Task P0.1.D — Unit tests
- **Owner**: Muscle
- **File**: `internal/handler/kafka_consumer_test.go`
- **Action**: 2 tests:
  - `TestRefreshTopics_NoChange`: stub discover return same set → `RefreshTopics` không recreate reader (count readers giữ nguyên 1).
  - `TestRefreshTopics_AddTopic`: stub discover return new set với 1 topic mới → readers list được rebuild (assert kc.currentTopics len tăng).
- **DoD**: tests PASS.

### Task P0.1.E — Smoke E2E P0.1
- **Owner**: Muscle
- **Action**:
  1. Pre-stage Mongo collection `payment_bills_smoke_p01` với 1 doc.
  2. PUT Debezium include list extend với collection mới.
  3. `nats pub cdc.cmd.kafka.refresh-topics ""`.
  4. Worker logs có "topic set changed, recreating reader".
  5. INSERT 1 doc mới → shadow landed trong 15s.
  6. KHÔNG restart worker.
- **DoD**: shadow row landed.

---

### Task P0.2.A — Add config fields
- **Owner**: Muscle
- **File**: `internal/config/config.go`
- **Action**: Add `AdminAPI struct { ListenAddr string }`, `Debezium struct { URL string }`, `SchemaRegistry struct { URL string }` if absent. Default `:8090`, env `ADMIN_API_LISTEN_ADDR`, `DEBEZIUM_URL`, `SCHEMA_REGISTRY_URL`.
- **DoD**: Build pass.

### Task P0.2.B — Create `cmd/admin-api/main.go`
- **Owner**: Muscle
- **File**: NEW
- **Action**: Per `02_plan_*.md` skeleton. Open `cdc_dw` DB, connect NATS, build admin.Server, listen.
- **DoD**: `go build ./cmd/admin-api` PASS.

### Task P0.2.C — Create `internal/admin/server.go` + `types.go`
- **Owner**: Muscle
- **File**: NEW (2 files)
- **Action**: Gin router (already in go.mod check), middleware Bearer auth, route `POST /v2/sources/register` → handler.
- **DoD**: Build PASS.

### Task P0.2.D — Implement `handleRegisterSource` + helpers
- **Owner**: Muscle
- **File**: `internal/admin/source_register.go` (NEW)
- **Action**: Per `02_plan_*.md` & `09_tasks_solution_*.md` chi tiết. 5 steps + rollback compensation 207 partial.
- **DoD**: Build PASS, mock unit test PASS.

### Task P0.2.E — Unit tests admin handler
- **Owner**: Muscle
- **File**: `internal/admin/server_test.go` (NEW)
- **Action**: sqlmock cho DB step 1, httptest server stub cho Debezium step 2 + Schema Registry step 3, mock NATS for step 4.
- **DoD**: `go test ./internal/admin/...` PASS.

### Task P0.2.F — Smoke E2E P0.2
- **Owner**: Muscle
- **Action**: Per "End-to-end smoke" section trong `02_plan_*.md`. Register collection mới, INSERT doc, verify shadow + master.
- **DoD**: response 200 với provisioning_state=active, shadow row landed, master row landed sau cron tick.

---

### Task X — Security review
- **Owner**: Muscle (`/security-agent`)
- **Action**: Sweep `cmd/admin-api` + `internal/admin/` for:
  - Token leak (logs, error strings).
  - SQL injection (raw queries với input → đã dùng parameterized via gorm).
  - SSRF (Debezium/SchemaRegistry URL từ config trusted, OK).
  - Listen addr default loopback only.
- **DoD**: 0 findings ≥ medium.

### Task Y — Append progress + lessons
- **Owner**: Brain (sau khi tất cả land)
- **File**:
  - `agent/memory/workspaces/feature-cdc-integration/05_progress.md` — APPEND closure cho mỗi P task.
  - `agent/memory/global/lessons.md` — APPEND Global Pattern: "Refresh-via-signal cho consumer group có topic set tĩnh" + "Transactional provisioning across DB + external HTTP API needs compensation, không phải auto-rollback".
- **DoD**: Files updated (APPEND only — CLAUDE.md §11).

---

## Sequencing rules

- P1.1.A → P1.1.B → P1.1.C (linear).
- P0.1.A → P0.1.B → P0.1.C → P0.1.D → P0.1.E (linear).
- P0.2.A → P0.2.B → P0.2.C → P0.2.D → P0.2.E → P0.2.F (linear).
- P0.1 phải land trước P0.2 vì P0.2 step 4 publish NATS dùng subject `cdc.cmd.kafka.refresh-topics` mà chỉ tồn tại sau khi P0.1.C subscribe.
- Task X chạy sau khi cả 3 phase code land.
- Task Y chạy sau Task X.

---

## Estimate

| Task | Effort |
|------|--------|
| P1.1.A+B+C | 30m |
| P0.1.A+B+C+D+E | 2h (refactor delicate) |
| P0.2.A-F | 4h (cmd mới, package mới, integration test) |
| Task X | 30m |
| Task Y | 15m |
| **Total** | **~7h** |

Note: estimate tham khảo. Muscle có quyền tự đánh giá lại sau khi đọc code.

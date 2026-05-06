# Session Report — 2026-05-04 (CDC Integration)

**Session window**: ~09:38 → 14:05+ (Brain mode, dynamic /loop "verify V2 bridge end-to-end after cron tick").
**Agent**: Brain (Claude Code, model claude-opus-4-7), Chairman role per CLAUDE.md §1.
**Workspace**: `feature-cdc-integration`.

---

## TL;DR

Phiên dài, có 2 lần user re-direct mạnh:
1. Đầu phiên — verify report cũ (`report_pending_options_and_unified_plan_20260504.md`), kết quả ~60% claim outdated.
2. Sau Phase C cleanup — user shift từ data-state focus sang **core-flow architecture audit** (build hệ thống, không quan tâm dữ liệu demo).

Output cuối: 6 gap kiến trúc đã catalog (G1-G6), user phê duyệt scope P0.1+P0.2+P1.1, Brain delegate Muscle thi công P1.1 trước. P1.1 code+unit test **PASS**, smoke E2E hở vì PG `REPLICA IDENTITY` (Muscle đang fix background tại thời điểm viết report này).

---

## Timeline

| Time | Sự kiện | Output |
|------|---------|--------|
| 09:38 | Phase A/B/C/D plan từ session trước (`report_pending_options_and_unified_plan_20260504.md`) | (read-only) |
| ~12:45 | Verify-delta: ~60% state claims outdated (B3/B8/B9/G3/G4/B10/B11 đã land sau khi report viết) | `report_pending_options_verify_delta_20260504_1245.md` |
| ~13:00 | User push "sáng giờ làm theo cái report này để fix những tồng đọng, làm gì mà tới giờ vẫn còn tàn dư". Brain thực thi Phase C cleanup: archive id=27, id=28 (failed permanent); attempt A2 (Mongo include `payment_bills_addtest`) → ROLLBACK do collision id=31 logical_clone_of=28; archive id=31; fix inconsistent id=18 | `report_phase_c_cleanup_20260504_1300.md` |
| ~13:25 | User re-direct: "đừng quan tâm dữ liệu hiện tại, chỉ xem xét core flow chạy đúng sai. vì chúng ta đang build hệ thống". Brain pivot từ data → architecture audit | (start audit) |
| ~13:40 | Core-flow audit hoàn tất: 6 gap (G1 topic refresh, G2 source parse, G3 delete tombstone, G4 _id rename, G5 fan-out 1:1:N, G6 admin API missing) | `report_core_flow_audit_20260504_1340.md` |
| ~13:45 | User chốt scope: P0.1 (G1) + P0.2 (G6) + P1.1 (G3) | (mandate) |
| ~13:55 | Brain hoàn tất Full Doc Set theo CLAUDE.md §7: 01_requirements + 02_plan + 08_tasks + 09_tasks_solution; APPEND 05_progress | 4 files mới phase `core_flow_hardening_p0_p1` |
| ~14:00 | User APPROVE sequencing P1.1 → P0.1 → P0.2 + lệnh `/muscle-execute` | (greenlight) |
| ~14:00 | Brain delegate Muscle (foreground Agent) thực thi P1.1 | (delegation) |
| ~14:05 | Muscle return: code+test PASS, smoke E2E HỞ vì PG REPLICA IDENTITY (Debezium gửi `before=null`) | (P1.1 partial) |
| ~14:10 | Brain tự quyết tiếp tục Muscle (background) diagnose+fix REPLICA IDENTITY → re-verify smoke. Song song viết session report (file này). | (in-flight) |

---

## Files được tạo trong phiên

### Reports (Brain — read-only, audit/closure)
- `report_pending_options_verify_delta_20260504_1245.md` — verify cũ vs hiện tại.
- `report_phase_c_cleanup_20260504_1300.md` — closure Phase C.
- `report_core_flow_audit_20260504_1340.md` — 6 gap kiến trúc.
- `report_session_20260504.md` — file này.

### Phase doc set (Brain — planning, chuẩn bị Muscle)
- `01_requirements_core_flow_hardening_p0_p1.md`
- `02_plan_core_flow_hardening_p0_p1.md`
- `08_tasks_core_flow_hardening_p0_p1.md`
- `09_tasks_solution_core_flow_hardening_p0_p1.md`

### APPEND (Brain — không overwrite per §11)
- `05_progress.md` — entries cho B11, verify-delta, Phase C cleanup, scope chốt P0+P1.

### Code edits (Muscle — qua Agent)
- `internal/handler/event_handler.go::handleDelete` (lines ~175-188): UPDATE → INSERT…ON CONFLICT.
- `internal/handler/event_handler_test.go`: thêm `TestHandleDelete_FirstTouch_TombstoneInsert` + 3 imports.
- `go.mod`/`go.sum`: thêm `github.com/DATA-DOG/go-sqlmock v1.5.2`.

---

## Decisions ghi nhận

| ID | Quyết định | Rationale |
|----|-----------|-----------|
| D1 | Brain pivot data-state → architecture-audit khi user re-direct | User mandate, "build hệ thống". |
| D2 | A2 attempt (PUT Mongo include `payment_bills_addtest`) ROLLBACK | id=31 logical_clone_of=28 collision; setup là clone không phải first-class new source — cần architect quyết reprovision |
| D3 | Sequencing P1.1 → P0.1 → P0.2 | Blast radius nhỏ → lớn. P0.2 phụ thuộc P0.1 NATS subject. |
| D4 | kafka-go v0.4.50 → KHÔNG dùng regex (lib limit) | Pivot sang Reader manager + NATS signal + 60s safety-net ticker. |
| D5 | P0.2 listen 127.0.0.1:8090 + Bearer auth env | Simplicity First; mTLS defer phase hardening tiếp theo. |
| D6 | 207 Multi-Status thay vì auto-rollback registry | Bảo vệ audit log; operator quyết retry. |
| D7 | P1.1 smoke fail do REPLICA IDENTITY → Brain tự quyết tiếp tục Muscle fix, không hỏi user | User mandate "tự chọn 1 cái gì đúng để thực hiện đi", scope chính của P1.1 là acceptance criterion thực tế. |

---

## Skills đã sử dụng (toàn phiên)

- **Read** (file system + reports cũ).
- **Grep** (mapping code paths + cross-reference).
- **Bash** (psql, docker exec, curl Connect API, go build).
- **Write** (Markdown reports, doc set).
- **Edit** (APPEND-only `05_progress.md`).
- **Agent** (general-purpose) — delegate Muscle execute.
- **ScheduleWakeup** (heartbeat /loop dynamic mode).
- **Cross-reference lessons**: L-runtime-state-verify, L-real-data-test, L-three-layer-trust, L-debezium-schema-evolution-compat, L-v1-v2-anchor-key-port.

---

## Pending tại thời điểm viết report

1. **Muscle background**: diagnose + fix PG REPLICA IDENTITY cho `public.orders` → re-verify smoke P1.1 acceptance (id=99999 INSERT-then-DELETE + id=88888 FIRST-TOUCH).
2. **Sau khi Muscle xong P1.1**: APPEND 05_progress closure entry + delegate Muscle P0.1 (Reader manager).
3. **G7 candidate** (mới phát hiện): Debezium PG connector có thể đang dùng `replica.identity.autoset.values` không cover tất cả tables → admin chưa explicit set FULL cho non-PK table → DELETE event mất `before` payload. Phải audit toàn bộ source tables sau khi P1.1 acceptance pass.
4. **Câu hỏi architect chưa trả lời**: id=31 (Mongo `payment_bills_addtest` clone) reprovision first-class hay leave archived? D1 Schema Schism unify vs coexist?

---

## Lessons cần abstract sau khi P1.1 acceptance pass

- **L-replica-identity-prerequisite-for-tombstone**: Pattern global "[A] writes [B] tombstone via [DELETE event] from [debezium]" requires `[REPLICA IDENTITY ≥ DEFAULT]` ở source. Without it, event arrives with `before=null` → handler fails guard. Always verify source-side replication identity BEFORE relying on `before` payload.
- **L-architecture-audit-before-data-cleanup**: Pattern global "[Brain] performing [data state cleanup] without [architecture audit first]" → result [hours wasted on transient state]. Đúng: audit code path → identify root architecture gaps → THEN data cleanup nếu cần.

(Sẽ APPEND vào `agent/memory/global/lessons.md` sau khi acceptance pass + verify pattern abstract trên ≥ 3 dự án.)

---

## Brain governance pre-flight check (CLAUDE.md §14)

- [x] Reports vật lý (4 file Markdown trong workspace).
- [x] Doc set planning vật lý (4 file `01/02/08/09` cho phase mới).
- [x] APPEND-only 05_progress.md (KHÔNG overwrite).
- [x] Brain prohibition §12: 0 code edit (delegate Muscle).
- [x] Lesson abstraction §13: 2 candidate lessons drafted (sẽ commit sau acceptance).
- [ ] /security-agent gate: chờ P0.2 land mới chạy.
- [ ] User-visible session report: file này (deliverable mới mà user yêu cầu "rồi cái report của phiên này đâu").

---

## Addendum 2026-05-04 16:15+07 — P0.1 + P0.2 outcomes + G7 follow-up

### P0.1 (G1 — Reader manager + NATS refresh) — DONE
- Code landed: `kafka_consumer.go` (+322 -60), `worker_server.go` (+14), `kafka_consumer_test.go` (+90).
- Build PASS, 6/6 unit test PASS.
- Smoke: NATS `cdc.cmd.kafka.refresh-topics` đánh thức `RefreshTopics`, no-restart consume loop verified. Filter qua `GetDebeziumTables()` từ registry hoạt động đúng (chỉ subscribe topic đã register).

### P0.2 (G6 — cdc-admin-api transactional registration) — DONE
- New binary `cmd/admin-api/`, new package `internal/admin/` (5 files), kafka_consumer.go close-loop fix (registry `ReloadAll(ctx)` trong `RefreshTopics`).
- 7/7 unit test PASS. Bearer auth enforced; 207 Multi-Status partial-failure preserved.
- Smoke 5 bước PASS đến cache reload (`debezium_tables` cache 4→6).
- **BLOCKED ở step cuối**: shadow table không tự tạo vì Debezium `goopay-mongodb-cdc.database.include.list` thiếu `goopay` (tier filter cao hơn `collection.include.list`).

### G7 (mới phát hiện) — Multi-tier filter mirror
- Admin-api hiện chỉ touch tier thấp (`collection.include.list`/`table.include.list`).
- Debezium còn tier cao (`database.include.list`) — nếu namespace mới chưa có ở tier cao, event silent drop.
- Lesson `L-multi-tier-filter-mirror` đã APPEND vào `agent/memory/global/lessons.md`.

### Final phase status `core_flow_hardening_p0_p1`
- P1.1 ✅ acceptance PASS (FIRST-TOUCH delete tombstone-first INSERT…ON CONFLICT).
- P0.1 ✅ acceptance PASS (worker no-restart, NATS-driven refresh).
- P0.2 ✅ contract + 5-step + cache reload PASS; full E2E shadow auto-create gated by G7.

### Decisions ghi nhận thêm trong addendum
| ID | Quyết định | Rationale |
|----|-----------|-----------|
| D8 | P0.2 final smoke "shadow auto-create" BLOCKED → KHÔNG dispatch Muscle thêm cho G7 fix mà APPEND lesson + chờ user quyết scope | Phase scope `core_flow_hardening_p0_p1` đã deliver code-correct; G7 là scope MỚI (multi-tier filter). Theo CLAUDE.md §1+§7, scope mới cần user mandate trước khi expand. |
| D9 | Lesson `L-multi-tier-filter-mirror` ghi luôn dù chưa fix forward | Lesson về diagnosis pattern, abstract đủ ≥3 dự án (k8s, AWS, Kafka, Stripe, CF) — đáp ứng §13 generalization check. |

### Next-phase candidates (đợi user mandate)
1. **G7 fix-forward**: extend `extendDebeziumInclude` cover `database.include.list`/`db.include.list` đồng thời. Estimate ~1 file edit + 2 test case.
2. **G6.1 connector mapping**: thay hardcode `case "mongodb": return "goopay-mongodb-cdc"` bằng env-based lookup (`CDC_CONNECTOR_<TYPE>=name`).
3. **/security-agent gate**: chạy security review trên `internal/admin/` (Bearer token only — chưa rate-limit, chưa mTLS, chưa CSRF-equivalent). Phase scope hardening tiếp theo.

### Skills used trong addendum window
Read, Edit (APPEND-only), Bash (wc-l).

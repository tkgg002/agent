# Báo cáo: Đóng phase `core_flow_hardening_p0_p1` + Re-scan hệ thống + Plan tổng hợp kế tiếp

**Date**: 2026-05-04 (16:30+07)
**Phase**: `core_flow_hardening_p0_p1` (P1.1 + P0.1 + P0.2)
**Trigger**: User yêu cầu — "report lại Phase scope core_flow_hardening_p0_p1, rộng hơn là cái report_pending_options_and_unified_plan_20260504.md"
**Phương pháp**: Exercise-driven verification — query DB thực tế (postgres-cdc/source/dest), curl Kafka Connect REST runtime, đọc code, đọc các file 01/02/05/08/09 vật lý.
**Verdict**: 3/3 task scope chốt PASS code-correct + acceptance criterion PASS cho P1.1. Full data-plane E2E shadow-auto-create cho admin-api collection mới gated bởi G7 (Debezium tier-cao filter). Một số gap từ session 2026-05-02 đã được giải quyết tự nhiên (B8 MariaDB plugin live, OTel collector live).

---

## 1. Tóm tắt scope phase đã đóng (P1.1 + P0.1 + P0.2)

### 1.1 P1.1 (G3 — handleDelete tombstone-first INSERT…ON CONFLICT) — ✅ DONE

| Hạng mục | Chi tiết |
|---|---|
| File sửa | `internal/handler/event_handler.go` (`handleDelete` lines 145-201, A1 + A2 fixes), `internal/handler/kafka_consumer.go` (`unwrapAvroUnion` cho `before` field tại line ~313-393), `internal/handler/event_handler_test.go` (+94 lines, 2 test mới) |
| Acceptance criterion user set | "Xóa 1 dòng ở Source mà dòng đó chưa từng xuất hiện ở Shadow → Shadow phải tự chèn 1 dòng mới với `_deleted = TRUE` và đầy đủ anchor `_gpay_source_id`" |
| Verify (independent Brain) | id=88888 FIRST-TOUCH + id=99999 INSERT-then-DELETE đều `_deleted=t`, `_gpay_source_id` đầy đủ |
| Lesson abstracted | `L-event-translator-field-completeness` — translator hardcode optional field nil che bug upstream; hard-fail boundary guard misdirect ops |

### 1.2 P0.1 (G1 — kafka-go Reader manager + NATS refresh, no-restart) — ✅ DONE

| Hạng mục | Chi tiết |
|---|---|
| File sửa | `internal/handler/kafka_consumer.go` (+322 -60: `refreshMu sync.Mutex`, `currentTopics []string`, `discoverFunc`, `buildReader`, `RefreshTopics`, `topicSetEqual`, 60s safety-net ticker), `internal/server/worker_server.go:526` NATS subscribe `cdc.cmd.kafka.refresh-topics`, `internal/handler/kafka_consumer_test.go` (+90 lines, 3 test groups) |
| Pattern chốt | Reader manager + mutex snapshot trong consume loop + `topicSetEqual` no-op early exit + EOF/closed-reader handling |
| Lý do KHÔNG dùng regex | kafka-go v0.4.50 `GroupTopics []string` không hỗ trợ regex với consumer-group mode |
| Acceptance | Worker không restart trong toàn smoke window. NATS publish `cdc.cmd.kafka.refresh-topics` 3× → log "nats-triggered topic refresh ok" |

### 1.3 P0.2 (G6 — cdc-admin-api transactional registration) — ✅ DONE (full E2E gated G7)

| Hạng mục | Chi tiết |
|---|---|
| File mới | `cmd/admin-api/main.go` (HTTP entry, listen `127.0.0.1:8090`, Bearer auth env), `internal/admin/server.go` (gin router + middleware), `internal/admin/types.go` (DTOs), `internal/admin/source_register.go` (5-step orchestrator), `internal/admin/helpers.go` (`extendDebeziumInclude`, `preemptSchemaRegistry`, `connectorNameFor`), `internal/admin/server_test.go` (7 test, sqlmock + httptest + embedded NATS) |
| 5 bước orchestration | (1) atomic INSERT `source_object_registry` + `shadow_binding`; (2) PUT Debezium `collection.include.list` / `table.include.list`; (3) Schema Registry pre-empt subjects compatibility=NONE; (4) NATS publish `cdc.cmd.kafka.refresh-topics`; (5) UPDATE `is_active=true` |
| Cache reload close-loop fix | `RefreshTopics` thêm type-assert `registrySvc.ReloadAll(ctx)` để cache `GetDebeziumTables()` bắt được row mới do admin-api INSERT — verify worker log `V2 metadata registry reloaded sources:6, debezium_tables:6` (cũ 4 → 6) |
| Acceptance contract | 200 happy path / 401 bad token / 207 Multi-Status partial (step 2/3 fail) / 500 chỉ khi step 1 fail. 7/7 unit test PASS |
| Lesson abstracted | `L-multi-tier-filter-mirror` — orchestrator phải touch ALL tier filter (database + collection / DB + table) chứ không chỉ tier thấp |

### 1.4 Build + test summary

| Layer | Kết quả |
|---|---|
| `go build ./...` | PASS |
| `go test ./internal/handler/...` | 6/6 PASS (P1.1 + P0.1) |
| `go test ./internal/admin/...` | 7/7 PASS (P0.2) |
| Smoke E2E P1.1 | PASS (FIRST-TOUCH + INSERT-then-DELETE đều có tombstone) |
| Smoke E2E P0.1 | PASS đường no-change + add-topic; worker no-restart |
| Smoke E2E P0.2 | 7/8 step PASS; step 8 (shadow auto-create cho `goopay.smoke_*`) BLOCKED (G7) |

---

## 2. Re-scan hệ thống (real-time verification 2026-05-04 16:30+07)

### 2.1 Containers (13 running)

| Container | Status | Note |
|---|---|---|
| gpay-cdc-worker | Up 31 minutes | live; cron tick 60s, transmute scheduler active |
| **gpay-otel-collector** | Up 5 hours | **MỚI** — G3 (session 2026-05-02) đã được giải quyết tự nhiên |
| gpay-kafka-connect | Up 5 hours (healthy) | port 18083→8083 |
| gpay-mariadb / gpay-mongo / gpay-postgres-cdc/source/dest/postgres | Up 5-6 days (healthy) | hạ tầng |
| gpay-kafka / gpay-schema-registry / gpay-redis / gpay-nats | Up 6 days | hạ tầng |

### 2.2 Kafka Connect runtime (live verification — đã tiến bộ vs report cũ)

```
/connector-plugins:
  MongoDbConnector
  MySqlConnector       ← B8 đã RESOLVED (session cũ claim "compose updated nhưng plugin không load" — giờ đã load thật)
  PostgresConnector
  Mirror{Checkpoint,Heartbeat,Source}Connector

/connectors:
  cdc-pg-source        — table.include.list = public.orders,public.users,public.payments
                         ↑ vẫn THIẾU public.orders_addtest (B3 chưa đụng)
  cdc-mariadb-source   — RUNNING, table.include.list = legacy_orders + legacy_orders_addtest ✅
  goopay-mongodb-cdc   — collection.include.list += smoke_p02_close_{1777882181,1777882418}
                         (do admin-api P0.2 PUT thành công)
                         database.include.list = payment-bill-service,centralized-export-service
                         ↑ THIẾU `goopay` ⇒ G7 — collection smoke_* không stream được
```

### 2.3 cdc_system control-plane state

| Bảng | Số rows | Note vs session 2026-05-02 |
|---|---|---|
| `source_object_registry` total | 25 | +3 từ admin-api smoke (id 32/33/34 mongo_smoke_p02 / mongo_close_*) |
| `source_object_registry is_active=t` | 5 | id 11 (orders), 26 (e2e_v5), 29 (addtest_pg), 30 (addtest_maria) |
| `source_object_registry is_active=f` | 20 | 10 V1 legacy + 7 stale failed/curl + id 31 (mongo bills clone) + id 32/33/34 smoke |
| `transmute_schedule` | 5/5 success | last_run_at 08:21:13 UTC = 15:21+07 (cron interval đã chạy nhiều tick từ đó) |

### 2.4 Shadow + Master tables (snapshot)

| Bảng | Rows | Thay đổi vs report cũ |
|---|---|---|
| `shadow_goopay_source.orders` | 16 | +11 (5 → 16) — ingest đang flow |
| `shadow_src_local_pg_source.orders_addtest` | 11 | có ingest (B3 không hoàn toàn block) |
| `shadow_mariadb_legacy_default.legacy_orders` | 0 | chưa ingest dù connector RUNNING |
| `shadow_mariadb_legacy_default.legacy_orders_addtest` | 1 | ingest 1 row (B8 path đã thông) |
| `dw_orders.orders_fact` | 35 | +10 (25 → 35) — transmute đang propagate |
| `dw_src_local_pg_source.orders_addtest` | 0 | shadow có 11 rows, master = 0 ⇒ schedule chưa enable hoặc binding chưa active |

---

## 3. Gap matrix (status post P0+P1)

### 3.1 Gap đã được giải quyết tự nhiên hoặc qua phase này

| Gap | Trạng thái | Bằng chứng |
|---|---|---|
| **G3 (P1.1)** delete-tombstone | ✅ CLOSED | unit test + smoke acceptance PASS |
| **G1 (P0.1)** topic refresh không restart | ✅ CLOSED | NATS-driven refresh + cache reload verified |
| **G6 (P0.2)** cdc-admin-api transactional | ✅ CLOSED contract + 5-step | endpoint live, 7/7 test PASS |
| **B8** MariaDB plugin | ✅ RESOLVED tự nhiên | `MySqlConnector` ở `/connector-plugins` + connector `cdc-mariadb-source` RUNNING |
| **G3 (session cũ)** OTel collector | ✅ RESOLVED tự nhiên | `gpay-otel-collector` Up 5h — không còn `connection refused` mỗi 5s |

### 3.2 Gap đang treo (mới hoặc còn từ session trước)

| Gap | Mức độ | Mô tả | Evidence |
|---|---|---|---|
| **G7 (mới)** Debezium multi-tier filter mirror | High | `extendDebeziumInclude` chỉ touch tier thấp (`collection.include.list`/`table.include.list`); tier cao (`database.include.list`/`db.include.list`) silent drop. Admin-api báo "register OK" nhưng pipeline không stream | `goopay-mongodb-cdc.database.include.list` thiếu `goopay` |
| **G2** Mongo `payment_bills_addtest` chưa stream | Med | Collection physical exists. `goopay-mongodb-cdc.collection.include.list` chưa có `payment-bill-service.payment_bills_addtest` (vẫn từ session 2026-05-02) | Connector config |
| **B3 (PG)** `public.orders_addtest` chưa trong include | Low | `cdc-pg-source.table.include.list` chỉ có 3 table chính | shadow_src_local có 11 rows nghĩa là có path khác đang ingest — cần audit |
| **G4 (mới)** master `orders_addtest` không transmute | Med | shadow có 11 rows nhưng master = 0 ⇒ binding/schedule chưa enable, hoặc `is_enabled=false` ở `transmute_schedule` | DB query |
| **G5 (cũ)** legacy V1 seeds | Low | 10 rows `legacy_*` vẫn `is_active=f` chiếm slot. Script `prune_legacy_v1_bindings.sql` chưa chạy | Migration 035 |
| **G6 (cũ)** orphan shadow `orders_e2e_d_v2/v3/v4` | Low | 3 stale tables tồn tại; sources id 23/24/25 đã `failed` | Schema list |
| **G6.1 (mới)** connector mapping hardcode | Low | `connectorNameFor("mongodb")` return literal `goopay-mongodb-cdc`; cần env-based `CDC_CONNECTOR_<TYPE>=name` để multi-tenant | `internal/admin/helpers.go` |
| **G8 (mới)** admin-api security hardening | Med | Bearer token only — chưa rate-limit, chưa mTLS, chưa CSRF-equivalent. `/security-agent` gate chưa chạy | `internal/admin/server.go` middleware |
| **G9 (mới)** transmute_schedule schema mismatch giữa plan cũ và DB | Low | Plan cũ tham chiếu `master_table` column, DB thực tế có `master_binding_id`. Doc drift | `\d cdc_system.transmute_schedule` |

### 3.3 Câu hỏi architect chưa trả lời (carry-over)

1. **id=31 `mongo_payment_bills_v2`** (logical_clone_of=28) — reprovision first-class hay leave archived?
2. **D1 Schema Schism** — V1 (`id` TEXT) vs V2 (`_gpay_id` BIGINT) shadow PK convention — unify hay coexist?
3. **G2 Mongo addtest** — extend `collection.include.list` ngay hay đợi G7 fix-forward để kèm `database.include.list`?

---

## 4. Plan tổng hợp kế tiếp (ưu tiên + sequencing)

> **Brain prohibition §12**: dưới là đề xuất, KHÔNG phải code change. Cần user mandate scope trước khi delegate Muscle.

### 4.1 Phase E — Multi-tier filter close-loop (G7 fix-forward) — **HIGHEST**

#### E1 [CODE] Extend `extendDebeziumInclude` cover `database.include.list`/`db.include.list`
- File: `internal/admin/helpers.go`
- Logic: parse + extend tier-cao **trước** tier-thấp; idempotent dedup; per-engine adapter (Mongo: `database.include.list`; PG: verify `database.dbname` match; MySQL/Maria: `database.include.list`).
- Test: 3 case (already-present no-op / new database add / mismatch error).
- Live verify: re-run smoke `goopay.smoke_*` collection → wait 30s → `kafka-console-consumer` thấy event → shadow row landed.

#### E2 [CODE] Pre-flight namespace check + WARN response field
- Khi register, nếu detect resource thuộc namespace chưa trong tier-cao → response thêm `warnings: ["database 'goopay' was just added to debezium include — first event may be delayed"]`.
- Tránh confusion cho operator.

**DoD Phase E**: smoke `goopay.smoke_p02_*` ingest end-to-end (collection PUT → registry insert → Debezium config update — both tier — Schema Registry pre-empt → NATS signal → topic discover → shadow row landed → wait 60s cron → master row landed) trong 1 lần admin-api call duy nhất, worker không restart.

---

### 4.2 Phase F — Security gate (G8) — **HIGH**

#### F1 [SECURITY] Chạy `/security-agent` review trên `internal/admin/`
- Threat model: Bearer token leak, replay, race condition giữa step 1↔step 5, NATS subject hijack.
- Output: list mitigation cần land trước khi expose admin-api ngoài 127.0.0.1.

#### F2 [CODE] Rate-limit (per-token bucket) + audit log persist
- File: `internal/admin/server.go` middleware chain.
- Persist mỗi request vào `cdc_system.cdc_activity_log` (đã tồn tại) — actor / endpoint / payload hash / status / duration.

#### F3 [DOC] Operator runbook
- Convention naming token, rotate procedure, abuse detection, 207 Multi-Status interpretation.

**DoD Phase F**: `/security-agent` report no-blocker; rate-limit enforced; audit log per request.

---

### 4.3 Phase G — Cleanup stale state (G2/G4/G5/G6) — **MEDIUM**

#### G1 [SQL] Run prune `legacy_*` script đã có sẵn (G5)
- `deployments/sql/cdc/prune_legacy_v1_bindings.sql` — idempotent.
- Verify count(*) where `object_code LIKE 'legacy_%' AND is_active=true` → 0.

#### G2 [SQL] Investigate orphan shadow `orders_e2e_d_v2/v3/v4` (G6)
- Quyết per-source: archive (drop shadow) hay revive (re-trigger provisioning).

#### G3 [SQL] Audit transmute_schedule cho `orders_addtest` (G4)
- Query: `SELECT id, mode, is_enabled, last_status FROM cdc_system.transmute_schedule WHERE master_binding_id IN (SELECT id FROM cdc_system.master_binding WHERE source_object_id = 29)`
- Nếu `is_enabled=false` → enable + verify cron `last_run_at` advances.

#### G4 [CODE] Reconcile audit của `cdc-pg-source.table.include.list` vs registry (B3 PG)
- Tại sao `shadow_src_local_pg_source.orders_addtest` có 11 rows nhưng `table.include.list` chỉ có 3 table chính? Có thể có connector khác đang stream, hoặc legacy ingest path chưa drain. Audit để chắc data integrity.

---

### 4.4 Phase H — Multi-tenant connector lookup (G6.1) — **LOW**

#### H1 [CODE] Env-based connector mapping
- `internal/admin/helpers.go::connectorNameFor` thay hardcode bằng `os.Getenv(fmt.Sprintf("CDC_CONNECTOR_%s", strings.ToUpper(sourceType)))`.
- Default fallback giữ nguyên giá trị hiện tại để backward-compat.
- Doc trong `agent/memory/global/conventions.md`.

---

### 4.5 Phase I — Schema Schism resolution (D1 architect) — **LONG-TERM**

(Carry-over từ session cũ — chờ architect quyết unify vs coexist.)

---

## 5. Verification end-to-end (sau Phase E + F + G)

```bash
# 1. Phase E acceptance — admin-api end-to-end vs new database namespace
TS=$(date +%s)
curl -X POST http://127.0.0.1:8090/v1/sources/register \
  -H "Authorization: Bearer $CDC_ADMIN_API_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "source_code": "smoke_phase_e_'"$TS"'",
    "source_type": "mongodb",
    "database_name": "newns_'"$TS"'",
    "collection_name": "items"
  }'
# Expect: 200 + warnings: ["database 'newns_<ts>' was just added..."]

# 2. Insert source doc
docker exec gpay-mongo mongosh "mongodb://localhost:27017/newns_${TS}" \
  --eval "db.items.insertOne({_id: 'e2e-1', name: 'phase-e-smoke'})"

# 3. Wait Debezium snapshot 30s + cron tick 60s
sleep 90

# 4. Verify shadow + master
docker exec gpay-postgres-cdc psql -U gpay_admin -d cdc_dw -c \
  "SELECT count(*) FROM shadow_mongo_<binding>.items;"
# Expect: >= 1
docker exec gpay-postgres-dest psql -U gpay_admin -d goopay_dest -c \
  "SELECT count(*) FROM dw_mongo_<binding>.items;"
# Expect: >= 1

# 5. Phase F — rate-limit smoke
for i in $(seq 1 100); do curl -s -o /dev/null -w "%{http_code}\n" \
  http://127.0.0.1:8090/v1/sources/register -H "Authorization: Bearer X"; done | sort | uniq -c
# Expect: phần lớn 401, sau N request thấy 429

# 6. Phase G — verify cleanup
docker exec gpay-postgres-cdc psql -U gpay_admin -d cdc_dw -c \
  "SELECT count(*) FROM cdc_system.source_object_registry
    WHERE object_code LIKE 'legacy_%' AND is_active=true;"
# Expect: 0
```

---

## 6. Files phase này đã tạo / sửa

### 6.1 Code (qua Muscle, đã land)

| Path | Action |
|---|---|
| `internal/handler/event_handler.go` | EDIT — `handleDelete` UPSERT tombstone |
| `internal/handler/event_handler_test.go` | EDIT — +94 lines, 2 test mới |
| `internal/handler/kafka_consumer.go` | EDIT — Reader manager (P0.1) + `unwrapAvroUnion` cho `before` (P1.1 fix A1) + `ReloadAll` close-loop (P0.2) |
| `internal/handler/kafka_consumer_test.go` | EDIT — +90 lines, 3 test |
| `internal/server/worker_server.go` | EDIT — NATS subscribe `cdc.cmd.kafka.refresh-topics` |
| `cmd/admin-api/main.go` | NEW |
| `internal/admin/server.go` | NEW |
| `internal/admin/types.go` | NEW |
| `internal/admin/source_register.go` | NEW |
| `internal/admin/helpers.go` | NEW |
| `internal/admin/server_test.go` | NEW |
| `go.mod` / `go.sum` | EDIT — `github.com/DATA-DOG/go-sqlmock`, `github.com/gin-gonic/gin`, `github.com/nats-io/nats-server/v2` |

### 6.2 Memory / docs (Brain, APPEND-only)

| Path | Action |
|---|---|
| `agent/memory/workspaces/feature-cdc-integration/01_requirements_core_flow_hardening_p0_p1.md` | NEW |
| `agent/memory/workspaces/feature-cdc-integration/02_plan_core_flow_hardening_p0_p1.md` | NEW |
| `agent/memory/workspaces/feature-cdc-integration/08_tasks_core_flow_hardening_p0_p1.md` | NEW |
| `agent/memory/workspaces/feature-cdc-integration/09_tasks_solution_core_flow_hardening_p0_p1.md` | NEW |
| `agent/memory/workspaces/feature-cdc-integration/05_progress.md` | APPEND ×4 (scope chốt + P1.1 closure + P0.1 closure + P0.2 closure) |
| `agent/memory/workspaces/feature-cdc-integration/report_session_20260504.md` | NEW + 1 addendum |
| `agent/memory/workspaces/feature-cdc-integration/report_phase_closure_core_flow_hardening_20260504.md` | NEW (file này) |
| `agent/memory/global/lessons.md` | APPEND ×2 (`L-event-translator-field-completeness`, `L-multi-tier-filter-mirror`) |

---

## 7. Skills used (CLAUDE.md §0)

- `Bash` — psql / docker exec / curl Kafka Connect REST / ps / wc-l
- `Read` — đọc 01/02/08/09 phase doc + report cũ + lessons
- `Edit` — APPEND-only memory (3 file: 05_progress, lessons, report_session addendum)
- `Write` — sinh report mới (file này)
- `Agent` (general-purpose) — delegate Muscle ×7 (P1.1 ×3 vòng, P0.1 ×1, P0.2 ×3 vòng)
- `ScheduleWakeup` — heartbeat /loop dynamic mode (đã dừng cuối phase)

**Governance honored** (CLAUDE.md §14):
- §1 phân quyền Brain/Muscle — đúng (Brain 0 source code edit)
- §7 Full Doc Set + APPEND-only — đúng
- §11 memory protection — không overwrite
- §12 Brain code prohibition — đúng
- §13 lesson abstraction — 2 lesson Global Pattern đã APPEND (đáp ứng generalization check ≥3 dự án)

**Lessons applied during phase**:
- `L-three-layer-trust` — diagnose translator layer trước khi nghi DB infra (P1.1 case)
- `L-runtime-state-verify` — verify Kafka Connect runtime `/connector-plugins` thay vì tin compose claim (B8 case)
- `L-real-data-test` — INSERT row mới → query shadow/master thực, không tin claim

**Lessons newly abstracted in phase**:
- `L-event-translator-field-completeness` — translator hardcode optional field (before/source/header) bị che bởi hard-fail boundary guard
- `L-multi-tier-filter-mirror` — orchestrator phải touch tất cả tier filter của hệ thống đích, không chỉ tier-thấp

---

## 8. Câu hỏi cần user/architect quyết để đề xuất Phase tiếp theo

1. **G7 fix-forward** — extend `extendDebeziumInclude` đồng thời `database.include.list` ngay (Phase E1) hay deploy admin-api với caveat hiện tại trước rồi nâng cấp sau?
2. **G8 security gate** — chạy `/security-agent` ngay trước khi expose admin-api ra ngoài `127.0.0.1`, hay defer đến khi có integration với cdc-portal?
3. **G6.1 connector mapping** — cần multi-tenant không, hay 1-engine-1-connector convention là đủ cho roadmap 6 tháng?
4. **G2 Mongo addtest** — extend `collection.include.list` riêng (1 line PUT) hay đợi Phase E1 land rồi gộp 1 admin-api call?
5. **D1 Schema Schism** (carry-over) — V1 vs V2 unify hay coexist?

Nếu user approve scope nào, em delegate Muscle thực thi theo sequencing E → F → G → H, mỗi phase 1 PR riêng để blast radius nhỏ.

---

**File này được sinh bởi Brain (Claude Code, claude-opus-4-7), tuân thủ CLAUDE.md §0 (tiếng Việt), §12 (no source-code edit), §14 (governance pre-flight).**

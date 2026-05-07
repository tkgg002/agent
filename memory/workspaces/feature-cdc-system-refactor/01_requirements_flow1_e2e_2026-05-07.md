# Requirements — Flow 1 (Connect Source) E2E lên-được

> **Author**: max (Brain) | **Date**: 2026-05-07 ICT
> **Driver**: Boss directive 2026-05-07 "bằng mọi giá phải lên đc flow1 này"
> **Predecessor**: `report_flow1_connect_source_2026-05-07.md` (overview + 6 gap đã liệt kê)
> **Audience**: x2 (Muscle, cms-lane). x2 review qua `09_tasks_solution_flow1_x2_2026-05-07.md` rồi mới execute.

---

## 1. Mục tiêu (Goal)

Operator chạy 1 lệnh → 60–120s sau, shadow PG có table mới + populated rows từ source. Toàn bộ flow auto, không cần SQL thủ công sau khi POST `/v2/sources/register`.

## 2. Scope

### In scope (Flow 1 = Wizard step 1–5)
- Tạo Debezium connector qua CMS proxy `POST /api/v1/system/connectors`
- Register source (admin-api `POST /v2/sources/register`) — 5 sub-step phải chạy đủ + state machine fire `cdc.cmd.shadow.bind`
- Worker handle `shadow.bind` → schema_adapter auto-create shadow schema + table với CDC + business cols
- shadow_binding `ddl_status` chuyển từ `pending` → `created`
- source_object_registry `provisioning_state` chuyển `draft` → `shadow_pending` → `shadow_active`
- Debezium snapshot/streaming → Kafka → worker consume → upsert shadow

### Out of scope
- Master layer (Wizard step 6–11) — Flow 2
- FE wizard UI render test — visual smoke khác buổi
- Track E (Mongo CDC) advanced features — workspace riêng
- Production deploy — Stage hiện tại Development/Local

## 3. Acceptance criteria E2E (DoD)

Run 1 happy-path smoke với 1 PG source mới (PG connector đang RUNNING — đảm bảo gốc OK):

| # | Check | Method | Expected |
|---|---|---|---|
| AC-1 | Source register OK | `curl POST /v2/sources/register` | HTTP 200, body `{provisioning_state:"active",steps_completed:["registry_insert","debezium_include_extend","schema_registry_preempt","worker_signal"]}` |
| AC-2 | Connector RUNNING | `curl :18083/connectors/<name>/status` | `connector.state="RUNNING"` + `tasks[].state="RUNNING"` (no FAILED) |
| AC-3 | Shadow schema tồn tại | `psql cdc-metadata SELECT 1 FROM information_schema.schemata WHERE schema_name='shadow_<db>'` | row found |
| AC-4 | Shadow table tồn tại với CDC cols | `\d shadow_<db>.<table>` | có pk + 8 CDC cols (`_raw_data,_source,_synced_at,_version,_hash,_deleted,_created_at,_updated_at`) + business cols inferred |
| AC-5 | `shadow_binding.ddl_status='created'` | `SELECT ddl_status FROM cdc_system.shadow_binding WHERE source_object_id=<N>` | `'created'` (không phải `'pending'`) |
| AC-6 | `source_object_registry.provisioning_state='shadow_active'` | `SELECT provisioning_state FROM cdc_system.source_object_registry WHERE id=<N>` | `'shadow_active'` (đã pass state machine) |
| AC-7 | Kafka topic có data | `docker exec gpay-kafka kafka-console-consumer --topic cdc.<prefix>.<db>.<table> --from-beginning --max-messages 1 --timeout-ms 30000` | ≥1 message |
| AC-8 | Shadow row count > 0 | `SELECT count(*) FROM shadow_<db>.<table>` sau snapshot | `count >= source_count` (within 60s of snapshot) |

**Bonus AC** (nếu time):
- AC-9: Test connection endpoint trả 200 trước khi register (Step 2 manual gap)
- AC-10: PG/MariaDB preflight gate active (Cascade Liability extend)

---

## 4. Constraints

- **Lane lock 2026-05-07 ICT effective** từ cms `b4a3461`:
  - max-Brain owns: worker code (`centralized-data-service/`), workspace docs, migrations cdc, PG metadata SQL
  - x2-Muscle owns: cms code (`cdc-cms-service/`), CMS test/build
- **§12 Brain Code Prohibition** — max plan + document, KHÔNG đụng .go/.ts/.sql trừ worker-lane (Boss override)
- **CLAUDE.md §11** — memory APPEND only
- **Auto mode** — execute fast, low-risk first
- **Boss approval gate** cho production-affecting actions (rebuild Debezium image, restart worker)

## 5. Available infrastructure (verified Phase A discovery)

| Component | Status | Notes |
|---|---|---|
| `gpay-postgres-cdc` (5433) | ✅ healthy | metadata + shadow PG |
| `gpay-postgres-source` (5435) | ✅ healthy + ping OK | PG source — dùng smoke happy-path |
| `gpay-postgres-shadow` (5436) | 🆕 alive (47h) | Container mới — code có biết không? G-5 |
| `gpay-postgres-dest` (5434) | ✅ healthy | dest DW (master) |
| `gpay-kafka-connect` (18083) | ✅ healthy | |
| `gpay-mongo` (17017) | ✅ healthy | tên container `gpay-mongo` (không `-source`) |
| `gpay-mariadb` (13307) | ✅ healthy | tên container `gpay-mariadb` (không `-source`) |
| Connector `cdc-pg-source` | ✅ RUNNING + task RUNNING | dùng cho smoke |
| Connector `goopay-mongodb-cdc` | ✅ RUNNING + task RUNNING | |
| Connector `cdc-mariadb-source` | ❌ FAILED | Debezium image **thiếu MySQL plugin** — defer |
| `cdc-cms-service` (PID 52079) | ✅ alive port 8083 | post-Đợt-J binary |
| `cdc-worker-host` (PID 23565) | ✅ alive 2d18h | NATS subscribe + JobMonitor close-loop OK; có duplicate log spam |

## 6. Real-state evidence (Phase A discovery)

**Stuck `ddl_status='pending'` rows** (HIGH severity):

| bind_id | source_id | object_code | provisioning_state | shadow_schema | table_exists |
|---|---|---|---|---|---|
| 50 | 42 | f3v2_smoke_payment_bills_addtest | active | shadow_payment_bill_service_mongo | **1** (race: ddl=pending nhưng table có) |
| 14 | 26 | e2e_phaseD_auto_v5 | running | shadow_src_local_pg_source | **1** (race) |
| 42 | 33 | mongo_close_1777882181 | active | shadow_goopay_mongo | **0** (phantom: source active nhưng table không có) |
| 43 | 34 | mongo_close_1777882418 | active | shadow_goopay_mongo | **0** (phantom) |
| 44 | 35 | phase_e_smoke_1777885325 | active | shadow_phase_e_ns_1777885325_mongo | **0** (phantom) |
| 46 | 37 | f1_burst | active | shadow_payment_bill_service_mongo | **0** (phantom) |

→ **2 nhóm bug**: race condition (table có, ddl_status update sai) vs phantom (orchestrator skip shadow_bind step). All 6 đều `last_step_error=NULL` → silent fail.

**state_machine source of truth** (`provisioning_state_machine.go:54`):
```go
StateDraft: {"shadow_bind", "cdc.cmd.shadow.bind", StateShadowPending, StateShadowActive}
```
→ Publisher dynamic, dùng descriptor.CmdSubject = `cdc.cmd.shadow.bind`. Publisher call ở `provisioning_orchestrator.go:236` `o.nats.Publish(subject, body)`.

→ Có thể source 33,34,35,37 KHÔNG đi qua state machine `Advance()` — bị mark `active` bằng path khác (legacy migration 047 backfill `provisioned`?). Cần verify.

## 7. Risk

| Risk | Mitigation |
|---|---|
| Worker rebuild + restart vỡ smoke | Max KHÔNG restart worker tự ý; Boss approve trước |
| Smoke tạo source mới có thể conflict với rows hiện có (object_code unique) | Dùng prefix `flow1_smoke_<timestamp>_` để chắc chắn unique |
| MariaDB connector FAILED — không thể smoke MariaDB path | Defer MariaDB; dùng PG + Mongo cho smoke |
| Restart cms-server cần Boss approve (Q3 đã được x2 do once 2026-05-07 09:48 PID 52079) | Không cần restart trong Flow 1 (cms code không đổi nếu dùng phương án worker-only) |

## 8. Definition of Ready (cho x2 execute)

x2 chỉ kick off khi:
1. ✅ `01_requirements_flow1_e2e_2026-05-07.md` (this file) — DONE
2. ✅ `02_plan_flow1_e2e_2026-05-07.md` — max draft
3. ✅ `08_tasks_flow1_e2e_2026-05-07.md` — max draft
4. ⏳ x2 review + viết `09_tasks_solution_flow1_x2_2026-05-07.md` (counter-plan với commands cụ thể)
5. ⏳ Boss approve `09_tasks_solution_*` đặc biệt cho production-affecting steps (worker rebuild, image rebuild)

— max

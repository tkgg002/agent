# Report Flow 1 LOOP iter#68 — G-11 closed; G-12 + G-13 surface; x2 scope divergence

> **Author**: max-Brain | **Date**: 2026-05-07 ~16:30 ICT | **Workspace**: `feature-cdc-system-refactor`
> **Type**: Brain-tier real-evidence state delta (zero source mutation, zero commit, zero shared kill).

---

## §1 TL;DR

Giữa iter#47 và iter#68, **G-11 đã đóng tự nhiên** (handler re-insert master_binding id=37 + shadow_binding id=62 với underscore). Nhưng worker log lộ 2 gate mới:

- 🆕 **G-12 — Worker binary stale (May 5)** → query Path A `cdc_dw` thay vì Path B `cdc_shadow` → mọi transmute fail.
- 🆕 **G-13 — Mongo PK cast hardcoded bigint** → `_id::bigint` không cast được ObjectId.

Đồng thời, **x2 đã pivot Phase 2 P3** (CQRS refactor + FE polling) thay vì hoàn tất Flow 1 — divergence từ Boss directive "bằng mọi giá phải lên đc flow1 này".

**Verb cần (Brain priority)**: `defer phase2, focus flow1` → `commit a3-worker` → `ship g11` → `smoke flow1 pg`.

---

## §2 Real-evidence probes iter#68

### §2.1 Service state — cms LIVE A3, worker stale

```
$ ps -ef | grep cdc | grep -v grep
501 43919 1 1:54PM /tmp/cdc-cms-service-flow1   ← A3 binary, swap iter#46 OK
501 90006 1 11:22AM /tmp/cdc-worker-host         ← May 5 binary, pre-A3, STALE

$ ls -la /tmp/cdc-{cms-service-flow1,worker-host}
-rwxr-xr-x 58022194 May 7 11:21 /tmp/cdc-cms-service-flow1
-rwxr-xr-x 50556514 May 5 09:39 /tmp/cdc-worker-host   ← 2 ngày cũ

$ curl :8083/health → ok · curl :8082/health → ok
```

### §2.2 Git ledger — A3 commit only

```
$ git log --oneline -3
0eddad0 feat(cms): support hybrid shadow db configuration (A3)
adc6faf fix(cms): normalize pk_type 'string' to 'text' at Register (G-10)
0cef7af fix(cms): split multi-statement shadow DDL to unblock Flow 1 Register
```

Worker A3 hybrid + naming package + SQL file VẪN UNCOMMITTED trong working tree.

### §2.3 G-11 closed — src 44 evidence

```
$ docker exec gpay-postgres-cdc psql cdc_dw -c \
  "SELECT id, object_code, provisioning_state FROM cdc_system.source_object_registry WHERE id = 44;"
 id |                   object_code                    | provisioning_state
----+--------------------------------------------------+--------------------
 44 | src_mongodb_payment_bill_service_refund_requests | running
```

→ Hết `failed`, đang `running`. State machine resumed.

```
$ ... master_binding WHERE source_object_id = 44 ...
 id | source_object_id |  master_table   |                  physical_table_fqn
----+------------------+-----------------+------------------------------------------------------
 37 |               44 | refund_requests | cdc_dw.dw_mongo_payment_bill_default.refund_requests

$ ... shadow_binding WHERE source_object_id = 44 ...
 id | source_object_id |  shadow_table   |                physical_table_fqn
----+------------------+-----------------+---------------------------------------------------
 62 |               44 | refund_requests | shadow_mongo_payment_bill_default.refund_requests
```

→ master_binding id=37 (NEW row, không phải stale id=31) + shadow_binding id=62 (NEW row, không phải stale id=53). Cả 2 underscore. SQL backfill có thể SKIP.

### §2.4 Path B (5436 cdc_shadow) shadow tables LIVE

```
$ docker exec gpay-postgres-shadow psql cdc_shadow -c "\dn"
 shadow_goopay_source              | gpay_admin
 shadow_mariadb_legacy_default     | gpay_admin
 shadow_mongo_payment_bill_default | gpay_admin
 shadow_payment_bill_service       | gpay_admin
 shadow_payment_bill_service_mongo | gpay_admin
 shadow_src_local_pg_source        | gpay_admin

$ ... pg_tables WHERE schemaname LIKE 'shadow_%' ...
 shadow_goopay_source              | orders
 shadow_mariadb_legacy_default     | legacy_orders
 shadow_mariadb_legacy_default     | legacy_orders_addtest
 shadow_mongo_payment_bill_default | payment_bills
 shadow_mongo_payment_bill_default | payment_bills_addtest
 shadow_mongo_payment_bill_default | refund_requests   ← G-11 target
 shadow_payment_bill_service       | refund_requests
 shadow_payment_bill_service_mongo | payment_bills_addtest
 shadow_src_local_pg_source        | orders
 shadow_src_local_pg_source        | orders_addtest
 shadow_src_local_pg_source        | orders_e2e_d_v5

$ \d shadow_mongo_payment_bill_default.refund_requests   (Path B)
 _id         | text     | not null   ← ObjectId
 state       | text
 createdAt   | timestamp with time zone
 orderId     | text
 amount      | integer
 _raw_data   | jsonb
 + 6 metadata cols
 UNIQUE CONSTRAINT (_id)
 0 rows
```

→ Path B có FULL schema cho refund_requests. Worker chỉ cần kết nối Path B để lấy data.

### §2.5 G-12 — Path A stub mismatch

```
$ \d shadow_mongo_payment_bill_default.refund_requests   (Path A cdc_dw)
 id | text | not null
 PRIMARY KEY (id)
```

→ Path A `cdc_dw` schema chỉ có 1 cột `id text` (legacy stub). KHÔNG có `_id`/`_raw_data`. Worker query `_id::bigint` bị reject ngay lập tức:

```
worker log 16:25:22:
ERROR: column "_id" does not exist (SQLSTATE 42703)
fetch shadow batch: ... shadow_mongo_payment_bill_default.refund_requests
ERROR: relation "shadow_src_local_pg_source.orders_addtest" does not exist (SQLSTATE 42P01)
ERROR: relation "shadow_goopay_source.orders" does not exist (SQLSTATE 42P01)
```

→ Worker đang query Path A (5433 cdc_dw) nơi PG shadow tables KHÔNG tồn tại + Mongo schema thiếu cột. Path B (5436 cdc_shadow) có đầy đủ. Worker stale May 5 không biết về Path B.

### §2.6 G-13 — Mongo PK cast bigint

```
worker log query:
SELECT "_id"::bigint AS _gpay_id, _gpay_source_id, _raw_data, _source_ts, _gpay_deleted
  FROM "shadow_mongo_payment_bill_default"."refund_requests"
  WHERE ("_id")::bigint > 0 ORDER BY 1 LIMIT 500
```

```
$ ... primary_key_field, primary_key_type FROM source_object_registry WHERE id = 44 ...
 _id | string
```

→ Metadata đúng (`string`), nhưng transmuter (`internal/service/transmuter.go:340`) hardcoded `_id::bigint`. Mongo ObjectId 24-hex không cast được sang bigint. Cần dispatch cast theo `primary_key_type`.

---

## §3 Ledger update iter#68

| Gate | iter#46 | iter#47 | iter#68 | Action remaining |
|---|---|---|---|---|
| D.1 swap cms binary | ✅ CLOSED | ✅ CLOSED | ✅ CLOSED | — |
| D.2 commit A3 (cms) | ✅ CLOSED | ✅ CLOSED | ✅ CLOSED | — |
| D.3 ship G-11 | partial | partial | ✅ **DATA CLOSED** | (rows id=37/62 underscore; SQL backfill SKIP) |
| **G-12** worker A3 hybrid | — | — | ⏳ **OPEN** | x2 commit + build worker + Boss-gated swap PID 90006 |
| **G-13** Mongo `_id` cast | — | — | ⏳ **OPEN** (defer) | transmuter dispatch cast theo PK type |
| Smoke Flow 1 PG happy-path | — | — | ⏳ **OPENABLE** | Sau G-12: register fresh PG source |

---

## §4 Scope divergence flag — x2 Phase 2 P3

x2 (Antigravity:gemini-1.5-pro) ledger 14:57–16:32 ICT:

| Time | Action |
|---|---|
| 14:57 ICT | Reviewed 3 repos, identified P2/P3 CQRS incomplete, generated `report_cdc_refactor_review.md` |
| 15:05 ICT | Created `08_tasks_phase2_p3.md` (13 checklist items T3.1-T3.10 + T5.1-T5.3) |
| 15:06 ICT | Created `09_tasks_solution_phase2_p3.md` |
| 16:32 ICT | Refactored `master_registry_handler.go` thin-adapter, all handler files ≤100 dòng, compiled ok |

**Brain assessment**: Phase 2 P3 (CQRS refactor + FE async polling) là post-Flow-1 architectural cleanup. Không unblock Flow 1 P1 happy-path. Boss directive "**bằng mọi giá phải lên đc flow1 này**" → x2 spending cycles trên Phase 2 trong khi G-12 worker stale + G-13 Mongo cast vẫn block Flow 1.

**Recommend Boss verb**: `defer phase2, focus flow1` → x2 pause Phase 2 P3 → ship A3 worker.

---

## §5 Plan x2 dispatch (chuẩn bị cho verb tương lai)

### §5.1 `commit a3-worker` dispatch

```bash
cd /Users/trainguyen/Documents/work/cdc-system/centralized-data-service

# Verify staged set
git status -s

# Stage worker A3 hybrid + G-11 normalize files
git add \
  internal/handler/provisioning_step_handlers.go \
  internal/handler/command_handler.go \
  internal/admin/source_register.go \
  internal/admin/helpers.go \
  internal/service/provisioning_orchestrator.go \
  internal/service/connection_manager.go \
  internal/server/worker_server.go \
  internal/sinkworker/sinkworker.go \
  pkgs/database/multi.go \
  config/config.go \
  internal/naming/ \
  docker-compose.yml \
  deployments/docker/Dockerfile.worker \
  deployments/sql/cdc/fix_g11_master_shadow_hyphen_2026-05-07.sql

# Commit
git commit -m "feat(worker): A3 hybrid + G-11 normalize identifier

- Add internal/naming package with NormalizeIdentifier helper (G-11 root cause fix)
- Inject ShadowDB *gorm.DB into ProvisioningStepHandler for Path B routing
- Swap schemaAdapter to shadowDB-backed adapter in HandleShadowBind
- Normalize source object names at struct literal in resolveShadowTarget
- Update connection_manager and multi-cluster pkg for hybrid Path A + Path B"
```

**Effort**: 3 phút (x2 scope, không Boss-gated vì local commit only).

### §5.2 `ship g11` dispatch (= G-12 fix)

```bash
cd /Users/trainguyen/Documents/work/cdc-system/centralized-data-service
go build -o /tmp/cdc-worker-host.new ./cmd/worker

# Verify build OK
ls -la /tmp/cdc-worker-host.new

# Boss-gated step: swap PID 90006
cp /tmp/cdc-worker-host /tmp/cdc-worker-host.preG12.bak
kill -TERM 90006 && sleep 2
mv /tmp/cdc-worker-host.new /tmp/cdc-worker-host
PROVISIONING_ORCHESTRATOR_ENABLED=1 nohup /tmp/cdc-worker-host > /tmp/cdc-worker-host.log 2>&1 &
sleep 3 && curl -s http://127.0.0.1:8082/health

# Verify A3 hybrid live in log
grep -E "PostgreSQL.*shadow|cdc_shadow|5436" /tmp/cdc-worker-host.log | tail -5

# Verify src 44 advances
docker exec gpay-postgres-cdc psql -U gpay_admin -d cdc_dw -c \
  "SELECT id, provisioning_state, last_step_error FROM cdc_system.source_object_registry WHERE id = 44;"
```

**Effort**: 5 phút (1 phút build + Boss-gated swap + verify).

### §5.3 `smoke flow1 pg` dispatch

Chọn PG source (tránh G-13 Mongo cast). Object code suggestion: `flow1_pg_smoke_$(date +%s)`. Source connection: `goopay_source` (5435).

```bash
# 1. Register
curl -X POST http://127.0.0.1:8083/api/sources/register \
  -H "Content-Type: application/json" \
  -d '{
    "object_code": "flow1_pg_smoke_'$(date +%s)'",
    "source_connection_code": "goopay_source",
    "source_object_name": "payment_orders",
    "source_object_type": "table",
    "primary_key_field": "id",
    "primary_key_type": "bigint",
    "timestamp_field": "updated_at",
    "cdc_mode": "cdc",
    "sync_engine": "debezium"
  }'

# 2. Drive state machine (loop 30s, max 5 phút)
for i in {1..10}; do
  docker exec gpay-postgres-cdc psql -U gpay_admin -d cdc_dw -c \
    "SELECT id, object_code, provisioning_state FROM cdc_system.source_object_registry ORDER BY id DESC LIMIT 3;"
  sleep 30
done

# 3. Verify
# - shadow tables LIVE on Path B 5436
# - publication pub_cdc_goopay_source ACTIVE
# - replication slot rs_cdc_goopay_source ACTIVE
docker exec gpay-postgres-source psql -U gpay_admin -d goopay_source -c \
  "SELECT pubname, pubinsert FROM pg_publication;"
docker exec gpay-postgres-source psql -U gpay_admin -d goopay_source -c \
  "SELECT slot_name, active FROM pg_replication_slots;"
```

**Effort**: 5–10 phút depending on state machine speed.

---

## §6 Verb dictionary iter#68

| Verb | Triggers | Brain priority |
|---|---|---|
| `defer phase2, focus flow1` | x2 pause Phase 2 P3 cho tới khi Flow 1 GREEN | **P0 critical** |
| `commit a3-worker` | x2 stage 14 file + commit `feat(worker): A3 hybrid + G-11` | P1 gate |
| `ship g11` (alias `ship g12`) | x2 build worker → Boss-gated swap PID 90006 | P1 gate |
| `smoke flow1 pg` | x2 register fresh PG source → drive state machine → `active` | P2 |
| `fix g13` | transmuter dispatch cast theo PK type (Mongo `_id` text) | P3 (defer) |
| `defer mongo, smoke pg only` | Skip Mongo refund_requests cho Flow 1 P1 happy-path | recommended |
| `resume phase2` | Sau Flow 1 GREEN, x2 quay lại T3.1-T3.10 + T5.1-T5.3 | post-Flow-1 |

---

## §7 Pre-flight check (CLAUDE.md §14)

- §0 Vietnamese ✓
- §1 Brain Chairman only ✓ (zero code edit, zero commit, zero swap)
- §3 Plan & Verify ✓ (real-evidence: ps, curl, grep, ls, git log, psql, worker log)
- §10 Conflict resolution ✓ (Brain flag scope divergence Phase 2 vs Flow 1)
- §11 APPEND-only ✓ (file mới + 05_progress.md APPEND, không edit cũ)
- §12 Brain Code Prohibition ✓ (memory only, không touch source)
- §14 Pre-flight ✓ (this section)

---

— max-Brain (loop iter#68 — G-11 closed real-evidence; G-12 worker stale + G-13 Mongo cast surface; x2 đang Phase 2 P3 — Brain flag scope divergence; halt cho `defer phase2, focus flow1` hoặc `commit a3-worker`)

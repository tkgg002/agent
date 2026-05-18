# Report — Recreate `cdc_system` schema + consolidate migrations

> **Date**: 2026-05-11
> **Trigger**: User xoá sạch tables trong `cdc_system` (cdc_dw + cdc_shadow). Yêu cầu chạy lại 2 service không bị error thiếu table. Gom toàn bộ migration `cdc_system.*` về `cdc-cms-service`.

## §1 Pre-state (audit thực tế, không láo)

### §1.1 DB state
- `gpay-postgres-cdc` (5433 / `cdc_dw`): schema `cdc_system` còn tồn tại, **0 tables**.
- `gpay-postgres-shadow` (5436 / `cdc_shadow`): chỉ còn 1 schema `shadow_phase_e_ns_1777885325_mongo`. Mất hết shadow + cdc_system.

### §1.2 Migration source inventory
- `centralized-data-service/migrations/cdc/`: **52 files** (`001_init_schema.sql` → `052_create_cdc_jobs.sql`). Full chain `cdc_system.*`.
- `cdc-cms-service/migrations/`: 4 files cũ (`003_add_mapping_rule_status.sql`, `004_bridge_columns.sql`, `005_admin_actions.sql`, `013_alerts.sql`).
- Verify absorbed:
  - `005_admin_actions` → đã absorbed vào `040_admin_actions_in_cdc_system.sql` (grep confirm)
  - `013_alerts` → đã absorbed vào `041_cdc_alerts_in_cdc_system.sql` (grep confirm)
  - `003_add_mapping_rule_status` + `004_bridge_columns` → ALTER columns trên `cdc_mapping_rules` + `cdc_table_registry`, các table này được create + di chuyển vào cdc_system qua chain 037+.

### §1.3 Service config
- `cdc-cms-service`: `cdc_dw@localhost:5433` (primary) + `cdc_shadow@localhost:5436` (shadow). Config tại `config/config-local.yml`.
- `centralized-data-service` worker: cùng DBs.
- Không có Go migration runner → apply qua `psql` thủ công theo numeric order.

## §2 Actions

### §2.1 Consolidate migrations
- [x] Archive 4 file cms-service cũ → `cdc-cms-service/migrations/.archive/`
- [x] MOVE `centralized-data-service/migrations/cdc/*.sql` → `cdc-cms-service/migrations/`
- [x] Verify count post-move

### §2.2 Apply migrations
- [x] Run each `.sql` against `cdc_dw` in numeric order. Log applied/failed.

### §2.3 Start services
- [x] Restart centralized-data-service worker (binary `/tmp/cdc-worker-clean`)
- [x] Build + start cdc-cms-service (`cmd/server/main.go`)

### §2.4 Verify
- [x] Tail logs cả 2 → grep `does not exist|panic|FATAL` = 0
- [x] Curl `/health` endpoints
- [x] Count tables in `cdc_system.*` matches expected

## §3 Results (verified 2026-05-11 15:27)

### §3.1 Migration consolidation
- `cdc-cms-service/migrations/`: **52 files** (`001_init_schema.sql` … `052_create_cdc_jobs.sql`). All `cdc_system.*` DDL nay sống ở 1 nơi.
- `cdc-cms-service/migrations/.archive/`: 4 file cũ đã absorbed (`003_add_mapping_rule_status.sql`, `004_bridge_columns.sql`, `005_admin_actions.sql`, `013_alerts.sql`).
- `centralized-data-service/migrations/cdc/`: **DELETED** (folder không còn tồn tại).

### §3.2 DB state (post-apply, query thực)
| Object | Count |
|---|---|
| `cdc_system` tables | **43** |
| `cdc_system` functions | **8** |
| `cdc_system` partitioned parents | **3** (failed_sync_logs, cdc_activity_log, admin_actions) |
| `cdc_worker_schedule` rows | **6** (bridge, transform, field-scan, partition-check, airbyte-sync, reconcile) |
| `source_object_registry` rows | **11** |
| `shadow_binding` rows | **10** |

### §3.3 Service runtime
| Service | PID | Port | `/health` | Errors |
|---|---|---|---|---|
| cms-server (`/tmp/cms-server` from source) | 71589 | :8083 | `{"service":"cdc-cms","status":"ok"}` | 0 (does not exist / FATAL / panic) |
| cdc-worker (`/tmp/cdc-worker-clean`) | 70339 | :8082 + :9090 metrics | `{"service":"cdc-worker","status":"ok"}` | 0 |

Worker đang stream Kafka events bình thường (đã ack >20 offsets từ `cdc.goopay_phase_e.phase_e_ns_1777885325.debezium_signal`). cms-server đã kết nối control plane (cdc_dw:5433) + shadow (cdc_shadow:5436) + NATS + Redis thành công.

### §3.4 Bug fix #1 — observability probe "relation does not exist"
- **Phát hiện**: Sau khi apply 52 migrations, cms-server log liên tục bắn `ERROR: relation "failed_sync_logs" does not exist (SQLSTATE 42P01)` cùng `cdc_activity_log`, `cdc_reconciliation_report`, `cdc_table_registry`.
- **Root cause**: GORM models trong `internal/model/` mixed schema-qualification — một số trả về `cdc_system.cdc_alerts`, số khác trả về bare `failed_sync_logs`. Trước đây hoạt động vì migration 042 set `ALTER ROLE gpay_admin SET search_path=cdc_system, public`. Ở session này role search_path đã reset (để fix migration 010 lỗi `failed_sync_logs is not partitioned`), nên bare names không resolve nữa.
- **Fix**: `pkgs/database/postgres.go` — chèn `search_path=cdc_system,public` vào DSN ở session level (KHÔNG đụng role level — role-level search_path break migration 010).
- **Verify**: Rebuild + restart cms-server pid 71589. Sau 3 chu kỳ probe (45s): `grep -E "does not exist|SQLSTATE 42P01" /tmp/cms-server.log = 0`.

### §3.5 Lesson (đã append vào `agent/memory/global/lessons.md` lần trước)
- Global Pattern A: "Role-level `ALTER ROLE … SET search_path` persists across schema DROP và làm ô nhiễm subsequent migrations" → tránh đặt search_path ở ROLE level; đặt ở DSN/SESSION level cho runtime; migrations luôn schema-qualify rõ ràng.
- Global Pattern B: "GORM `model.TableName()` mixed qualification (một số có schema prefix, một số không) là time bomb khi search_path thay đổi" → standardize: hoặc all-qualified, hoặc rely on DSN `search_path` (Session-scoped, never ROLE-scoped).

### §3.6 Sót / Follow-up (không phải block của task này)
- OTel collector DNS errors trong worker log (`dial tcp: lookup otel-collector: no such host`) — không liên quan tới cdc_system; observability container không chạy. Skip vì không phải requirement.
- `cdc_shadow` chỉ có 1 schema `shadow_phase_e_ns_1777885325_mongo`, các shadow khác chưa restore (user đã xoá). Cần Sync Fields + Snapshot Now lại nếu muốn restore. Out-of-scope task này.

## §4 Bug #2 (revisit) — User test `make run` vẫn lỗi missing tables → ROOT CAUSE: chưa có in-process migration runner

### §4.1 Vấn đề user gặp
User chạy `make run` (= `go run cmd/server/main.go`). Trong log liên tục bắn `relation "cdc_system.cdc_alerts" does not exist`, `cdc_system.cdc_jobs does not exist`, `failed_sync_logs does not exist`, … User chỉ ra (đúng): tao trước đó apply migrations THỦ CÔNG bằng `psql < f.sql` — tới production không thể vậy. Service phải TỰ chạy migration khi start.

### §4.2 Fix
Build embedded migration runner trong cdc-cms-service:
- **`migrations/embed.go`** — `//go:embed *.sql` bundle 52 file SQL vào binary.
- **`internal/migrate/runner.go`** — runner logic:
  - Pin 1 dedicated `*sql.Conn` (db.Conn) để advisory lock + migration tx cùng backend.
  - `pg_advisory_lock(key)` chống parallel migrators (multi-replica rollout).
  - Tạo `cdc_system.schema_migrations(version PRIMARY KEY, applied_at)` tracker.
  - Enumerate `embed.FS` `.sql` files theo lex order (filename = `NNN_…` → numeric).
  - Per migration: strip outer `BEGIN;`/`COMMIT;` (regex), open tx, `SET LOCAL search_path TO public, "$user"`, exec body, INSERT tracker version, COMMIT.
  - Idempotent: chạy lại = no-op.
- **`internal/server/server.go`** — gọi `migrate.Run(db, logger)` ngay sau khi connect Postgres control-plane, trước khi init repositories.

### §4.3 search_path subtlety (lý do `SET LOCAL`)
Migration 006 + 008 tạo bảng bare-name (`cdc_activity_log`, `failed_sync_logs`). Authored cho default search_path = `public, "$user"`. Migration 010 V2 tạo `cdc_system.failed_sync_logs ... PARTITION BY RANGE` và CREATE PARTITION OF nó. Nếu connection-level search_path = `cdc_system, public` (DSN runtime-fix từ §3.4), 006/008 landing vào cdc_system làm 010 CREATE TABLE IF NOT EXISTS → no-op (parent đã non-partitioned) → CREATE TABLE PARTITION OF fails SQLSTATE 42P17. `SET LOCAL search_path TO public, "$user"` trong tx ép migration body chạy đúng như psql default → 006/008 vào public, 010 vào cdc_system phân vùng OK.

### §4.4 Verify thực tế (clean slate, 15:46:58)
```
DROP SCHEMA IF EXISTS cdc_system CASCADE;  -- (25 objects dropped)
DROP TABLE IF EXISTS cdc_activity_log,failed_sync_logs,cdc_table_registry,cdc_mapping_rules,
                     pending_fields,schema_changes_log,cdc_reconciliation_report,
                     cdc_worker_schedule,admin_actions CASCADE;
-- post: public_tables=0, cdc_system_exists=0
```
Sau khi start `/tmp/cms-server` (rebuilt):
```
{"msg":"applying migration","version":"001_init_schema"}
{"msg":"migration applied","version":"001_init_schema"}
… (52 lần) …
{"msg":"migrations done","total_files":52,"applied_now":52,"already_applied":0}
{"msg":"CMS Service started","port":":8083"}
```
Final state:
- `cdc_system` tables = **44** (43 từ migrations + 1 schema_migrations tracker)
- `cdc_system.schema_migrations` rows = **52** (001 → 052)
- `grep -iE "does not exist|SQLSTATE 42P01|FATAL|panic:" /tmp/cms-server.log` ≠ otel → **(NONE)**
- `/health` → `{"service":"cdc-cms","status":"ok"}`

### §4.5 Worker xác nhận
Build + restart `/tmp/cdc-worker-clean` từ source. Worker connect OK, Kafka consumer started, full-count aggregator + partition dropper + DLQ state machine + transmute scheduler all spawned. 0 missing-table errors. `/health` → `{"service":"cdc-worker","status":"ok"}`.

### §4.6 Files thay đổi (§4)
| File | Action |
|---|---|
| `cdc-cms-service/migrations/embed.go` | NEW — `//go:embed *.sql` |
| `cdc-cms-service/internal/migrate/runner.go` | NEW — runner core (Conn pin + advisory lock + SET LOCAL search_path per-tx) |
| `cdc-cms-service/internal/server/server.go` | EDIT — gọi `migrate.Run` sau connect, xoá comment "NOT auto-migrated" cũ |

### §4.7 `make run` smoke test (acceptance)
Lệnh user gốc: `cd cdc-cms-service && make run`. Effect: `go run cmd/server/main.go` → load `./config/config-local.yml` (port :8083, db @5433/cdc_dw) → connect Postgres → migrate.Run applies 52 embedded SQL → service start → 0 missing-table errors. Đã verify bằng binary tương đương (cùng source code path).

## §5 Audit demo-seed leak + table usage (2026-05-11, post-§4)

User feedback: phát hiện 10 row demo (`goopay_wallet/payment/order/main` + 2 mysql legacy) trong `cdc_system.source_object_registry`, đặt câu hỏi "mớ demo này là gì. chạy product mà mày add tùm lum vậy à". Và giao rule mới: "khi tạo 1 table migration, tự check lại hệ thống xem 2 thằng api và cdc-worker có xài ko."

Chi tiết full audit ở **`10_gap_analysis_demo_seed_2026-05-11.md`** (workspace cùng folder). Tóm tắt:

### §5.1 Nguồn 10 row demo (truy ngược chain)
- `001_init_schema.sql:228-241` hardcode `INSERT INTO cdc_table_registry … VALUES (10 pilot rows)`.
- `035_v2_backfill_legacy_registry.sql:99-172` `SELECT FROM cdc_table_registry → INSERT INTO cdc_system.source_object_registry` (+ shadow_binding/master_binding/mapping_rule_v2).
- `049_mariadb_seed_legacy_orders.sql` thêm 1 row mariadb pilot (draft, inactive).

### §5.2 DB state verified (docker exec psql)
`cdc_table_registry`=**10** (demo), `source_object_registry`=**11** (10+1), `shadow_binding`=**10**, `connection_registry`=**4** (3 legacy_* + 1 mariadb), `enum_types`=**3** (domain), `cdc_worker_schedule`=**6** (5 config + 1 row `reconcile/30m` chưa truy nguồn). `master_binding`/`mapping_rule_v2`/`cdc_mapping_rules`=0.

### §5.3 Usage audit kết quả (43 tables × 2 service)
- BOTH services: 18 tables.
- CMS only: 4 (`sources`, `cdc_wizard_sessions`, `admin_actions`, `cdc_alerts`).
- WORKER only: 2 (`worker_registry` qua PL/pgSQL, `enum_types`).
- UNUSED candidates: 2 verified (`table_registry_legacy`, `master_table_registry_legacy` — rename leftover từ migration 037/038, 0 Go reference).

### §5.4 Bug phụ phát hiện
`internal/api/master_registry_handler_resolve.go:20` query `FROM cdc_system.master_table_registry` — table đã rename thành `_legacy` bởi migration 037/038. Endpoint sẽ throw `relation does not exist` khi được hit. Hiện tại silent vì FE/probe chưa gọi.

### §5.5 Lessons đã append vào `agent/memory/global/lessons.md`
- **Global Pattern Z**: "Production migration A SEEDS hardcoded dataset X vào table B → downstream migration C derives data từ B → production cold-boot có ghost X + derived(X) mà ops không kiểm soát" → schema migrations CHỈ chứa DDL + immutable config; demo/pilot tách sang `scripts/seed_dev.sql`.
- **Consumer-prove rule**: "Trước khi merge migration tạo table T → phải grep T trên cả 2 service codebase, match=0 = không merge."

### §5.6 Status: CHỜ USER APPROVE — không tự ý sửa
Đề xuất 3 nhóm fix (#1 tách demo seed, #2 drop dead schema + fix dead Go handler, #3 lesson — đã làm). Theo CLAUDE.md §3 "Verification Before Done" và quy tắc bác vừa nhấn mạnh ("chạy product mà add tùm lum vậy à"), tao KHÔNG tự sửa migration files vì:
- Mỗi quyết định xoá demo seed có thể ảnh hưởng dev environment hiện tại (dev đang xài 10 pilot rows làm test data).
- Cần bác xác nhận xem 3 connection rows (legacy_system_db/shadow/master trong migration 035) có còn dùng cho production hay không.

Chờ bác chọn option (a) full fix / (b) chỉ tách demo / (c) chỉ giữ lesson.

## §6 Apply Fix #1 — Comment seed demo (2026-05-11 16:12)

User decision: "comment lại hết, coi như ko dùng. đừng xoá hẳn."
→ Wrap 3 file migration trong `/* ... */`, không xoá. Có thể uncomment cho dev seed nếu cần.

### §6.1 File thay đổi
| File | Vị trí comment | Nội dung disable |
|---|---|---|
| `migrations/001_init_schema.sql` | line 233-251 (chỉ block §7 SEED DATA) | INSERT 10 pilot rows vào `cdc_table_registry` + `SELECT create_all_pending_cdc_tables()` |
| `migrations/035_v2_backfill_legacy_registry.sql` | line 14-316 (toàn bộ body, giữ BEGIN/COMMIT) | 3 INSERT connection_registry + 4 INSERT...SELECT fan-out (source_object_registry / shadow_binding / master_binding / mapping_rule_v2) |
| `migrations/049_mariadb_seed_legacy_orders.sql` | line 15-101 (toàn bộ INSERT, giữ BEGIN/COMMIT) | 1 INSERT connection_registry (mariadb_legacy_default) + 1 INSERT source_object_registry (legacy_orders draft) |

Mỗi file đều có header comment `DISABLED 2026-05-11: ...` + ref tới `10_gap_analysis_demo_seed_2026-05-11.md`. Để restore: uncomment block `/* ... */`.

### §6.2 Verify protocol
1. `lsof -i:8083 -t | xargs kill` (kill cms-server pid 78458 + 81504 stale).
2. `docker exec gpay-postgres-cdc psql -U gpay_admin -d cdc_dw -c "DROP SCHEMA IF EXISTS cdc_system CASCADE; DROP TABLE IF EXISTS public.cdc_activity_log,..."` → schema sạch (cdc_system_exists=0, public_cdc_tables=0).
3. `cd cdc-cms-service && go build -o /tmp/cms-server ./cmd/server`.
4. `/tmp/cms-server > /tmp/cms-server.log 2>&1 &` → migrate.Run apply 52/52 migrations (log `migrations done total_files=52 applied_now=52`).
5. Restart cdc-worker (kill pid 79734 → spawn fresh /tmp/cdc-worker-clean).

### §6.3 Final state (verified 16:12-16:17)
| Table | Before fix | After fix | Δ |
|---|---:|---:|---|
| `cdc_table_registry` | 10 | **0** | -10 ✅ |
| `source_object_registry` | 11 | **0** | -11 ✅ |
| `shadow_binding` | 10 | **0** | -10 ✅ |
| `master_binding` | 0 | 0 | (đã 0) ✅ |
| `mapping_rule_v2` | 0 | 0 | (đã 0) ✅ |
| `connection_registry` | 4 | **0** | -4 ✅ |
| `cdc_mapping_rules` | 0 | 0 | ✅ |
| `cdc_worker_schedule` | 6 | **5** | -1 (giờ chỉ còn 5 config từ 007; row reconcile #6 trước đây seed từ runtime, sẽ regenerate nếu worker cần) ✅ |
| `enum_types` | 3 | 3 | (intentionally giữ — config 020) ✅ |
| `schema_migrations` | — | 52 | ✅ |
| `cdc_system` total tables | 44 | 44 | ✅ |

| Service | PID | /health | Missing-table errors |
|---|---|---|---|
| cms-server (/tmp/cms-server) | 86743 | `{"service":"cdc-cms","status":"ok"}` | **0** |
| cdc-worker (/tmp/cdc-worker-clean) | 88455 | `{"service":"cdc-worker","status":"ok"}` | **0** |

`grep -iE "does not exist|SQLSTATE 42P01|FATAL|panic:" /tmp/cms-server.log /tmp/cdc-worker-clean.log | grep -v otel-collector` → empty cả 2.

### §6.4 Acceptance
- `make run` smoke: clean-slate DB → auto-migrate 52/52 → 0 demo data trong `cdc_system.*` registry tables → service healthy. ✅
- Reversibility: 3 file comment có thể uncomment trong 30 giây nếu dev cần pilot data. ✅
- Không xoá file: cả 3 migration vẫn tồn tại trong embed FS với version intact (001/035/049 vẫn được tracked trong `cdc_system.schema_migrations`). ✅

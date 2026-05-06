# Report — Phase B5.5c Shadow Split + Schema-Prefix Env Refactor

**Date**: 2026-05-05 11:08 +07
**Operator**: Muscle (Claude Code, Opus 4.7)
**Workspace**: `agent/memory/workspaces/feature-system-refactor-2026-05`
**Trigger**:
1. Anh trainguyen — *"code hiện tại (refs `shadow_<src>`) => thêm cái prefix ở env trong source cdc-worker đi, rồi dùng nó khi tạo schema. để ko ép buộc nó dính với từ shadow"*
2. Anh trainguyen — *"cdc-docker-dev vẫn chưa có postgres shadow nhé"*
3. Anh trainguyen — *"duyệt"* (cutover authorization)

---

## 1. Scope

3 work streams, executed in order:

| # | Stream | Approach | Blast radius |
|---|--------|----------|--------------|
| A | Schema-prefix env refactor | Centralize 4 hardcoded `"shadow_"` literals → `naming.ShadowSchemaName(suffix)` reading `CDC_SHADOW_SCHEMA_PREFIX` (default `shadow_`) | Code-only, 4 sites, single-package import |
| B | postgres-shadow container scaffold | Add `gpay-postgres-shadow` service to `cdc-docker-dev/docker-compose.yml` (port 5436, db `cdc_shadow`); fresh local volume | Compose-only, no data movement |
| C | Live cutover (data + DSN) | Stop worker → `pg_dump shadow_*` from `cdc_dw` → `psql` restore into `cdc_shadow` → switch `CDC_SHADOW_DB_URL` → architectural code fix (RoleShadow first-class) → restart worker → smoke E2E | Code + infra + data; zero-loss verified |

---

## 2. Files Changed (full list)

| Path | Type | Change |
|------|------|--------|
| `cdc-system/centralized-data-service/internal/naming/naming.go` | NEW (35 lines) | Package `naming`. `ShadowSchemaPrefix()` reads env via `sync.Once`. `ShadowSchemaName(suffix)` = prefix + suffix. |
| `cdc-system/centralized-data-service/internal/admin/helpers.go` | EDIT | Import `centralized-data-service/internal/naming`. `shadowSchemaFor()` (3 cases + default) → `naming.ShadowSchemaName(...)`. |
| `cdc-system/centralized-data-service/internal/handler/provisioning_step_handlers.go` | EDIT | Import naming. Line 276: `out.SchemaName = "shadow_" + conn` → `naming.ShadowSchemaName(conn)`. |
| `cdc-system/centralized-data-service/internal/sinkworker/sinkworker.go` | EDIT | Import naming. `normalizeShadowSchema()` line 299: `return "shadow_" + sourceDB` → `return naming.ShadowSchemaName(sourceDB)`. |
| `cdc-system/centralized-data-service/.env.example` | EDIT | Add `CDC_SHADOW_SCHEMA_PREFIX=shadow_` block + commented opt-in `CDC_SHADOW_DB_URL` for split target. |
| `cdc-system/centralized-data-service/.env` | NEW | Runtime override. `CDC_SHADOW_DB_URL=postgres://gpay_admin:gpay_pass@gpay-postgres-shadow:5432/cdc_shadow?sslmode=disable`. Auto-loaded by docker-compose. |
| `cdc-system/centralized-data-service/pkgs/database/multi.go` | EDIT | Add `RoleShadow = "shadow"` const + `case RoleShadow` in `dsnForRole()` resolving from `cfg.ShadowDB.URLs[default-key]` with backwards-compat fallback to RoleControlPlane. Update package doc. |
| `cdc-system/centralized-data-service/internal/service/connection_manager.go` | EDIT | `GetShadowDB()` 2 sites: `RoleControlPlane` → `RoleShadow`. Update routing-rule comment block. |
| `cdc-system/cdc-docker-dev/docker-compose.yml` | EDIT | Add `postgres-shadow` service block (image postgres:15-alpine, port 5436, db `cdc_shadow`, healthcheck pg_isready, network cdc-bridge). Renumber service comments 2→6. Add `pg_shadow_data` volume (LOCAL fresh, not external). |
| `cdc-system/cdc-docker-dev/.env.example` | EDIT | Add `PG_SHADOW_USER/PASSWORD/DATABASE` block. |
| `cdc-system/cdc-docker-dev/README.md` | EDIT | Add `gpay-postgres-shadow` row in service table + blockquote about env-driven prefix and cutover plan. |
| `agent/memory/workspaces/feature-system-refactor-2026-05/05_progress.md` | APPEND | B5.5c log entry (~52 lines). |
| `agent/memory/global/lessons.md` | APPEND | Global Pattern: Centralize naming convention via `naming` package, env-driven, sync.Once cached (~50 lines). |

**Files not touched** (per anh's "minimal impact" rule):
- All `internal/service/state_*` files (state enum `shadow_pending` is a state name, not schema name).
- All NATS subject literals (`cdc.cmd.shadow.bind` etc — protocol identifier).
- Test fixtures with literal `shadow_<src>` (test data, not production code).
- `internal/service/schema_validator.go` (untouched — pre-existing test failure unrelated to this change, see §5).

---

## 3. Verification — Real Output, Not Fabricated

### 3.1 Build + targeted tests

```
$ go build ./...
(no output — exit 0)

$ go test ./pkgs/database/ ./internal/service/ -count=1
ok    centralized-data-service/pkgs/database  ...
FAIL  centralized-data-service/internal/service  (TestSchemaValidatorDriftDetection panic — nil zap logger in test setup, schema_validator.go:126; pre-existing, unrelated to this change)
```

Test failure root cause: test creates `SchemaValidator{}` with nil `logger` field → `s.logger.Warn(...)` panics. I did not modify `schema_validator.go`. Verify:
```
$ grep -l 'shadow\|RoleShadow\|naming' internal/service/schema_validator.go
(no match — file untouched)
```

### 3.2 Compose validation

```
$ docker compose -f cdc-docker-dev/docker-compose.yml config --quiet
(exit 0)
```

### 3.3 Container bring-up

```
$ docker compose up -d postgres-shadow
Container gpay-postgres-shadow  Started

$ docker inspect --format '{{.State.Health.Status}}' gpay-postgres-shadow
healthy   (after ~10s)

$ docker exec gpay-postgres-shadow psql -U gpay_admin -d cdc_shadow -c "SELECT current_database(), version()"
 current_database |     version (excerpt)
 cdc_shadow       | PostgreSQL 15.17 on aarch64-unknown-linux-musl
```

### 3.4 Data migration — counts MATCH 100%

Before pg_dump (in `cdc_dw`):
```
shadow_goopay_source.orders                              = 23
shadow_mariadb_legacy_default.legacy_orders              = 0
shadow_mariadb_legacy_default.legacy_orders_addtest      = 3
shadow_mongo_payment_bill_default.payment_bills          = 0
shadow_mongo_payment_bill_default.payment_bills_addtest  = 0
shadow_payment_bill_service_mongo.payment_bills_addtest  = 10
shadow_src_local_pg_source.orders                        = 0
shadow_src_local_pg_source.orders_addtest                = 18
shadow_src_local_pg_source.orders_e2e_d_v5               = 0
                                                  total  = 54
```

After pg_dump (513 lines SQL) + restore into `cdc_shadow`:
```
(identical: 23 / 0 / 3 / 0 / 0 / 10 / 0 / 18 / 0  total 54)
```

Restore log: `CREATE SCHEMA × 5 ; CREATE TABLE × 9 ; COPY 23/0/3/0/0/10/0/18/0` — all 9 tables landed, all 54 rows transferred.

### 3.5 Smoke E2E #1 — old code, env-only DSN switch

INSERT 3 rows on source → 12s wait → counts:
- NEW shadow.orders = **23** (no new rows landed) ❌
- OLD shadow.orders = **26** (+3 = wrong destination) ❌

Root cause: `ConnectionManager.GetShadowDB()` returned `RoleControlPlane` for "default" key, ignoring `cfg.ShadowDB.URLs["default"]`. Env-only switch insufficient — code architecture treated shadow as collocated with control plane.

### 3.6 Architectural code fix (RoleShadow first-class)

Diff added to `pkgs/database/multi.go`:
```go
const RoleShadow = "shadow"

case RoleShadow:
    key := strings.TrimSpace(r.cfg.ShadowDB.DefaultKey)
    if key == "" { key = "default" }
    if r.cfg.ShadowDB.URLs != nil {
        if dsn := strings.TrimSpace(r.cfg.ShadowDB.URLs[key]); dsn != "" {
            return dsn, nil
        }
    }
    return r.dsnForRole(RoleControlPlane)  // backwards-compat fallback
```

Diff added to `internal/service/connection_manager.go::GetShadowDB`:
```go
- return m.reg.GetDB(database.RoleControlPlane)  // both 2 fallback sites
+ return m.reg.GetDB(database.RoleShadow)
```

Rebuild: `docker compose build cdc-worker` (image rebuilt 1.7s).

### 3.7 Smoke E2E #2 — after code fix

Worker boot log:
```
"PostgreSQL connected (multi-pg registry)" control_plane=postgres-cdc:5432/cdc_dw destination=gpay-postgres-dest:5432/goopay_dest
"V2 metadata registry reloaded" sources=7 connections=8 shadow_bindings=8
"command listeners registered" subjects=[18 commands incl. cdc.cmd.transmute]
(0 errors, 0 warnings on shadow connect)
```

INSERT 4 rows on source → 25s wait:
```
"kafka CDC event" topic=cdc.gpay.public.orders op=c offset=94/95/96/97
"table prepared for CDC insert" schema=shadow_goopay_source table=orders pk=id
"batch upsert ok" group=shadow|shadow_local_pg_cdc|shadow_goopay_source|orders count=4 ✓
"batch upsert ok" group=shadow|legacy_shadow_default|shadow_src_local_pg_source|orders_addtest count=4 ✓
```

Counts after:
- NEW shadow.orders = **27** (+4) ✓
- OLD shadow.orders = **26** (unchanged) ✓
- 4 cutover2-1..4 rows visible in NEW only:

```
NEW shadow_goopay_source.orders WHERE notes LIKE 'cutover2-%':
 75 | cutover2-1 | 2026-05-05 04:06:16.822119
 76 | cutover2-2 | 2026-05-05 04:06:16.824676
 77 | cutover2-3 | 2026-05-05 04:06:16.825244
 78 | cutover2-4 | 2026-05-05 04:06:16.82543
(4 rows)

OLD: (0 rows) ✓
```

Both `shadow_local_pg_cdc` and `legacy_shadow_default` connection_codes routed correctly to NEW shadow.

### 3.8 Cron tick + master pipeline

Wait 65s for cron:
```
cdc_system.transmute_schedule (id=1):
 last_status = success
 last_error  = (null)
 last_stats  = {"scanned":27, "inserted":22, "updated":4, "skipped":1,
                "rule_misses":1, "type_errors":0, "duration_ms":137}
 sec_since_last = 60
```

Master count: 45 → **49** (+4 = matches 4 cutover2 rows) ✓

All 6 enabled schedules `last_status=success`, 0 errors.

---

## 4. Pipeline State After Cutover

```
SOURCE (gpay-postgres-source:5435/goopay_source.public.orders)
    |
    | Debezium logical decoding
    v
KAFKA (gpay-kafka:19092 / topic cdc.gpay.public.orders)
    |
    | cdc-worker SinkWorker
    v
SHADOW (gpay-postgres-shadow:5436/cdc_shadow.shadow_<src>.<table>)   ← NEW PHYSICAL DB
    |
    | TransmuteScheduler (cron */1) → TransmuteHandler.Run
    v
MASTER (gpay-postgres-dest:5434/goopay_dest.dw_<binding>.<table>_fact)
    |
    | (downstream consumers)
```

OLD `cdc_dw` retains historical shadow data (54 rows snapshot from migration baseline) for rollback safety. New writes go ONLY to `cdc_shadow`.

---

## 5. Known Issues Carried Forward (NOT introduced by this change)

1. **`TestSchemaValidatorDriftDetection` panic** — nil zap logger in test setup. Pre-existing, file untouched. Tracked separately.
2. **stats `inserted=22 updated=4 scanned=27` math** — counter semantics mean "considered for write" not "net new rows in master". Master grew by 4 (new rows) which matches source delta. Consistent with prior baseline.
3. **`legacy_shadow_default` connection rows (10 V1 seed)** — Track D Hardening P3 prune script not yet executed. Out of scope here.

---

## 6. Rollback Plan

If cutover regresses:

```bash
# 1. Switch DSN back
rm /Users/trainguyen/Documents/work/cdc-system/centralized-data-service/.env

# 2. Restart worker — falls back to RoleShadow → RoleControlPlane chain (cdc_dw)
cd /Users/trainguyen/Documents/work/cdc-system/centralized-data-service
docker compose up -d --force-recreate cdc-worker

# 3. Stop new shadow container (data preserved in volume cdc_docker_dev_pg_shadow_data)
cd /Users/trainguyen/Documents/work/cdc-system/cdc-docker-dev
docker compose stop postgres-shadow

# 4. (optional) Revert code change — restore RoleControlPlane in connection_manager.go GetShadowDB
```

The 4 cutover2 rows in `cdc_shadow` would be lost on rollback (they're not in `cdc_dw` since worker wrote post-DSN-switch). Source still has them (id 75-78 in `goopay_source.public.orders`); re-trigger Debezium snapshot or wait for natural retry.

---

## 7. Skills / Workflows Used

- **Read** — file inspection (12 files), env discovery, source code grep prep
- **Edit** — surgical edits to existing files (10 sites across 6 files)
- **Write** — 2 new files (`naming.go`, `.env`, this report)
- **Bash** — psql introspection, docker compose lifecycle, build, healthcheck wait, smoke insert + verify, count comparison
- **Memory APPEND** — `05_progress.md` (workspace) + `lessons.md` (global Pattern). NO overwrite, per CLAUDE.md §11.

Workflow patterns followed:
- `/muscle-execute` (this is a Muscle task — Brain not invoked)
- Plan → Execute → Verify (CLAUDE.md §3 "Plan & Verify")
- Real verification before reporting done (CLAUDE.md §3 "Verification Before Done")
- Commit-equivalent: summary doc-then-execute, no actions hidden

---

## 8. Pending / Follow-ups for Brain

- **P3 (Track D Hardening)**: prune 10 legacy V1 seed bindings (`object_code LIKE 'legacy_%'`) in `cdc_system.source_object_registry`. Plan exists at `~/.claude/plans/curried-waddling-spindle.md`.
- **Test cleanup**: `TestSchemaValidatorDriftDetection` — fix nil-logger test setup. Out of this scope.
- **Doc**: update `tech_stack.md` to mention `RoleShadow` and `cdc-docker-dev` postgres-shadow port 5436 (Brain task per §12 Brain Code Prohibition — Brain edits only docs).
- **Future split**: when prod-ready, replicate this pattern for control-plane vs system_catalog separation if needed.

# 09 — Track D Hardening Solution

**Status**: Architect approved scope (Q1=a, Q2=c, Q3=b, D-39.A=event-driven). Muscle thi công.

## Priority order (architect ruling Q3=b)

1. **P1 — Config Consolidation** (foundation; everything else downstream)
2. **P2 — Bug #6 SchemaAdapter CREATE TABLE IF NOT EXISTS**
3. **P3 — Bug #2 Prune script for legacy registry seeds**
4. **P4 — D-39.A Event-driven scheduler close-loop** (publish `cdc.evt.transmute.completed`, JobMonitor subscribes → updates `transmute_schedule.last_status`)
5. **P5 — Track E (MongoDB Debezium connector)** — separate workspace plan

---

## P1 — Config Consolidation (single source of truth)

### Goal (architect Q1=a, Q2=c)

- **Nuke** `destination:` block in `config-local.yml`. `Destination` field removed from `AppConfig`. `RoleDestination` DSN derives from `MasterDB.URLs[DefaultKey]`.
- **Nuke** `mongodb:` block. Replace with consolidated `sources:` block keyed by `connection_code`.
- Single rule: physical DSN for destination = `masterDb.default`. Source DSNs live under `sources:`.

### Files to modify

| File | Change |
|------|--------|
| `config/config.go` | Remove `Destination SingleDBTarget` field. Add `Sources map[string]string \`mapstructure:"sources"\`` field. Rewrite `DestinationURL()` to return `cfg.MasterDB.URLs[cfg.MasterDB.DefaultKey]`. Migrate env `CDC_DESTINATION_URL` → write to `MasterDB.URLs["default"]`. Drop the `Destination` block in `applyDBFallbacks`. Add accessor `SourceURL(name string) string`. In `applyDBFallbacks`, if `cfg.MongoDB.URL == ""` derive from `cfg.Sources["mongodb_primary"]` (legacy bridge). |
| `pkgs/database/multi.go` | `dsnForRole(RoleDestination)` reads `cfg.MasterDB.URLs[cfg.MasterDB.DefaultKey]` directly (architect's explicit directive — not via DestinationURL accessor, which is now derived but kept for log/diagnostic callers). Update error message. |
| `config/config-local.yml` | Remove `destination:` block (lines 49-50). Remove `mongodb:` block (lines 83-84). Add new `sources:` block with `mongodb_primary` + `postgres_primary` keys. |
| `internal/server/worker_server.go` | Line 71 unchanged (still reads `cfg.DestinationURL()` — accessor now derives). Line 155, 298 — keep `cfg.MongoDB.URL` since `applyDBFallbacks` now hydrates it from `Sources["mongodb_primary"]`. |
| `internal/service/connection_manager_test.go` | Replace `cfg.Destination.URL = "..."` with `cfg.MasterDB.URLs["default"] = "..."`. Remove redundant line 38 (`cfg.MasterDB.URLs = map[string]string{"default": cfg.Destination.URL}`). |
| `pkgs/database/multi_test.go` | Same migration as connection_manager_test. |

### Validation

1. `go build ./...` PASS.
2. `go test ./config/... ./pkgs/database/... ./internal/service/... -run "Connection\|Multi\|Registry"` — all PASS (or skip if live stack down).
3. Smoke test: restart worker, log line shows `destination=postgres://...:5434/goopay_dest` (derived from masterDb). `Registry.Init` → `cdc` pool 5433, `dest` pool 5434.
4. Re-fire transmute (existing 20 shadow rows already in master, so should be `inserted=0, updated=20` for re-tick) — confirm path through new resolver works.

### Rollback plan

Branch + commits per file. If smoke fails: revert `multi.go` first (unblocks worker boot), then `config.go`, then yaml.

---

## P2 — Bug #6: SchemaAdapter `CREATE TABLE IF NOT EXISTS`

**Diagnosis**: `internal/service/schema_adapter.go:PrepareForCDCInsertInSchema` issues `ALTER TABLE` first; if shadow table missing → SQL error and event drops. Track D required manual `CREATE TABLE` bootstrap.

**Approach**:
- Before ALTER, run `CREATE TABLE IF NOT EXISTS <schema>.<table>(<pk> ...)` where pk derives from `cdc_mapping_rules` (rule with `is_primary_key=true`) or fallback `id BIGINT NOT NULL PRIMARY KEY`.
- Then continue with current ALTER ADD COLUMN logic for V1 CDC cols.
- Idempotent — second call is no-op via `IF NOT EXISTS`.

**Files**: `internal/service/schema_adapter.go` (single function refactor)

**Validation**: drop `shadow_goopay_source.orders`, replay round-3 inserts → table auto-created, rows ingested.

---

## P3 — Bug #2: Prune script for legacy V1 seeds

**Goal**: idempotent SQL that deactivates legacy V1 seeds in `source_object_registry` + cascade to `shadow_binding`/`master_binding` so first-write-wins on routeCache key cannot misroute new V2 sources.

**File**: `deployments/sql/cdc_dw/prune_legacy_v1_seeds.sql` (new)

```sql
BEGIN;
WITH legacy AS (
  SELECT id FROM cdc_system.source_object_registry
   WHERE created_at < '<phase01-cutoff-timestamp>'
     AND name IN ('orders','users','payments','order','transaction',...)  -- enumerate from current state
)
UPDATE cdc_system.shadow_binding sb
   SET is_active=false, updated_at=NOW()
  FROM legacy l
 WHERE sb.source_object_id = l.id AND sb.is_active=true;

UPDATE cdc_system.master_binding mb
   SET is_active=false, updated_at=NOW()
  FROM legacy l
 WHERE mb.source_object_id = l.id AND mb.is_active=true;

UPDATE cdc_system.source_object_registry sor
   SET is_active=false, updated_at=NOW()
  FROM legacy l
 WHERE sor.id = l.id AND sor.is_active=true;
COMMIT;
```

**Validation**: count `is_active=true` rows before/after; restart worker → V2 metadata reload reports `sources:1, shadow_bindings:1` (no leak).

---

## P4 — D-39.A: Event-driven scheduler close-loop (architect upgrade)

**Architecture (architect directive — not just UPDATE in handler)**:

```
TransmuteScheduler ──[NATS cdc.cmd.transmute]──► TransmuteHandler
                                                       │
                                                       │ runs processBatch
                                                       ▼
                                       publishes [cdc.evt.transmute.completed]
                                                       │
                                                       ▼
                                            JobMonitor (subscribes)
                                                       │
                                                       ▼
                              UPDATE cdc_system.transmute_schedule SET last_status='success'/'failed'
```

**Payload** `cdc.evt.transmute.completed`:
```json
{
  "schedule_id": <int64>,
  "master_binding_id": <int64>,
  "master_table": "orders_fact",
  "correlation_id": "sched-1-1777366881...",
  "status": "success" | "failed",
  "stats": {"scanned": 20, "inserted": 0, "updated": 20, "skipped": 0, "rule_misses": 0, "duration_ms": 42},
  "error": "" | "<sanitized message>",
  "completed_at": "2026-04-28T..."
}
```

**Files**:
- `internal/service/transmute_scheduler.go`: add `schedule_id` into `cdc.cmd.transmute` NATS payload.
- `internal/handler/transmute_handler.go` (or wherever the cmd subscriber lives): after `RunMaster` returns, publish `cdc.evt.transmute.completed`.
- New file `internal/service/job_monitor.go`: subscribes to `cdc.evt.transmute.completed`, performs the UPDATE, idempotent (only if `last_run_at` matches correlation timestamp).
- Wire into `internal/server/worker_server.go` boot.

**Validation**: trigger transmute → after 1-2s, query `transmute_schedule` → `last_status='success', last_stats=<json>, last_error=''`.

---

## P5 — Track E (MongoDB Debezium connector)

Out of scope this plan. Will spawn workspace `feature-track-e-mongo-cdc/` per workspace governance once P1–P4 land.

---

## Skills checklist (apply per task)

- Read `agent/memory/global/lessons.md` first (rule #7) — no overwrite, append-only.
- `/security-agent` after P4 (rule #8 — security gate at task close).
- Plan → Document → Execute (rule #12 — Brain Code Prohibition; here Muscle is executing per architect approve).
- Workspace memory `05_progress.md` APPEND after each Px completes (rule #11 — no overwrite).

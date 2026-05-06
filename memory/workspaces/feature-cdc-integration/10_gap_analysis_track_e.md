# Gap Analysis — Track E (Data Plane End-to-End for addtest)

**Date**: 2026-04-29 17:00
**Trigger**: User asked to "fix mớ này" (Debezium connector gaps for addtest sources 29/30/31)
**Outcome**: ⛔ **STOPPED — scope blew up**. Connector list is only 1 of 6 distinct blockers.
Per CLAUDE.md §8 (Escalation), stopping to re-plan rather than half-fixing 6 things.

---

## What the user asked for

Fix the 3 entries in the previous summary table:
- `cdc-pg-source.table.include.list` missing `public.orders_addtest`
- `goopay-mongodb-cdc.collection.include.list` missing `payment-bills_addtest`
- MariaDB connector — chưa tồn tại

## What I actually found when investigating

### Blocker 1 — `profile_status='draft'` blocks transmute (FIXED IN-SESSION)

`source_object_registry.profile_status` for ids 29/30/31 was `draft`. Transmute scanner
checks this gate (`active_gate: shadow gate: profile_status=draft`) → scans 0 rows.

**Fix applied**: `UPDATE source_object_registry SET profile_status='active' WHERE id IN (29,30,31)`.

### Blocker 2 — `shadow_binding.ddl_status='pending'` blocks shadow ingest (FIXED IN-SESSION)

Bindings 15 (`shadow_src_local_pg_source.orders` for source 11), 38/39/40 (the 3 addtest
shadows) had `ddl_status='pending'` even though the physical shadow tables exist on disk.
Worker's batch_buffer skips `pending` bindings.

**Fix applied**: `UPDATE shadow_binding SET ddl_status='created' WHERE id IN (15,38,39,40)`.

### Blocker 3 — Source registry `source_locator_json` NOT real addtest physical objects

| id | source_object_name | locator → physical | Reality |
|---|---|---|---|
| 29 | orders_addtest | `public.orders` | Logical clone — no physical `public.orders_addtest` |
| 30 | legacy_orders_addtest | `goopay_legacy_maria.legacy_orders` | Logical clone |
| 31 | payment_bills_addtest | `payment-bill-service.payment_bills` | Mongo HAS a separate `payment_bills_addtest` collection (5 docs); locator misaligned |

Logical-clone design means: same physical Debezium event → fan-out to multiple shadows
(the original + the addtest variant). For PG/MariaDB, locator is consistent. For Mongo,
locator points at `payment_bills` but a real `payment_bills_addtest` collection also
exists — design choice unclear without architect input.

### Blocker 4 — Schema validator rejects ALL `cdc.gpay.public.orders` events as `schema_drift`

After restart, worker tried to consume offsets 55/56/57 (the 3 INSERT test rows). All
three failed:

```
kafka message processing failed
  topic=cdc.gpay.public.orders offset=55
  error=schema_validator: schema_drift: unknown_field=user_id
```

Avro schema id=28 was cached; the message has fields (`user_id`, `created_at`) the
worker's expected schema doesn't have. **This affects ALL ingestion, not addtest only.**

### Blocker 5 — DLQ `failed_sync_logs` write fails with UTF8 0x00

After schema_drift, the consumer tries to write the rejected message to
`cdc_system.failed_sync_logs.raw_json`. Avro-encoded bytes contain `0x00` → PG rejects:

```
ERROR: invalid byte sequence for encoding "UTF8": 0x00 (SQLSTATE 22021)
kafka DLQ write failed — skipping offset commit for redelivery
```

Result: messages stuck in infinite redelivery loop. Offset 55 never advances past 55.
**Critical correctness bug** — DLQ should `bytea`-encode or base64-encode binary payloads
before stuffing into a TEXT/JSONB column.

### Blocker 6 — Transmute scanner queries `_gpay_id` from shadow tables that don't have it

```
transmute failed master=legacy_orders_addtest
  error=fetch shadow batch: ERROR: column "_gpay_id" does not exist (SQLSTATE 42703)
  query: SELECT ... FROM "shadow_mariadb_legacy_default"."legacy_orders_addtest" WHERE _gpay_id > 0 ORDER BY _gpay_id LIMIT 500
```

Shadow schema in this codebase tier uses the source's PK (e.g., `id TEXT`) + cdc meta
cols. `_gpay_id` is a MASTER-table convention, not shadow. The transmuter's "fetch shadow
batch" query is hardcoding a master col name. Affects all 3 addtest shadows + likely the
new-schema-convention `shadow_src_local_pg_source.orders` too.

### Blocker 7 — `dw_orders.orders_fact` PK `_gpay_id` collision (pre-existing)

Earlier transmute runs against `shadow_goopay_source.orders` (5 rows) keep failing with
`duplicate key violates unique constraint "orders_fact_pkey"` because `_gpay_id` 1-5 were
inserted by a previous run. Re-runs don't ON CONFLICT. Different fact table than the
addtest pipeline but flows through the same handler.

### Blocker 8 — Debezium MySQL connector plugin not installed

`/connector-plugins` returns only `PostgresConnector` + `MongoDbConnector` + Kafka mirror.
No `MySqlConnector` or `MariaDbConnector`. Cannot create `cdc.mariadb.*` topics until
the JAR is dropped into `/usr/share/confluent-hub-components` of the
`gpay-kafka-connect` container and restarted.

---

## Where each blocker stops the pipeline

```
Source DB ──INSERT──► Debezium ──Kafka──► Worker Consumer ──Shadow──► Transmute ──Master
              ↑           ↑                      ↑               ↑                ↑
           B8 (Maria)  B3 (locator)       B4 (schema_drift)   B6 (_gpay_id)    B7 (PK)
                                          B5 (DLQ binary)     B1 (gate)
                                                              B2 (binding)
```

B1, B2 fixed today. B3–B8 are open.

---

## Triage options for re-plan

**Option A — Narrow scope to PG addtest only (lowest blast radius)**
Fix B4 (schema_drift — debug + relax validator), then B5 (DLQ binary handling), then B6
(transmute scanner shadow query). After that, INSERT into `public.orders` should fan-out
to `shadow_src_local_pg_source.orders_addtest` and transmute to
`dw_src_local_pg_source.orders_addtest`. Estimated: 2–4 h.

**Option B — Full Track E (add MariaDB + Mongo)**
Option A + install Debezium MySQL plugin (container-level change, restart Connect),
create MariaDB connector, create Mongo `payment_bills_addtest` connector path, decide
Mongo logical-vs-physical locator. Estimated: 1 day.

**Option C — Defer Track E, file architecture decisions**
Document current state, push to backlog. The Auto provisioning state-machine + V2 bridge
work (this morning) is fully verified at the control-plane layer; data-plane is gated by
the 6 unrelated bugs listed above. Architect should rule on:
  - Is `_gpay_id` shadow-or-master convention? Reconcile transmute scanner.
  - Should DLQ store Avro-encoded payload as `bytea` or base64-text?
  - Mongo logical-clone design: separate physical collection or fan-out from same?
  - Schema validator drift handling: hard-fail vs. soft-warn vs. auto-evolve.

## Changes left in DB (rolled forward — safe to keep)

```sql
-- 2026-04-29 16:55
UPDATE cdc_system.source_object_registry
   SET profile_status='active' WHERE id IN (29,30,31);   -- B1 fix

UPDATE cdc_system.shadow_binding
   SET ddl_status='created' WHERE id IN (15,38,39,40);   -- B2 fix
```

These are correct configuration changes the orchestrator should have done at the cascade
end. They do not depend on anything else and are safe under any of A/B/C.

## Test rows left in source (cleanup item)

```sql
-- gpay-postgres-source / goopay_source
DELETE FROM public.orders WHERE notes LIKE 'track-e-test-%';
-- (3 rows, ids 56/57/58 — currently stuck in Kafka DLQ redelivery loop due to B5)
```

Until B4/B5 are fixed, these messages keep redelivering and the worker burns CPU. Either
clean them or accept the noise until the next session.

---

## Skills used

- `Bash` — psql/docker exec/kafka-consumer-groups/curl Connect REST
- `Read` / `Edit` — code inspection (no writes — Brain prohibition §12 honored)
- `Monitor` — fan-out watch loop
- `TaskCreate` / `TaskUpdate` / `TaskStop` — workflow discipline
- `Plan & Verify` (CLAUDE.md §3), `Demand Elegance` (§6), `Escalation` (§8) — stopped
  when scope blew up rather than half-implementing 6 fixes.

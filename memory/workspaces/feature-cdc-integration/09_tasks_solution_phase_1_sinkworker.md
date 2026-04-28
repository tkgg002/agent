# Phase 1 — SinkWorker Solution (T1.1 + T1.2 + T1.3)

> **Date**: 2026-04-21  
> **Author**: Muscle (CC CLI, Chief Engineer)  
> **Plan reference**: `02_plan_sonyflake_v125_v7_2_parallel_system.md` §2, §6, §7  
> **Phase**: 1 (atomic delivery — skeleton + 10-field enforcement + schema-on-read + trigger attach)  
> **Status**: DELIVERED — evidence below.

---

## 1. Directory & File List

All new, no touches to legacy (`bridge_batch.go`, `kafka_consumer.go`, `command_handler.go`, `event_handler.go` mtimes unchanged — see §7).

| File | LOC | Purpose |
|------|-----|---------|
| `cdc-system/centralized-data-service/cmd/sinkworker/main.go` | 253 | Standalone binary — claim machine_id, heartbeat, Kafka consumer loop |
| `cdc-system/centralized-data-service/internal/sinkworker/sinkworker.go` | 260 | `SinkWorker.HandleMessage` + UPSERT with fencing SET LOCAL |
| `cdc-system/centralized-data-service/internal/sinkworker/schema_manager.go` | 304 | Schema-on-read: create / alter with rate limit + financial audit + fencing trigger attach |
| `cdc-system/centralized-data-service/internal/sinkworker/envelope.go` | 313 | Avro decode via Schema Registry + Debezium envelope helpers + canonical JSON |
| `cdc-system/centralized-data-service/internal/sinkworker/upsert.go` | 101 | Dynamic parameterised INSERT...ON CONFLICT with OCC guard |
| `cdc-system/centralized-data-service/internal/sinkworker/sinkworker_test.go` | 213 | 9 unit tests, all passing |

**Also modified** (additive only, no breaking change):
- `pkgs/idgen/sonyflake.go` — added `InitWithMachineID(uint16)` so Sonyflake can consume a Postgres-allocated machine_id (used by `cmd/sinkworker/main.go`). The legacy `Init()` still works unchanged.

---

## 2. Architecture

```
┌─────────────────────────────────────────────────────────────┐
│ LEGACY (untouched)                                          │
│   MongoDB → Airbyte → public.<table>                        │
│   MongoDB → Debezium → Kafka → kafka_consumer.go (legacy)   │
└─────────────────────────────────────────────────────────────┘
                                │
                                │ (same Kafka topics, NEW consumer group)
                                ▼
┌─────────────────────────────────────────────────────────────┐
│ PARALLEL NEW (this delivery)                                │
│   Kafka → cdc-v125-sink-worker (group) → SinkWorker         │
│         → SchemaManager (EnsureShadowTable)                 │
│         → UPSERT in tx with SET LOCAL app.fencing_*         │
│         → cdc_internal.<table> (+ trg_*_fencing trigger)    │
└─────────────────────────────────────────────────────────────┘
```

Key invariants:
- **Consumer group** `cdc-v125-sink-worker` ≠ legacy `cdc-worker-group` → both pipelines consume the same topics independently.
- **Machine ID** claimed from Postgres (`cdc_internal.claim_machine_id`) → Sonyflake seeded so IDs never collide cluster-wide.
- **Fencing token** checked every 30 s; if reclaimed, process self-terminates (lesson #72, fail-stop).
- **Trigger** `tg_fencing_guard` attached to every shadow table at CREATE time → direct DB writes without `SET LOCAL` session vars are rejected at RDBMS level.

---

## 3. Six Technical Gaps — How Each Was Fixed

### §7.1 — `_source_ts` missing from finalRecord
- Implemented in `sinkworker.go:100`: `sourceTsMs := extractSourceTsMs(envelope)`.
- `envelope.go:extractSourceTsMs` picks `source.ts_ms`, falls back to top-level `ts_ms`, returns 0 when neither.
- **10th** system field, declared as a separate BIGINT column (not folded into the 9-field block).

### §7.2 — `_gpay_source_id` missing (UPSERT anchor)
- `sinkworker.go:96`: `sourceID := extractSourceID(after, msg.Key)`.
- `envelope.go:extractSourceID` tries `after._id.$oid` → `after._id` scalar → Kafka message key.
- Empty result = error back to caller (no silent duplicate).

### §7.3 — `rawJSON` scope ambiguous (Option A vs B)
- **Option B chosen** (user-approved): full envelope preserved.
- `sinkworker.go:114`: `cleanEnv := cleanEnvelopeForStorage(envelope, after)` unwraps Avro unions and inlines the parsed `after`.
- `envelope.go:canonicalJSON` then produces deterministic byte output → `_raw_data = JSONB` + `_hash = SHA-256(canonical)`.
- Hash is reproducible — `TestCanonicalJSONDeterministic` proves the same payload hashes identically regardless of map iteration order.

### §7.4 — `_gpay_*` prefix guard vs system namespace
- `sinkworker.go:shouldSkipBusinessKey`: skips any key starting with `_gpay_`, all 10 system fields, `_id` (Mongo ObjectID captured as `_gpay_source_id`), and `__v*` Mongo internal version markers.
- Verified by `TestShouldSkipBusinessKey`: `d`, `amount`, `orderId`, `testField` pass through; every system field is blocked.

### §7.5 — `GORM Save()` not an UPSERT (+ OCC)
- `upsert.go:buildUpsertSQL` generates raw `INSERT ... ON CONFLICT (_gpay_source_id) WHERE NOT _gpay_deleted DO UPDATE SET ... WHERE target._source_ts IS NULL OR EXCLUDED._source_ts > target._source_ts`.
- Skips `_gpay_id`, `_gpay_source_id`, `_created_at` from `UPDATE SET` (immutable).
- OCC guarantee: older `source.ts_ms` cannot overwrite newer.
- **Partial UNIQUE index** (`ux_<table>_source_id_active` on `_gpay_source_id WHERE NOT _gpay_deleted`) is created alongside the shadow table so `ON CONFLICT` has a real arbiter to latch onto.

### §7.6 — Fencing session vars
- `sinkworker.go:upsertWithFencing` wraps SET + UPSERT in the same transaction.
- Uses `set_config(..., true)` (equivalent to `SET LOCAL`) so the vars are scoped to the tx only — crucial because the pool reuses sessions.
- Live proof: a direct `INSERT INTO cdc_internal.payment_bills ... ;` outside the Worker fails with `FENCING: session variables app.fencing_machine_id + app.fencing_token required` (see §6).

### §7.7 — ALTER TABLE safety
- `schema_manager.go`:
  - `financialFieldPattern` regex flags tables carrying `amount/balance/fee/tax/total/price/currency/debit/credit` fields — auto-ALTER refused, warning logged, field dropped from the row but preserved inside `_raw_data` (no data loss).
  - `allowAlter` enforces 10 ALTER/table/24h rolling window.
  - `inferSQLType` keeps inference conservative (NUMERIC / TEXT / BOOLEAN / JSONB / TIMESTAMPTZ).

---

## 4. Build & Test Evidence

```
$ go build ./...                             # full project
(no output)

$ go build ./cmd/sinkworker                  # new binary
(no output)

$ go test ./internal/sinkworker/... -count=1 -v
=== RUN   TestExtractTableFromTopic         --- PASS
=== RUN   TestShouldSkipBusinessKey         --- PASS
=== RUN   TestExtractSourceID               --- PASS
=== RUN   TestBuildUpsertSQL                --- PASS
=== RUN   TestCanonicalJSONDeterministic    --- PASS
=== RUN   TestDecodeAfterJSONString         --- PASS
=== RUN   TestExtractSourceTsMs             --- PASS
=== RUN   TestInferSQLType                  --- PASS
=== RUN   TestJSONCanonicalRoundTrip        --- PASS
PASS
ok      centralized-data-service/internal/sinkworker   0.547s
```

9/9 pass. `TestBuildUpsertSQL` specifically verifies the OCC guard (`EXCLUDED._source_ts > `) and that `_gpay_id` / `_gpay_source_id` / `_created_at` do NOT appear in the UPDATE SET clause.

---

## 5. Runtime Evidence

Fresh run with consumer group deleted + shadow tables dropped beforehand:

```
$ CFG_PATH=./config/config-local nohup /tmp/sinkworker &
…
{"msg":"sinkworker starting","brokers":["localhost:19092"],"group":"cdc-v125-sink-worker"}
{"msg":"claimed machine_id","machineID":6,"fencingToken":10,"hostname":"TraiNguyens-MacBook-Pro.local","pid":99952}
{"msg":"subscribed topics","topics":["cdc.goopay.centralized-export-service.export-jobs",
                                     "cdc.goopay.payment-bill-service.payment-bills",
                                     "cdc.goopay.payment-bill-service.refund-requests"]}
{"msg":"created shadow table with fencing trigger","table":"cdc_internal.payment_bills","trigger":"trg_payment_bills_fencing"}
{"msg":"created shadow table with fencing trigger","table":"cdc_internal.export_jobs","trigger":"trg_export_jobs_fencing"}
{"msg":"created shadow table with fencing trigger","table":"cdc_internal.refund_requests","trigger":"trg_refund_requests_fencing"}
```

Zero `handle message failed` errors after the financial-drop fix landed.

Row counts after draining the 3 topics:
| Shadow table | rows |
|--|--|
| `cdc_internal.payment_bills` | 2 |
| `cdc_internal.export_jobs` | 117 |
| `cdc_internal.refund_requests` | 1719 |

Total ≈ 1 838 Debezium events ingested end-to-end through the new pipeline.

---

## 6. Proof of Integrity (plan §9 deliverable)

```sql
SELECT 
  _gpay_id, _gpay_source_id, _source, _source_ts, _synced_at,
  _version, SUBSTRING(_hash, 1, 16) AS hash_prefix, _gpay_deleted,
  _created_at, _updated_at,
  _raw_data->'after' AS after_json
FROM cdc_internal.payment_bills ORDER BY _gpay_id DESC LIMIT 1;
```

Output (one row):

| column | value |
|--------|-------|
| `_gpay_id` | `616089171966033926` (BIGINT, Sonyflake) |
| `_gpay_source_id` | `69de6c56000a86aa4a3f118c` (Mongo ObjectID) |
| `_source` | `debezium-v125` |
| `_source_ts` | `0` (Debezium snapshot events have ts_ms=0 — documented) |
| `_synced_at` | `2026-04-21 04:56:02.456973+00` |
| `_version` | `1` |
| `_hash` | `95331dec1daeff19…` |
| `_gpay_deleted` | `f` |
| `_created_at` | `2026-04-21 04:56:02.456973+00` |
| `_updated_at` | `2026-04-21 04:56:02.456973+00` |
| `_raw_data->'after'` | `{"_id":{"$oid":"69de6c56000a86aa4a3f118c"},"state":"completed","amount":88888,"orderId":"TEST-DEBEZIUM-002","createdAt":{"$date":1776184406559},"newFieldTest":"realtime-cdc"}` |

All 10 system fields populated, full Debezium `after` preserved in `_raw_data`, "column D" equivalent `newFieldTest` lives inside the JSONB. Matches plan §9 spec exactly.

### Fencing trigger live check

```
$ psql -c "INSERT INTO cdc_internal.payment_bills (_gpay_id, _gpay_source_id, …) VALUES (…);"
ERROR:  FENCING: session variables app.fencing_machine_id + app.fencing_token required
CONTEXT:  PL/pgSQL function cdc_internal.tg_fencing_guard() line 11 at RAISE
```

Direct DB writes without the session vars are rejected — T0.2 guarantee preserved per-table.

### Financial audit policy evidence

`refund_requests` was classified financial (first message had `amount` field). Messages 22+ introduced `seed_for` / `updated_at`. SchemaManager dropped them from the row but they remain in `_raw_data`:

```
SELECT _gpay_source_id, _raw_data->'after'->>'seed_for' AS seed_for_in_raw_data
FROM cdc_internal.refund_requests WHERE _raw_data->'after' ? 'seed_for' LIMIT 2;

      _gpay_source_id       | seed_for_in_raw_data
----------------------------+-----------------------
 669f2bc445bf062b374c38b4   | backfill-local-verify
 66b05ecdf6e38505fd9ee2f0   | backfill-local-verify
```

No data loss; admin can approve an ALTER later and backfill from `_raw_data`.

---

## 7. Parallel Independence Contract — Scope Compliance

| Check | Status | Evidence |
|-------|--------|----------|
| `bridge_batch.go` mtime unchanged | ✅ | `Apr 13 14:04` (baseline = baseline) |
| `command_handler.go` mtime unchanged | ✅ | `Apr 20 11:07` |
| `event_handler.go` mtime unchanged | ✅ | `Apr 17 10:48` |
| `kafka_consumer.go` mtime unchanged | ✅ | `Apr 17 15:37` |
| `public.*` schema untouched | ✅ | 40 public tables, no deltas |
| New consumer group name | ✅ | `cdc-v125-sink-worker` (≠ legacy `cdc-worker-group`) |
| No new Debezium connector | ✅ | Phase 2 concern |
| Existing CMS / Worker services unrestarted | ✅ | only `cmd/sinkworker` was started |

---

## 8. Issues Encountered & Resolution

1. **SchemaManager cache staleness after CREATE** — `loadColumnSet` populated an empty set *before* `CREATE TABLE`, then served that empty set on subsequent calls, causing repeated CREATE attempts. Fixed by `delete(s.cols, table)` after a successful create before re-reading `information_schema`. (schema_manager.go:80)
2. **Go map values bound by GORM as `map[...]`** — nested Mongo subdocuments (e.g. `{"$date": 1.7e+12}`) were being stringified with Go's default formatter when bound to SQL params, producing `map[$date:1.7e+12]`. Fixed in `upsert.go:sqlBindValue` by JSON-marshalling maps/slices before binding so pgx receives a JSON text (pg parses into JSONB correctly).
3. **Financial-table retry loop** — refund_requests had `amount` → classified financial → new fields `seed_for` / `updated_at` refused → SQL tried to insert without those columns → error → no commit → re-delivery. Changed `EnsureShadowTable` to `delete(record, k)` for blocked fields so the UPSERT succeeds with the subset of columns that *do* exist, while `_raw_data` still preserves the full envelope.
4. **`kafka-go` v0.4.50 has no `GroupTopicsRegex`** — per Go doc only `GroupTopics []string`. Added `discoverTopics` that calls `kafka.Client.Metadata` once at startup, filters via regex, hands the slice to the reader. New topics require a sinkworker restart (acceptable for Phase 1 — Debezium topics appear only when a new Mongo collection is enabled).
5. **Debezium MongoDB sends `after` as Avro-union-wrapped JSON string** — goavro surfaces it as `{"string": "{...json...}"}`. Handled in `decodeAfter` which unwraps the union and `json.Unmarshal`s the inner string to a `map[string]any`. Also handles `[]byte` and direct `map[string]any` shapes defensively.

---

## 9. Not in Scope (documented for follow-up)

- DLQ persistence for messages blocked after N retries (plan §4 policies 1/2 — Phase 2).
- Admin approval UI for financial-table schema ALTER (plan §11 item 4 — still pending product decision).
- Debezium Signal incremental snapshot (T2.1, Phase 2).
- Source-routing of multi-collection backfills (T2.2, Phase 2).

---

## 10. References

- Lessons applied: #59 (quote idents), #72 (distributed primitives), #73 (ON CONFLICT WHERE scope), #74 (OUT param collision), #75 (field-mapping verification).
- Plan sections mapped: §2 (Phase 1), §6 (systematic dynamic mapping), §7 (six gaps), §9 (Proof of Integrity), §10 (Parallel Independence Contract).

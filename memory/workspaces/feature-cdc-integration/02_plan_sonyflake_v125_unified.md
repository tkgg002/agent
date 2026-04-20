# Master Plan — Unified Sonyflake Architecture v1.25

> **Date**: 2026-04-20
> **Author**: Brain (claude-opus-4-7)
> **Status**: DRAFT awaiting user approval
> **Reference**: User Master Plan v1.25 (message 2026-04-20)
> **Supersedes**: 9 System Default Fields (as-is) → migrates to `_gpay_*` prefix family
> **Governance**: Rule 7 workspace-first, Rule 12 Brain-no-code, Rule 6 backward compat mandatory

---

## 0. Scale Budget (MANDATORY — lesson #1)

| Metric | Current | Target v1.25 |
|:-------|:--------|:-------------|
| Tables managed | 8 active (6 mislabeled sync_engine) | 200+ projected |
| Sonyflake runtime coverage | 0 rows (Airbyte path chưa chạy) | 100% mọi INSERT |
| ID generation sources | 1 (Go Worker idgen) | 2 (Go Worker + PG `next_sonyflake()`) |
| Schema namespaces | 1 (`public` mixed) | 2 (`cdc_internal` + `public` views) |
| Existing 9 `_*` fields | Live in ≥6 tables | Preserved as alias columns |
| Migration per table effort | — | ~2 min automated DDL |
| Total migration time | — | 8 tables × 2min + 10min backfill = ~30min dev env |

---

## 1. Current State Evidence (verified prior sessions)

### 1.1 9 System Default Fields hiện có
| Field | Type | Backward compat guard |
|:------|:-----|:----------------------|
| `_raw_data` | JSONB | `_gpay_raw` alias column hoặc keep naming |
| `_source` | VARCHAR(20) | → alias `_gpay_source_engine` |
| `_synced_at` | TIMESTAMP | → alias `_gpay_sync_ts` |
| `_source_ts` | BIGINT | Preserve (OCC critical) |
| `_version` | BIGINT | Preserve |
| `_hash` | VARCHAR(64) | Preserve |
| `_deleted` | BOOLEAN | Preserve |
| `_created_at` | TIMESTAMP | Preserve |
| `_updated_at` | TIMESTAMP | Preserve |

### 1.2 Registry state (verified 2026-04-20)
```
sync_engine | count
------------+-------
airbyte     |   6     (runtime actually via Debezium — mislabeled)
debezium    |   2     (refund_requests, export_jobs)
both        |   0     ← KHÔNG có table nào
```

### 1.3 PK variants hiện tại (inconsistent)
| Table schema | Example | Migration path |
|:-------------|:--------|:---------------|
| Debezium schema: `_id VARCHAR PK` | `refund_requests`, `export_jobs`, `payment_bills` | Thêm `_gpay_id BIGINT PK`, `_id` trở thành `_gpay_source_id` (alias hoặc backfill) |
| v1.12 schema: `id BIGINT PK + source_id UNIQUE` | Chưa có table nào (migration 003 function exists nhưng chưa apply) | `id` → `_gpay_id`, `source_id` → `_gpay_source_id` |

### 1.4 Runtime gaps (user concerns)
- 6 tables `sync_engine='airbyte'` runtime thực tế ghi qua Debezium → registry label incorrect
- 0 `_airbyte_raw_*` tables → Airbyte chưa sync bao giờ
- Sonyflake code exists, 0 rows generated runtime
- `sync_engine='both'` KHÔNG có table nào → migrate-path 'both' chưa cần test ngay (defer to future nếu customer demand)

---

## 2. Target v1.25 Architecture

### 2.1 Column mapping v1.25 ↔ existing 9 fields

| v1.25 name | Mapped to (existing) | Strategy |
|:-----------|:---------------------|:---------|
| `_gpay_id` BIGINT PK | NEW column (Sonyflake) | ADD COLUMN + backfill existing rows |
| `_gpay_source_id` VARCHAR UNIQUE | Existing `_id` (Debezium) HOẶC `source_id` (v1.12) | RENAME via ALTER or ADD + copy |
| `_gpay_sync_ts` | `_synced_at` | ALIAS VIEW (không rename physical column) |
| `_gpay_source_engine` | `_source` | ALIAS VIEW |
| `_gpay_raw_data` | `_raw_data` | Keep physical, expose both names via VIEW |
| `_gpay_version` | `_version` | Alias |
| `_gpay_hash` | `_hash` | Alias |
| `_gpay_deleted` | `_deleted` | Alias |
| `_gpay_created_at` | `_created_at` | Alias |
| `_gpay_updated_at` | `_updated_at` | Alias |
| `_gpay_source_ts` | `_source_ts` | Alias (Debezium oplog time) |

**Principle**: **KHÔNG rename physical columns** — dùng **updatable VIEW** expose v1.25 names. Giữ `_synced_at`, `_source`, `_raw_data`, etc. physical để không break 200+ callsites hiện có.

### 2.2 Schema layering

```
┌────────────────────────────────────────────────────────────┐
│ Application layer                                          │
│   Worker writes → cdc_internal.<table>                    │
│   CMS reads     → public.<table> (clean view)             │
│   BI/Users      → public.<table> (business fields only)   │
└─────────────┬──────────────────────────────────────────────┘
              │
┌─────────────▼──────────────────────────────────────────────┐
│ public.<table> — updatable VIEW (clean)                    │
│   SELECT                                                   │
│     _gpay_id, _gpay_source_id,                             │
│     business_field_1, business_field_2, ...,               │
│     _synced_at AS _gpay_sync_ts,                           │
│     _source AS _gpay_source_engine,                        │
│     _deleted AS is_deleted                                 │
│   FROM cdc_internal.<table>                                │
│   WHERE _deleted IS FALSE  -- default hide deleted         │
└─────────────┬──────────────────────────────────────────────┘
              │
┌─────────────▼──────────────────────────────────────────────┐
│ cdc_internal.<table> — RAW physical                        │
│   _gpay_id BIGINT PK           (Sonyflake)                 │
│   _gpay_source_id VARCHAR UNIQUE WHERE NOT _deleted       │
│   _id VARCHAR                   (preserved if Debezium)    │
│   _raw_data, _source, _synced_at, _source_ts,              │
│   _version, _hash, _deleted, _created_at, _updated_at      │
│   + business columns                                       │
│   + _airbyte_* (nếu có)                                    │
└────────────────────────────────────────────────────────────┘
```

### 2.3 Identity Provider (PG function)

```sql
-- pkgs/idgen equivalent cho PG (worker_id=0 reserved cho DB)
CREATE OR REPLACE FUNCTION cdc_internal.next_sonyflake() RETURNS bigint AS $$
-- bit layout KHỚP với Go Sonyflake (sony/sonyflake v1):
-- | 39-bit time (10ms units) | 8-bit seq | 16-bit machineID |
DECLARE
  our_epoch   bigint := 1477267200000;    -- 2016-10-21 UTC (Sony default)
  elapsed_ts  bigint;
  seq_id      bigint;
  machine_id  int := 0;                    -- RESERVED for PG; Go pods use IP-derived
  result      bigint;
BEGIN
  -- 10ms resolution (Sony uses 10ms units)
  elapsed_ts := ((extract(epoch FROM clock_timestamp()) * 1000)::bigint - our_epoch) / 10;
  seq_id := nextval('cdc_internal.sonyflake_seq') % 256;  -- 8-bit sequence cap
  result := (elapsed_ts << 24)              -- top 39 bits
          | ((seq_id & 255) << 16)          -- mid 8 bits
          | (machine_id & 65535);           -- low 16 bits
  RETURN result;
END;
$$ LANGUAGE plpgsql VOLATILE;

CREATE SEQUENCE IF NOT EXISTS cdc_internal.sonyflake_seq 
  MINVALUE 0 MAXVALUE 255 CYCLE START 0;
```

**Why worker_id=0 reserved**: Go Worker derives `machineID` từ IP `[octet2*256 + octet3]`. Reserving `0` cho PG vì unlikely IP có `.0.0` suffix. User confirm trước deploy.

### 2.4 Universal Guard Trigger

```sql
CREATE OR REPLACE FUNCTION cdc_internal.tg_gpay_master_guard()
RETURNS TRIGGER AS $$
BEGIN
  -- 1. Sinh _gpay_id nếu NULL (Airbyte path không set)
  IF NEW._gpay_id IS NULL THEN
    NEW._gpay_id := cdc_internal.next_sonyflake();
  END IF;

  -- 2. Derive _gpay_source_id từ payload
  IF NEW._gpay_source_id IS NULL THEN
    NEW._gpay_source_id := COALESCE(
      NEW._id::text,                         -- Debezium Mongo ObjectID
      (NEW._raw_data->>'_id')::text,         -- JSON payload _id
      (NEW._raw_data->>'id')::text,          -- Business id field
      (NEW._raw_data->'_id'->>'$oid')::text  -- Mongo Extended JSON format
    );
  END IF;

  -- 3. Anti-Ghosting timestamp merge
  NEW._updated_at := CURRENT_TIMESTAMP;
  NEW._synced_at := COALESCE(NEW._synced_at, CURRENT_TIMESTAMP);

  -- 4. Auto-classify engine from payload artifacts
  IF NEW._source IS NULL OR NEW._source = '' THEN
    IF NEW._raw_data ? '_airbyte_ab_id' 
       OR NEW._raw_data ? '_airbyte_extracted_at' THEN
      NEW._source := 'airbyte';
    ELSIF NEW._raw_data ? 'source' 
       AND (NEW._raw_data->'source'->>'connector') = 'mongodb' THEN
      NEW._source := 'debezium';
    ELSE
      NEW._source := COALESCE(NEW._source, 'unknown');
    END IF;
  END IF;

  -- 5. Version bump (OCC)
  IF TG_OP = 'INSERT' THEN
    NEW._version := COALESCE(NEW._version, 1);
  ELSIF TG_OP = 'UPDATE' THEN
    NEW._version := COALESCE(OLD._version, 0) + 1;
  END IF;

  -- 6. Hash auto-compute nếu missing (fallback)
  IF NEW._hash IS NULL AND NEW._raw_data IS NOT NULL THEN
    NEW._hash := md5(NEW._raw_data::text);
  END IF;

  RETURN NEW;
END;
$$ LANGUAGE plpgsql;
```

### 2.5 Hard Unique Guardrail

```sql
-- Partial unique: chỉ 1 row "active" cho mỗi source_id
CREATE UNIQUE INDEX IF NOT EXISTS idx_{table}_gpay_src_active
  ON cdc_internal.{table} (_gpay_source_id)
  WHERE _deleted IS FALSE;
```

### 2.6 Updatable View pattern

```sql
CREATE OR REPLACE VIEW public.{table} AS
SELECT
  _gpay_id,
  _gpay_source_id,
  {business_columns},
  _synced_at     AS _gpay_sync_ts,
  _source        AS _gpay_source_engine,
  _source_ts     AS _gpay_source_ts,
  _version       AS _gpay_version,
  _deleted       AS _gpay_deleted,
  _created_at    AS _gpay_created_at,
  _updated_at    AS _gpay_updated_at
FROM cdc_internal.{table}
WHERE _deleted IS FALSE;

-- INSERT/UPDATE on view routes to underlying (since view is simple) — 
-- explicit INSTEAD OF triggers if view has JOINs (not our case).
```

---

## 3. Migration Phases (backward-compat mandatory)

### Phase 0 — Preparation (NO data change)

**0.1 Create `cdc_internal` schema**
```sql
CREATE SCHEMA IF NOT EXISTS cdc_internal;
-- Set search_path for migrations temporarily
```

**0.2 Install `next_sonyflake()` function + sequence**

**0.3 Install `tg_gpay_master_guard()` function (chưa attach)**

**0.4 Update Go Worker**:
- Add flag `v125_mode` in config (default false — legacy mode)
- When `v125_mode=true`: write to `cdc_internal.<table>`, allow `_gpay_id NULL` (trigger sinh) hoặc pass Sonyflake explicitly
- When `v125_mode=false` (current): unchanged

**Verify**: 
- Code build pass cả Worker + CMS
- Existing 9-field behavior unchanged
- `cdc_internal.next_sonyflake()` testable qua psql

### Phase 1 — Per-table migration (online, per-table tested)

Mỗi table migrate theo pattern:

**1.1 ADD `_gpay_id` column** (NULL allowed initially)
```sql
ALTER TABLE public.{table} ADD COLUMN IF NOT EXISTS _gpay_id BIGINT;
```

**1.2 Backfill `_gpay_id` cho existing rows**
```sql
UPDATE public.{table} 
SET _gpay_id = cdc_internal.next_sonyflake() 
WHERE _gpay_id IS NULL;
```
(With lock: `LOCK TABLE ... IN SHARE ROW EXCLUSIVE`)

**1.3 ADD `_gpay_source_id` column + backfill từ `_id` hoặc `source_id`**
```sql
ALTER TABLE public.{table} ADD COLUMN IF NOT EXISTS _gpay_source_id VARCHAR(200);
UPDATE public.{table} SET _gpay_source_id = COALESCE(_id::text, source_id);
```

**1.4 Add NOT NULL + PK constraint**
```sql
ALTER TABLE public.{table} 
  ALTER COLUMN _gpay_id SET NOT NULL,
  ADD CONSTRAINT pk_{table}_gpay UNIQUE (_gpay_id);
-- Don't drop old PK yet — keep both for now
```

**1.5 Add Partial Unique Index**
```sql
CREATE UNIQUE INDEX CONCURRENTLY IF NOT EXISTS idx_{table}_gpay_src_active
  ON public.{table} (_gpay_source_id)
  WHERE _deleted IS FALSE;
```

**1.6 Attach trigger** (Guard trigger set Sonyflake + source_id + version)
```sql
CREATE TRIGGER trg_{table}_gpay_guard
BEFORE INSERT OR UPDATE ON public.{table}
FOR EACH ROW EXECUTE FUNCTION cdc_internal.tg_gpay_master_guard();
```

**1.7 Move table to `cdc_internal` schema** (optional, can defer)
```sql
ALTER TABLE public.{table} SET SCHEMA cdc_internal;
```

**1.8 Create `public.{table}` view** (clean exposure)
```sql
CREATE OR REPLACE VIEW public.{table} AS
SELECT ... FROM cdc_internal.{table} WHERE _deleted IS FALSE;
```

**1.9 Update Worker config**: switch table to `v125_mode=true`

**1.10 Verify**: 
- SELECT count from view == count from internal (minus deleted)
- Sample 10 rows: `_gpay_id NOT NULL`, `_gpay_source_id NOT NULL`
- New write: trigger populates `_gpay_id` nếu Go Worker không set

**1.11 Drop old PK (sau khi verify stable)**
```sql
-- e.g., drop _id PK, make _gpay_id PRIMARY KEY instead
ALTER TABLE cdc_internal.{table} DROP CONSTRAINT {table}_pkey;
ALTER TABLE cdc_internal.{table} ADD PRIMARY KEY (_gpay_id);
```

### Phase 2 — Recon + FE updates

**2.1 Recon agent**:
- Source Agent: query Mongo dùng `_gpay_source_id` ↔ Mongo `_id` contract
- Dest Agent: query PG `cdc_internal.<table>._gpay_source_id` 
- Dual-engine recon: support `sync_engine='both'` với schema duy nhất (v1.25)

**2.2 FE DataIntegrity**:
- Display `_gpay_id` (Sonyflake BIGINT) thay `id`/`_id`
- Column `Source Engine` renders `_gpay_source_engine`
- Hide `_airbyte_*` + internal columns trên user-facing views

**2.3 CMS API**:
- Registry register table → auto run Phase 1 steps via NATS `cdc.cmd.migrate-to-v125`
- Response include v125_status per table

### Phase 3 — Cleanup + decommission legacy

**3.1 After all tables migrated + stable N days**:
- Drop `_id`, `id`, `source_id` old PKs if redundant
- Remove `v125_mode=false` branch in Worker
- Remove migration 001/002/003 `create_cdc_table()` function (replace với v1.25 variant)

---

## 4. Backward Compatibility Matrix

| Existing feature | Preserved? | How |
|:-----------------|:-----------|:----|
| 9 `_*` fields physical columns | ✅ Yes | Kept, aliased via VIEW |
| `_source_ts` OCC logic | ✅ Yes | Unchanged, trigger respect existing NEW._source_ts |
| Worker Go `idgen.NextID()` | ✅ Yes | Still generates ID, passed as `_gpay_id` explicit |
| Debezium Kafka consume path | ✅ Yes | Worker writes `_gpay_id` (from Go Sonyflake) + `_gpay_source_id=_id` |
| Airbyte bridge batch path | ✅ Yes | Worker writes `_gpay_id` (or NULL → trigger sinh) |
| Existing queries on `_id` | ⚠️ Via view alias | `public.<table>` view preserves old names during transition |
| Recon XOR hash semantic | ✅ Yes | `_gpay_source_id` replaces `_id` in hash input |
| Backfill service `_source_ts` | ✅ Yes | Unchanged |
| `sync_engine='both'` | ✅ Improved | Both paths converge on `_gpay_id` + `_gpay_source_id` |
| CMS DataIntegrity existing API | ✅ Yes | Schema unchanged; new fields are additive |

**Key principle**: **ADDITIVE migration**. Thêm `_gpay_*` → backfill → transition writes → later cleanup. Mỗi step riêng rollback-able.

---

## 5. Rollback Plan

| Phase | Rollback trigger | Action |
|:------|:-----------------|:-------|
| 0 Preparation | Function bug | `DROP FUNCTION next_sonyflake, tg_gpay_master_guard` — no data touched |
| 1.1-1.6 Per-table ADD | `_gpay_id` backfill fail | `ALTER TABLE DROP COLUMN _gpay_id, _gpay_source_id; DROP INDEX idx_*; DROP TRIGGER trg_*` — ADD COLUMN reversible |
| 1.7 Schema move | View breaks client | `ALTER TABLE SET SCHEMA public` — data untouched |
| 1.8 View creation | View query slow | `DROP VIEW public.<table>` + clients revert to direct table access |
| 1.10 Worker v125_mode | Write failures | Flip config `v125_mode=false` — revert to legacy path, existing rows still valid |
| 1.11 Drop old PK | Recon breaks | **BLOCKED until N days stable**. Only run this after all other phases verified |

**Maximum reversibility until Phase 1.11** (drop old PK). User controls timing.

---

## 6. Risks & Mitigations

| Risk | Impact | Mitigation |
|:-----|:-------|:-----------|
| PG + Go Sonyflake collision (same machineID) | Duplicate `_gpay_id` | worker_id=0 reserved PG; Go uses IP-derived ≥1; verify post-migration SELECT duplicates |
| Epoch mismatch PG ↔ Go | IDs out of order | Explicit constant `1477267200000` in both; unit test generate 100 IDs each side, sort compare |
| Clock skew PG vs Go pods | IDs go backward | NTP enforcement; Sonyflake tolerates 10ms drift; monitor `clock_diff` metric |
| Trigger fires 2x (insert + update) | Version double-bump | `TG_OP` branch correct (test coverage) |
| Existing queries with `WHERE _id = '...'` | Break after column drop | Keep `_id` column until Phase 3; view aliases; staged deprecation |
| Recon hash semantic differs | False positive drift | Unit test: hash 10 docs cũ vs mới — must match before Phase 1.9 cutover |
| CMS dashboards filtered `_source='airbyte'` | Still work | `_source` column preserved |
| Partial unique index blocks legitimate soft-delete reuse | Insert fail with same source_id | Test: delete row (set `_deleted=true`), insert new with same `_gpay_source_id` — must succeed |
| `_gpay_source_id` NULL for edge cases | Fail unique guard | Trigger fallback chain 4 tiers (NEW._id, _raw_data->>_id, _raw_data->>id, _raw_data->_id->>$oid); log warn + reject INSERT if all NULL |

---

## 7. Test Strategy

### 7.1 Unit tests (Worker + PG)
- `TestNextSonyflake_MonotonicSequence` (PG): 1000 calls → strictly increasing
- `TestNextSonyflake_NoCollisionWithGo` (cross): Go generates 1000 IDs (machineID=1), PG generates 1000 IDs (machineID=0) → union size = 2000
- `TestTriggerPopulates_GpayFields`: INSERT với `_gpay_id=NULL` → trigger sinh, row valid
- `TestTriggerIdempotent`: INSERT same row twice → second UPDATE, `_version++`
- `TestPartialUnique_SoftDelete`: insert+delete+insert same source_id → success (3 rows total, 1 active)
- `TestHashSemantic_Preserved`: compare `_hash` before/after migration on sample rows

### 7.2 Integration tests
- `TestDebeziumPath_v125`: Debezium message → Worker → INSERT với `_gpay_id` populated
- `TestAirbytePath_v125`: Airbyte bridge batch → Worker pass NULL → trigger sinh → row valid
- `TestBothPath_NoDuplicate`: Insert same row from Airbyte + Debezium → 1 row only, latest wins by `_source_ts`
- `TestReconAfterMigration`: Tier 1 count pre/post migration → equal
- `TestViewSelect_CleanFields`: `SELECT * FROM public.<table>` hide `_airbyte_*` + internal fields

### 7.3 Load tests
- 1M rows backfill on test table → measure time + DB load
- 10K/sec dual-path inserts → verify zero duplicate, unique index hit rate

---

## 8. Task Breakdown (for Muscle execution — awaits approval)

### Phase 0 — Foundation (2-3h)
- **T0.1**: Migration 018 — `cdc_internal` schema + `next_sonyflake()` + sequence + `tg_gpay_master_guard()`
- **T0.2**: Go Worker config flag `v125_mode` per-table
- **T0.3**: Unit tests PG + Go (cross-collision + monotonic)

### Phase 1 — Per-table migration harness (4-6h)
- **T1.1**: Migration function `migrate_table_to_v125(table_name)` — idempotent DDL sequence (ADD → backfill → constraint → trigger → view)
- **T1.2**: NATS command `cdc.cmd.migrate-to-v125` → Worker handler
- **T1.3**: CMS endpoint `POST /api/registry/:id/migrate-v125`
- **T1.4**: Per-table verify checks (count match, PK integrity, trigger fires)

### Phase 2 — Recon + FE (3-4h)
- **T2.1**: Source agent use `_gpay_source_id` (unified, not `_id` or `source_id`)
- **T2.2**: Dest agent query `cdc_internal.<table>` via schema config
- **T2.3**: FE DataIntegrity column rename `_gpay_*` + hide internal
- **T2.4**: CMS registry create path → auto Phase 1 for new tables

### Phase 3 — Cleanup (defer 2 weeks)
- **T3.1**: Drop old PK columns
- **T3.2**: Retire `create_cdc_table()` old function
- **T3.3**: Remove `v125_mode=false` legacy branch

### Total effort: ~10-14h (Phases 0-2), Phase 3 = 1-2h later

---

## 9. Decision Required Before Execute

User confirm trước khi Muscle start:

1. **Schema strategy**: VIEW-based alias (ADDITIVE, recommended) hay RENAME columns (breaking, fewer columns)?
2. **Worker ID 0 reservation**: OK dành cho PG? (Go pods unlikely collision — cluster IP scheme có `.0.0` IP không?)
3. **Rollout timing**: Migrate tất cả 8 tables batch hay 1-by-1 với stability verification giữa?
4. **Old PK drop timeline**: N days = 7, 14, 30? (Trade-off risk vs cleanup)
5. **Airbyte activation**: Trong Phase 1 có enable Airbyte bridge schedule không? Hay giữ Debezium-only cho đến sau migration?
6. **Registry label correction**: 6 tables mislabeled `sync_engine='airbyte'` — update sang `'debezium'` TRƯỚC khi migrate (true state) hay GIỮ label và fix Airbyte integration trong Phase 2?

---

## 10. Anti-patterns Rejected

- ❌ Rename columns trực tiếp (`ALTER TABLE ... RENAME COLUMN _id TO _gpay_source_id`) — breaks 200+ callsites
- ❌ Drop 9 `_*` fields — recon OCC depends on `_source_ts`, `_hash`, `_version`
- ❌ Single migration all-or-nothing — phải reversible per table
- ❌ Force Sonyflake PG-only — Go Worker path works, no need rewrite
- ❌ Skip partial unique index — duplicates can sneak in during double-write race

---

## 11. Success Criteria (Definition of Done)

- [ ] All 8 active tables migrated với `_gpay_id` populated 100%
- [ ] All 8 tables có partial unique index active
- [ ] `next_sonyflake()` PG + Go coexist, 0 collisions in 10K × 2 test
- [ ] Existing 9 field queries unchanged (recon, heal, backfill OK)
- [ ] FE DataIntegrity displays `_gpay_*` columns
- [ ] Recon Tier 1/2/3 pass with new schema
- [ ] `sync_engine='both'` works without duplicate (test dual-path write)
- [ ] Rollback tested on 1 table successfully
- [ ] Zero production incidents post-migration 7 days
- [ ] Docs updated in `agent/memory/workspaces/feature-cdc-integration/`

---

## 12. Related documents

- `02_plan_data_integrity_v3.md` — recon strategy (preserved)
- `04_decisions_recon_systematic_v4.md` — auto-detect timestamp (preserved)
- Migration 001/003/009/016/017 — existing schema layers
- Lesson #1 Scale Budget, #60 ADR passive, #64 Cross-store hash, #65 Per-entity band-aid — applied

---

## 13. Out of scope (future iteration)

- Multi-region Sonyflake (different epoch per region)
- Horizontal partition by `_gpay_id` high bits (time-based partitioning)
- Sonyflake ID → external system exports (Kafka outbox with `_gpay_id` as key)
- ML-based anomaly detection on `_gpay_id` distribution

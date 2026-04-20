# Master Plan — Unified Sonyflake Architecture v1.25 RECONSTRUCTION

> **Date**: 2026-04-20 (revised)
> **Author**: Brain (claude-opus-4-7)
> **Supersedes**: `02_plan_sonyflake_v125_unified.md` (v1 — rejected by user: passive band-aid)
> **Mode**: **RECONSTRUCTION**, not migration. Physical clean slate + Forced cutover + Single Identity Authority.
> **Reference**: Lesson #67 (band-aid vs reconstruction), User critique 6 points 2026-04-20

---

## 0. User's 6 rejected design patterns (v1 plan)

| # | v1 flaw | User critique | v2 correction |
|:--|:--------|:--------------|:--------------|
| 1 | VIEW alias ẩn `_airbyte_*` rác | Physical layer vẫn chứa rác → VACUUM/Backup chậm | **Bóc tách** business fields thành columns thật, DROP `_airbyte_*` physically |
| 2 | `IF NEW._gpay_id IS NULL` Trigger | Go Worker pass sai ID → DB chấp nhận lỗi | **FORCE DB sinh cho Airbyte path**. Go Debezium path: **STRICT VALIDATE** format + machineID range |
| 3 | `_gpay_sync_ts` → alias `_synced_at` | Spaghetti naming | **Drop `_synced_at`, `_source`, etc.** Rename physical columns sang `_gpay_*` uniform |
| 4 | `COALESCE(debezium_ts, _ab_cdc_updated_at)` | Quên OCC `_source_ts` migration 009 | `_source_ts` → rename `_gpay_source_ts`, OCC guard `WHERE _gpay_source_ts < EXCLUDED` **preserved** |
| 5 | Giữ PK cũ đến Phase 3 | Dual-index phình IO | **DROP old PK trong cùng migration transaction**. Single-PK aggressive cutover |
| 6 | Worker ID 0 mặc định | Không verify Go machineID range | **Phase -1**: inspect Go IP range cluster → allocate PG worker_id ≥ max(go_range)+1 |

---

## 1. Reconstruction Principles

1. **Physical Clean Slate**: `cdc_internal.<table>` chỉ có columns business + `_gpay_*`. Zero `_airbyte_*`, zero `_ab_cdc_*`, zero `_id`/`source_id` legacy.
2. **Single Identity Authority**: 
   - **Airbyte path** → DB trigger FORCE sinh `_gpay_id` (Go Worker không handle Airbyte).
   - **Debezium path** → Go Worker sinh `_gpay_id`, DB VALIDATE (reject format/range invalid).
3. **Uniform Naming**: Mọi metadata prefix `_gpay_*`. Không alias. Không VIEW hiding.
4. **Preserved Earned Patterns**: `_source_ts` OCC (migration 009) → rename `_gpay_source_ts`, giữ semantic + WHERE guard.
5. **Aggressive Cutover**: DROP old PK + old metadata columns cùng transaction migration. Không dual-PK, không defer.
6. **Environment Verified**: Go Worker IP range queried **pre-design**, không assumption.

---

## 2. Target Schema — `cdc_internal.<table>` (v1.25 pure)

```sql
CREATE TABLE cdc_internal.<table> (
  -- IDENTITY (v1.25 _gpay_* prefix, no alias)
  _gpay_id            BIGINT PRIMARY KEY,             -- Sonyflake (Go OR PG-generated, validated)
  _gpay_source_id     VARCHAR(200) NOT NULL,          -- Mongo _id or business PK, anchor for upsert
  
  -- LINEAGE + TIMING (v1.25 replacing legacy _*)
  _gpay_source_engine VARCHAR(20) NOT NULL,           -- 'airbyte' | 'debezium' (no 'both' since single-path per table)
  _gpay_source_ts     BIGINT,                         -- Debezium oplog ms epoch; Airbyte path NULL → fallback timestamp_field
  _gpay_sync_ts       TIMESTAMPTZ NOT NULL DEFAULT NOW(), -- DW write time
  _gpay_created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(), -- Immutable
  _gpay_updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(), -- Refreshed on upsert
  
  -- INTEGRITY (v1.25)
  _gpay_raw_data      JSONB NOT NULL DEFAULT '{}'::jsonb,  -- Full source payload (POST-strip airbyte metadata)
  _gpay_hash          VARCHAR(64),                    -- SHA256(_gpay_raw_data) for content dedup
  _gpay_version       BIGINT NOT NULL DEFAULT 1,      -- OCC counter
  _gpay_deleted       BOOLEAN NOT NULL DEFAULT FALSE, -- Soft delete
  
  -- BUSINESS FIELDS (extracted from _gpay_raw_data → typed columns)
  {business_col_1}    {type},
  {business_col_2}    {type},
  ...
  
  -- Enforcement
  CONSTRAINT chk_source_engine CHECK (_gpay_source_engine IN ('airbyte','debezium'))
);

-- Hard unique: active row per source_id
CREATE UNIQUE INDEX idx_<table>_gpay_src_active
  ON cdc_internal.<table> (_gpay_source_id)
  WHERE _gpay_deleted IS FALSE;

-- OCC guard helper index
CREATE INDEX idx_<table>_gpay_src_ts
  ON cdc_internal.<table> (_gpay_source_ts)
  WHERE _gpay_source_ts IS NOT NULL;

-- Query index
CREATE INDEX idx_<table>_gpay_updated_at
  ON cdc_internal.<table> (_gpay_updated_at DESC);
```

**ZERO**:
- `_id`, `id`, `source_id` (old PKs dropped)
- `_airbyte_ab_id`, `_airbyte_extracted_at`, `_airbyte_loaded_at`, `_airbyte_data_hash`, `_ab_cdc_*`
- `_raw_data`, `_source`, `_synced_at`, `_source_ts`, `_version`, `_hash`, `_deleted`, `_created_at`, `_updated_at` (legacy 9 fields — all renamed to `_gpay_*`)

**Business fields** extracted vào typed columns TẠI migration time, không để trong `_gpay_raw_data` (raw là audit only, queries use typed columns).

---

## 3. Identity Provider — Strict Authority

### 3.1 PG `next_sonyflake()` — bit layout exact match Go sonyflake

**Pre-design verification (Phase -1)**:
```bash
# Query Go Worker runtime log để biết machineID đã dùng
grep "sonyflake initialized" /tmp/worker*.log
# Output: "machineID":2622,"ip":"192.168.10.62"
# → Current machineID = 2622 (from IP octet2*256 + octet3 = 10*256 + 62 = 2622)

# Determine IP range allocation
# Cluster subnet? If 192.168.10.0/24 → machineID ∈ [2560-2815]
# PG worker_id must be OUTSIDE Go range
```

**Rule**: PG `worker_id` = **value NOT in Go cluster IP range**. Safest: reserve range **0-999** cho PG (assuming Go IPs always have high octet2 in private subnet 10.*, 172.16-31.*, 192.168.*).

**Verified for this deployment**: Go machineID=2622 → PG can use 0-2559 or 2816+. Plan **use PG worker_id = 1** (not 0, để reserve 0 cho sentinel/test).

### 3.2 Function

```sql
CREATE SEQUENCE cdc_internal.sonyflake_seq MINVALUE 0 MAXVALUE 255 CYCLE START 0;

CREATE OR REPLACE FUNCTION cdc_internal.next_sonyflake() RETURNS bigint AS $$
DECLARE
  -- EXACT epoch match Go Sony Sonyflake default 2016-10-21 UTC
  our_epoch  bigint := 1477267200000;
  elapsed    bigint;
  seq_id     bigint;
  -- PG allocated machineID, verified OUT OF Go range at Phase -1
  worker_id  int    := 1;
  result     bigint;
BEGIN
  elapsed := ((extract(epoch FROM clock_timestamp()) * 1000)::bigint - our_epoch) / 10;
  
  -- Fail-fast if clock regresses below epoch
  IF elapsed < 0 THEN
    RAISE EXCEPTION 'Clock before epoch 2016-10-21 — NTP sync broken';
  END IF;
  
  seq_id := nextval('cdc_internal.sonyflake_seq') % 256;
  
  -- Bit layout: 39-bit time | 8-bit seq | 16-bit machineID
  result := (elapsed << 24) | ((seq_id & 255) << 16) | (worker_id & 65535);
  RETURN result;
END;
$$ LANGUAGE plpgsql VOLATILE;
```

### 3.3 Validation function — Strict Go-provided ID check

```sql
CREATE OR REPLACE FUNCTION cdc_internal.validate_sonyflake(sf bigint) RETURNS void AS $$
DECLARE
  -- Go worker IDs allowed range (queried Phase -1)
  go_min_machine_id int := 2560;
  go_max_machine_id int := 2815;
  extracted_machine int;
  extracted_time    bigint;
BEGIN
  IF sf IS NULL OR sf < 0 THEN
    RAISE EXCEPTION 'Invalid sonyflake: NULL or negative';
  END IF;
  
  extracted_machine := (sf & 65535);
  extracted_time    := sf >> 24;
  
  -- Machine ID must be either PG's (1) or Go's allocated range
  IF extracted_machine != 1 
     AND (extracted_machine < go_min_machine_id OR extracted_machine > go_max_machine_id) THEN
    RAISE EXCEPTION 'Sonyflake machineID % out of allowed ranges [PG=1] or [Go=%..% ]', 
      extracted_machine, go_min_machine_id, go_max_machine_id;
  END IF;
  
  -- Timestamp must be in reasonable range (not future > 1 day, not before 2016)
  IF extracted_time < 0 OR extracted_time > ((extract(epoch FROM clock_timestamp()) * 1000 - 1477267200000) / 10 + 8640000) THEN
    RAISE EXCEPTION 'Sonyflake timestamp out of reasonable range';
  END IF;
END;
$$ LANGUAGE plpgsql IMMUTABLE;
```

### 3.4 Universal Guard Trigger — Force Authority

```sql
CREATE OR REPLACE FUNCTION cdc_internal.tg_gpay_guard()
RETURNS TRIGGER AS $$
BEGIN
  -- === IDENTITY AUTHORITY ===
  IF NEW._gpay_source_engine IS NULL THEN
    RAISE EXCEPTION '_gpay_source_engine required (airbyte|debezium)';
  END IF;
  
  CASE NEW._gpay_source_engine
    WHEN 'airbyte' THEN
      -- Airbyte path: DB FORCE sinh (Go Worker không touch Airbyte)
      -- ID từ Go = error (rejected)
      IF NEW._gpay_id IS NOT NULL THEN
        RAISE EXCEPTION 'Airbyte path: _gpay_id must be NULL (DB-generated only)';
      END IF;
      NEW._gpay_id := cdc_internal.next_sonyflake();
      
    WHEN 'debezium' THEN
      -- Debezium path: Go Worker sinh + DB validate STRICT
      IF NEW._gpay_id IS NULL THEN
        RAISE EXCEPTION 'Debezium path: _gpay_id required (Go Worker must provide)';
      END IF;
      PERFORM cdc_internal.validate_sonyflake(NEW._gpay_id);
  END CASE;
  
  -- === SOURCE_ID ANCHOR ===
  IF NEW._gpay_source_id IS NULL THEN
    NEW._gpay_source_id := COALESCE(
      (NEW._gpay_raw_data->>'_id'),
      (NEW._gpay_raw_data->'_id'->>'$oid'),
      (NEW._gpay_raw_data->>'id')
    );
    IF NEW._gpay_source_id IS NULL THEN
      RAISE EXCEPTION '_gpay_source_id cannot be derived from _gpay_raw_data (no _id, _id.$oid, or id field)';
    END IF;
  END IF;
  
  -- === TIMING ===
  NEW._gpay_updated_at := CURRENT_TIMESTAMP;
  IF TG_OP = 'INSERT' THEN
    NEW._gpay_sync_ts := COALESCE(NEW._gpay_sync_ts, CURRENT_TIMESTAMP);
    NEW._gpay_created_at := COALESCE(NEW._gpay_created_at, CURRENT_TIMESTAMP);
    NEW._gpay_version := COALESCE(NEW._gpay_version, 1);
  ELSIF TG_OP = 'UPDATE' THEN
    NEW._gpay_version := COALESCE(OLD._gpay_version, 0) + 1;
    -- Immutable fields
    NEW._gpay_created_at := OLD._gpay_created_at;
    NEW._gpay_id := OLD._gpay_id;  -- Never mutate PK
    NEW._gpay_source_id := OLD._gpay_source_id;  -- Never mutate anchor
  END IF;
  
  -- === INTEGRITY ===
  IF NEW._gpay_hash IS NULL THEN
    NEW._gpay_hash := encode(digest(NEW._gpay_raw_data::text, 'sha256'), 'hex');
  END IF;
  
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;
```

**Không có** `IF NULL fallback` cho identity — **explicit REJECT** wrong input.

---

## 4. OCC — Earned Preservation (`_source_ts` → `_gpay_source_ts`)

Migration 009 OCC pattern **unchanged semantically**, chỉ rename physical column:

```sql
-- BEFORE v1.25:
INSERT INTO tbl (...) VALUES (...)
ON CONFLICT (_id) DO UPDATE SET ...
WHERE tbl._source_ts IS NULL OR tbl._source_ts < EXCLUDED._source_ts;

-- AFTER v1.25 (identical semantic, new names):
INSERT INTO cdc_internal.<table> (...) VALUES (...)
ON CONFLICT (_gpay_source_id) DO UPDATE SET ...
WHERE cdc_internal.<table>._gpay_source_ts IS NULL 
   OR cdc_internal.<table>._gpay_source_ts < EXCLUDED._gpay_source_ts;
```

**KHÔNG** COALESCE Airbyte timestamp với Debezium — mỗi table single-path engine, không cross-engine race.

**Anti-Ghosting** được xử lý ở tầng REGISTRY level: `sync_engine` enforce single value per table. Không có dual-write race cần resolve ở row level.

---

## 5. Phase Plan — Aggressive Cutover

### Phase -1: Environment Verification (user-blocking)

Output `10_gap_analysis_sonyflake_environment.md`:
- Go Worker machineID values observed (grep worker logs)
- Cluster IP subnet(s)
- Reserved PG worker_id (out of Go range)
- Existing row count per table
- Business columns vs raw data field mapping

### Phase 0: Foundation

- Migration 018_sonyflake_v125_foundation.sql:
  - `CREATE SCHEMA cdc_internal`
  - `CREATE FUNCTION next_sonyflake, validate_sonyflake, tg_gpay_guard`
  - `CREATE SEQUENCE sonyflake_seq`

### Phase 1: Per-Table Reconstruction (for each of 8 tables)

**1 transaction per table** — all-or-nothing:

```sql
BEGIN;

-- 1. Create new table với v1.25 schema pure
CREATE TABLE cdc_internal.<table>_v125 (
  _gpay_id BIGINT PRIMARY KEY,
  _gpay_source_id VARCHAR(200) NOT NULL,
  _gpay_source_engine VARCHAR(20) NOT NULL,
  _gpay_source_ts BIGINT,
  _gpay_sync_ts TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  _gpay_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  _gpay_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  _gpay_raw_data JSONB NOT NULL DEFAULT '{}'::jsonb,
  _gpay_hash VARCHAR(64),
  _gpay_version BIGINT NOT NULL DEFAULT 1,
  _gpay_deleted BOOLEAN NOT NULL DEFAULT FALSE,
  {business_cols_typed},
  CONSTRAINT chk_engine CHECK (_gpay_source_engine IN ('airbyte','debezium'))
);

-- 2. Copy data từ public.<table> sang cdc_internal.<table>_v125
--    - Generate _gpay_id cho existing rows (từ old PK nếu là BIGINT, else sinh mới)
--    - Map: _id → _gpay_source_id
--    - Map: _source_ts → _gpay_source_ts
--    - Map: _source → _gpay_source_engine
--    - Strip _airbyte_* from _raw_data → _gpay_raw_data
--    - Extract business fields từ _raw_data → typed columns
INSERT INTO cdc_internal.<table>_v125 (
  _gpay_id, _gpay_source_id, _gpay_source_engine, _gpay_source_ts,
  _gpay_sync_ts, _gpay_created_at, _gpay_updated_at,
  _gpay_raw_data, _gpay_hash, _gpay_version, _gpay_deleted,
  {business_cols}
)
SELECT
  CASE 
    WHEN _source = 'airbyte' THEN cdc_internal.next_sonyflake()
    WHEN _source = 'debezium' AND id IS NOT NULL AND id > 0 THEN id  -- keep Go-generated if exists
    ELSE cdc_internal.next_sonyflake()  -- fallback
  END,
  COALESCE(_id::text, source_id, id::text),
  _source,
  _source_ts,
  _synced_at,
  _created_at,
  _updated_at,
  _raw_data - ARRAY['_airbyte_ab_id','_airbyte_extracted_at','_airbyte_loaded_at','_airbyte_data_hash','_ab_cdc_lsn','_ab_cdc_updated_at','_ab_cdc_deleted_at'],
  _hash,
  _version,
  _deleted,
  {business_col_extractions}
FROM public.<table>;

-- 3. Indexes
CREATE UNIQUE INDEX idx_<table>_gpay_src_active
  ON cdc_internal.<table>_v125 (_gpay_source_id)
  WHERE _gpay_deleted IS FALSE;
CREATE INDEX idx_<table>_gpay_src_ts
  ON cdc_internal.<table>_v125 (_gpay_source_ts)
  WHERE _gpay_source_ts IS NOT NULL;
CREATE INDEX idx_<table>_gpay_updated_at
  ON cdc_internal.<table>_v125 (_gpay_updated_at DESC);

-- 4. Trigger
CREATE TRIGGER trg_<table>_gpay_guard
  BEFORE INSERT OR UPDATE ON cdc_internal.<table>_v125
  FOR EACH ROW EXECUTE FUNCTION cdc_internal.tg_gpay_guard();

-- 5. Verify count match
DO $$
DECLARE old_cnt bigint; new_cnt bigint;
BEGIN
  SELECT COUNT(*) INTO old_cnt FROM public.<table>;
  SELECT COUNT(*) INTO new_cnt FROM cdc_internal.<table>_v125;
  IF old_cnt != new_cnt THEN
    RAISE EXCEPTION 'Row count mismatch: old=%, new=%', old_cnt, new_cnt;
  END IF;
END $$;

-- 6. Atomic swap: rename old → _legacy, new → canonical
ALTER TABLE public.<table> RENAME TO <table>_legacy;
ALTER TABLE cdc_internal.<table>_v125 RENAME TO <table>;
ALTER TABLE cdc_internal.<table> SET SCHEMA public;  -- cdc_internal.<table> accessible via public.<table>
-- Actually: keep in cdc_internal, CMS + Worker reference cdc_internal.<table> directly

COMMIT;
```

**Post-Phase 1**: `public.<table>_legacy` exists for 24h rollback window; `cdc_internal.<table>` is canonical.

### Phase 2: Worker + CMS code update

- **Worker kafka_consumer.go**: write target `cdc_internal.<table>` với `_gpay_*` columns, `_gpay_source_engine='debezium'`, Go sinh `_gpay_id`
- **Worker bridge_batch.go**: write target `cdc_internal.<table>`, `_gpay_source_engine='airbyte'`, **_gpay_id = NULL** (trigger sinh)
- **Recon**: source_id = `_gpay_source_id`, OCC column = `_gpay_source_ts`
- **CMS API**: response columns rename
- **FE**: display `_gpay_*`, hide/remove `_id`/`source_id` legacy references

### Phase 3: Drop legacy (after 7-day stability)

- DROP public.<table>_legacy
- Drop migration 001/002/003 `create_cdc_table()` function (replaced by v1.25)
- Remove all code references to `_synced_at`, `_source`, `_raw_data`, etc.

---

## 6. Anti-band-aid enforcement

### What v2 REMOVES compared to v1

| v1 Feature | v2 Action | Reason |
|:-----------|:----------|:-------|
| `public.<table>` VIEW aliasing | **REMOVED** — canonical table in `cdc_internal` | View = hiding, not cleaning |
| Alias `_gpay_sync_ts ↔ _synced_at` | **REMOVED** — physical rename `_synced_at` → `_gpay_sync_ts` | Single naming |
| `IF NULL fallback` in Trigger | **REMOVED** — explicit REJECT with EXCEPTION | Force authority |
| Keep old PK 7-30 days | **REMOVED** — dropped in Phase 1 same transaction | No dual-index waste |
| Worker ID 0 default | **REMOVED** — Phase -1 verify then allocate ID=1 | No assumption |
| COALESCE debezium/airbyte timestamp | **REMOVED** — single-path per table, no race | OCC via `_gpay_source_ts` only |
| Keep `_airbyte_*` in `_raw_data` | **REMOVED** — stripped in copy INSERT | Physical clean |

---

## 7. 6 User Decisions (revised, state-verified)

1. **PG worker_id allocation**: Phase -1 queries Go IP range. **Propose: worker_id=1** (Go uses 2560-2815 observed). Confirm?
2. **Business columns extraction**: plan per-table SELECT extraction from `_raw_data`. Brain produce list per-table in Phase -1 output?
3. **Drop legacy timeline**: 24h (aggressive) or 7d (standard safety)?
4. **Registry label correction**: 6 mislabeled `sync_engine='airbyte'` → flip to `debezium` **IN Phase 1** (source of truth) hay Phase 2 code update?
5. **Airbyte activation**: Phase 2 OR defer Phase 4 (after system stable on Debezium-only)?
6. **Rollout order**: 8 tables parallel (single migration script) hay sequential (lower risk, 3-4h longer)?

---

## 8. Rollback Plan (Phase 1 transaction boundary)

| Failure point | Rollback |
|:--------------|:---------|
| COPY INSERT fail | Transaction ROLLBACK — no change |
| Count mismatch | Transaction ROLLBACK |
| Trigger test fail post-rename | Manual: `ALTER TABLE public.<table>_legacy RENAME TO <table>; ALTER TABLE cdc_internal.<table> RENAME TO <table>_failed; DROP cdc_internal.<table>_failed` |
| Worker code fail Phase 2 | Worker config flag `v125_mode=false` → write back to `public.<table>_legacy` (stored 24h) |

Phase 1 failure window: 24h (legacy retained). After → full commit, only roll-forward.

---

## 9. Success Criteria

- [ ] 8 tables in `cdc_internal` with `_gpay_*` schema only
- [ ] `public.<table>` does NOT exist (no VIEW, no alias — canonical is cdc_internal)
- [ ] `_id`, `source_id`, `_synced_at`, `_source`, `_raw_data`, `_version`, `_hash`, `_deleted`, `_created_at`, `_updated_at`, `_source_ts` — **all physically dropped**
- [ ] `_airbyte_*` columns gone from `_gpay_raw_data` JSONB
- [ ] `_gpay_id` 100% populated via `next_sonyflake()` (PG) or Go `idgen.NextID()` (Worker)
- [ ] Trigger rejects invalid input (test: INSERT with Airbyte path + preset _gpay_id → EXCEPTION)
- [ ] Partial unique index prevents soft-delete duplicate (test: active row per source_id = exactly 1)
- [ ] OCC via `_gpay_source_ts` preserves recon drift detection (test: 1 drift scenario pre/post, identical result)
- [ ] Recon Tier 1/2/3 pass with `cdc_internal.<table>` target
- [ ] FE DataIntegrity shows `_gpay_*` columns, hides legacy
- [ ] 0 references to `_id`/`source_id` in Worker + CMS + FE codebase (grep audit)
- [ ] Load test 10K/sec inserts with 0 collision (Go + PG Sonyflake)

---

## 10. Out of scope

- `sync_engine='both'` dual-path — 0 tables have this state, defer until customer demands
- Multi-region Sonyflake (different epoch)
- JSONB → typed column auto-migration tool (manual per-table in Phase 1)

---

## 11. Related Lessons Applied

- #1 Scale Budget — 200+ tables target
- #60 ADR passive — this v2 replaces passive v1
- #64 Cross-store hash — PG + Go Sonyflake byte-exact
- #65 Per-entity band-aid — avoided via batch migration script
- #67 Passive vs Reconstruction — v2 IS reconstruction

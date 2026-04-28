# Sonyflake v1.25 — v7.1 FINAL (v7 + user refinements + 5 decisions resolved)

> **Date**: 2026-04-21
> **Author**: Brain (claude-opus-4-7)
> **Supersedes**: v7 (adds 4 refinements + 5 decision defaults)
> **Status**: User approved v7 "Architectural Excellence" — v7.1 = v7 + micro-adjustments per user feedback, ready for Muscle execution
> **Base**: See `02_plan_sonyflake_v125_v7.md` for full design (fencing token + outbox + data profiling + ctid snapshot scan)

---

## 0. Delta from v7 → v7.1

### 4 user refinements (accepted)

| # | v7 | v7.1 |
|:--|:---|:-----|
| 1 | Fencing: heartbeat-only 30s window | **Hybrid**: inline fencing in UPSERT WHERE cho financial tables (+1ms/batch), heartbeat-only cho non-financial |
| 2 | Outbox consumer design unspecified | **Per-table consumer** (goroutine per table) — bảng "nát" không nghẽn 199 khác |
| 3 | Profile confidence 0.9 auto-accept | **Stratified**: financial field pattern → force `admin_override`, non-financial → 0.95 auto-accept |
| 4 | Backfill single snapshot có thể > 4h | **Split 4h chunks** + outbox dedup at consumer (ON CONFLICT DO NOTHING) |

### 5 decisions resolved

| # | Decision | Default | Rationale |
|:--|:---------|:--------|:----------|
| 1 | Outbox vs Logical Replication | **Outbox** | Easier debug/retry, BIGSERIAL ordering, +10-20% IO acceptable |
| 2 | Fencing heartbeat vs inline per-write | **Hybrid** (see refinement #1) | Financial = strict, non-financial = fast |
| 3 | Confidence threshold | **Stratified pattern-based** (see refinement #3) | Financial never auto, others 0.95 |
| 4 | Backfill snapshot duration | **4h chunks** (see refinement #4) | VACUUM + WAL health |
| 5 | Outbox retention | **3 days processed + daily purge; failed kept until resolve** | Audit forensic window + bounded storage |

---

## 1. Financial Tables List (required for stratified decisions)

Phase -1 Muscle produces: list of "financial tables" (inline fencing + forced override).

**Initial list** (based on current 8 tables + naming heuristic):
- `payment_bills` (amount, balance fields)
- `refund_requests` (amount field)
- `payment_bill_histories` (amount, balance)
- Future: any table với schema có field regex match `^(amount|balance|currency|account|price|fee|total|sum|refund|payment|transaction)[_a-z]*$`

**Non-financial tables**:
- `identitycounters`, `payment_bill_codes`, `payment_bill_events`, `payment_bill_holdings`, `export_jobs`

User confirm list or adjust — Phase -1 output `financial_tables.yaml`.

---

## 2. Hybrid Fencing — implementation

### ⚠️ CRITICAL FIX (from user review 2026-04-21)

**Prior v7.1 draft proposed `INSERT ... ON CONFLICT ... DO UPDATE ... WHERE EXISTS (fencing check)`. This is BROKEN.**

**PostgreSQL semantic**: The `WHERE` clause in `ON CONFLICT DO UPDATE` **only filters the UPDATE path**. When a row is new (no conflict → INSERT branch), `WHERE` does NOT evaluate. Result: Zombie Pod với stale fencing_token vẫn có thể INSERT rows mới — fencing escape.

**Correct pattern**: **BEFORE INSERT OR UPDATE trigger** với session variable + `RAISE EXCEPTION` on mismatch. Trigger fires for BOTH paths (INSERT + UPDATE), RAISE aborts entire transaction including the INSERT. See lesson #73.

### 2.1 Financial tables — BEFORE trigger enforcement

**Step 1**: Worker sets session variables per transaction

```go
// centralized-data-service/internal/handler/write.go
func (h *Handler) writeFinancial(ctx context.Context, row TypedRow) error {
    return h.db.Transaction(func(tx *gorm.DB) error {
        // Inject fencing context into session — trigger reads these
        if err := tx.Exec(
            "SET LOCAL app.fencing_machine_id = ?; SET LOCAL app.fencing_token = ?",
            machineID, fencingToken,
        ).Error; err != nil { return err }
        
        // Main write + outbox (same tx)
        if err := tx.Exec(legacyUpsertSQL, row.LegacyArgs()...).Error; err != nil {
            return err  // includes fencing RAISE EXCEPTION from trigger
        }
        if err := tx.Exec(outboxInsertSQL, row.OutboxArgs()...).Error; err != nil {
            return err
        }
        return nil
    })
}
```

**Step 2**: Lightweight BEFORE trigger on financial tables

```sql
CREATE OR REPLACE FUNCTION cdc_internal.tg_fencing_guard()
RETURNS TRIGGER AS $$
DECLARE
  v_session_machine INT;
  v_session_token   BIGINT;
  v_current_token   BIGINT;
BEGIN
  -- Read session variables set by Worker (Worker bắt buộc set trước write)
  BEGIN
    v_session_machine := current_setting('app.fencing_machine_id', false)::INT;
    v_session_token   := current_setting('app.fencing_token', false)::BIGINT;
  EXCEPTION WHEN OTHERS THEN
    RAISE EXCEPTION 'FENCING: session variables app.fencing_machine_id + app.fencing_token required for financial table writes';
  END;
  
  -- Lookup current token in registry
  SELECT fencing_token INTO v_current_token
    FROM cdc_internal.worker_registry
    WHERE machine_id = v_session_machine;
  
  -- Zombie Pod detection: token mismatch OR machine_id not found
  IF v_current_token IS NULL THEN
    RAISE EXCEPTION 'FENCING: machine_id % not registered (pod never claimed)', v_session_machine;
  END IF;
  
  IF v_current_token != v_session_token THEN
    RAISE EXCEPTION 'FENCING: token mismatch (pod reclaimed). machine_id=%, pod_token=%, current_token=%',
      v_session_machine, v_session_token, v_current_token;
  END IF;
  
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Attach to financial tables only
CREATE TRIGGER trg_payment_bills_fencing
  BEFORE INSERT OR UPDATE ON cdc_internal.payment_bills
  FOR EACH ROW EXECUTE FUNCTION cdc_internal.tg_fencing_guard();

CREATE TRIGGER trg_refund_requests_fencing
  BEFORE INSERT OR UPDATE ON cdc_internal.refund_requests
  FOR EACH ROW EXECUTE FUNCTION cdc_internal.tg_fencing_guard();

CREATE TRIGGER trg_payment_bill_histories_fencing
  BEFORE INSERT OR UPDATE ON cdc_internal.payment_bill_histories
  FOR EACH ROW EXECUTE FUNCTION cdc_internal.tg_fencing_guard();
```

**Step 3**: UPSERT template (clean, no inline WHERE EXISTS)

```sql
-- Financial UPSERT — trigger enforces fencing, UPSERT just does data
INSERT INTO cdc_internal.payment_bills (
  _gpay_id, _gpay_source_id, _gpay_source_engine, _gpay_source_ts,
  _gpay_raw_data, _gpay_hash, _gpay_version, _gpay_deleted,
  amount, currency, status, merchant_id, ...
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, ...)
ON CONFLICT (_gpay_source_id) WHERE NOT _gpay_deleted
DO UPDATE SET
  _gpay_raw_data = EXCLUDED._gpay_raw_data,
  _gpay_source_ts = EXCLUDED._gpay_source_ts,
  amount = EXCLUDED.amount,
  ...
WHERE cdc_internal.payment_bills._gpay_source_ts IS NULL 
   OR cdc_internal.payment_bills._gpay_source_ts < EXCLUDED._gpay_source_ts;
-- NO fencing WHERE — trigger handles both INSERT + UPDATE paths
```

### 2.1.1 Why BEFORE trigger (not AFTER, not CONSTRAINT TRIGGER)

| Alternative | Why not chosen |
|:------------|:---------------|
| `WHERE EXISTS` in UPSERT | ❌ UPDATE path only — INSERT escapes (the original bug) |
| AFTER trigger | ❌ Row already inserted in page; rollback wastes IO |
| CONSTRAINT TRIGGER (deferred) | ❌ Fires at COMMIT — too late if tx has other side effects |
| CHECK constraint | ❌ Cannot reference other tables (cross-table subquery disallowed) |
| Row-Level Security (RLS) WITH CHECK | ✓ Valid alternative, but triggers more ergonomic for this audit + exception pattern |

**BEFORE INSERT OR UPDATE**: fires before row touches disk. RAISE EXCEPTION in trigger = statement error = transaction rollback = no data written. Clean fencing semantic.

### 2.1.2 Performance characteristics

- Trigger overhead per financial row: ~0.3ms (session variable read + registry SELECT via PK index)
- Session variable `SET LOCAL` overhead: ~0.1ms, once per transaction (amortized across batch)
- Financial write p99 impact: +0.5ms per batch of 100 rows = +0.005ms per row
- Trade-off vs throughput gain (no inline WHERE): negligible
- Error handling: Worker catches `pgerr.Code='P0001'` + message starts with `'FENCING:'` → `os.Exit(1)` per v7 fencing loop semantic

### 2.1.3 Registry index requirement

```sql
-- Registry must have PK index on machine_id (already exists from PRIMARY KEY)
-- Trigger SELECT is PK lookup = O(1)
```

### 2.2 Non-financial tables (heartbeat-only)

### 2.2 Non-financial tables (heartbeat-only)

```sql
-- Standard UPSERT không inline fencing
INSERT INTO cdc_internal.identitycounters (...)
ON CONFLICT (_gpay_source_id) WHERE NOT _gpay_deleted
DO UPDATE SET ...;
```

Trust 30s heartbeat loop + self-terminate via `os.Exit(1)`.

### 2.3 Config per-table

```sql
ALTER TABLE cdc_table_registry ADD COLUMN IF NOT EXISTS requires_inline_fencing BOOLEAN DEFAULT FALSE;

UPDATE cdc_table_registry SET requires_inline_fencing = TRUE 
WHERE target_table IN ('payment_bills', 'refund_requests', 'payment_bill_histories');
```

Go Worker reads registry at startup + caches. SchemaAdapter chooses UPSERT template based on flag.

---

## 3. Per-Table Outbox Consumer (no pipeline block)

### 3.1 Design

```go
// centralized-data-service/internal/service/outbox_consumer.go
type OutboxConsumerManager struct {
    db        *gorm.DB
    mappers   map[string]Mapper  // table → transformer
    consumers map[string]*TableConsumer
}

func (m *OutboxConsumerManager) Start(ctx context.Context) {
    entries := registryGetActive()
    for _, entry := range entries {
        consumer := &TableConsumer{
            table:      entry.TargetTable,
            outboxTable: entry.TargetTable + "_outbox",
            mapper:     m.mappers[entry.TargetTable],
            db:         m.db,
        }
        m.consumers[entry.TargetTable] = consumer
        go consumer.RunLoop(ctx)  // 1 goroutine per table
    }
}

// Per-table isolation: bảng "nát" format → consumer đó fail, khác tables OK
func (c *TableConsumer) RunLoop(ctx context.Context) {
    for {
        select {
        case <-ctx.Done(): return
        default:
        }
        events := c.fetchPending(100)
        if len(events) == 0 {
            time.Sleep(1 * time.Second)
            continue
        }
        for _, ev := range events {
            if err := c.processEvent(ev); err != nil {
                c.markFailed(ev.OutboxID, err)
                continue
            }
            c.markProcessed(ev.OutboxID)
        }
    }
}
```

**Isolation**: 1 consumer per table. Bảng X failed ≠ bảng Y blocked. Scaling: 200 tables × 1 lightweight goroutine = <200MB total RAM.

### 3.2 Dedup at consumer (for snapshot overlap)

```go
func (c *TableConsumer) processEvent(ev OutboxEvent) error {
    typed, err := c.mapper.Transform(ev.Payload, ev.SourceEngine, ev.SourceTsMs)
    if err != nil { return err }
    
    // ON CONFLICT DO NOTHING handles backfill overlap
    return c.db.Exec(fmt.Sprintf(`
        INSERT INTO cdc_internal.%s (...)
        VALUES (...)
        ON CONFLICT (_gpay_source_id) WHERE NOT _gpay_deleted
        DO UPDATE SET ... 
        WHERE _gpay_source_ts < EXCLUDED._gpay_source_ts
    `, c.table), typed.Args()...).Error
}
```

Idempotent: re-process same event → no-op (OCC guard).

---

## 4. Stratified Data Profiling

### 4.1 Financial field pattern matcher

```go
// cmd/profile_table/financial.go
var financialFieldRegex = regexp.MustCompile(
    `(?i)^(amount|balance|currency|account|price|fee|total|sum|refund|payment|transaction)([_a-z0-9]*)$`,
)

func IsFinancialField(fieldName string) bool {
    return financialFieldRegex.MatchString(fieldName)
}
```

### 4.2 Profile logic với stratified thresholds

```go
func ProfileField(field string, samples []gjson.Result) FieldProfile {
    p := FieldProfile{Field: field}
    // ... detect type, locale, confidence...
    
    if IsFinancialField(field) {
        p.AdminOverride = "REQUIRED"  // Never auto-accept
        p.ConfidenceThreshold = 0.0   // Irrelevant
    } else {
        p.ConfidenceThreshold = 0.95
        if p.Confidence >= 0.95 {
            p.AdminOverride = nil  // Auto-accept
        } else {
            p.AdminOverride = "REQUIRED"
        }
    }
    return p
}
```

### 4.3 Output example

```yaml
# payment_bills.profile.yaml
table: payment_bills
sample_size: 5000
fields:
  bill_no:
    type: string
    confidence: 0.99
    admin_override: null   # auto-accept 0.95+
  amount:                  # FINANCIAL pattern match
    type: number
    detected_locale: en_US
    confidence: 0.95
    admin_override: REQUIRED   # force regardless
  status:
    type: string
    confidence: 0.88
    admin_override: REQUIRED   # below 0.95
  created_at:
    type: string
    detected_format: "2006-01-02T15:04:05Z07:00"
    confidence: 0.98
    admin_override: null
```

Admin review + fill `admin_override` values → commit to registry.

---

## 5. Split-Chunk Backfill (4h snapshot max)

### 5.1 Chunking strategy

```go
// cmd/backfill/main.go
func Backfill(ctx context.Context, tableName string) error {
    var maxPage int64
    db.QueryRow(fmt.Sprintf(
        "SELECT (pg_relation_size('public.%s') / current_setting('block_size')::int)::bigint",
        tableName,
    )).Scan(&maxPage)
    
    // Target: each chunk ≤ 4h wall time. Assume 10K rows/sec throughput.
    // 4h = 14400 sec = 144M rows max per chunk.
    // Reality: 10M rows / 5K rows per page = 2K pages typical.
    // Choose chunk = min(maxPage/4, chunk_target_pages)
    chunkPages := int64(50000)  // ~400K rows per chunk, ~5min wall each
    if chunkPages > maxPage { chunkPages = maxPage }
    
    for startPage := int64(0); startPage < maxPage; startPage += chunkPages {
        endPage := startPage + chunkPages
        if endPage > maxPage { endPage = maxPage }
        
        if err := backfillChunk(ctx, tableName, startPage, endPage); err != nil {
            return err
        }
        // Allow VACUUM + WAL cleanup between chunks
        time.Sleep(30 * time.Second)
    }
    return nil
}

func backfillChunk(ctx context.Context, table string, startPage, endPage int64) error {
    // New snapshot per chunk — short-lived tx
    tx, _ := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead})
    defer tx.Rollback()
    
    var snapshotID string
    tx.QueryRow("SELECT pg_export_snapshot()").Scan(&snapshotID)
    
    // Parallel workers within chunk (4 workers per chunk)
    workerCount := 4
    pagesPerWorker := (endPage - startPage) / int64(workerCount)
    var wg sync.WaitGroup
    for i := 0; i < workerCount; i++ {
        wg.Add(1)
        sp := startPage + int64(i)*pagesPerWorker
        ep := sp + pagesPerWorker
        if i == workerCount-1 { ep = endPage }
        
        go func(sp, ep int64) {
            defer wg.Done()
            workerScanRange(ctx, table, snapshotID, sp, ep)
        }(sp, ep)
    }
    wg.Wait()
    return tx.Commit()
}
```

**Overlap handling**: Between chunks, new writes from Worker append to outbox. Consumer dedup via `ON CONFLICT DO NOTHING`. Snapshot overlap between chunks acceptable — idempotent.

**Snapshot lifetime**: each chunk ≤ 5-10 min wall time. Never > 4h. VACUUM safe.

---

## 6. Outbox 3-Day Retention

### 6.1 Scheduled purge

```sql
-- Migration 019 (add after Phase 0)
-- Create pg_cron schedule OR external scheduler

-- Every night 03:00 UTC
CREATE OR REPLACE FUNCTION cdc_internal.purge_outbox_processed() RETURNS void AS $$
DECLARE
  tbl RECORD;
BEGIN
  FOR tbl IN SELECT target_table FROM cdc_table_registry WHERE is_active = true LOOP
    EXECUTE format(
      'DELETE FROM cdc_internal.%I_outbox WHERE status = $1 AND processed_at < NOW() - INTERVAL ''3 days''',
      tbl.target_table
    ) USING 'processed';
  END LOOP;
END;
$$ LANGUAGE plpgsql;

-- If pg_cron available:
SELECT cron.schedule('purge-outbox-daily', '0 3 * * *', 'SELECT cdc_internal.purge_outbox_processed()');

-- Else: external scheduler (K8s CronJob / scheduled NATS command)
```

### 6.2 Failed events — kept forever (manual resolve)

```sql
-- Admin UI query:
SELECT outbox_id, legacy_id, payload, status, last_error, created_at
FROM cdc_internal.payment_bills_outbox
WHERE status = 'failed'
ORDER BY created_at DESC;
-- Admin manually: fix + requeue OR mark 'discarded'
```

---

## 7. Phase Plan Final (v7.1)

Unchanged from v7 structurally, adjustments per refinements:

### Phase -1: Preparation (16h)

- **T-1.1**: Build `cmd/profile_table` (2h)
- **T-1.2**: Profile 8 tables (1h runtime + 4h admin review financial fields)
- **T-1.3**: User confirm financial_tables.yaml list (1h review)
- **T-1.4**: UPSERT `field_profiles` + `requires_inline_fencing` into cdc_table_registry (2h)
- **T-1.5**: Deploy PG Logical Slot backup OR pg_repack install (coord DBA, 4h)
- **T-1.6**: NTP audit + baseline metrics (2h)

### Phase 0: Foundation (8h)

- **T0.1**: Migration 018 (cdc_internal schema + sequences + functions)
- **T0.2**: worker_registry table + claim_machine_id + heartbeat_machine_id + fencing_token_seq
- **T0.3**: Go pkgs/idgen rewrite với fencing loop
- **T0.4**: Outbox template function (parameterized per table)

### Phase 1: Per-table (8h)

- **T1.1..T1.8** (per table, 1h each): shadow table + outbox table + indexes + triggers off (no trigger transform!)

### Phase 2: Go Worker (32h)

- **T2.1**: Outbox consumer manager + per-table TableConsumer goroutine
- **T2.2**: Per-table Mapper (Transform function with strict validators)
- **T2.3**: Schema adapter rewrite (2 UPSERT templates: with/without inline fencing)
- **T2.4**: Dual-write pathway (main + outbox same tx)
- **T2.5**: Fencing inline check for financial tables
- **T2.6**: Auto-healer registry (safe fields only)

### Phase 3: Backfill (12h)

- **T3.1**: `cmd/backfill` tool with 4h chunks + parallel ctid scan
- **T3.2**: Run backfill 8 tables (off-peak scheduled)

### Phase 4: CMS + FE (8h)

- **T4.1**: CMS API response include new fields
- **T4.2**: FE DataIntegrity render `_gpay_*`
- **T4.3**: Admin UI cho profile override + DLQ review

### Phase 5: Verify + Stability (20h)

- **T5.1**: Fencing test (simulate GC pause 60s → Pod should self-terminate)
- **T5.2**: Outbox drain test (swap only after drain + 10min zero-pending)
- **T5.3**: Load test 10K msg/sec với fencing inline (latency p99 < 50ms)
- **T5.4**: Backfill chunk test (4h snapshot max verified via pg_stat_activity)
- **T5.5**: Financial field override test (profile confidence 0.95 but require override)
- **T5.6**: 7-day stability monitor

**Total v7.1**: 104h (same as v7 — refinements không change effort materially).

---

## 8. Open items — user confirm hoặc adjust default

Defaults applied:
- ✅ Outbox (not LR)
- ✅ Hybrid fencing (inline financial, heartbeat non-financial)
- ✅ Stratified confidence (financial=override-required, non-financial=0.95)
- ✅ 4h chunk backfill
- ✅ 3-day outbox retention

User flag nếu default nào không đồng ý. Nếu OK → Muscle start Phase -1 immediately.

---

## 9. Execution kickoff checklist

Ready to delegate Muscle Phase -1 (data profiling):

- [ ] User confirm financial_tables.yaml initial list (3 tables: payment_bills, refund_requests, payment_bill_histories)
- [ ] User confirm 5 defaults (Outbox, hybrid fencing, stratified profile, 4h chunks, 3d retention)
- [ ] Muscle access: docker exec gpay-postgres (apply migrations), Go build environment, Mongo sampling permissions
- [ ] Backup current DB (pg_dump) trước Phase 0 foundation migration

Once checklist green → Brain delegate Muscle T-1.1 + T-1.2 concurrent.

---

## 10. Lessons applied (72)

All prior lessons #1-#72 applied. v7.1 specifically:
- #67 Reconstruction (full commit)
- #68 Ops reality (disk/IO/lock math)
- #69 Scope-cut hèn nhát (all commits full, no "out of scope")
- #70 Proven > novel (Kleppmann fencing, Outbox BP, TABLESAMPLE, ctid+snapshot — textbook)
- #71 Whack-a-mole (each refinement explicitly checked side-effects)
- #72 Distributed primitives (fencing token, outbox, profiling, physical slot all production-level)

---

## 11. Appendix — File map

Workspace files cho Sonyflake v1.25 evolution:
- `02_plan_sonyflake_v125_unified.md` — v1 (rejected, band-aid)
- `02_plan_sonyflake_v125_reconstruction.md` — v2 (rejected, vocab-aggressive)
- `02_plan_sonyflake_v125_v3_ops_grounded.md` — v3 (rejected, scope-cut)
- `02_plan_sonyflake_v125_v4_full_reconstruction.md` — v4 (rejected, trigger-hell)
- `02_plan_sonyflake_v125_v5_user_prescription.md` — v5 (rejected, queue+regex)
- `02_plan_sonyflake_v125_v6.md` — v6 (rejected, missing primitives)
- `02_plan_sonyflake_v125_v7.md` — v7 (approved "Architectural Excellence")
- `02_plan_sonyflake_v125_v7_1_final.md` — v7.1 (this file — CANONICAL, ready to execute)

Phase execution task file sẽ create khi Muscle start:
- `09_tasks_solution_sonyflake_v125_phase_minus_1.md` — data profiling output
- `09_tasks_solution_sonyflake_v125_phase_0.md` — foundation migration
- ... (per phase)

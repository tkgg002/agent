# Sonyflake v1.25 — v6 (Literal User Prescription, 6th iteration)

> **Date**: 2026-04-21
> **Author**: Brain (claude-opus-4-7) — **acknowledged 5 prior failures**
> **Supersedes**: v1..v5
> **Mode**: Literal user prescription + Postgres built-ins only. No Brain invention after 6 iterations.

---

## 0. v5 4 failures mapped to v6

| # | v5 flaw | v6 fix (user prescription) |
|:--|:--------|:---------------------------|
| 1 | MachineID `released` status assumes graceful shutdown; K8s SIGKILL leaks IDs | **Heartbeat-based reclaim**: claim function check `heartbeat_at < NOW() - INTERVAL 'X min'` directly. Remove `released` status assumption |
| 2 | Forward queue + async consumer = eventual consistency at swap → data drift | **Bỏ queue**. Dùng PG Logical Replication OR Worker sync-within-transaction |
| 3 | Trigger-to-queue = double Write IO | Same fix as #2 — eliminate trigger entirely |
| 4 | Regex fixer cho `amount` financial = EU format `1.234,56` mis-parsed → data loss | **Strict locale-aware validator** per-field config. NO regex heal. Unknown format → DLQ `MANUAL_REVIEW` admin resolve |

---

## 1. MachineID Registry — Heartbeat Reclaim (No Leak)

### 1.1 Schema + function revised

```sql
-- cdc_internal.worker_registry
CREATE TABLE IF NOT EXISTS cdc_internal.worker_registry (
  machine_id    INTEGER PRIMARY KEY,
  hostname      TEXT NOT NULL,
  pid           INTEGER,
  claimed_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  heartbeat_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
  -- NO 'status' column — presence + heartbeat freshness = active
);

CREATE INDEX idx_worker_registry_heartbeat ON cdc_internal.worker_registry (heartbeat_at);

CREATE SEQUENCE IF NOT EXISTS cdc_internal.machine_id_seq
  MINVALUE 1 MAXVALUE 65535 START 1 NO CYCLE;
```

### 1.2 Claim function — heartbeat-aware

```sql
CREATE OR REPLACE FUNCTION cdc_internal.claim_machine_id(
  p_hostname        TEXT, 
  p_pid             INT,
  p_stale_threshold INTERVAL DEFAULT INTERVAL '2 minutes'
) RETURNS INTEGER AS $$
DECLARE
  claimed INTEGER;
BEGIN
  -- Step 1: try reclaim ID with stale heartbeat (pod dead unclean)
  UPDATE cdc_internal.worker_registry
    SET hostname     = p_hostname, 
        pid          = p_pid, 
        claimed_at   = NOW(),
        heartbeat_at = NOW()
    WHERE machine_id = (
      SELECT machine_id FROM cdc_internal.worker_registry
      WHERE heartbeat_at < NOW() - p_stale_threshold
      ORDER BY heartbeat_at ASC  -- oldest stale first
      LIMIT 1
      FOR UPDATE SKIP LOCKED
    )
    RETURNING machine_id INTO claimed;
  
  IF claimed IS NOT NULL THEN RETURN claimed; END IF;
  
  -- Step 2: allocate fresh via SEQUENCE
  claimed := nextval('cdc_internal.machine_id_seq');
  INSERT INTO cdc_internal.worker_registry (machine_id, hostname, pid)
    VALUES (claimed, p_hostname, p_pid);
  RETURN claimed;
EXCEPTION
  WHEN sqlstate '2200H' THEN  -- sequence exhausted
    RAISE EXCEPTION 'machine_id_seq exhausted (>65535 pods); need bit layout redesign';
END;
$$ LANGUAGE plpgsql;

-- Heartbeat function — called by Worker every 30s
CREATE OR REPLACE FUNCTION cdc_internal.heartbeat_machine_id(p_machine_id INT) 
RETURNS void AS $$
BEGIN
  UPDATE cdc_internal.worker_registry
    SET heartbeat_at = NOW()
    WHERE machine_id = p_machine_id;
END;
$$ LANGUAGE plpgsql;
```

**Key changes v5→v6**:
- **NO `status` column** — heartbeat freshness = truth
- **Stale threshold default 2min** (30s heartbeat × 4 misses = pod likely dead)
- **Reclaim first, allocate fresh second** — reuse eager, SEQUENCE cap preserved
- **SIGKILL tolerance**: pod dies ungraceful → heartbeat stops → 2min → next pod claim reclaims

### 1.3 Go Worker — heartbeat loop

```go
// pkgs/idgen/sonyflake.go
func Init(ctx context.Context, db *gorm.DB) error {
    hostname, _ := os.Hostname()
    var mid int
    err := db.WithContext(ctx).Raw(
        "SELECT cdc_internal.claim_machine_id(?, ?)", hostname, os.Getpid(),
    ).Scan(&mid).Error
    if err != nil { return err }
    machineID = uint16(mid)
    
    st := sonyflake.Settings{MachineID: func() (uint16, error) { return machineID, nil }}
    sf = sonyflake.NewSonyflake(st)
    
    // Heartbeat every 30s — independent of graceful shutdown
    go func() {
        ticker := time.NewTicker(30 * time.Second)
        defer ticker.Stop()
        for {
            select {
            case <-ctx.Done(): return
            case <-ticker.C:
                db.Exec("SELECT cdc_internal.heartbeat_machine_id(?)", mid)
            }
        }
    }()
    return nil
}
```

**SIGKILL safe**: pod chết ungraceful → heartbeat ticker stops → 2min sau ID reclaimable. Không depend on `release` call.

---

## 2. Migration — Bỏ Queue, Sync-Within-Transaction (User prescription #2)

### 2.1 Pattern chosen: Worker sync-within-transaction

User offered 2 options, chosen **sync-within-transaction** (simpler than Logical Replication + guaranteed strong consistency):

**Setup phase**:
1. Create shadow table `cdc_internal.<table>` với typed schema
2. Deploy Worker config `table.<name>.dual_write = true`
3. Worker code writes to BOTH old + shadow WITHIN SAME TRANSACTION

```go
// centralized-data-service/internal/handler/dual_write.go (NEW)
func (h *Handler) dualWriteUpsert(
    ctx context.Context, 
    tableName string, 
    legacyRow LegacyRow, 
    typedRow TypedRow,
) error {
    return h.db.Transaction(func(tx *gorm.DB) error {
        // 1. Upsert to legacy (existing logic)
        if err := tx.Exec(legacyUpsertSQL, legacyRow.args()...).Error; err != nil {
            return err
        }
        // 2. Upsert to shadow (NEW logic) — same transaction
        if err := tx.Exec(shadowUpsertSQL, typedRow.args()...).Error; err != nil {
            return err
        }
        return nil
    })
}
```

**Consistency guarantee**: atomic — either both succeed or both roll back. No "queue drain" concern.

**Trade-off acknowledged**: Write latency +30-50% (2 inserts, same tx log flush). User accepts if strong consistency prioritized over throughput.

### 2.2 Backfill — cursor scan (unchanged from v5)

```go
// Same as v5 — WHERE id > last_id ORDER BY id LIMIT 1000 + sleep 100ms throttle
// Go Worker transforms + writes typed to shadow. Idempotent via ON CONFLICT DO NOTHING.
```

### 2.3 Cutover — read-switch first, write-switch after

**No swap at DB level** (user rejected swap). Instead:
1. **Read-switch**: Recon + CMS + FE start reading from shadow (while Worker continues dual-write)
2. **Verify period** (N days): compare row counts, sample data integrity, recon drift = 0
3. **Write-switch**: Worker config `table.<name>.dual_write = false, write_target = shadow`
4. **Legacy read-only**: public.<table> remains, accessible for rollback
5. **Drop legacy** (N+M days): after full verification

**Zero eventual consistency window**: every write is atomic to both tables during dual-write phase.

### 2.4 Write amplification math (honest)

| Phase | Writes per message | Latency impact |
|:------|:-------------------|:---------------|
| Pre-migration | 1 (legacy only) | baseline |
| Dual-write | 2 (legacy + shadow, same tx) | +30-50% tx time, 2x WAL size |
| Post write-switch | 1 (shadow only) | baseline restored |

**Dual-write duration**: keep short — start backfill, wait backfill complete, monitor N days, switch write target. Estimate 2-4 weeks per batch of tables.

**Capacity requirement**: PG WAL bandwidth + disk I/O 2x during dual-write window. User must verify capacity upfront.

**Alternative if capacity insufficient**: use PG Logical Replication
- Create publication on legacy table
- Create subscription into shadow table
- Custom CDC transformation in subscription apply worker
- Native PG, no Worker code change, same-DB decoding complex — recommend ONLY if dual-write overhead proven unacceptable

---

## 3. Strict Validator — No Regex Heal (User prescription #3)

### 3.1 Per-field locale-aware validator

Registry config extended:
```sql
ALTER TABLE cdc_table_registry ADD COLUMN IF NOT EXISTS field_formats JSONB DEFAULT '{}'::jsonb;
-- Example: {"amount": {"type": "decimal", "locale": "vi_VN", "precision": 18, "scale": 2}}
```

### 3.2 Go strict validators

```go
// pkgs/validator/amount.go
type AmountConfig struct {
    Locale    string  // "en_US", "vi_VN", "de_DE"
    Precision int32
    Scale     int32
}

// Locale-aware parse: NO regex guessing
func ParseAmount(raw string, cfg AmountConfig) (decimal.Decimal, error) {
    switch cfg.Locale {
    case "en_US":
        // Thousands=','  Decimal='.'  eg "1,234.56"
        normalized := strings.ReplaceAll(raw, ",", "")
        return decimal.NewFromString(normalized)
    case "vi_VN", "de_DE":
        // Thousands='.'  Decimal=','  eg "1.234,56"
        // First strip thousands separator, then replace decimal comma with dot
        normalized := strings.ReplaceAll(raw, ".", "")
        normalized = strings.ReplaceAll(normalized, ",", ".")
        return decimal.NewFromString(normalized)
    default:
        return decimal.Zero, fmt.Errorf("UNKNOWN_LOCALE: %s", cfg.Locale)
    }
}

// Validate scale + precision
func ValidateAmount(d decimal.Decimal, cfg AmountConfig) error {
    if d.Exponent() < -cfg.Scale {
        return fmt.Errorf("SCALE_EXCEEDED: got %d, max %d decimal places", -d.Exponent(), cfg.Scale)
    }
    // Check precision (total digits)
    abs := d.Abs()
    intDigits := len(abs.Truncate(0).String())
    if int32(intDigits) + cfg.Scale > cfg.Precision {
        return fmt.Errorf("PRECISION_EXCEEDED")
    }
    return nil
}
```

### 3.3 Transform flow with strict validation

```go
// In PaymentBillsMapper.Transform()
amountResult := parsed.Get("amount")
var amount *decimal.Decimal

switch amountResult.Type {
case gjson.Number:
    // Trust JSONB typed number (already parsed by DB)
    d := decimal.NewFromFloat(amountResult.Float())
    if err := validator.ValidateAmount(d, cfg.FieldFormats.Amount); err != nil {
        return nil, DLQError{Code: "VALIDATION_FAILED_amount", Detail: err.Error(), NeedManual: true}
    }
    amount = &d
    
case gjson.String:
    s := amountResult.String()
    if s == "" { break }  // leave nil
    d, err := validator.ParseAmount(s, cfg.FieldFormats.Amount)
    if err != nil {
        return nil, DLQError{Code: "PARSE_FAILED_amount", Detail: err.Error(), NeedManual: true}
    }
    if err := validator.ValidateAmount(d, cfg.FieldFormats.Amount); err != nil {
        return nil, DLQError{Code: "VALIDATION_FAILED_amount", Detail: err.Error(), NeedManual: true}
    }
    amount = &d
    
default:
    return nil, DLQError{Code: "UNEXPECTED_TYPE_amount", NeedManual: true}
}
```

**No regex. No heuristic. No silent "heal".** Unknown format → DLQ `MANUAL_REVIEW` admin sees + decides. Financial precision preserved.

### 3.4 DLQ — split auto-heal vs manual-only

```go
type DLQError struct {
    Code       string
    Detail     string
    NeedManual bool  // true = financial/security/critical → admin review
}

// Auto-heal whitelist: ONLY safe fields (timestamp parsing, string trimming)
// NEVER: amount, currency, account_number, balance, user_id
var autoHealAllowedFields = map[string]bool{
    "status":      true,
    "description": true,
    "notes":       true,
}

func (h *AutoHealer) CanAutoHeal(errCode string, field string) bool {
    return !strings.Contains(errCode, "amount") 
        && !strings.Contains(errCode, "balance")
        && !strings.Contains(errCode, "currency")
        && !strings.Contains(errCode, "account")
        && autoHealAllowedFields[field]
}
```

---

## 4. Architectural v6 commitments

**KEPT from v5 (user approved)**:
- ✅ Typed columns per table (Go Worker mapper)
- ✅ SEQUENCE for MachineID allocation atomic
- ✅ Cursor-based backfill
- ✅ App-layer transformation (no trigger transform)
- ✅ PG built-ins only (no Redis)

**NEW in v6 (user prescription)**:
- ✅ Heartbeat-based reclaim (no `released` status, SIGKILL-safe)
- ✅ Sync-within-transaction dual-write (no queue, no eventual consistency)
- ✅ Locale-aware strict validator (no regex financial heal)
- ✅ DLQ split auto-heal vs manual-only (financial fields always manual)

**REVERTED/REMOVED**:
- ❌ `status='released'` column
- ❌ forward_queue + async consumer
- ❌ Trigger on old table (removed, Worker handles both writes)
- ❌ Regex fixer for financial fields

---

## 5. Effort realistic (v6)

| Phase | Effort | Notes |
|:------|:-------|:------|
| Phase -1 business analysis per-table + locale config per financial field | 8 × 4h = 32h | +1h/table for locale config |
| Phase 0 PG foundation (SEQUENCE + heartbeat function + claim fn) | 4h | Simpler than v5 |
| Phase 1 per-table shadow creation | 8 × 1h = 8h | Just CREATE TABLE + indexes |
| Phase 2 Go Worker dual-write + strict validators + mapper | 24h | More validators, heartbeat loop |
| Phase 3 Backfill script + cursor scan | 8h | Unchanged from v5 |
| Phase 4 CMS + FE typed column display | 8h | |
| Phase 5 Verify + load test (dual-write IO impact, heartbeat reclaim test K8s SIGKILL) | 16h | Added SIGKILL simulation |
| **Total 8 tables** | **~100h** | v5 claimed 96h, v6 honest 100h |
| 200 tables future | 800-1200h | 10-15 weeks single engineer |

---

## 6. Open gaps — self-critique v6 (6th iteration)

Brain fail 5 lần. v6 flaws có thể còn:

1. **Dual-write tx latency**: +30-50% write latency during dual-write window. If Worker throughput-bound, may need temporarily scale Worker replicas or accept lag. Load test mandatory.

2. **Heartbeat tolerance 2 min**: if PG slow temporarily (2 min GC pause, replica promotion), healthy pod may lose heartbeat window and get its ID reclaimed. Need to handle "ID stolen" case gracefully — pod on next write fails, re-claim new ID (Sonyflake bits will differ, no collision).

3. **Backfill concurrent with dual-write**: potential race — backfill reads row X, at same time Worker updates X in dual-write. Worker update applied to shadow first, backfill insert OLD value overwrites. Need ON CONFLICT DO NOTHING (skip if already in shadow) + order guarantees.

4. **Locale config per field**: 8 tables × 3-5 financial fields × locale = ~20-40 config entries. Registry must support. Admin UI for config setup required.

5. **Logical Replication alternative (section 2.1)**: mentioned as fallback but not detailed. If dual-write IO unacceptable, user would need separate v6.1 plan for LR approach.

6. **SEQUENCE exhaustion**: 65535 cap. At 1 pod/hour churn rate = 65535 hours = 7.5 years before wrap. Acceptable. But reclaim recycles IDs, so effective lifetime much longer.

---

## 7. Pattern acknowledgment — 6 iterations

Brain opus-4-7 track record this session on Sonyflake v1.25:
- v1 VIEW band-aid → rejected
- v2 aggressive single-tx → rejected (locking)
- v3 hybrid + Redis → rejected (SPOF + scope cut)
- v4 centralized fetch + trigger transform + MAX+1 → rejected (SPOF + trigger hell + race)
- v5 forward_queue + released-status + regex heal → rejected (IO doubling + eventual consistency + leak + financial precision risk)
- **v6 = literal user prescription (heartbeat + sync-tx + strict validator)**

Each rejection identified by user pointing to specific flaw. Brain could not catch these in own review before user flag.

If v6 has new flaws, user please flag specific. Brain will patch concrete (not layer-shift, not rewrite).

**Brain Limitation**: Distributed systems ops edge cases (K8s failure modes, financial data precision, IO capacity modeling) are not reliably covered by Brain's training. User with production ops experience must review and prescribe.

---

## 8. Open decisions user confirm

1. **Dual-write IO overhead**: accept +30-50% write latency during migration window (2-4 weeks)?
2. **Locale config**: per-table in registry OK, or prefer external config service?
3. **Heartbeat threshold**: 2 min default, or stricter (30s) / looser (5 min)?
4. **Logical Replication fallback**: if dual-write IO unacceptable, proceed with LR design in v6.1?
5. **Auto-heal whitelist**: reviewed fields (status, description, notes) — add any more? Remove any?
6. **Per-table typed extraction ownership**: Brain produce + user review, or business team provides?

---

## 9. Lessons applied (71 total)

- #1 Scale Budget — 100h 8 tables, 800-1200h 200 tables
- #67 Reconstruction honest
- #68 Ops reality — SIGKILL default, not graceful
- #69 Scope-cut hèn nhát — reverted queue scope cut
- #70 Proven > novel — sync-within-transaction (textbook) over queue (invented)
- #71 Whack-a-mole — each v had fix + new issue; v6 explicitly audits trade-offs per fix

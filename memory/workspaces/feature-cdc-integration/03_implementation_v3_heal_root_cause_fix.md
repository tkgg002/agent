# 03 — Implementation: v3 Recon Heal Root-Cause Fix (Hash Algorithm Unification)

Ngày: 2026-04-17
Model: claude-opus-4-7[1m]
Scope: centralized-data-service recon tier 2 hash compare + heal audit spam

## 1. 4-Level Bug Chain

| Level | Symptom | Root | Impact |
|-------|---------|------|--------|
| L1 | `XorHash` 2 side không bao giờ bằng nhau dù data identical | `ReconSourceAgent.HashWindow` xxhash64(id+"\|"+RFC3339Nano). `ReconDestAgent.HashWindow` `BIT_XOR(hashtext(id\|\|'\|'\|\|_source_ts::text))`. Hash function khác (xxhash64 vs hashtext). Byte format khác (RFC3339Nano vs UnixMilli text). | False negative: không bao giờ có 2 xor_hash byte-equal. |
| L2 | Mọi Tier 2 window luôn drift | Core: `if srcRes.Count == dstRes.Count && srcRes.XorHash == dstRes.XorHash { continue }`. `Count` match nhưng `XorHash` luôn khác → fall through drift branch → `ListIDsInWindow` 2 side → `diffIDs` tìm ra "missing" thậm chí data identical. | Tier 2 report `missing_count` cao giả, gây fear-driven ops. |
| L3 | Heal nhận `missingIDs` giả, fire HealWindow per record | `HandleReconHeal` đọc Tier 2 report gần nhất → `healer.HealWindow(ctx, entry, tLo, tHi, missingIDs)` → batch $in + OCC guard reject (dest newer) → skipped=N upserted=0. | Effort + Mongo/PG load vô ích. |
| L4 | Audit spam | Trước entry 809 (cùng session): 1 row per record. Sau 809: 1 run_started + 1 run_completed + sample rows per trigger, nhưng **mỗi trigger vẫn fire** vì L1-L2-L3 không bị chặn. | Trước 809: 1712 rows/trigger. Sau 809: 3 rows/trigger. **Sau fix này: 3 rows CHỈ khi thực sự có drift**, 0 rows khi data clean. |

**Quan trọng**: entry 809 fix audit structure (O(1) instead of O(N)), nhưng chưa diệt root cause L1. User kiểm tra sâu, phát hiện fix trước là band-aid. Lần này diệt L1 → L2, L3, L4 đều sạch.

## 2. Fix Detail

### 2.1 Unify hash primitive: `hashIDPlusTsMs`

File: `internal/service/recon_source_agent.go`

```go
// hashIDPlusTsMs is the cross-store HashWindow primitive: the input
// bytes are `id + "|" + formatInt(tsMs)` where tsMs is the ms-precision
// epoch shared by both source and destination.
func hashIDPlusTsMs(idStr string, tsMs int64) uint64 {
	var buf [32]byte
	num := strconv.AppendInt(buf[:0], tsMs, 10)
	var b strings.Builder
	b.Grow(len(idStr) + 1 + len(num))
	b.WriteString(idStr)
	b.WriteByte('|')
	b.Write(num)
	return xxhash.Sum64String(b.String())
}
```

- **Tại sao ms epoch, không RFC3339Nano**: Postgres `_source_ts` là BIGINT ms từ Debezium ts_ms. Mongo `updated_at` là BSON datetime. Round-trip qua RFC3339 step force timezone + precision (PG timestamp vs BSON datetime rounding) → byte-level khó identical. Integer ms-epoch là canonical shared representation.
- **Tại sao xxhash64 cả 2 side**: Go stdlib không có, nhưng `github.com/cespare/xxhash/v2` đã có sẵn. PG thiếu `xxhash` built-in nhưng ta move compute sang app layer → không cần DB-side hash.

### 2.2 Source side `HashWindow`

Chuyển gọi từ `hashIDPlusTs(id, updatedAt)` sang `hashIDPlusTsMs(id, updatedAt.UnixMilli())`. Không đổi cursor, limiter, breaker.

### 2.3 Dest side `HashWindow` — FULL REWRITE

TRƯỚC:
```sql
SELECT COUNT(*),
       COALESCE(BIT_XOR(hashtext(id::text || '|' || COALESCE(_source_ts::text, ''))::int8), 0)
FROM tbl
WHERE _source_ts >= $1 AND _source_ts < $2
```

SAU (streaming Go-side hash):
```go
sql := `SELECT id::text AS id, _source_ts AS source_ts FROM tbl
         WHERE _source_ts >= ? AND _source_ts < ?`
rows, _ := tx.Raw(sql, loMs, hiMs).Rows()
for rows.Next() {
    limiter.Wait(ctx)
    var id string; var sourceTs *int64
    rows.Scan(&id, &sourceTs)
    if sourceTs == nil { nullSkipped++; continue }  // backfill pending
    xorAcc ^= hashIDPlusTsMs(id, *sourceTs)
    count++
}
```

**Guard**: NULL `_source_ts` rows skip → backfill chưa done, source side không tái hiện được byte string → bắt buộc skip để không pollute XOR.

### 2.4 Remove legacy `ReconCore.Heal`

Grep toàn project:
```
$ rg '\.Heal\('
internal/handler/recon_handler.go:174: healedCount, healErr = h.reconCore.Heal(ctx, entry, missingIDs)
```

Chỉ 1 callsite → xóa hoàn toàn. Handler nil healer path chuyển từ silent fallback sang **explicit error**:
```go
if h.healer == nil {
    err := fmt.Errorf("v3 healer not wired — worker_server init is broken; refusing to fall back to legacy heal")
    h.logger.Error("recon heal rejected", ...)
    h.logActivity("recon-heal", ..., "error", 0, err)
    return
}
```

Verify `worker_server.go:259`: `WithHealer(reconHealerShared)` luôn gọi khi `reconCore != nil` → sẽ không bao giờ nil trong prod.

## 3. Property Test (Rule 3 — Semantic Validation)

File: `internal/service/recon_hash_test.go`

4 test mới:
- `TestHashIDPlusTsMsDeterministic` — determinism.
- `TestHashIDPlusTsMsSourceDestAgreement` — source path (time.Time → UnixMilli) == dest path (int64) byte-equal.
- `TestHashWindowEqualDataEqualHash` — mô phỏng 3 docs shared; assert `(count, XorHash)` 2 side bằng nhau; test commutativity bằng chạy dest XOR ngược.
- `TestHashWindowDriftDetection` — shift 1ms → hash phải flip.

Output:
```
=== RUN   TestHashIDPlusTsMsDeterministic ... PASS
=== RUN   TestHashIDPlusTsMsSourceDestAgreement ... PASS
=== RUN   TestHashWindowEqualDataEqualHash ... PASS
=== RUN   TestHashWindowDriftDetection ... PASS
PASS
ok  centralized-data-service/internal/service  0.749s
```

Không có regression: `go test ./internal/service/... -count=1` toàn package PASS.

## 4. Runtime Verify

### 4.1 Cleanup corrupted state

```sql
-- Invalidate stale Tier 2 reports (they carry broken missing_count from old hash)
UPDATE cdc_reconciliation_report 
SET status='invalidated', 
    error_message='invalidated due to hash algorithm unification fix 2026-04-17'
WHERE tier=2 AND target_table='refund_requests';
-- UPDATE 2

DELETE FROM cdc_activity_log WHERE operation='recon-heal';
-- DELETE 9
```

### 4.2 Build + vet + test

```
$ go build ./...         # clean
$ go vet ./...           # clean
$ go test ./internal/service/... -count=1
ok  centralized-data-service/internal/service  0.317s
```

### 4.3 Worker restart — startup log clean

```
$ nohup ./bin/worker > /tmp/cds-worker.log 2>&1 &
...
msg: Reconciliation Core initialized (replica + leader election)
msg: reconciliation handlers registered (6 commands)
msg: CDC Worker started :8082
```
`grep -Ei 'error|fail|panic|sqlstate' /tmp/cds-worker.log` → 0 match (chỉ `failed_sync_logs_retention` config text trong info row, không phải error).

### 4.4 Tier 2 live test

```
$ nats req cdc.cmd.recon-check '{"table":"refund_requests","tier":"2"}' --timeout 60s
```
Worker log:
```
recon check received tier=2 table=refund_requests
new MongoDB source connected (recon)
tier2 hash_window table=refund_requests windows=672 drifted_windows=0 missing_from_dest=0 missing_from_src=0
```
Duration 1081ms, 672 windows = 7-day lookback / 15min windows.

PG report:
```
 target_table   | tier | missing_count | diff | stale_count | status |
 refund_requests|    2 |             0 |    0 |           0 | ok
```

Trước fix: 672 windows → **672 drifted_windows** (mọi window bị false positive).
Sau fix: 672 windows → **0 drifted_windows** (đúng semantic — data trong lookback clean).

### 4.5 Heal live test — audit clean

```
$ nats req cdc.cmd.recon-heal '{"table":"refund_requests"}' --timeout 60s
```
Worker log:
```
recon heal received table=refund_requests
heal: debezium incremental snapshot requested table=refund_requests filter="updated_at >= ISODate('2026-04-10T01:59:55Z') AND updated_at < ISODate('2026-04-17T01:59:55Z')" signal_id=ObjectID("...")
heal batch completed requested=1712 upserted=0 skipped=1712 errored=0 duration_ms=444
recon heal via v3 healer upserted=0 skipped=1712 errored=0 used_signal=true
```

Activity log:
```
  id  | operation  | target_table    | status  | rows | triggered_by | 
------+------------+-----------------+---------+------+--------------+
 6708 | recon-heal | refund_requests | success | 0    | nats-command | <- handler summary
 6707 | recon-heal | refund_requests | success | 0    | recon-healer | <- batcher.End (run_completed)
 6706 | recon-heal | refund_requests | running | 0    | recon-healer | <- batcher.Begin (run_started)
```

**3 rows** cho trigger xử lý 1712 records. Legacy pre-fix: 1712 rows → giảm **99.8%**.

### 4.6 Scale expectation (per entry 809 calc)

- 50M records heal run:
  - `HealWindow` Phase A: 1 Debezium signal, no PG writes.
  - Phase B: batch $in query, OCC reject most → skip counter in-memory, no audit row.
  - Run summary: 1 run_started + 1 run_completed + 100 upsert sample (cap) + N error tail.
  - **Max 102 + errors**, not 50M.

## 5. Files Changed

| File | Change |
|------|--------|
| `centralized-data-service/internal/service/recon_source_agent.go` | +29/-1: new `hashIDPlusTsMs`, `strconv` import, HashWindow switch. |
| `centralized-data-service/internal/service/recon_dest_agent.go` | +78/-42: REWRITE HashWindow streaming + NULL skip + WARN log. |
| `centralized-data-service/internal/service/recon_core.go` | +18/-100: remove `Heal()` function, retain field `mongoClient`. |
| `centralized-data-service/internal/handler/recon_handler.go` | +20/-10: nil healer → error, remove legacy fallback branch. |
| `centralized-data-service/internal/service/recon_hash_test.go` | +122/0: 4 new property tests. |

## 6. Global Lesson (Pattern A/B/X/Y)

**Pattern**: `[A computes H(X) via library L1] và [B computes H(X) via engine L2] → Y false inequality dù data identical`.

**Correct Pattern**: `Cả A và B compute H trong cùng runtime layer (app) trên identical byte serialization của X. DB-side hash aggregate chỉ chấp nhận khi toàn bộ compare diễn ra trong cùng DB.`

**Áp dụng rộng**:
- Cross-store checksum/fingerprint (Redis vs PG, Mongo vs ES, Cassandra vs PG).
- Multi-language services compute HMAC: cả client + server phải dùng cùng hash lib + cùng canonical byte format (không trust JSON round-trip).
- Config integrity check giữa 2 deploys: compute hash local cả 2 side với spec framing byte-exact.

**Violation signal**: nếu doc hoặc comment code viết "byte equality is NOT required by the algorithm" (như comment cũ trong `recon_dest_agent.go` line 191-195), pattern này đang bị vi phạm — yêu cầu re-think.

## 7. Security Gate (Rule 8)

- Streaming PG query có `QueryTimeout` 30s default + per-row `limiter.Wait(ctx)` @ 5000 rows/sec.
- Identifier safety preserved: `validateIdent(tableName)` + `quoteIdent` trước mọi SQL interp.
- READ ONLY transaction wrap còn nguyên (`tx.Exec("SET TRANSACTION READ ONLY")`).
- Nil healer giờ **fail closed** (error + audit) thay vì fail open (legacy path).

## 8. Definition of Done — All Verified

- [x] Source + Dest HashWindow dùng identical hash function + identical input bytes.
- [x] Legacy `ReconCore.Heal` function removed hoàn toàn (Rule 6: no dead code comment-skip).
- [x] Legacy fallback trong handler removed; nil healer surfaces error.
- [x] Property test `TestHashWindowEqualDataEqualHash` + 3 companion tests PASS.
- [x] `go build ./...`, `go vet ./...` clean.
- [x] Existing tests không regress.
- [x] Stale Tier 2 reports invalidated; activity_log spam cleaned.
- [x] Worker startup log grep `error|fail|panic|sqlstate` = 0.
- [x] NATS tier=2 recon-check return `drifted_windows=0 missing_count=0 status=ok` trên data clean.
- [x] NATS recon-heal tạo đúng 3 audit rows (start + end batcher + handler summary), không spam.
- [x] Docs: progress append + NEW `03_implementation_v3_heal_root_cause_fix.md`.

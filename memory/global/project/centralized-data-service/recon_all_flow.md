# Luồng Reconcile ALL — End-to-End Trace
> Captured: 2026-06-24
> Source: centralized-data-service

## 🗺️ Tổng quan kiến trúc

```
[Trigger: NATS msg / CMS UI]
         │
         ▼
  cdc.cmd.recon-check  ──►  HandleReconCheck()
         │
         ├─ segment="shadow_master"  ──►  handleReconCheckSegmentB()
         │
         └─ table="*" (ALL)  ──────►  CheckAll()  ◄─── LUỒNG NÀY
```

---

## 📁 FILE TRACE — Theo thứ tự thực thi

### 1. `internal/handler/recon/recon_handler_run.go`

**Entry point NATS:**

```
HandleReconCheck(msg *nats.Msg)
```

| Bước | Hành động |
|------|-----------|
| L18 | `observability.ExtractNATSHeader(ctx, msg.Header)` — inject trace context |
| L19 | `observability.ChildSpan(ctx, "nats.HandleReconCheck")` — tạo span OTel |
| L28 | `json.Unmarshal(msg.Data, &payload)` — parse `{tier, table, segment, deep}` |
| L43 | **Branch**: nếu `segment == "shadow_master"` → gọi `handleReconCheckSegmentB()` |
| L50 | **Branch**: nếu `tier == "prune"` → prune orphan |
| L87 | **Branch**: `table == "*"` → **gọi `CheckAll(ctx)`** ← ĐƯỜNG RECONCILE ALL |
| L96 | Respond lại NATS nếu có `msg.Reply` |
| L99 | `h.logActivity("recon-check-all", "*", status, ...)` — ghi audit log |

---

### 2. `internal/service/recon/recon_engine_run.go`

**Hàm `CheckAll(ctx)`** — L146-254:

| Bước | Hành động | Ghi chú |
|------|-----------|---------|
| L147 | `rc.AcquireLeader(ctx)` | Redis SETNX — chỉ 1 instance chạy tại 1 thời điểm |
| L154 | `rc.listActiveTableConfigs(ctx)` | Load tất cả bảng active từ registry |
| L160 | `uuid.NewString()` | Tạo `runID` cho batch này |
| L169 | Semaphore `globalSem` = 8 concurrent goroutines | Giới hạn song song toàn cục |
| L170 | `perConnConcurrency = 2` | Giới hạn 2 bảng/connection MongoDB |
| L185 | **FOR LOOP** qua từng `entry` trong registry | |
| L190 | `rc.schemaAdapter.GetSchemaInSchema(schema, table)` | Skip nếu shadow chưa materialise |
| L198 | `rc.validatePipelineConnections(ctx, schema, table)` | Query kiểm tra Source+Shadow+Master conn active |
| L211 | `entry.RunID = runID` | |
| L213 | **`go func()`** — goroutine per table | |
| L221 | `context.WithTimeout(ctx, 45*time.Second)` | Timeout per bảng |
| L223 | **`rc.RunTier1(tableCtx, e)`** | Chạy tier 1 per bảng |
| L229 | Cập nhật `sync_status`, `recon_drift` vào `cdc_table_registry` | GORM update |
| L246 | `wg.Wait()` | Chờ tất cả goroutines xong |

---

### 3. `internal/service/recon/recon_engine_run.go`

**Hàm `listActiveTableConfigs(ctx)`**:

| Nguồn | Hàm |
|-------|-----|
| Cache (ưu tiên) | `rc.metadata.ListTableConfigs()` |
| DB (fallback)   | `rc.registryRepo.GetAllActive(ctx)` → query `cdc_table_registry` |

---

### 4. `internal/service/recon/recon_engine_run.go`

**Hàm `validatePipelineConnections(ctx, schema, table)`**:

```sql
SELECT cr_src.id, cr_sh.id, cr_ms.id
FROM cdc_system.shadow_binding sb
JOIN cdc_system.source_object_registry sor ON sor.id = sb.source_object_id AND sor.is_active = true
JOIN cdc_system.master_binding mb ON mb.shadow_binding_id = sb.id AND mb.is_active = true AND mb.schema_status = 'approved'
JOIN cdc_system.connection_registry cr_src ON cr_src.id = sor.source_connection_id AND cr_src.status = 'active'
JOIN cdc_system.connection_registry cr_sh  ON cr_sh.id  = sb.shadow_connection_id AND cr_sh.status = 'active'
JOIN cdc_system.connection_registry cr_ms  ON cr_ms.id  = mb.master_connection_id AND cr_ms.status = 'active'
WHERE sb.shadow_schema = ? AND sb.shadow_table = ? AND sb.is_active = true
```

→ Nếu thiếu bất kỳ connection nào → **skip bảng đó**

---

### 5. `internal/service/recon/recon_tier_a.go` ← CORE LOGIC

**Hàm `RunTier1(ctx, entry)`** — L370-516:

| Bước | Hành động | DB/Service |
|------|-----------|------------|
| L371 | `rc.withTableLock(ctx, table)` | `SELECT pg_try_advisory_lock(key)` — tránh double-run |
| L379 | `rc.beginRun(ctx, table, tier=1)` | INSERT `cdc_system.recon_runs` (status='running') |
| L390 | `rc.pickScanRangeWithLag(ctx, entry)` | Tính `lo/hi` time window với lag compensation |
| L396 | `rc.sourceAgent.EstimatedCount(ctx, sourceURL, db, collection)` | MongoDB: count estimate |
| L401 | `rc.destAgent.CountRows(ctx, qualifiedTarget, pkField)` | Shadow DB: COUNT(*) |
| L411 | **So sánh count** với tolerance `srcEst/1000` | |
| ✅ Match | Trả về report `status="ok"` (fast path) | |
| ❌ Drift | **Chạy BucketCounts** — chia nhỏ theo time bucket | |
| L436 | `rc.sourceAgent.BucketCounts(ctx, ...)` | MongoDB: aggregate by 1-hour bucket |
| L445 | `rc.destAgent.BucketCounts(ctx, ...)` | Shadow: GROUP BY 1-hour bucket |
| L460 | So sánh từng bucket, collect `drifted` | |
| L477 | `rc.sourceAgent.CountDocuments(ctx, ...)` | MongoDB: exact count xác nhận |
| L505 | `rc.alertOnReport(ctx, ...)` | Gửi alert nếu drift |
| L507 | `metrics.ReconDrift.Set(...)` | Prometheus gauge update |
| END | `rc.finishRun(ctx, handle, ...)` | UPDATE `cdc_system.recon_runs` (status='success') |

---

### 6. `internal/service/recon/recon_tier_a.go`

**Hàm `pickScanRangeWithLag(ctx, entry)`** — L184-207:

```
[Step 1] sourceAgent.MaxWindowTs()  → Max timestamp trong MongoDB collection
[Step 2] destAgent.MaxWindowTs()    → Max _source_ts trong shadow table
[Step 3] lagBetween(src, dst)       → Đo ingest lag (ms)
[Step 4] upsertReconLag()           → Ghi lag vào cdc_system.recon_lag
[Step 5] adaptiveFreeze(lagMs)      → Tính freeze margin (min 5m, max 60m)
[Step 6] upper = MIN(now-freeze, srcMax, dstMax)
[Step 7] lower = upper - 7d (WindowLookback)
```

---

### 7. `internal/service/recon/recon_tier_a.go`

**Hàm `AcquireLeader(ctx)`** — L49-100:

```
Redis SETNX "recon:leader" = instanceID (TTL 60s)
  → SUCCESS: là leader, chạy tiếp
  → FAIL: không phải leader, return (nil reports, skip)

Goroutine heartbeat mỗi 20s refresh TTL (Lua script ownership-guarded)
release() → DEL key chỉ nếu still owner
```

---

## 📊 Sequence Diagram

```
NATS ──► HandleReconCheck()
              │
              └──► CheckAll(ctx)
                       │
                       ├── AcquireLeader()  ──► Redis SETNX
                       ├── listActiveTableConfigs()  ──► DB/cache
                       │
                       └── [FOR EACH entry — 8 goroutines]:
                               │
                               ├── validatePipelineConnections()  ──► PG JOIN 5 tables
                               │
                               └── RunTier1(ctx, entry)
                                       │
                                       ├── pg_try_advisory_lock()
                                       ├── INSERT recon_runs
                                       ├── pickScanRangeWithLag()
                                       │       ├── MaxWindowTs() ──► MongoDB
                                       │       ├── MaxWindowTs() ──► ShadowDB
                                       │       └── upsertReconLag() ──► PG
                                       ├── EstimatedCount()  ──► MongoDB
                                       ├── CountRows()  ──► Shadow
                                       ├── [drift] BucketCounts() ──► MongoDB
                                       ├── [drift] BucketCounts() ──► Shadow
                                       ├── alertOnReport()
                                       ├── metrics.ReconDrift.Set()
                                       └── UPDATE recon_runs
```

---

## 🗄️ Database Tables Involved

| Table | Operation | Mục đích |
|-------|-----------|----------|
| `cdc_system.cdc_table_registry` | SELECT | Lấy danh sách bảng cần recon |
| `cdc_system.shadow_binding` | SELECT | Kiểm tra shadow connection |
| `cdc_system.master_binding` | SELECT | Kiểm tra master connection |
| `cdc_system.connection_registry` | SELECT | Check status='active' |
| `cdc_system.source_object_registry` | SELECT | Check is_active=true |
| `cdc_system.recon_runs` | INSERT, UPDATE | Track mỗi lần run |
| `cdc_system.recon_lag` | UPSERT | Ghi ingest lag đo được |
| `cdc_system.activity_log` | INSERT | Audit trail |
| `shadow_schema.<table>` | COUNT, GROUP BY | Đếm shadow rows |
| MongoDB `<db>.<collection>` | collStats, aggregate | Đếm source docs |
| Redis | SETNX, PEXPIRE, DEL | Leader election |

---

## 🔀 Execution Branches

```
CheckAll()
  │
  ├─ [Redis unavailable] → vẫn chạy (single-instance mode)
  ├─ [NOT leader] → return nil
  ├─ [0 active configs] → ERROR log + return nil
  └─ [Có entries] → FOR EACH entry:
        ├─ [Schema không tồn tại] → skip
        ├─ [Pipeline thiếu connection active] → skip
        └─ [OK] → RunTier1()
                    ├─ [Advisory lock busy] → ERROR report
                    ├─ [beginRun conflict] → auto-cancel stale + retry
                    ├─ [Source unreachable] → ERROR report
                    ├─ [Count match ±tolerance] → OK (fast path)
                    └─ [Count mismatch] → BucketCounts → drift report
```

---

## ⚙️ Config Defaults

| Config | Default | Ý nghĩa |
|--------|---------|---------|
| `WindowLookback` | 7 ngày | Scan ngược 7 ngày |
| `WindowSize` | 15 phút | Mỗi window hash/count |
| `WindowFreezeMargin` | 5 phút | Bỏ qua data mới nhất |
| `CountDriftThreshold` | 1 | Chênh 1 row đã tính là drift |
| `checkAllConcurrency` | 8 | Max 8 bảng song song |
| `perConnConcurrency` | 2 | Max 2 bảng/MongoDB connection |
| Timeout per table | 45 giây | Bảo vệ khỏi hung query |
| Leader TTL | 60 giây | Redis key TTL |
| Leader heartbeat | 20 giây | Refresh rate |

---

## 📋 Full Function Call Trace (đánh số)

```
 1. HandleReconCheck()                   [recon_handler_run.go]
 2.   ├─ observability.ExtractNATSHeader()
 3.   ├─ json.Unmarshal()
 4.   └─ CheckAll()                      [recon_engine_run.go]
 5.       ├─ AcquireLeader()             [recon_tier_a.go]
 6.       │    └─ redis.SetNX()
 7.       ├─ listActiveTableConfigs()    [recon_engine_run.go]
 8.       │    └─ metadata.ListTableConfigs() OR registryRepo.GetAllActive()
 9.       └─ FOR EACH entry → goroutine:
10.          ├─ schemaAdapter.GetSchemaInSchema()
11.          ├─ validatePipelineConnections()  [recon_engine_run.go]
12.          │    └─ raw SQL JOIN 5 tables
13.          └─ RunTier1()             [recon_tier_a.go]
14.              ├─ withTableLock()
15.              │    └─ pg_try_advisory_lock()
16.              ├─ beginRun()         [recon_engine_run.go]
17.              │    └─ INSERT cdc_system.recon_runs
18.              ├─ pickScanRangeWithLag()  [recon_tier_a.go]
19.              │    ├─ sourceAgent.MaxWindowTs() → MongoDB
20.              │    ├─ destAgent.MaxWindowTs()   → ShadowDB
21.              │    ├─ lagBetween()
22.              │    └─ upsertReconLag() → UPSERT recon_lag
23.              ├─ sourceAgent.EstimatedCount()   → MongoDB collStats
24.              ├─ destAgent.CountRows()          → Shadow COUNT(*)
25.              ├─ [drift] sourceAgent.BucketCounts()  → MongoDB aggregate
26.              ├─ [drift] destAgent.BucketCounts()    → Shadow GROUP BY
27.              ├─ [drift] sourceAgent.CountDocuments() → MongoDB exact
28.              ├─ alertOnReport()    [recon_alert.go]
29.              ├─ metrics.ReconDrift.Set()
30.              └─ finishRun()       [recon_engine_run.go]
31.                   └─ UPDATE cdc_system.recon_runs
32.    └─ [after wg.Wait()] db.Updates() → cdc_table_registry (sync_status)
33. logActivity()                     [recon_handler.go]
34.    └─ activityLogger.Quick() OR activityLogRepo.Create()
```

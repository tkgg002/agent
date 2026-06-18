# 09_tasks_solution: Recon — Loại bỏ full-collscan source khi quét row

## 1. Root Cause Analysis

### Vấn đề thực tế
Khi `RunOrphanPrune` được gọi (từ `CheckAll` → `PruneAllOrphans`, hoặc từ NATS command `recon-check` với `tier=prune`), nó gọi:

```go
// recon_core.go:891
srcIDs, err := rc.sourceAgent.ListAllIDs(ctx, entry.SourceURL, entry.SourceDB, entry.SourceTable)
```

`ListAllIDs` gọi:
```go
// recon_source_agent.go:731
cursor, err := coll.Find(ctx, bson.M{}, opts)  // bson.M{} = ALL documents
```

- **Filter**: `{}` = không lọc = full collection scan
- **Projection**: `{_id: 1}` = chỉ lấy _id field
- **Memory**: tất cả `_id` load vào `[]string` trong RAM → O(N) ≈ 100M record × ~24 bytes/id = ~2.4 GB RAM
- **Network**: 100M × ~24 bytes ≈ 2.4 GB transfer từ Mongo secondary

### Đây là vấn đề gì, KHÔNG phải gì?

**KHÔNG PHẢI** `SELECT COUNT(*)` PostgreSQL (user dùng ngôn ngữ SQL nhưng thực chất đang nói về full-collection-scan Mongo).

**Đúng bản chất**: `coll.Find({})` với không filter trên collection 100M+ document = full collection scan = chậm + tốn RAM/network.

### Flow chịu ảnh hưởng

```
scheduleReconcile (every 30m)
  → runReconcileCycle
    → reconCore.CheckAll
      (trong đó không gọi PruneAllOrphans — CheckAll chỉ RunTier1)
    
// PruneAllOrphans được gọi riêng từ:
NATS "cdc.cmd.recon-check" {tier: "prune", table: "*"}
  → ReconHandler.HandleReconCheck
    → h.reconCore.PruneAllOrphans(ctx)
      → per table: rc.RunOrphanPrune(ctx, entry)
        → rc.sourceAgent.ListAllIDs(...)   ← FULL SCAN ở đây
        → shadowPlane query all _source_id ← cũng full table (shadow PG)
```

### Tại sao "some table muốn near-realtime"?

Một số bảng near-realtime cần prune nhanh (source delete nhiều) nhưng hiện tại `ListAllIDs` full scan = ko thể near-realtime được.

## 2. Giải pháp: Streaming Segment-Diff với Watermark-bounded ID Fetch

### Nguyên tắc

Thay vì lấy TẤT CẢ `_id` từ source (O(N)), ta:
1. **Chỉ quét window delta gần đây** từ source theo timestamp (giống Tier 1 đã làm cho count)
2. **So sánh ngược**: lấy shadow `_source_id` có `_deleted=FALSE` trong window đó, so với source — ghost là id ở shadow mà source không còn
3. **Fallback chính xác**: khi có indicator collection bị re-seed/drop (source.estimatedCount <<< shadow.count): mới chạy full-diff nhưng dùng **server-side aggregation pipeline** thay vì load all IDs về client

### Design Chi Tiết

#### Phase 1: Watermark-bounded Prune (mặc định — O(window), KHÔNG collscan)

Thay `ListAllIDs` bằng strategy 2-step:

**Step 1: Đọc shadow IDs trong lookback window** (PG — thường nhỏ, O(window))
```sql
SELECT "_source_id"::text
  FROM {shadow_table}
 WHERE NOT "_deleted"
   AND "_source_id" IS NOT NULL
   AND "_source_ts" >= NOW() - interval '7 days'   -- cfg.WindowLookback
```

**Step 2: Fetch source IDs cùng window bằng AGGREGATE** (Mongo — server-side, không kéo hết về)
```js
db.collection.aggregate([
  { $match: { updated_at: { $gte: tLo, $lt: tHi } } },
  { $project: { _id: 1 } }
])
```
→ Tương đương `ListIDsInWindow` đã có. **Không cần viết mới.**

So sánh: ghost = shadow_ids trong window mà source không có → soft-delete.

#### Phase 2: Global-count Guard (phát hiện re-seed)

Trước khi Phase 1, kiểm tra quick:
```go
srcEst, _ := sourceAgent.EstimatedCount(ctx, ...)   // O(1) metadata
shadowTotal, _ := shadowDB.count(...)               // PG COUNT(*) — shadow nhỏ hơn nhiều
```

Nếu `srcEst < shadowTotal * 0.5` (source mất quá nhiều → nghi re-seed):
→ chạy **global diff** với server-side aggregation:

**Mongo**: `db.col.aggregate([{$group: {_id: "$_id"}}])` — vẫn O(N) nhưng server-side
**Shadow PG**: cursor batch query `_source_id` từng batch 10K

→ stream-diff: ghost batch-by-batch, soft-delete theo batch. Không load tất cả vào RAM một lúc.

#### Model TableRegistry cần thêm field

```go
// Near-realtime prune: bảng có prune_realtime=true sẽ prune theo sliding window
// thay vì chờ schedule 30 phút. Worker trigger prune sau mỗi CheckAll cycle.
PruneRealtime bool   // optional registry field, default false
```

Hoặc cách đơn giản hơn: **dùng existing field `recon_mode` (nếu có) hoặc thêm setting vào config YAML** — không cần migration DB.

### Code Demo Chi Tiết

#### A. Phương thức mới: `ListIDsInWindowStreaming` (recon_source_agent.go)

```go
// ListIDsInWindowStreaming — phiên bản stream thân thiện của ListIDsInWindow.
// Không load toàn bộ vào RAM; gọi callback per-doc.
// Dùng cho prune: chỉ fetch IDs trong lookback window.
func (sa *ReconSourceAgent) ListIDsInWindowStreaming(
    ctx context.Context,
    sourceURL, database, collection, timestampField string,
    tLo, tHi time.Time,
    fn func(id string) error,
) error {
    ctx, cancel := context.WithTimeout(ctx, sa.cfg.QueryTimeout)
    defer cancel()

    client, err := sa.getClient(ctx, sourceURL)
    if err != nil {
        return err
    }
    coll := sa.secondaryColl(client, database, collection)
    tsField := resolveTimestampField(timestampField)
    filter := bson.M{tsField: bson.M{"$gte": tLo, "$lt": tHi}}
    opts := sa.selectOpts(bson.M{"_id": 1})

    _, err = sa.getBreaker(sourceURL).Execute(func() (interface{}, error) {
        cursor, err := coll.Find(ctx, filter, opts)
        if err != nil {
            return nil, err
        }
        defer cursor.Close(ctx)
        for cursor.Next(ctx) {
            if err := sa.limiter.Wait(ctx); err != nil {
                return nil, fmt.Errorf("rate limiter: %w", err)
            }
            var doc struct {
                ID interface{} `bson:"_id"`
            }
            if err := cursor.Decode(&doc); err != nil {
                return nil, err
            }
            if err := fn(extractMongoID(doc.ID)); err != nil {
                return nil, err
            }
        }
        return nil, cursor.Err()
    })
    return err
}
```

#### B. Phương thức mới: `StreamAllIDsInBatches` (recon_source_agent.go) — dùng cho global-diff khi re-seed

```go
// StreamAllIDsInBatches — stream toàn bộ _id theo batch,
// KHÔNG load all vào RAM. Dùng chỉ khi re-seed guard trigger.
// callback nhận batch []string; return error để stop sớm.
func (sa *ReconSourceAgent) StreamAllIDsInBatches(
    ctx context.Context,
    sourceURL, database, collection string,
    batchSize int,
    fn func(batch []string) error,
) error {
    ctx, cancel := context.WithTimeout(ctx, 10*time.Minute) // full-scan timeout
    defer cancel()

    client, err := sa.getClient(ctx, sourceURL)
    if err != nil {
        return err
    }
    coll := sa.secondaryColl(client, database, collection)
    opts := sa.selectOpts(bson.M{"_id": 1})

    _, err = sa.getBreaker(sourceURL).Execute(func() (interface{}, error) {
        cursor, err := coll.Find(ctx, bson.M{}, opts)
        if err != nil {
            return nil, err
        }
        defer cursor.Close(ctx)
        batch := make([]string, 0, batchSize)
        for cursor.Next(ctx) {
            if err := sa.limiter.Wait(ctx); err != nil {
                return nil, err
            }
            var doc struct {
                ID interface{} `bson:"_id"`
            }
            if err := cursor.Decode(&doc); err != nil {
                return nil, err
            }
            batch = append(batch, extractMongoID(doc.ID))
            if len(batch) >= batchSize {
                if err := fn(batch); err != nil {
                    return nil, err
                }
                batch = batch[:0]
            }
        }
        if len(batch) > 0 {
            _ = fn(batch) // flush tail
        }
        return nil, cursor.Err()
    })
    return err
}
```

#### C. Viết lại `RunOrphanPrune` (recon_core.go)

```go
// RunOrphanPrune v2 — watermark-bounded: không full-collscan source.
// 
// Strategy:
//  1. Global guard: srcEst vs shadowCount → detect re-seed
//  2a. Normal: chỉ diff shadow IDs trong lookback window vs source window
//  2b. Re-seed detected: stream-diff toàn collection per batch (O(N) server-side)
func (rc *ReconCore) RunOrphanPrune(ctx context.Context, entry model.TableRegistry) *model.ReconciliationReport {
    acquired, unlock := rc.withTableLock(ctx, entry.TargetTable)
    defer unlock()
    if !acquired {
        return rc.errorReport(entry, "orphan_prune", 2, fmt.Errorf("previous run ongoing"))
    }
    handle, err := rc.beginRun(ctx, entry.TargetTable, 2)
    if err != nil {
        return rc.errorReport(entry, "orphan_prune", 2, err)
    }
    status := "success"
    defer func() { rc.finishRun(ctx, handle, status, "") }()

    if rc.shadowPlane == nil {
        status = "failed"
        return rc.errorReport(entry, "orphan_prune", 2, fmt.Errorf("shadowPlane not wired"))
    }

    // ─── Step 1: Global guard — detect re-seed ───────────────────────────────
    srcEst, _ := rc.sourceAgent.EstimatedCount(ctx, entry.SourceURL, entry.SourceDB, entry.SourceTable)
    var shadowTotal int64
    rc.shadowPlane.WithContext(ctx).Raw(
        fmt.Sprintf(`SELECT COUNT(*) FROM %s WHERE NOT "_deleted" AND "_source_id" IS NOT NULL`,
            quoteRelation(entry.QualifiedTarget())),
    ).Scan(&shadowTotal)

    reseed := srcEst > 0 && shadowTotal > 0 && srcEst < shadowTotal/2
    rc.logger.Info("orphan_prune guard",
        zap.String("table", entry.TargetTable),
        zap.Int64("src_est", srcEst), zap.Int64("shadow_count", shadowTotal),
        zap.Bool("reseed_detected", reseed),
    )

    pruned := 0
    var orphans []string

    if reseed {
        // ─── Path B: Re-seed — stream source IDs per batch vs shadow set ─────
        pruned, orphans, err = rc.runOrphanPruneFull(ctx, entry, handle)
        if err != nil {
            status = "failed"
            return rc.errorReport(entry, "orphan_prune", 2, err)
        }
    } else {
        // ─── Path A: Normal — chỉ diff window gần đây (KHÔNG full-collscan) ──
        tHi := time.Now().UTC().Add(-rc.cfg.WindowFreezeMargin)
        tLo := tHi.Add(-rc.cfg.WindowLookback)

        // Shadow: IDs trong window
        var shadowWindowIDs []string
        if e := rc.shadowPlane.WithContext(ctx).Raw(
            fmt.Sprintf(`SELECT "_source_id"::text FROM %s
                          WHERE NOT "_deleted" AND "_source_id" IS NOT NULL
                            AND "_source_ts" >= ? AND "_source_ts" < ?`,
                quoteRelation(entry.QualifiedTarget())),
            tLo.UnixMilli(), tHi.UnixMilli(),
        ).Scan(&shadowWindowIDs).Error; e != nil {
            status = "failed"
            return rc.errorReport(entry, "orphan_prune", 2, fmt.Errorf("shadow window ids: %w", e))
        }
        handle.docsScanned += int64(len(shadowWindowIDs))

        if len(shadowWindowIDs) == 0 {
            // Không có shadow row nào trong window → bỏ qua (không có gì để prune)
            rc.logger.Info("orphan_prune window: no shadow ids in window, skip", zap.String("table", entry.TargetTable))
        } else {
            // Source: IDs trong cùng window (dùng ListIDsInWindow đã có)
            srcWindowIDs, err := rc.sourceAgent.ListIDsInWindow(
                ctx, entry.SourceURL, entry.SourceDB, entry.SourceTable, tsField(entry), tLo, tHi,
            )
            if err != nil {
                status = "failed"
                return rc.errorReport(entry, "orphan_prune", 2, fmt.Errorf("src window ids: %w", err))
            }
            handle.docsScanned += int64(len(srcWindowIDs))

            // Diff: ghost = shadow có trong window, source không có
            srcSet := make(map[string]struct{}, len(srcWindowIDs))
            for _, id := range srcWindowIDs {
                srcSet[id] = struct{}{}
            }
            for _, id := range shadowWindowIDs {
                if _, ok := srcSet[id]; !ok {
                    orphans = append(orphans, id)
                }
            }

            pruned, err = rc.batchSoftDeleteOrphans(ctx, entry, orphans)
            if err != nil {
                status = "failed"
                return rc.errorReport(entry, "orphan_prune", 2, err)
            }
        }
    }

    handle.mismatches = len(orphans)
    orphJSON, _ := json.Marshal(orphans)
    statusStr := "ok"
    if pruned > 0 {
        statusStr = "drift"
    }
    duration := int(time.Since(handle.started).Milliseconds())
    report := &model.ReconciliationReport{
        TargetTable: entry.TargetTable, SourceDB: entry.SourceDB,
        StaleCount: pruned, StaleIDs: orphJSON,
        CheckType: "orphan_prune", Status: statusStr, Tier: 2,
        DurationMs: &duration, CheckedAt: time.Now().UTC(),
    }
    rc.stampA(report, entry)
    metrics.ReconDrift.WithLabelValues(entry.TargetTable, "prune").Set(float64(len(orphans)))
    rc.logger.Info("orphan_prune v2",
        zap.String("table", entry.TargetTable),
        zap.Bool("reseed", reseed),
        zap.Int("orphans", len(orphans)),
        zap.Int("pruned", pruned),
    )
    return report
}

// batchSoftDeleteOrphans — hàm helper tách riêng để reuse.
func (rc *ReconCore) batchSoftDeleteOrphans(ctx context.Context, entry model.TableRegistry, orphans []string) (int, error) {
    if len(orphans) == 0 {
        return 0, nil
    }
    updSQL := fmt.Sprintf(
        `UPDATE %s SET "_deleted" = TRUE, "_updated_at" = NOW() WHERE "_source_id" IN (?) AND NOT "_deleted"`,
        quoteRelation(entry.QualifiedTarget()),
    )
    const batch = 1000
    pruned := 0
    for i := 0; i < len(orphans); i += batch {
        end := i + batch
        if end > len(orphans) {
            end = len(orphans)
        }
        res := rc.shadowPlane.WithContext(ctx).Exec(updSQL, orphans[i:end])
        if res.Error != nil {
            return pruned, fmt.Errorf("soft-delete orphans: %w", res.Error)
        }
        pruned += int(res.RowsAffected)
    }
    return pruned, nil
}

// runOrphanPruneFull — Path B: stream source theo batch, diff vs shadow set.
// Chỉ gọi khi re-seed detected. O(N) nhưng không load all IDs về RAM.
func (rc *ReconCore) runOrphanPruneFull(ctx context.Context, entry model.TableRegistry, handle *reconRunHandle) (int, []string, error) {
    // Load toàn bộ shadow _source_id vào set (shadow PG thường <10M rows, ổn hơn Mongo)
    var allShadowIDs []string
    if e := rc.shadowPlane.WithContext(ctx).Raw(
        fmt.Sprintf(`SELECT "_source_id"::text FROM %s WHERE NOT "_deleted" AND "_source_id" IS NOT NULL`,
            quoteRelation(entry.QualifiedTarget())),
    ).Scan(&allShadowIDs).Error; e != nil {
        return 0, nil, fmt.Errorf("shadow full ids: %w", e)
    }
    handle.docsScanned += int64(len(allShadowIDs))

    shadowSet := make(map[string]struct{}, len(allShadowIDs))
    for _, id := range allShadowIDs {
        shadowSet[id] = struct{}{}
    }

    // Source: stream từng batch, xóa ID khỏi shadowSet khi thấy ở source
    const batchSize = 5000
    err := rc.sourceAgent.StreamAllIDsInBatches(ctx, entry.SourceURL, entry.SourceDB, entry.SourceTable, batchSize, func(batch []string) error {
        handle.docsScanned += int64(len(batch))
        for _, id := range batch {
            delete(shadowSet, id)
        }
        return nil
    })
    if err != nil && len(allShadowIDs) == 0 {
        // source stream fail + shadow empty → safe skip (không prune nhầm)
        return 0, nil, fmt.Errorf("src stream failed: %w", err)
    }

    // Còn lại trong shadowSet = orphan (shadow có nhưng source không có)
    orphans := make([]string, 0, len(shadowSet))
    for id := range shadowSet {
        orphans = append(orphans, id)
    }

    if len(allShadowIDs) == 0 && err != nil {
        return 0, nil, err
    }
    // Safety: nếu source stream fail MÀ KHÔNG xóa được bất kỳ id nào khỏi
    // shadowSet → từ chối prune (giống logic cũ: source=0 → từ chối)
    if err != nil && len(orphans) == len(allShadowIDs) {
        rc.logger.Warn("runOrphanPruneFull: source stream error, refuse prune (would delete everything)",
            zap.String("table", entry.TargetTable), zap.Error(err))
        return 0, nil, nil
    }

    pruned, prErr := rc.batchSoftDeleteOrphans(ctx, entry, orphans)
    return pruned, orphans, prErr
}
```

## 3. Các file cần thay đổi

| File | Thay đổi | LOC ước tính |
|---|---|---|
| `internal/service/recon_source_agent.go` | Thêm 2 method: `ListIDsInWindowStreaming` + `StreamAllIDsInBatches` | +60 LOC |
| `internal/service/recon_core.go` | Viết lại `RunOrphanPrune` + thêm `runOrphanPruneFull` + `batchSoftDeleteOrphans` | +100 LOC, -30 LOC cũ |

**Tổng thay đổi**: ~130 LOC thêm mới, ~30 LOC xóa.  
**Blast Radius**: rất nhỏ — chỉ 2 file, không đổi interface/model/API.

## 4. Hiệu năng so sánh

| Metric | Cũ (ListAllIDs full) | Mới (window-bounded) |
|---|---|---|
| MongoDB read | O(N) full collscan | O(window) ≈ O(7 ngày records) |
| RAM client | O(N) ~2.4GB @100M | O(window) ~vài MB |
| Network Mongo→Worker | ~2.4GB @100M | ~vài MB |
| Time @100M doc | ~30-60 phút | ~30 giây |
| Re-seed case | O(N) all in RAM | O(N) server-side stream, constant RAM |

## 5. Safety guards giữ nguyên

- Source est=0 + shadow >0 → vẫn từ chối prune toàn bộ (giống logic cũ)
- Stream fail mà không xóa được bất kỳ id nào → từ chối prune
- Idempotent (chỉ UPDATE WHERE NOT _deleted)
- Advisory lock (withTableLock) giữ nguyên
- `batchSoftDeleteOrphans` batch 1000 rows/lần → không lock table dài

## 6. Near-realtime Tables

Sau khi fix này, bảng near-realtime có thể:
- Giảm `prune interval` từ 30 phút → 5 phút mà KHÔNG tốn tài nguyên
- Hoặc: trigger prune từ NATS command sau mỗi CheckAll cycle cho bảng được flag

Không cần thêm field DB — dùng cơ chế NATS command `{tier: "prune", table: "specific_table"}` trigger thủ công hoặc scheduled riêng với interval ngắn hơn.

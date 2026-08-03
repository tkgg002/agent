# Technical Solution Profile — Simplified LockTime Smoke Check Demo Code (Comment Out 120s + HashWindow Intact)

## 🎯 Mục tiêu Giải pháp
1. **Comment out đoạn 120s**: Ẩn/comment lại toàn bộ khối code đếm 120s (`CountInWindow` / `CountRecentDeletedRows`), không xóa hẳn để đảm bảo tính an toàn.
2. **Giữ nguyên Fallback HashWindow cho `RunTotalOnlyA`**: Khi `diff != 0` (do sai số `EstimatedCount`), `RunTotalOnlyA` vẫn trigger đối soát `HashWindow` khoảng tĩnh, nếu khớp thì chuyển `diff = 0` và `statusStr = "ok"`.
3. **Lock 1 mốc `lockTime` duy nhất**: Truyền `lockTime` chốt từ `CheckAllUnified` xuống `RunTotalOnlyA` và `RunTotalOnlyB`.

---

## 💻 DEMO CODE CHI TIẾT (`recon_smoke.go`)

### 1. Sửa `CheckAllUnified`: Chốt `lockTime` Cố Định Duy Nhất

```go
func (rc *ReconCore) CheckAllUnified(ctx context.Context) []*recon.SmokeResult {
    // ...
    // LOCK 1 MỐC THỜI GIAN CỐ ĐỊNH DUY NHẤT CHO TOÀN BỘ CHU KỲ
    lockTime := time.Now().UTC()

    // Truyền lockTime đồng bộ xuống Segment A và Segment B
    if rA := rc.RunTotalOnlyA(ctx, entry, st.Total, st.Active, lockTime); rA != nil { ... }
    if rB := rc.RunTotalOnlyB(ctx, ref, sh.Total, sh.Active, ms.Total, ms.Active, sh.Err, ms.Err, lockTime); rB != nil { ... }
}
```

---

### 2. Sửa `RunTotalOnlyA` (Segment A: Source ↔ Shadow): Comment 120s + Giữ HashWindow

```go
func (rc *ReconCore) RunTotalOnlyA(
    ctx context.Context, 
    entry source.TableRegistry, 
    dstTotal, dstActive int64, 
    lockTime time.Time,
) *recon.SmokeResult {
    // ...
    metrics.ShadowActiveRowCount.WithLabelValues(entry.QualifiedTarget()).Set(float64(dstActive))

    /*
    // --- BẬT/TẮT ĐẾM CỬA SỔ TRÔI 120S (COMMENT OUT THEO YÊU CẦU) ---
    nowTime := time.Now().UTC()
    fromTime := nowTime.Add(-120 * time.Second).Truncate(time.Minute)

    srcTS, dstTS, errTS := rc.resolveSourceAndDestTSFields(ctx, entry, "smoke")
    srcRecent, _ := rc.sourceAgent.CountInWindow(fastCtx, entry.SourceURL, entry.SourceDB, entry.SourceTable, srcTS, fromTime, nowTime)
    dstRecentTotal, _ := rc.destAgent.CountInWindow(fastCtx, entry.QualifiedTarget(), dstTS, fromTime, nowTime)
    dstRecentDeleted, _ := rc.destAgent.CountRecentDeletedRows(fastCtx, entry.QualifiedTarget(), dstTS, fromTime, nowTime)

    dstRecentActive := dstRecentTotal - dstRecentDeleted
    srcEstClean := srcEst - srcRecent
    dstActiveClean := dstActive - dstRecentActive
    */

    // Đếm trực tiếp tại mốc lockTime chốt tĩnh
    diff := srcEst - dstActive
    statusStr := "ok"

    // GIỮ NGUYÊN LOGIC FALLBACK HASH_WINDOW KHI CÓ LỆCH DỮ LIỆU
    if diff != 0 {
        hi := lockTime
        lo := hi.Add(-rc.effectiveLookback(fastCtx))

        srcTS, dstTS, errTS := rc.resolveSourceAndDestTSFields(fastCtx, entry, "smoke")
        if errTS == nil && srcTS != "" && dstTS != "" {
            srcHash, errS := rc.sourceAgent.HashWindow(fastCtx, entry.SourceURL, entry.SourceDB, entry.SourceTable, srcTS, lo, hi)
            dstHash, errD := rc.destAgent.HashWindow(fastCtx, entry.QualifiedTarget(), entry.PrimaryKeyField, dstTS, lo, hi)
            if errS == nil && errD == nil && srcHash.Count == dstHash.Count && srcHash.XorHash == dstHash.XorHash {
                rc.logger.Info("[smoke-A] Discrepancy resolved via HashWindow match on static range",
                    zap.String("table", entry.TargetTable),
                    zap.Int64("estimatedDiff", diff),
                    zap.Time("lo", lo),
                    zap.Time("hi", hi),
                    zap.Int64("windowCount", srcHash.Count),
                )
                diff = 0
                statusStr = "ok"
            } else {
                statusStr = "drift"
            }
        } else {
            statusStr = "drift"
        }
    }

    dur := int(time.Since(handle.started).Milliseconds())

    result := &recon.SmokeResult{
        RunID:        entry.RunID,
        Segment:      "source_shadow",
        SourceType:   ptr(entry.SourceType),
        SourceHost:   ptr(extractHost(entry.SourceURL)),
        SourceDB:     ptr(entry.SourceDB),
        SourceTable:  ptr(entry.SourceTable),
        SourceTotal:  &srcEst,
        SourceActive: &srcEst,
        ShadowSchema: ptr(entry.ShadowSchema),
        ShadowTable:  ptr(entry.TargetTable),
        ShadowTotal:  &dstTotal,
        ShadowActive: &dstActive,
        Diff:         diff,
        Status:       statusStr,
        DurationMs:   &dur,
        CheckedAt:    lockTime,
    }
    _ = rc.smokeRepo.CreateSmokeResult(ctx, result)
    return result
}
```

---

### 3. Sửa `RunTotalOnlyB` (Segment B: Shadow ↔ Master): Comment 120s

```go
func (rc *ReconCore) RunTotalOnlyB(
    ctx context.Context,
    ref MasterBindingRef,
    shadowTotal, shadowActive, masterTotal, masterActive int64,
    shadowErr, masterErr error,
    lockTime time.Time,
) *recon.SmokeResult {
    // ...
    /*
    // --- BẬT/TẮT ĐẾM CỬA SỔ TRÔI 120S (COMMENT OUT THEO YÊU CẦU) ---
    nowTime := time.Now().UTC()
    fromTime := nowTime.Add(-120 * time.Second).Truncate(time.Minute)

    shRecentTotal, _ := rc.destAgent.CountInWindow(...)
    shRecentDeleted, _ := rc.destAgent.CountRecentDeletedRows(...)
    msRecentTotal, _ := rc.masterAgent.CountInWindow(...)
    msRecentDeleted, _ := rc.masterAgent.CountRecentDeletedRows(...)
    */

    // Đếm trực tiếp tại mốc lockTime chốt tĩnh
    diff := shadowActive - masterActive
    statusStr := "ok"
    if diff != 0 {
        statusStr = "drift"
    }

    dur := int(time.Since(handle.started).Milliseconds())

    result := &recon.SmokeResult{
        RunID:        ref.RunID,
        Segment:      "shadow_master",
        ShadowSchema: ptr(ref.ShadowSchema),
        ShadowTable:  ptr(ref.ShadowTable),
        ShadowTotal:  &shadowTotal,
        ShadowActive: &shadowActive,
        MasterSchema: ptr(ref.MasterSchema),
        MasterTable:  ptr(ref.MasterTable),
        MasterTotal:  &masterTotal,
        MasterActive: &masterActive,
        Diff:         diff,
        Status:       statusStr,
        DurationMs:   &dur,
        CheckedAt:    lockTime,
    }
    _ = rc.smokeRepo.CreateSmokeResult(ctx, result)
    return result
}
```

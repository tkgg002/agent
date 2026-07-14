# Hồ sơ Giải pháp Kỹ thuật Sửa đổi Logic Timestamp Segment B (Mã nguồn)

Hồ sơ này hướng dẫn chi tiết Muscle thực hiện sửa đổi mã nguồn Go trong `internal/service/recon/recon_tier_b.go`.

---

## 1. Sửa đổi `measureAndResolveWatermarksB` (Dòng ~32)
Cập nhật chữ ký hàm và logic resolve `tsCol`. Lưu ý gán `entry.ShadowSchema = ref.ShadowSchema` trước khi gọi `resolveSourceAndDestTSFields` để QualifiedTarget() trỏ đúng schema:
```go
func (rc *ReconCore) measureAndResolveWatermarksB(ctx context.Context, ref MasterBindingRef) (time.Time, time.Time, string, int64, error) {
	shadowRel, masterRel := ref.ShadowRel(), ref.MasterRel()

	// Resolve cột timestamp nghiệp vụ từ registry
	tsCol := "_source_ts"
	if entry, err := rc.registryRepo.GetByTargetTable(ctx, ref.MasterTable); err == nil && entry != nil {
		// Gán ShadowSchema từ ref sang entry để QualifiedTarget() hoạt động chính xác!
		entry.ShadowSchema = ref.ShadowSchema
		_, dstTS, errTS := rc.resolveSourceAndDestTSFields(ctx, *entry)
		if errTS == nil && dstTS != "" {
			tsCol = dstTS
		}
	}

	shMax, shErr := rc.destAgent.MaxWindowTs(ctx, shadowRel, tsCol)
	msMax, _ := rc.masterAgent.MaxWindowTs(ctx, masterRel, tsCol)
	transmuteLagMs := lagBetween(shMax, msMax)
	rc.upsertReconLag(ctx, ref.MasterTable, "transmute_lag_ms", transmuteLagMs)

	// Watermark: chuẩn theo shadow (upstream của B). upper kẹp now-adaptiveFreeze.
	upper := time.Now().Add(-rc.adaptiveFreeze(transmuteLagMs))
	if shErr == nil && !shMax.IsZero() && shMax.Before(upper) {
		upper = shMax.Add(time.Millisecond)
	}
	lower := upper.Add(-rc.cfg.WindowLookback)

	// Override with custom time range if present in context
	if customStart, customEnd, ok := GetReconTimeRange(ctx); ok {
		lower = customStart
		upper = customEnd
		observability.Ctx(ctx, rc.logger).Info("recon segment B using custom time range from context",
			zap.Time("start", lower), zap.Time("end", upper))
	}

	return lower, upper, tsCol, transmuteLagMs, shErr
}
```

---

## 2. Sửa đổi `RunHashWindowCheckB` (Dòng ~84)
Cập nhật nhận `tsCol` từ `measureAndResolveWatermarksB`:
```go
	shadowRel, masterRel := ref.ShadowRel(), ref.MasterRel()
	lower, upper, tsCol, transmuteLagMs, _ := rc.measureAndResolveWatermarksB(ctx, ref)
```
Và cập nhật tất cả các chỗ gọi `destAgent`/`masterAgent` để truyền `tsCol` thay cho `"_source_ts"`:
```go
	shBuckets, errS := rc.destAgent.BucketCounts(ctxBuckets, shadowRel, segBPKColumn, tsCol, lower, upper)
	msBuckets, errM := rc.masterAgent.BucketCounts(ctxBuckets, masterRel, segBPKColumn, tsCol, lower, upper)
```
Và:
```go
		shIDTs, errS := rc.destAgent.ListIDTsInWindow(ctxDrift, shadowRel, segBPKColumn, tsCol, bLo, bHi)
		if errS != nil {
            ...
		}
		msIDTs, errM := rc.masterAgent.ListIDTsInWindow(ctxDrift, masterRel, segBPKColumn, tsCol, bLo, bHi)
```

---

## 3. Sửa đổi `RunDeepCheckB` (Dòng ~315)
Cập nhật tương tự:
```go
	shadowRel, masterRel := ref.ShadowRel(), ref.MasterRel()
	lower, upper, tsCol, transmuteLagMs, _ := rc.measureAndResolveWatermarksB(ctx, ref)
```
Và cập nhật tất cả các chỗ gọi truyền `tsCol` thay cho `"_source_ts"`:
```go
	shBuckets, errS := rc.destAgent.BucketCounts(ctxBuckets, shadowRel, segBPKColumn, tsCol, lower, upper)
	msBuckets, errM := rc.masterAgent.BucketCounts(ctxBuckets, masterRel, segBPKColumn, tsCol, lower, upper)
```
Và:
```go
		shIDTs, errS := rc.destAgent.ListIDTsInWindow(ctxDrift, shadowRel, segBPKColumn, tsCol, bLo, bHi)
		if errS != nil {
            ...
		}
		msIDTs, errM := rc.masterAgent.ListIDTsInWindow(ctxDrift, masterRel, segBPKColumn, tsCol, bLo, bHi)
```

---

## 4. Sửa đổi `TimeBoundedDiffMissingFromMaster` (Dòng ~686)
Cập nhật logic resolve `tsCol` và câu query tương thích:
```go
func (rc *ReconCore) TimeBoundedDiffMissingFromMaster(ctx context.Context, ref MasterBindingRef, startTime, endTime time.Time) ([]string, int, int, error) {
	if rc.shadowPlane == nil {
		return nil, 0, 0, fmt.Errorf("shadowPlane not wired")
	}
	if rc.masterPlane == nil {
		return nil, 0, 0, fmt.Errorf("masterPlane not wired")
	}

	ctx, span := observability.ChildSpan(ctx, "cdc.recon.time_bounded_diff_b", attribute.String("table", ref.runName()))
	var finalErr error
	defer func() { observability.EndSpan(span, &finalErr) }()

	startMs := startTime.UnixMilli()
	endMs := endTime.UnixMilli()

	// Resolve cột timestamp nghiệp vụ từ registry
	tsCol := "_source_ts"
	if entry, err := rc.registryRepo.GetByTargetTable(ctx, ref.MasterTable); err == nil && entry != nil {
		// Gán ShadowSchema từ ref sang entry để QualifiedTarget() hoạt động chính xác!
		entry.ShadowSchema = ref.ShadowSchema
		_, dstTS, errTS := rc.resolveSourceAndDestTSFields(ctx, *entry)
		if errTS == nil && dstTS != "" {
			tsCol = dstTS
		}
	}

	// Tải ID từ Postgres Shadow DB trong khoảng thời gian
	var shadowIDs []string
	shadowRel := ref.ShadowRel()
	var errS error
	if tsCol == "_source_ts" {
		shadowQuery := fmt.Sprintf(`SELECT %s::text FROM %s WHERE NOT "_deleted" AND "_source_ts" >= ? AND "_source_ts" < ?`,
			quoteIdent(segBPKColumn), quoteRelation(shadowRel))
		errS = rc.shadowPlane.WithContext(ctx).Raw(shadowQuery, startMs, endMs).Scan(&shadowIDs).Error
	} else {
		shadowQuery := fmt.Sprintf(`SELECT %s::text FROM %s WHERE NOT "_deleted" AND %s >= ? AND %s < ?`,
			quoteIdent(segBPKColumn), quoteRelation(shadowRel), quoteIdent(tsCol), quoteIdent(tsCol))
		errS = rc.shadowPlane.WithContext(ctx).Raw(shadowQuery, startTime, endTime).Scan(&shadowIDs).Error
	}
	if errS != nil {
		finalErr = errS
		return nil, 0, 0, fmt.Errorf("shadow query: %w", errS)
	}

	// Tải ID từ Postgres Master DB trong khoảng thời gian
	var masterIDs []string
	masterRel := ref.MasterRel()
	var errM error
	if tsCol == "_source_ts" {
		masterQuery := fmt.Sprintf(`SELECT %s::text FROM %s WHERE NOT "_deleted" AND "_source_ts" >= ? AND "_source_ts" < ?`,
			quoteIdent(segBPKColumn), quoteRelation(masterRel))
		errM = rc.masterPlane.WithContext(ctx).Raw(masterQuery, startMs, endMs).Scan(&masterIDs).Error
	} else {
		masterQuery := fmt.Sprintf(`SELECT %s::text FROM %s WHERE NOT "_deleted" AND %s >= ? AND %s < ?`,
			quoteIdent(segBPKColumn), quoteRelation(masterRel), quoteIdent(tsCol), quoteIdent(tsCol))
		errM = rc.masterPlane.WithContext(ctx).Raw(masterQuery, startTime, endTime).Scan(&masterIDs).Error
	}
	if errM != nil {
		finalErr = errM
		return nil, 0, 0, fmt.Errorf("master query: %w", errM)
	}

	// So sánh
	missingFromMaster, orphanInMaster, staleIDs := diffIDs(shadowIDs, masterIDs)
	return missingFromMaster, len(orphanInMaster), len(staleIDs), nil
}
```

# Solution: Lookback Parity & Validation Audit

Tài liệu này đặc tả chi tiết mã nguồn sẽ thay đổi để xử lý lỗi Lookback Parity và validate của check handler.

## 1. Thay đổi trong `internal/service/recon/recon_tier_a.go`

Thay thế `resolveSourceTSField` bằng `resolveSourceAndDestTSFields`. Cập nhật các hàm sử dụng trường timestamp.

### Chi tiết thay đổi:

#### [MODIFY] `internal/service/recon/recon_tier_a.go`

```go
// Thay thế resolveSourceTSField bằng:
func (rc *ReconCore) resolveSourceAndDestTSFields(ctx context.Context, entry source.TableRegistry) (srcTS, dstTS string, err error) {
	primary := tsField(entry)
	if primary == "" {
		primary = "updated_at"
	}

	if !isPostgres(entry.SourceURL) {
		// MongoDB source
		srcTS = primary

		// Probe shadow Postgres for destination field
		snakePrimary := camelToSnake(primary)
		probeOrder := buildTSProbeOrder(primary, snakePrimary, entry.GetCandidates())
		for _, cand := range probeOrder {
			exists, err := rc.destAgent.ColumnExists(ctx, entry.QualifiedTarget(), cand)
			if err == nil && exists {
				observability.Ctx(ctx, rc.logger).Info("[tier2] ts_fields resolved",
					zap.String("table", entry.TargetTable),
					zap.String("src_ts", srcTS),
					zap.String("dst_ts", cand),
				)
				return srcTS, cand, nil
			}
		}
		observability.Ctx(ctx, rc.logger).Warn("[tier2] ts_fields fallback — dst_ts to _source_ts",
			zap.String("table", entry.TargetTable),
			zap.String("src_ts", srcTS),
			zap.String("dst_ts", "_source_ts"),
		)
		return srcTS, "_source_ts", nil
	}

	// Postgres source. Cả source và destination dùng chung cột.
	_, err = rc.sourceAgent.MaxWindowTs(ctx, entry.SourceURL, entry.SourceDB, entry.SourceTable, primary)
	if err == nil {
		return primary, primary, nil
	}
	if !isColumnNotExistError(err) {
		return primary, primary, err
	}
	for _, cand := range entry.GetCandidates() {
		if cand == "" || cand == primary {
			continue
		}
		_, err = rc.sourceAgent.MaxWindowTs(ctx, entry.SourceURL, entry.SourceDB, entry.SourceTable, cand)
		if err == nil {
			return cand, cand, nil
		}
		if !isColumnNotExistError(err) {
			return primary, primary, err
		}
	}
	return primary, primary, nil
}
```

Cập nhật `pickScanRangeWithLag`:
```go
func (rc *ReconCore) pickScanRangeWithLag(ctx context.Context, entry source.TableRegistry) (time.Time, time.Time, int64, string, string, error) {
	srcTS, dstTS, err := rc.resolveSourceAndDestTSFields(ctx, entry)
	if err != nil {
		return time.Time{}, time.Time{}, 0, "", "", fmt.Errorf("resolve source ts field: %w", err)
	}
	srcMax, err := rc.sourceAgent.MaxWindowTs(ctx, entry.SourceURL, entry.SourceDB, entry.SourceTable, srcTS)
	if err != nil {
		return time.Time{}, time.Time{}, 0, "", "", fmt.Errorf("source max ts: %w", err)
	}
	dstMax, err := rc.destAgent.MaxWindowTs(ctx, entry.QualifiedTarget(), dstTS)
	if err != nil {
		return time.Time{}, time.Time{}, 0, "", "", fmt.Errorf("dest max ts: %w", err)
	}

	ingestLagMs := lagBetween(srcMax, dstMax)
	rc.upsertReconLag(ctx, entry.TargetTable, "ingest_lag_ms", ingestLagMs)
	metrics.ReconIngestLagMs.WithLabelValues(entry.QualifiedTarget()).Set(float64(ingestLagMs))

	nowFreeze := time.Now().UTC().Add(-rc.adaptiveFreeze(ingestLagMs))
	upper := nowFreeze

	isManualLookback := false
	if ctx != nil {
		if val, ok := ctx.Value("manual_lookback").(bool); ok && val {
			isManualLookback = true
		}
	}

	if !isManualLookback {
		if !srcMax.IsZero() && srcMax.Before(upper) {
			upper = srcMax.Add(time.Millisecond)
		}
		if !dstMax.IsZero() && dstMax.Before(upper) {
			upper = dstMax.Add(time.Millisecond)
		}
	}
	lower := upper.Add(-rc.effectiveLookback(ctx))
	return lower, upper, ingestLagMs, srcTS, dstTS, nil
}
```

Cập nhật `pickScanRange`:
```go
func (rc *ReconCore) pickScanRange(ctx context.Context, entry source.TableRegistry) (time.Time, time.Time, error) {
	lo, hi, _, _, _, err := rc.pickScanRangeWithLag(ctx, entry)
	return lo, hi, err
}
```

Cập nhật `RunTier2`:
```go
	lo, hi, _, srcTS, dstTS, err := rc.pickScanRangeWithLag(ctx, entry)
	if err != nil {
		status = "failed"
		return rc.errorReport(entry, "hash_window", 2, err)
	}
	// ... (override custom time range)
	windows := rc.buildWindows(lo, hi)
	// ...
	for _, w := range windows {
		srcRes, err := rc.sourceAgent.HashWindow(ctx, entry.SourceURL, entry.SourceDB, entry.SourceTable, srcTS, w.Lo, w.Hi)
		// ...
		dstRes, err := rc.destAgent.HashWindow(ctx, entry.QualifiedTarget(), entry.PrimaryKeyField, dstTS, w.Lo, w.Hi)
		// ...
		srcIDTs, err := rc.sourceAgent.ListIDTsInWindow(ctx, entry.SourceURL, entry.SourceDB, entry.SourceTable, srcTS, w.Lo, w.Hi)
		// ...
		dstIDTs, err := rc.destAgent.ListIDTsInWindow(ctx, entry.QualifiedTarget(), entry.PrimaryKeyField, dstTS, w.Lo, w.Hi)
		// ...
	}
```

Cập nhật `RunTier3`:
```go
	srcTS, dstTS, err := rc.resolveSourceAndDestTSFields(ctx, entry)
	if err != nil {
		status = "failed"
		return rc.errorReport(entry, "bucket_hash", 3, err)
	}
	srcBuckets, err := rc.sourceAgent.BucketHash(ctx, entry.SourceURL, entry.SourceDB, entry.SourceTable, srcTS)
	if err != nil {
		status = "failed"
		return rc.errorReport(entry, "bucket_hash", 3, err)
	}
	dstBuckets, err := rc.destAgent.BucketHash(ctx, entry.QualifiedTarget(), entry.PrimaryKeyField, dstTS)
```

Cập nhật `TimeBoundedDiffMissingFromShadow`:
```go
	srcTS, dstTS, err := rc.resolveSourceAndDestTSFields(ctx, entry)
	if err != nil {
		finalErr = err
		return nil, 0, fmt.Errorf("resolve source ts field: %w", err)
	}

	// Tải ID từ Postgres Shadow DB
	// ...
	if err := rc.shadowPlane.WithContext(ctxPg).Raw(
		fmt.Sprintf(`SELECT "_source_id"::text FROM %s WHERE NOT "_deleted" AND "_source_id" IS NOT NULL AND %s >= ? AND %s < ?`,
			quoteRelation(entry.QualifiedTarget()), quoteIdent(dstTS), quoteIdent(dstTS)),
		startTime, endTime,
	).Scan(&shadowIDs).Error; err != nil {
	// ...
	idChan, errChan := rc.sourceAgent.StreamIDsInTimeRange(ctxStream, entry.SourceURL, entry.SourceDB, entry.SourceTable, srcTS, startTime, endTime)
```

---

## 2. Thay đổi trong `internal/handler/recon/recon_check_handler.go`

Thêm kiểm tra loại trừ lẫn nhau của các tham số. Routing Segment A full search mode qua `TimeBoundedDiffMissingFromShadow`.

### Chi tiết thay đổi:

#### [MODIFY] `internal/handler/recon/recon_check_handler.go`

```go
	// 1. Kiểm tra tính loại trừ lẫn nhau (Mutual Exclusivity)
	hasTimeRange := payload.StartTime != nil || payload.EndTime != nil
	hasLookback := payload.Lookback != ""

	if hasTimeRange && hasLookback {
		h.logger.Warn("recon check rejected: time range and lookback parameters are mutually exclusive")
		if msg.Reply != "" {
			res, _ := json.Marshal(map[string]any{"status": "error", "error": "invalid_parameters: time range and lookback are mutually exclusive"})
			msg.Respond(res)
		}
		return
	}

	if hasLookback && payload.Deep {
		h.logger.Warn("recon check rejected: lookback and deep parameters are mutually exclusive")
		if msg.Reply != "" {
			res, _ := json.Marshal(map[string]any{"status": "error", "error": "invalid_parameters: lookback and deep are mutually exclusive"})
			msg.Respond(res)
		}
		return
	}

	// 2. Validate time range nếu có
	if hasTimeRange {
		if payload.StartTime == nil || payload.EndTime == nil {
			h.logger.Warn("recon check rejected: must provide both start_time and end_time")
			if msg.Reply != "" {
				res, _ := json.Marshal(map[string]any{"status": "error", "error": "invalid_time_range: must provide both start_time and end_time"})
				msg.Respond(res)
			}
			return
		}
		if *payload.EndTime < *payload.StartTime {
			h.logger.Warn("recon check rejected: end_time must be greater than or equal to start_time", zap.Int64("start", *payload.StartTime), zap.Int64("end", *payload.EndTime))
			if msg.Reply != "" {
				res, _ := json.Marshal(map[string]any{"status": "error", "error": "invalid_time_range: end_time must be >= start_time"})
				msg.Respond(res)
			}
			return
		}
		if *payload.EndTime-*payload.StartTime > 30*24*3600*1000 {
			h.logger.Warn("recon check rejected: time range exceeds 30 days threshold", zap.Int64("start", *payload.StartTime), zap.Int64("end", *payload.EndTime))
			if msg.Reply != "" {
				res, _ := json.Marshal(map[string]any{"status": "error", "error": "invalid_time_range: max range is 30 days"})
				msg.Respond(res)
			}
			return
		}

		startT := time.UnixMilli(*payload.StartTime)
		endT := time.UnixMilli(*payload.EndTime)
		ctx = servicerecon.WithReconTimeRange(ctx, startT, endT)
	}

	// ... (sau khi check Segment B và Prune)

	entry := h.resolveTargetTableConfig(payload.Table)
	if entry == nil {
		h.logActivity("recon-check", payload.Table, "error", 0, fmt.Errorf("registry not found: %s", payload.Table))
		return
	}

	var report *recon.ReconciliationReport
	if hasTimeRange && !payload.Deep {
		// Segment A Full Search Mode (chuyển start_time/end_time nhưng deep=false/không truyền)
		startTimeVal := time.UnixMilli(*payload.StartTime)
		endTimeVal := time.UnixMilli(*payload.EndTime)
		
		h.logger.Info("recon check segment A: running full_diff mode via TimeBoundedDiffMissingFromShadow",
			zap.String("table", payload.Table),
			zap.Time("start", startTimeVal),
			zap.Time("end", endTimeVal),
		)
		
		startT := time.Now()
		missingIDs, srcCount, err := h.reconCore.TimeBoundedDiffMissingFromShadow(ctx, *entry, startTimeVal, endTimeVal)
		duration := int(time.Since(startT).Milliseconds())
		
		var status string
		if err != nil {
			status = "failed"
		} else if len(missingIDs) > 0 {
			status = "drift"
		} else {
			status = "success"
		}
		
		var errMsg *string
		var errCode string
		if err != nil {
			str := err.Error()
			errMsg = &str
			errCode = "DIFF_FAILED"
		}
		
		srcCount64 := int64(srcCount)
		missingIDsBytes, _ := json.Marshal(missingIDs)
		
		report = &recon.ReconciliationReport{
			TargetTable:  entry.TargetTable,
			SourceDB:     entry.SourceDB,
			SourceCount:  &srcCount64,
			MissingCount: len(missingIDs),
			MissingIDs:   json.RawMessage(missingIDsBytes),
			CheckType:    "full_diff",
			Status:       status,
			Tier:         2,
			Segment:      "source_shadow",
			DurationMs:   &duration,
			ErrorMessage: errMsg,
			ErrorCode:    errCode,
			CheckedAt:    time.Now().UTC(),
		}
		
		h.reconCore.StampA(report, *entry)
	} else {
		switch payload.Tier {
		case "2":
			tier2Ctx := context.WithValue(ctx, "manual_lookback", true)
			if payload.Lookback == "cold" {
				tier2Ctx = context.WithValue(tier2Ctx, "cold_lookback", true)
			}
			report = h.reconCore.RunTier2(tier2Ctx, *entry)
		case "3":
			report = h.reconCore.RunTier3(ctx, *entry)
		default:
			report = h.reconCore.RunTier1(ctx, *entry)
		}
	}
```

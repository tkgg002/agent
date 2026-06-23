# Giải pháp kỹ thuật chi tiết: Phân tách recon_tier_a.go

Tài liệu này chứa mã nguồn chi tiết cho các file helper mới được phân tách từ `recon_tier_a.go`.

---

## 1. `recon_tier_a_lock.go`
```go
package recon

import (
	"context"

	"go.uber.org/zap"
)

// withTableLock acquires an advisory lock on the target table.
func (rc *ReconCore) withTableLock(ctx context.Context, table string) (bool, func()) {
	key := advisoryLockKey("recon_" + table)
	var acquired bool
	if err := rc.db.WithContext(ctx).Raw(
		"SELECT pg_try_advisory_lock(?)", key,
	).Scan(&acquired).Error; err != nil {
		rc.logger.Warn("advisory lock acquire failed — assuming NOT acquired",
			zap.String("table", table), zap.Error(err))
		return false, func() {}
	}
	if !acquired {
		return false, func() {}
	}
	unlock := func() {
		rc.db.Exec("SELECT pg_advisory_unlock(?)", key)
	}
	return true, unlock
}

// AcquireLeader attempts to become the scheduled-recon leader. Returns a
// cancel func that releases the lock + stops the heartbeat. Safe to call
// when Redis is not configured — in that case it always returns
// (true, noop) so single-instance deployments work unchanged.
func (rc *ReconCore) AcquireLeader(ctx context.Context) (bool, func()) {
	if rc.redis == nil {
		return true, func() {}
	}
	client := rc.redis.RawClient()
	if client == nil {
		return true, func() {}
	}

	ok, err := client.SetNX(ctx, rc.cfg.LeaderLockKey, rc.cfg.InstanceID, rc.cfg.LeaderLockTTL).Result()
	if err != nil {
		rc.logger.Warn("leader acquire failed — skipping scheduled run",
			zap.Error(err))
		return false, func() {}
	}
	if !ok {
		return false, func() {}
	}

	hbCtx, cancel := context.WithCancel(ctx)
	go func() {
		ticker := time.NewTicker(rc.cfg.LeaderHeartbeat)
		defer ticker.Stop()
		for {
			select {
			case <-hbCtx.Done():
				return
			case <-ticker.C:
				// Refresh TTL using an ownership-guarded Lua script so we
				// don't accidentally extend a stolen lock.
				script := `if redis.call("get", KEYS[1]) == ARGV[1] then
					return redis.call("pexpire", KEYS[1], ARGV[2])
				else return 0 end`
				_, _ = client.Eval(hbCtx, script,
					[]string{rc.cfg.LeaderLockKey},
					rc.cfg.InstanceID, int64(rc.cfg.LeaderLockTTL/time.Millisecond),
				).Result()
			}
		}
	}()

	release := func() {
		cancel()
		// Release lock only if still ours.
		script := `if redis.call("get", KEYS[1]) == ARGV[1] then
			return redis.call("del", KEYS[1])
		else return 0 end`
		_, _ = client.Eval(context.Background(), script,
			[]string{rc.cfg.LeaderLockKey}, rc.cfg.InstanceID).Result()
	}
	return true, release
}
```

---

## 2. `recon_tier_a_helpers.go`
```go
package recon

import (
	"centralized-data-service/internal/model/source"
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
)

// buildWindows splits the time range [from, to] into contiguous, non-overlapping
// windows of length rc.cfg.WindowSize.
func (rc *ReconCore) buildWindows(from, to time.Time) []window {
	var ws []window
	if to.Before(from) {
		return ws
	}
	cur := from
	for cur.Before(to) {
		next := cur.Add(rc.cfg.WindowSize)
		if next.After(to) {
			next = to
		}
		ws = append(ws, window{Lo: cur, Hi: next})
		cur = next
	}
	return ws
}

// tsField returns the entry's Mongo timestamp field. Helper keeps the
// nil-pointer dereference in one place + lets us unit-test the default
// fallback behavior when registry value is not set.
func tsField(entry source.TableRegistry) string {
	if entry.TimestampField == nil {
		return ""
	}
	return *entry.TimestampField
}

// adaptiveFreeze computes the freeze margin dynamically based on the actual
// observed ingest lag (lagMs). High lag implies wider margin, preventing
// false-positives while pipeline catches up. Clamped to [WindowFreezeMargin, 60m].
func (rc *ReconCore) adaptiveFreeze(lagMs int64) time.Duration {
	base := rc.cfg.WindowFreezeMargin // default 5m
	m := base + time.Duration(lagMs)*time.Millisecond
	if m < base {
		m = base
	}
	if m > 60*time.Minute {
		m = 60 * time.Minute
	}
	return m
}

// lagBetween calculates watermark difference between upstream and downstream.
func lagBetween(upstream, downstream time.Time) int64 {
	if upstream.IsZero() || downstream.IsZero() {
		return 0
	}
	d := upstream.Sub(downstream).Milliseconds()
	if d < 0 {
		return 0
	}
	return d
}

// upsertReconLag records measured ingest/transmute lag into the database.
func (rc *ReconCore) upsertReconLag(ctx context.Context, table, col string, lagMs int64) {
	if col != "ingest_lag_ms" && col != "transmute_lag_ms" {
		return
	}
	if err := rc.db.WithContext(ctx).Exec(fmt.Sprintf(
		`INSERT INTO cdc_system.recon_lag (table_name, %s, measured_at)
		 VALUES (?, ?, NOW())
		 ON CONFLICT (table_name) DO UPDATE SET %s = EXCLUDED.%s, measured_at = NOW()`,
		col, col, col), table, lagMs).Error; err != nil {
		rc.logger.Warn("recon_lag upsert failed", zap.String("table", table), zap.Error(err))
	}
}

// pickScanRange is a backward compatibility wrapper for RunTier2.
func (rc *ReconCore) pickScanRange(ctx context.Context, entry source.TableRegistry) (time.Time, time.Time, error) {
	lo, hi, _, err := rc.pickScanRangeWithLag(ctx, entry)
	return lo, hi, err
}

// pickScanRangeWithLag resolves the lower and upper watermarks dynamically
// taking current replication lag into account.
func (rc *ReconCore) pickScanRangeWithLag(ctx context.Context, entry source.TableRegistry) (time.Time, time.Time, int64, error) {
	srcMax, err := rc.sourceAgent.MaxWindowTs(ctx, entry.SourceURL, entry.SourceDB, entry.SourceTable, tsField(entry))
	if err != nil {
		return time.Time{}, time.Time{}, 0, fmt.Errorf("source max ts: %w", err)
	}
	dstMax, err := rc.destAgent.MaxWindowTs(ctx, entry.QualifiedTarget())
	if err != nil {
		return time.Time{}, time.Time{}, 0, fmt.Errorf("dest max ts: %w", err)
	}

	ingestLagMs := lagBetween(srcMax, dstMax)
	rc.upsertReconLag(ctx, entry.TargetTable, "ingest_lag_ms", ingestLagMs)

	nowFreeze := time.Now().UTC().Add(-rc.adaptiveFreeze(ingestLagMs))
	upper := nowFreeze
	if !srcMax.IsZero() && srcMax.Before(upper) {
		upper = srcMax.Add(time.Millisecond)
	}
	if !dstMax.IsZero() && dstMax.Before(upper) {
		upper = dstMax.Add(time.Millisecond)
	}
	lower := upper.Add(-rc.cfg.WindowLookback)
	return lower, upper, ingestLagMs, nil
}
```

---

## 3. `recon_tier_a_run.go`
```go
package recon

import (
	"centralized-data-service/internal/model/source"
	"centralized-data-service/internal/model/system"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"centralized-data-service/pkgs/metrics"

	"go.uber.org/zap"
)

// RunTier1 performs O(1) total count check, falling back to aggregate-bucket comparison on mismatches.
func (rc *ReconCore) RunTier1(ctx context.Context, entry source.TableRegistry) *system.ReconciliationReport {
	acquired, unlock := rc.withTableLock(ctx, entry.TargetTable)
	defer unlock()
	if !acquired {
		rc.logger.Info("tier1 skipped — previous run ongoing",
			zap.String("table", entry.TargetTable))
		return rc.errorReport(entry, "count", 1, fmt.Errorf("previous run ongoing"))
	}

	handle, err := rc.beginRun(ctx, entry.TargetTable, 1)
	if err != nil {
		rc.logger.Error("tier1 beginRun failed", zap.Error(err))
		return rc.errorReport(entry, "count", 1, err)
	}

	status := "success"
	defer func() {
		rc.finishRun(ctx, handle, status, "")
	}()

	lo, hi, _, err := rc.pickScanRangeWithLag(ctx, entry)
	if err != nil {
		status = "failed"
		return rc.errorReport(entry, "count", 1, err)
	}

	srcEst, errE := rc.sourceAgent.EstimatedCount(ctx, entry.SourceURL, entry.SourceDB, entry.SourceTable)
	if errE != nil {
		status = "failed"
		return rc.errorReport(entry, "count", 1, fmt.Errorf("src estimated count: %w", errE))
	}
	dstTotal, errD := rc.destAgent.CountRows(ctx, entry.QualifiedTarget(), entry.PrimaryKeyField)
	if errD != nil {
		status = "failed"
		return rc.errorReport(entry, "count", 1, fmt.Errorf("dst count: %w", errD))
	}

	estTolerance := srcEst / 1000
	if estTolerance < 1 {
		estTolerance = 1
	}
	if abs(srcEst-dstTotal) <= estTolerance {
		duration := int(time.Since(handle.started).Milliseconds())
		report := &system.ReconciliationReport{
			TargetTable: entry.TargetTable, SourceDB: entry.SourceDB,
			SourceCount: &srcEst, DestCount: dstTotal, Diff: srcEst - dstTotal,
			TotalSourceCount: &srcEst, TotalDestCount: &dstTotal,
			CheckType: "count_total", Status: "ok", Tier: 1,
			DurationMs: &duration, CheckedAt: time.Now().UTC(),
		}
		rc.stampA(report, entry)
		metrics.ReconDrift.WithLabelValues(entry.TargetTable, "1").Set(0)
		rc.logger.Info("tier0 count_total ok",
			zap.String("table", entry.TargetTable),
			zap.Int64("src_est", srcEst), zap.Int64("dst", dstTotal))
		return report
	}

	type winDrift struct {
		Lo          time.Time `json:"lo"`
		Hi          time.Time `json:"hi"`
		SourceCount int64     `json:"source_count"`
		DestCount   int64     `json:"dest_count"`
	}
	var srcBuckets map[int64]int64
	for _, f := range append([]string{tsField(entry)}, entry.GetCandidates()...) {
		srcBuckets, err = rc.sourceAgent.BucketCounts(ctx, entry.SourceURL, entry.SourceDB, entry.SourceTable, f, lo, hi)
		if err != nil {
			status = "failed"
			return rc.errorReport(entry, "count", 1, fmt.Errorf("src bucket counts (%s): %w", f, err))
		}
		if len(srcBuckets) > 0 {
			break
		}
	}
	dstBuckets, err := rc.destAgent.BucketCounts(ctx, entry.QualifiedTarget(), entry.PrimaryKeyField, lo, hi)
	if err != nil {
		status = "failed"
		return rc.errorReport(entry, "count", 1, fmt.Errorf("dst bucket counts: %w", err))
	}

	var drifted []winDrift
	var totalSrc, totalDst int64
	keys := make(map[int64]struct{}, len(srcBuckets)+len(dstBuckets))
	for k := range srcBuckets {
		keys[k] = struct{}{}
	}
	for k := range dstBuckets {
		keys[k] = struct{}{}
	}
	for k := range keys {
		s, d := srcBuckets[k], dstBuckets[k].Count
		totalSrc += s
		totalDst += d
		if abs(s-d) >= rc.cfg.CountDriftThreshold {
			bLo := time.UnixMilli(k).UTC()
			drifted = append(drifted, winDrift{Lo: bLo, Hi: bLo.Add(time.Hour), SourceCount: s, DestCount: d})
		}
	}
	handle.windowsCount = len(keys)
	handle.docsScanned = totalSrc + totalDst
	handle.mismatches = len(drifted)

	statusStr := "ok"
	if len(drifted) > 0 {
		statusStr = "drift"
	} else {
		if exact, err := rc.sourceAgent.CountDocuments(ctx, entry.SourceURL, entry.SourceDB, entry.SourceTable); err == nil {
			srcEst = exact
			if abs(exact-dstTotal) > 0 {
				statusStr = "drift"
			}
		}
	}

	driftJSON, _ := json.Marshal(drifted)
	duration := int(time.Since(handle.started).Milliseconds())
	srcTotal := totalSrc
	report := &system.ReconciliationReport{
		TargetTable:      entry.TargetTable,
		SourceDB:         entry.SourceDB,
		SourceCount:      &srcTotal,
		TotalSourceCount: &srcEst,
		TotalDestCount:   &dstTotal,
		DestCount:        totalDst,
		Diff:             totalSrc - totalDst,
		StaleCount:       len(drifted),
		StaleIDs:         driftJSON,
		CheckType:        "count_windowed",
		Status:           statusStr,
		Tier:             1,
		DurationMs:       &duration,
		CheckedAt:        time.Now().UTC(),
	}
	rc.stampA(report, entry)
	rc.alertOnReport(ctx, "source_shadow", entry.TargetTable, statusStr, 0, report.Diff)

	metrics.ReconDrift.WithLabelValues(entry.TargetTable, "1").Set(float64(len(drifted)))

	rc.logger.Info("tier1 bucket_aggregate",
		zap.String("table", entry.TargetTable),
		zap.Int("buckets", handle.windowsCount),
		zap.Int("drifted_buckets", len(drifted)),
		zap.Int64("total_src", totalSrc),
		zap.Int64("total_dst", totalDst),
	)
	return report
}

// RunTier2 drills down into drifted windows using XOR hash, and lists IDs to find exact diffs.
func (rc *ReconCore) RunTier2(ctx context.Context, entry source.TableRegistry) *system.ReconciliationReport {
	acquired, unlock := rc.withTableLock(ctx, entry.TargetTable)
	defer unlock()
	if !acquired {
		rc.logger.Info("tier2 skipped — previous run ongoing",
			zap.String("table", entry.TargetTable))
		return rc.errorReport(entry, "hash_window", 2, fmt.Errorf("previous run ongoing"))
	}

	handle, err := rc.beginRun(ctx, entry.TargetTable, 2)
	if err != nil {
		return rc.errorReport(entry, "hash_window", 2, err)
	}
	status := "success"
	defer func() { rc.finishRun(ctx, handle, status, "") }()

	lo, hi, err := rc.pickScanRange(ctx, entry)
	if err != nil {
		status = "failed"
		return rc.errorReport(entry, "hash_window", 2, err)
	}
	windows := rc.buildWindows(lo, hi)
	handle.windowsCount = len(windows)

	var missingFromDest []string
	var missingFromSrc []string
	var driftedWindows int

	for _, w := range windows {
		srcRes, err := rc.sourceAgent.HashWindow(ctx, entry.SourceURL, entry.SourceDB, entry.SourceTable, tsField(entry), w.Lo, w.Hi)
		if err != nil {
			status = "failed"
			return rc.errorReport(entry, "hash_window", 2, fmt.Errorf("src hash window %v: %w", w.Lo, err))
		}
		dstRes, err := rc.destAgent.HashWindow(ctx, entry.QualifiedTarget(), entry.PrimaryKeyField, w.Lo, w.Hi)
		if err != nil {
			status = "failed"
			return rc.errorReport(entry, "hash_window", 2, fmt.Errorf("dst hash window %v: %w", w.Lo, err))
		}
		handle.docsScanned += srcRes.Count + dstRes.Count

		if srcRes.Count == dstRes.Count && srcRes.XorHash == dstRes.XorHash {
			continue
		}
		driftedWindows++

		srcIDs, err := rc.sourceAgent.ListIDsInWindow(ctx, entry.SourceURL, entry.SourceDB, entry.SourceTable, tsField(entry), w.Lo, w.Hi)
		if err != nil {
			rc.logger.Warn("src list ids failed", zap.Error(err))
			continue
		}
		dstIDs, err := rc.destAgent.ListIDsInWindow(ctx, entry.QualifiedTarget(), entry.PrimaryKeyField, w.Lo, w.Hi)
		if err != nil {
			rc.logger.Warn("dst list ids failed", zap.Error(err))
			continue
		}
		mFromDst, mFromSrc := diffIDs(srcIDs, dstIDs)
		missingFromDest = append(missingFromDest, mFromDst...)
		missingFromSrc = append(missingFromSrc, mFromSrc...)
	}

	handle.mismatches = len(missingFromDest) + len(missingFromSrc)

	missingJSON, _ := json.Marshal(missingFromDest)
	staleJSON, _ := json.Marshal(map[string][]string{
		"missing_from_dest": missingFromDest,
		"missing_from_src":  missingFromSrc,
	})

	statusStr := "ok"
	if driftedWindows > 0 {
		statusStr = "drift"
	}

	duration := int(time.Since(handle.started).Milliseconds())
	report := &system.ReconciliationReport{
		TargetTable:  entry.TargetTable,
		SourceDB:     entry.SourceDB,
		MissingCount: len(missingFromDest),
		MissingIDs:   missingJSON,
		StaleCount:   driftedWindows,
		StaleIDs:     staleJSON,
		CheckType:    "hash_window",
		Status:       statusStr,
		Tier:         2,
		DurationMs:   &duration,
		CheckedAt:    time.Now().UTC(),
	}
	rc.stampA(report, entry)

	metrics.ReconDrift.WithLabelValues(entry.TargetTable, "2").Set(float64(handle.mismatches))

	rc.logger.Info("tier2 hash_window",
		zap.String("table", entry.TargetTable),
		zap.Int("windows", len(windows)),
		zap.Int("drifted_windows", driftedWindows),
		zap.Int("missing_from_dest", len(missingFromDest)),
		zap.Int("missing_from_src", len(missingFromSrc)),
	)
	return report
}

// RunTier3 computes the 256-bucket fingerprint on both sides and diffs them (whole-table scan).
func (rc *ReconCore) RunTier3(ctx context.Context, entry source.TableRegistry) *system.ReconciliationReport {
	acquired, unlock := rc.withTableLock(ctx, entry.TargetTable)
	defer unlock()
	if !acquired {
		return rc.errorReport(entry, "bucket_hash", 3, fmt.Errorf("previous run ongoing"))
	}

	handle, err := rc.beginRun(ctx, entry.TargetTable, 3)
	if err != nil {
		return rc.errorReport(entry, "bucket_hash", 3, err)
	}
	status := "success"
	defer func() { rc.finishRun(ctx, handle, status, "") }()

	if !rc.isOffPeak(time.Now().UTC()) {
		rc.logger.Info("tier3 skipped — outside off-peak window",
			zap.String("table", entry.TargetTable),
			zap.Int("off_peak_start", rc.cfg.Tier3OffPeakStart),
			zap.Int("off_peak_end", rc.cfg.Tier3OffPeakEnd),
		)
		status = "cancelled"
		return rc.errorReport(entry, "bucket_hash", 3, fmt.Errorf("outside off-peak window"))
	}

	destCount, err := rc.destAgent.CountRows(ctx, entry.QualifiedTarget(), entry.PrimaryKeyField)
	if err != nil {
		status = "failed"
		return rc.errorReport(entry, "bucket_hash", 3, err)
	}
	if destCount > rc.cfg.Tier3MaxDocsPerRun {
		rc.logger.Warn("tier3 budget exceeded — falling back to 7-day window scan",
			zap.String("table", entry.TargetTable),
			zap.Int64("dest_count", destCount),
			zap.Int64("budget", rc.cfg.Tier3MaxDocsPerRun),
		)
		handle.mismatches = 0
		return rc.RunTier2(ctx, entry)
	}

	srcBuckets, err := rc.sourceAgent.BucketHash(ctx, entry.SourceURL, entry.SourceDB, entry.SourceTable, tsField(entry))
	if err != nil {
		status = "failed"
		return rc.errorReport(entry, "bucket_hash", 3, err)
	}
	dstBuckets, err := rc.destAgent.BucketHash(ctx, entry.QualifiedTarget(), entry.PrimaryKeyField)
	if err != nil {
		status = "failed"
		return rc.errorReport(entry, "bucket_hash", 3, err)
	}
	handle.docsScanned = srcBuckets.Total + dstBuckets.Total

	var driftedBuckets []int
	for i := 0; i < 256; i++ {
		if srcBuckets.Buckets[i] != dstBuckets.Buckets[i] {
			driftedBuckets = append(driftedBuckets, i)
		}
	}
	handle.mismatches = len(driftedBuckets)

	statusStr := "ok"
	if len(driftedBuckets) > 0 {
		statusStr = "drift"
	}

	payload := map[string]interface{}{
		"drifted_buckets": driftedBuckets,
		"src_total":       srcBuckets.Total,
		"dst_total":       dstBuckets.Total,
	}
	staleJSON, _ := json.Marshal(payload)
	duration := int(time.Since(handle.started).Milliseconds())
	srcBucketTotal := srcBuckets.Total
	report := &system.ReconciliationReport{
		TargetTable: entry.TargetTable,
		SourceDB:    entry.SourceDB,
		SourceCount: &srcBucketTotal,
		DestCount:   dstBuckets.Total,
		Diff:        srcBuckets.Total - dstBuckets.Total,
		StaleCount:  len(driftedBuckets),
		StaleIDs:    staleJSON,
		CheckType:   "bucket_hash",
		Status:      statusStr,
		Tier:        3,
		DurationMs:  &duration,
		CheckedAt:   time.Now().UTC(),
	}
	rc.stampA(report, entry)

	metrics.ReconDrift.WithLabelValues(entry.TargetTable, "3").Set(float64(len(driftedBuckets)))

	rc.logger.Info("tier3 bucket_hash",
		zap.String("table", entry.TargetTable),
		zap.Int("drifted_buckets", len(driftedBuckets)),
		zap.Int64("src_total", srcBuckets.Total),
		zap.Int64("dst_total", dstBuckets.Total),
	)
	return report
}
```

---

## 4. `recon_tier_a_prune.go`
```go
package recon

import (
	"centralized-data-service/internal/model/source"
	"centralized-data-service/internal/model/system"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"centralized-data-service/pkgs/metrics"

	"go.uber.org/zap"
)

// RunOrphanPrune soft-deletes shadow rows whose _source_id is no longer in MongoDB.
func (rc *ReconCore) RunOrphanPrune(ctx context.Context, entry source.TableRegistry) *system.ReconciliationReport {
	acquired, unlock := rc.withTableLock(ctx, entry.TargetTable)
	defer unlock()
	if !acquired {
		rc.logger.Info("orphan_prune skipped — previous run ongoing", zap.String("table", entry.TargetTable))
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

	var dstIDs []string
	if e := rc.shadowPlane.WithContext(ctx).Raw(
		fmt.Sprintf(`SELECT "_source_id"::text FROM %s WHERE NOT "_deleted" AND "_source_id" IS NOT NULL`,
			quoteRelation(entry.QualifiedTarget())),
	).Scan(&dstIDs).Error; e != nil {
		status = "failed"
		return rc.errorReport(entry, "orphan_prune", 2, fmt.Errorf("dst list ids: %w", e))
	}

	shadowSet := make(map[string]struct{}, len(dstIDs))
	for _, id := range dstIDs {
		shadowSet[id] = struct{}{}
	}

	idChan, errChan := rc.sourceAgent.StreamAllIDs(ctx, entry.SourceURL, entry.SourceDB, entry.SourceTable)
	srcCount := 0
	var streamErr error

	for idChan != nil || errChan != nil {
		select {
		case id, ok := <-idChan:
			if !ok {
				idChan = nil
				break
			}
			srcCount++
			handle.docsScanned++
			delete(shadowSet, id)

		case err, ok := <-errChan:
			if !ok {
				errChan = nil
				break
			}
			if err != nil {
				streamErr = err
			}
		}
	}

	handle.docsScanned += int64(len(dstIDs))

	if srcCount == 0 && len(dstIDs) > 0 {
		rc.logger.Warn("orphan_prune skip — source returned 0 ids (refuse to prune entire shadow)",
			zap.String("table", entry.TargetTable),
			zap.Int("shadow_ids", len(dstIDs)),
			zap.NamedError("stream_err", streamErr),
		)
		duration := int(time.Since(handle.started).Milliseconds())
		report := &system.ReconciliationReport{
			TargetTable: entry.TargetTable, SourceDB: entry.SourceDB,
			MissingCount: 0, StaleCount: 0, CheckType: "orphan_prune",
			Status: "warning", Tier: 2, DurationMs: &duration, CheckedAt: time.Now().UTC(),
		}
		rc.stampA(report, entry)
		return report
	}

	if streamErr != nil {
		rc.logger.Warn("orphan_prune: source stream had error (partial data)",
			zap.String("table", entry.TargetTable),
			zap.Int("src_ids_read", srcCount),
			zap.Error(streamErr),
		)
	}

	orphans := make([]string, 0, len(shadowSet))
	for id := range shadowSet {
		orphans = append(orphans, id)
	}

	pruned := 0
	if len(orphans) > 0 {
		updSQL := fmt.Sprintf(
			`UPDATE %s SET "_deleted" = TRUE, "_updated_at" = NOW() WHERE "_source_id" IN (?) AND NOT "_deleted"`,
			quoteRelation(entry.QualifiedTarget()),
		)
		const batch = 1000
		for i := 0; i < len(orphans); i += batch {
			end := i + batch
			if end > len(orphans) {
				end = len(orphans)
			}
			res := rc.shadowPlane.WithContext(ctx).Exec(updSQL, orphans[i:end])
			if res.Error != nil {
				status = "failed"
				return rc.errorReport(entry, "orphan_prune", 2, fmt.Errorf("soft-delete orphans: %w", res.Error))
			}
			pruned += int(res.RowsAffected)
		}
	}
	handle.mismatches = len(orphans)

	orphJSON, _ := json.Marshal(orphans)
	statusStr := "ok"
	if pruned > 0 {
		statusStr = "drift"
	}
	duration := int(time.Since(handle.started).Milliseconds())
	report := &system.ReconciliationReport{
		TargetTable:  entry.TargetTable,
		SourceDB:     entry.SourceDB,
		MissingCount: 0,
		StaleCount:   pruned,
		StaleIDs:     orphJSON,
		CheckType:    "orphan_prune",
		Status:       statusStr,
		Tier:         2,
		DurationMs:   &duration,
		CheckedAt:    time.Now().UTC(),
	}
	rc.stampA(report, entry)
	metrics.ReconDrift.WithLabelValues(entry.TargetTable, "prune").Set(float64(len(orphans)))
	rc.logger.Info("orphan_prune v2",
		zap.String("table", entry.TargetTable),
		zap.Int("src_ids_streamed", srcCount),
		zap.Int("shadow_ids", len(dstIDs)),
		zap.Int("orphans", len(orphans)),
		zap.Int("pruned", pruned))
	return report
}

// PruneAllOrphans runs RunOrphanPrune for all active tables.
func (rc *ReconCore) PruneAllOrphans(ctx context.Context) []*system.ReconciliationReport {
	entries := rc.listActiveTableConfigs(ctx)
	reports := make([]*system.ReconciliationReport, 0, len(entries))
	for _, e := range entries {
		reports = append(reports, rc.RunOrphanPrune(ctx, e))
	}
	return reports
}
```

---

## 5. `recon_models.go` (Thay đổi)
Chúng ta sẽ chuyển `reconRunHandle` từ `recon_tier_a.go` sang `recon_models.go` để các file trong package đều import được.

```go
// [MODIFY] recon_models.go
// Thêm:
type reconRunHandle struct {
	id           string
	table        string
	tier         int
	started      time.Time
	docsScanned  int64
	windowsCount int
	mismatches   int
	healActions  int
}
```

---

## 6. `recon_tier_a.go` (Rút gọn)
```go
// Package service — recon_tier_a.go
//
// Tier A: Source ↔ Shadow reconciliation module.
// Contains count checks, XOR window hashes, whole-table fingerprinting (buckets),
// and orphan prune daemon logic. Separated into Lock, Helpers, Run, and Prune.
package recon
```

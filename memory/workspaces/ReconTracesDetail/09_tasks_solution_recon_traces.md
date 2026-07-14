# Hồ sơ Giải pháp Kỹ thuật Chi tiết - Tối ưu hóa Tracing & Hiệu năng Reconciliation

Tài liệu này chứa đặc tả thiết kế chi tiết của các thay đổi mã nguồn trong centralized-data-service để tối ưu hóa hiệu năng hash_window và ngăn ngừa Span Storm.

## 1. pkgs/observability/trace_helpers.go
Khai báo context key và helper methods để quản lý cờ bypass tracing:

```go
type contextKey string
const skipTraceKey contextKey = "recon.skip_window_trace"

// ContextWithSkipTrace returns a new context with the skip trace flag set.
func ContextWithSkipTrace(ctx context.Context) context.Context {
	return context.WithValue(ctx, skipTraceKey, true)
}

// IsTraceSkipped returns true if the context contains the skip trace flag.
func IsTraceSkipped(ctx context.Context) bool {
	val, ok := ctx.Value(skipTraceKey).(bool)
	return ok && val
}
```

## 2. internal/service/recon/recon_hash.go
Cập nhật `HashWindow` kiểm tra cờ bypass qua helper và import `oteltrace "go.opentelemetry.io/otel/trace"`:

```go
func (sa *ReconSourceAgent) HashWindow(ctx context.Context, sourceURL, database, collection, timestampField string, tLo, tHi time.Time) (*WindowResult, error) {
	var span oteltrace.Span
	if !observability.IsTraceSkipped(ctx) {
		ctx, span = observability.ChildSpan(ctx, "recon.source.hash_window",
			attribute.String("db.database", database),
			attribute.String("db.collection", collection),
			attribute.String("recon.timestamp_field", timestampField),
			attribute.String("recon.t_lo", tLo.Format(time.RFC3339)),
			attribute.String("recon.t_hi", tHi.Format(time.RFC3339)),
			attribute.Bool("db.is_postgres", isPostgres(sourceURL)),
		)
	}
	defer func() {
		if span != nil {
			span.End()
		}
	}()
    // ... logic giữ nguyên ...
}
```

## 3. internal/service/recon/recon_dest_hash.go
Cập nhật `HashWindow` kiểm tra cờ bypass qua helper và import `oteltrace "go.opentelemetry.io/otel/trace"`:

```go
func (da *ReconDestAgent) HashWindow(ctx context.Context, tableName, pkColumn, timestampField string, tLo, tHi time.Time) (*WindowResult, error) {
	var span oteltrace.Span
	if !observability.IsTraceSkipped(ctx) {
		ctx, span = observability.ChildSpan(ctx, "pg.hash_window",
			attribute.String("db.table", tableName),
			attribute.String("recon.timestamp_field", timestampField),
			attribute.String("recon.t_lo", tLo.Format(time.RFC3339)),
			attribute.String("recon.t_hi", tHi.Format(time.RFC3339)),
		)
	}
	defer func() {
		if span != nil {
			span.End()
		}
	}()
    // ... logic giữ nguyên ...
}
```

## 4. internal/service/recon/recon_tier_a.go
Thực hiện **Global Hash Verification & Block Partitioning** trong `RunHashWindowCheck` và inject cờ bypass:

```go
	// 1. Global Hash Check & Block Partitioning (Ngưỡng 7 ngày)
	const maxGlobalDays = 7
	diffDays := hi.Sub(lo).Hours() / 24

	if diffDays <= maxGlobalDays {
		ctxVerify, spanVerify := observability.ChildSpan(ctx, "cdc.recon.verify_global_range", attribute.String("table", entry.TargetTable))
		srcGlobal, errS := rc.sourceAgent.HashWindow(ctxVerify, entry.SourceURL, entry.SourceDB, entry.SourceTable, srcTS, lo, hi)
		dstGlobal, errD := rc.destAgent.HashWindow(ctxVerify, entry.QualifiedTarget(), entry.PrimaryKeyField, dstTS, lo, hi)
		observability.EndSpan(spanVerify, nil)

		if errS == nil && errD == nil && srcGlobal.Count == dstGlobal.Count && srcGlobal.XorHash == dstGlobal.XorHash {
			observability.Ctx(ctx, rc.logger).Info("[tier2] global hash match — no drift detected in range",
				zap.String("table", entry.TargetTable),
				zap.Int64("count", srcGlobal.Count),
				zap.Time("lo", lo),
				zap.Time("hi", hi),
			)
			duration := int(time.Since(handle.started).Milliseconds())
			report := &recon.ReconciliationReport{
				TargetTable: entry.QualifiedTarget(), SourceDB: entry.SourceDB,
				SourceCount: &srcGlobal.Count, DestCount: dstGlobal.Count, Diff: 0,
				CheckType: "hash_window", Status: "ok",
				Segment: segmentSourceShadow, DurationMs: &duration, CheckedAt: time.Now().UTC(),
			}
			rc.stampA(report, entry)
			rc.finishRun(ctx, handle, "success", "")
			return report
		}
	} else {
		// Dải thời gian > 7 ngày -> Chia thành các block 7 ngày để tránh Table Scan lớn
		allMatched := true
		var totalCount int64
		curBlock := lo
		
		ctxBlock, spanBlock := observability.ChildSpan(ctx, "cdc.recon.verify_global_blocks", attribute.String("table", entry.TargetTable))
		for curBlock.Before(hi) {
			nextBlock := curBlock.Add(maxGlobalDays * 24 * time.Hour)
			if nextBlock.After(hi) {
				nextBlock = hi
			}
			
			srcBlock, errS := rc.sourceAgent.HashWindow(ctxBlock, entry.SourceURL, entry.SourceDB, entry.SourceTable, srcTS, curBlock, nextBlock)
			dstBlock, errD := rc.destAgent.HashWindow(ctxBlock, entry.QualifiedTarget(), entry.PrimaryKeyField, dstTS, curBlock, nextBlock)
			
			if errS != nil || errD != nil || srcBlock.Count != dstBlock.Count || srcBlock.XorHash != dstBlock.XorHash {
				allMatched = false
				break
			}
			totalCount += srcBlock.Count
			curBlock = nextBlock
		}
		observability.EndSpan(spanBlock, nil)

		if allMatched {
			observability.Ctx(ctx, rc.logger).Info("[tier2] global blocks hash match — no drift detected in range",
				zap.String("table", entry.TargetTable),
				zap.Int64("count", totalCount),
				zap.Time("lo", lo),
				zap.Time("hi", hi),
			)
			duration := int(time.Since(handle.started).Milliseconds())
			report := &recon.ReconciliationReport{
				TargetTable: entry.QualifiedTarget(), SourceDB: entry.SourceDB,
				SourceCount: &totalCount, DestCount: totalCount, Diff: 0,
				CheckType: "hash_window", Status: "ok",
				Segment: segmentSourceShadow, DurationMs: &duration, CheckedAt: time.Now().UTC(),
			}
			rc.stampA(report, entry)
			rc.finishRun(ctx, handle, "success", "")
			return report
		}
	}

	// 2. Fallback window loop
	windows := rc.buildWindows(lo, hi)
	handle.windowsCount = len(windows)
    // ...
	ctxLoop, spanLoop := observability.ChildSpan(ctx, "cdc.recon.window_loop", attribute.Int("windows_count", len(windows)))
	defer spanLoop.End()

	// Smart Tracing: bypass window sạch
	ctxLoop = observability.ContextWithSkipTrace(ctxLoop)
    // ...
```

## 5. internal/service/recon/recon_tier_b.go
Bypass trace cho vòng lặp window Segment B:

```go
	ctxLoop, spanLoop := observability.ChildSpan(ctx, "cdc.recon.window_loop_b", attribute.Int("windows_count", len(bucketKeys)))
	defer spanLoop.End()

	// Smart Tracing: bypass window sạch
	ctxLoop = observability.ContextWithSkipTrace(ctxLoop)
```

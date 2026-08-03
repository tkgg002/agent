# 09 Tasks Solution: Giải Pháp Kỹ Thuật Phân Cấp OTel Trace Spans

## 1. Thiết kế thay đổi trong `recon_stream_bucket_engine.go`

Thay thế `drillSubWindows` hiện tại (đang tạo 1 `cdc.recon.hash_window` duy nhất cho cả ngày) bằng việc tạo `cdc.recon.hash_window` riêng cho từng cửa sổ 15 phút:

```go
func (e *ChunkStreamBucketEngine) drillSubWindows(
	ctx context.Context,
	entry source.TableRegistry,
	dayStart, dayEnd time.Time,
	srcTS, dstTS, srcPK, dstPK string,
	dayIdx int,
	staleAcc *StaleIDsPayload,
) []DriftWindow {
	var drifts []DriftWindow

	for i := 0; i < subWindowsPerDay; i++ {
		subStart := dayStart.Add(time.Duration(i) * subWindowDuration)
		if !subStart.Before(dayEnd) {
			break
		}
		subEnd := subStart.Add(subWindowDuration)
		if subEnd.After(dayEnd) {
			subEnd = dayEnd
		}

		// 1. Parent sub-window span: cdc.recon.hash_window: <table> [HH:MM:SS -> HH:MM:SS]
		subWinTitle := fmt.Sprintf("cdc.recon.hash_window: %s [%s -> %s]",
			entry.TargetTable,
			subStart.UTC().Format("15:04:05"),
			subEnd.UTC().Format("15:04:05"),
		)
		ctxSubWin, subWinSpan := observability.ChildSpan(ctx, subWinTitle,
			attribute.String("recon.table", entry.TargetTable),
			attribute.String("recon.sub_start", subStart.Format(time.RFC3339)),
			attribute.String("recon.sub_end", subEnd.Format(time.RFC3339)),
		)

		// 2. Hash source & dest (Child Spans của ctxSubWin)
		srcResult, srcErr := e.sourceAgent.HashWindow(ctxSubWin, entry.SourceURL, entry.SourceDB, entry.SourceTable, srcTS, subStart, subEnd)
		if srcErr != nil {
			e.logger.Warn("source sub-window hash failed", zap.Int("day", dayIdx), zap.Int("bucket", i), zap.Error(srcErr))
			subWinSpan.End()
			continue
		}

		dstResult, dstErr := e.destAgent.HashWindow(ctxSubWin, entry.QualifiedTarget(), dstPK, dstTS, subStart, subEnd)
		if dstErr != nil {
			e.logger.Warn("dest sub-window hash failed", zap.Int("day", dayIdx), zap.Int("bucket", i), zap.Error(dstErr))
			subWinSpan.End()
			continue
		}

		// 3. Nếu lệch -> Tạo cdc.recon.drift_drill_down là Child Span của ctxSubWin
		if srcResult.XorHash != dstResult.XorHash || srcResult.Count != dstResult.Count {
			drifts = append(drifts, DriftWindow{
				StartTime: subStart,
				EndTime:   subEnd,
				SrcHash:   srcResult.XorHash,
				DstHash:   dstResult.XorHash,
				SrcCount:  srcResult.Count,
				DstCount:  dstResult.Count,
			})

			if staleAcc != nil && e.sourceAgent != nil && e.destAgent != nil {
				driftTitle := fmt.Sprintf("cdc.recon.drift_drill_down: %s [%s -> %s]",
					entry.TargetTable,
					subStart.UTC().Format("15:04:05"),
					subEnd.UTC().Format("15:04:05"),
				)
				ctxDiff, diffSpan := observability.ChildSpan(ctxSubWin, driftTitle,
					attribute.String("recon.table", entry.TargetTable),
					attribute.String("recon.sub_start", subStart.Format(time.RFC3339)),
					attribute.String("recon.sub_end", subEnd.Format(time.RFC3339)),
				)

				// 4. ListIDTsInWindow (Child Spans của ctxDiff)
				srcIDTs, errS := e.sourceAgent.ListIDTsInWindow(ctxDiff, entry.SourceURL, entry.SourceDB, entry.SourceTable, srcTS, subStart, subEnd)
				dstIDTs, errD := e.destAgent.ListIDTsInWindow(ctxDiff, entry.QualifiedTarget(), dstPK, dstTS, subStart, subEnd)
				if errS == nil && errD == nil {
					mShadow, mMaster, mm := diffIDTs(srcIDTs, dstIDTs)
					staleAcc.MissingFromShadow = append(staleAcc.MissingFromShadow, mShadow...)
					staleAcc.MissingFromMaster = append(staleAcc.MissingFromMaster, mMaster...)
					staleAcc.Mismatched = append(staleAcc.Mismatched, mm...)
					diffSpan.SetAttributes(
						attribute.Int("recon.missing_from_shadow", len(mShadow)),
						attribute.Int("recon.missing_from_master", len(mMaster)),
						attribute.Int("recon.mismatched", len(mm)),
					)
				}
				diffSpan.End()
			}
		}
		subWinSpan.End()
	}

	return drifts
}
```

## 2. Thiết kế thay đổi trong các file Agent (Naming Spans)

- `recon_hash.go`: `recon.source.hash_window: <collection> [HH:MM:SS -> HH:MM:SS]`
- `recon_dest_hash.go`: `pg.hash_window: <table> [HH:MM:SS -> HH:MM:SS]`
- `recon_stream.go`: `recon.source.diff_idts: <collection> [HH:MM:SS -> HH:MM:SS]`
- `recon_dest_query.go`: `pg.diff_idts: <table> [HH:MM:SS -> HH:MM:SS]`

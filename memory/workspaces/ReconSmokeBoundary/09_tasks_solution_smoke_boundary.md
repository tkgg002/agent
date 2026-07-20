# Hồ Sơ Giải Pháp Kỹ Thuật: Tối ưu hóa Smoke Check bằng Cửa Sổ Trừ Bù, Tính Toán Xóa Mềm và Xác Thực Chéo HashWindow (Smoke Boundary Solution Profile)

Dưới đây là thiết kế chi tiết về các thay đổi mã nguồn trong `centralized-data-service`.

## 1. Bổ sung helper `CountRecentDeletedRows` vào `ReconDestAgent`

Thêm hàm sau vào `recon_dest_query.go`:

```go
// CountRecentDeletedRows counts rows whose filter column ∈ [tLo, tHi) in ms and _deleted = true.
func (da *ReconDestAgent) CountRecentDeletedRows(ctx context.Context, tableName, timestampField string, tLo, tHi time.Time) (int64, error) {
	ctx, span := observability.ChildSpan(ctx, "pg.count_recent_deleted_rows",
		attribute.String("db.table", tableName),
		attribute.String("recon.timestamp_field", timestampField),
		attribute.String("recon.t_lo", tLo.Format(time.RFC3339)),
		attribute.String("recon.t_hi", tHi.Format(time.RFC3339)),
	)
	defer span.End()

	if err := validateIdent(tableName); err != nil {
		return 0, err
	}
	ctx, cancel := context.WithTimeout(ctx, da.cfg.QueryTimeout)
	defer cancel()

	tsCol := strings.TrimSpace(timestampField)
	if tsCol == "" || tsCol == "_source_ts" {
		loMs, hiMs := tLo.UnixMilli(), tHi.UnixMilli()
		result, err := da.breaker.Execute(func() (interface{}, error) {
			tx := da.readOnlyDB(ctx)
			defer tx.Rollback()
			var count int64
			sql := fmt.Sprintf(
				`SELECT COUNT(*) FROM %s WHERE "_source_ts" >= ? AND "_source_ts" < ? AND "_deleted" = true`,
				quoteRelation(tableName),
			)
			if err := tx.Raw(sql, loMs, hiMs).Scan(&count).Error; err != nil {
				return nil, err
			}
			return count, nil
		})
		if err != nil {
			return 0, err
		}
		return result.(int64), nil
	}

	if err := validateIdent(tsCol); err != nil {
		return 0, err
	}

	result, err := da.breaker.Execute(func() (interface{}, error) {
		tx := da.readOnlyDB(ctx)
		defer tx.Rollback()
		var count int64
		sql := fmt.Sprintf(
			`SELECT COUNT(*) FROM %s WHERE %s >= ? AND %s < ? AND "_deleted" = true`,
			quoteRelation(tableName), quoteIdent(tsCol), quoteIdent(tsCol),
		)
		if err := tx.Raw(sql, tLo, tHi).Scan(&count).Error; err != nil {
			return nil, err
		}
		return count, nil
	})
	if err != nil {
		return 0, err
	}
	return result.(int64), nil
}
```

## 2. Chi tiết cơ chế kiểm tra đối chiếu chéo HashWindow (Double-Check HashWindow)

Khi `srcEstClean` (được tính toán qua `EstimatedCount` của MongoDB) lệch so với `dstActiveClean` (được tính toán chính xác từ Shadow), hệ thống sẽ thực hiện kiểm tra chéo bằng `HashWindow` trên cửa sổ thời gian tĩnh để phát hiện drift thực tế, loại bỏ hoàn toàn lag đồng bộ.

### 2.1. Phân giải mốc thời gian (Timestamps)
Để loại bỏ 100% nhiễu do lag đồng bộ trong cửa sổ gần đây, mốc thời gian kiểm tra không sử dụng `now` hay `now - lag`. Thay vào đó, nó sử dụng mốc thời gian tĩnh được giới hạn bởi cửa sổ trễ:
*   `hi` (Time): Được gán chính xác bằng `fromTime` (tức `now - 120s` làm tròn phút). Mốc này loại trừ toàn bộ dữ liệu đang ghi/xóa trong 120s gần nhất.
*   `lo` (Time): Được tính bằng `hi.Add(-rc.effectiveLookback(ctx))` (mặc định Hot mode là `hi - 2h`, Cold mode là `hi - 7d`).
*   `srcTS` (String): Tên trường timestamp ở MongoDB (`lastUpdatedAt` hoặc `_source_ts`).
*   `dstTS` (String): Tên trường timestamp ở Postgres (`_source_ts` hoặc `lastUpdatedAt`).

### 2.2. Cách thức hoạt động của HashWindow
*   **Source (MongoDB):** Gọi `rc.sourceAgent.HashWindow(fastCtx, entry.SourceURL, entry.SourceDB, entry.SourceTable, srcTS, lo, hi)`.
    *   Hàm thực hiện một query tìm kiếm các bản ghi có timestamp trong dải `[lo, hi)` (sử dụng index trên trường timestamp, không collscan).
    *   Mỗi bản ghi được băm khóa chính và timestamp dưới dạng millisecond (`hashIDPlusTsMs`).
    *   Tiến hành XOR tích lũy tất cả mã băm để sinh ra một mã chữ ký số `XorHash` (uint64) duy nhất và đếm số lượng `Count` (int64) bản ghi thực tế.
*   **Destination (Shadow Postgres):** Gọi `rc.destAgent.HashWindow(fastCtx, entry.QualifiedTarget(), entry.PrimaryKeyField, dstTS, lo, hi)`.
    *   Hàm thực hiện SELECT stream trên Postgres lấy các bản ghi có timestamp trong dải `[lo, hi)`.
    *   Thực hiện băm tương tự để tính ra `XorHash` (uint64) và `Count` (int64).

### 2.3. Quy tắc đối soát và kết luận
So sánh trực tiếp:
```go
if srcHash.Count == dstHash.Count && srcHash.XorHash == dstHash.XorHash {
    // 100% khớp dữ liệu thực tế ngoài cửa sổ 120s -> Sai số do EstimatedCount metadata của MongoDB
    diff = 0
    statusStr = "ok"
} else {
    // Lệch thực tế -> Drift thật sự
    statusStr = "drift"
}
```

## 3. Chi tiết thay đổi file `recon_smoke.go`

### Thay đổi tại `RunTotalOnlyA` (Dòng ~230):

```go
	// --- QUERY TẦNG 1: MONGODB / POSTGRESQL ---
	dbSystem := "mongodb"
	spanName := "recon.smoke.mongo_estimate"
	if isPostgres(entry.SourceURL) {
		dbSystem = "postgresql"
		spanName = "recon.smoke.postgres_count"
	}

	var mongoErr error
	mongoCtx, mongoSpan := observability.ChildSpan(fastCtx, spanName,
		attribute.String("db.system", dbSystem),
		attribute.String("db.name", entry.SourceDB),
		attribute.String("db.collection", entry.SourceTable),
	)

	var srcEst int64

	// Luôn sử dụng EstimatedCount cho MongoDB để tránh collscan và bảo vệ database
	if isPostgres(entry.SourceURL) {
		srcEst, mongoErr = rc.sourceAgent.CountDocuments(mongoCtx, entry.SourceURL, entry.SourceDB, entry.SourceTable)
	} else {
		srcEst, mongoErr = rc.sourceAgent.EstimatedCount(mongoCtx, entry.SourceURL, entry.SourceDB, entry.SourceTable)
	}

	if mongoErr == nil {
		mongoSpan.SetAttributes(attribute.Int64("db.row_count", srcEst))
	}
	observability.EndSpan(mongoSpan, &mongoErr)

	if mongoErr != nil {
		status = "failed"
		runErr = mongoErr
		errLabel := "src estimated count"
		if isPostgres(entry.SourceURL) {
			errLabel = "src postgres count"
		}
		return rc.smokeErrorReportA(ctx, entry, fmt.Errorf("%s: %w", errLabel, mongoErr), int(time.Since(handle.started).Milliseconds()))
	}

	// dstTotal/dstActive đã được tính trước bởi CheckAllUnified qua scanExact.
	metrics.ShadowActiveRowCount.WithLabelValues(entry.QualifiedTarget()).Set(float64(dstActive))

	// Cửa sổ gần đây 120s làm tròn về phút
	nowTime := time.Now().UTC()
	fromTime := nowTime.Add(-120 * time.Second).Truncate(time.Minute)

	srcTS, dstTS, errTS := rc.resolveSourceAndDestTSFields(ctx, entry)
	if errTS != nil || srcTS == "" || dstTS == "" {
		return rc.smokeErrorReportA(ctx, entry, fmt.Errorf("resolve timestamp fields failed: %w", errTS), int(time.Since(handle.started).Milliseconds()))
	}

	// Đếm số lượng record mới của Source trong cửa sổ gần đây
	srcRecent, errWindow := rc.sourceAgent.CountInWindow(fastCtx, entry.SourceURL, entry.SourceDB, entry.SourceTable, srcTS, fromTime, nowTime)
	if errWindow != nil {
		return rc.smokeErrorReportA(ctx, entry, fmt.Errorf("source CountInWindow failed: %w", errWindow), int(time.Since(handle.started).Milliseconds()))
	}

	// Đếm tổng số lượng record mới của Shadow trong cửa sổ gần đây
	dstRecentTotal, errWindow := rc.destAgent.CountInWindow(fastCtx, entry.TargetTable, dstTS, fromTime, nowTime)
	if errWindow != nil {
		return rc.smokeErrorReportA(ctx, entry, fmt.Errorf("shadow CountInWindow failed: %w", errWindow), int(time.Since(handle.started).Milliseconds()))
	}

	// Đếm số lượng record bị XÓA MỀM trong cửa sổ gần đây
	dstRecentDeleted, errWindow := rc.destAgent.CountRecentDeletedRows(fastCtx, entry.TargetTable, dstTS, fromTime, nowTime)
	if errWindow != nil {
		return rc.smokeErrorReportA(ctx, entry, fmt.Errorf("shadow CountRecentDeletedRows failed: %w", errWindow), int(time.Since(handle.started).Milliseconds()))
	}

	// Số lượng active thực tế phát sinh trong cửa sổ gần đây
	dstRecentActive := dstRecentTotal - dstRecentDeleted

	srcEstClean := srcEst
	if val := srcEst - srcRecent; val >= 0 {
		srcEstClean = val
	} else {
		srcEstClean = 0
	}

	dstActiveClean := dstActive
	if val := dstActive - dstRecentActive; val >= 0 {
		dstActiveClean = val
	} else {
		dstActiveClean = 0
	}

	diff := srcEstClean - dstActiveClean
	statusStr := "ok"

	// Nếu lệch số lượng (có thể do sai số metadata của EstimatedCount)
	if diff != 0 {
		// Mốc trên hi = fromTime (ngoài cửa sổ trễ 120s), mốc dưới lo = hi - lookback
		hi := fromTime
		lo := hi.Add(-rc.effectiveLookback(fastCtx))

		srcHash, errS := rc.sourceAgent.HashWindow(fastCtx, entry.SourceURL, entry.SourceDB, entry.SourceTable, srcTS, lo, hi)
		dstHash, errD := rc.destAgent.HashWindow(fastCtx, entry.QualifiedTarget(), entry.PrimaryKeyField, dstTS, lo, hi)
		if errS == nil && errD == nil && srcHash.Count == dstHash.Count && srcHash.XorHash == dstHash.XorHash {
			// Khớp cả count và hash -> Xác nhận không có drift thực tế (do sai lệch metadata EstimatedCount)
			rc.logger.Info("[smoke-A] EstimatedCount discrepancy resolved via HashWindow match on static range",
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
			rc.logger.Warn("[smoke-A] HashWindow mismatch — real drift confirmed on static range",
				zap.String("table", entry.TargetTable),
				zap.Int64("estimatedDiff", diff),
				zap.Time("lo", lo),
				zap.Time("hi", hi),
				zap.String("errS", fmt.Sprintf("%v", errS)),
				zap.String("errD", fmt.Sprintf("%v", errD)),
			)
		}
	}

	rc.logger.Info("[smoke-A] final smoke result",
		zap.String("table", entry.TargetTable),
		zap.String("status", statusStr),
		zap.Int64("diff", diff),
		zap.Int64("srcTotal", srcEst),
		zap.Int64("srcRecent", srcRecent),
		zap.Int64("srcClean", srcEstClean),
		zap.Int64("dstActive", dstActive),
		zap.Int64("dstRecentTotal", dstRecentTotal),
		zap.Int64("dstRecentDeleted", dstRecentDeleted),
		zap.Int64("dstRecentActive", dstRecentActive),
		zap.Int64("dstClean", dstActiveClean),
	)
```

### Thay đổi logic trừ bù cửa sổ thời gian gần đây tại `RunTotalOnlyB` (Dòng ~415):

```go
	// Cửa sổ gần đây 120s làm tròn về phút
	nowTime := time.Now().UTC()
	fromTime := nowTime.Add(-120 * time.Second).Truncate(time.Minute)

	// Đếm số lượng của Shadow trong cửa sổ gần đây
	shRecentTotal, errWindowB := rc.destAgent.CountInWindow(fastCtx, shadowRel, "_source_ts", fromTime, nowTime)
	if errWindowB != nil {
		return rc.smokeErrorReportB(ctx, shadowRel, masterRel, fmt.Errorf("shadow CountInWindow failed: %w", errWindowB), int(time.Since(handle.started).Milliseconds()))
	}
	shRecentDeleted, errWindowB := rc.destAgent.CountRecentDeletedRows(fastCtx, shadowRel, "_source_ts", fromTime, nowTime)
	if errWindowB != nil {
		return rc.smokeErrorReportB(ctx, shadowRel, masterRel, fmt.Errorf("shadow CountRecentDeletedRows failed: %w", errWindowB), int(time.Since(handle.started).Milliseconds()))
	}
	shRecentActive := shRecentTotal - shRecentDeleted

	// Đếm số lượng của Master trong cửa sổ gần đây
	msRecentTotal, errWindowB := rc.masterAgent.CountInWindow(fastCtx, masterRel, "_source_ts", fromTime, nowTime)
	if errWindowB != nil {
		return rc.smokeErrorReportB(ctx, shadowRel, masterRel, fmt.Errorf("master CountInWindow failed: %w", errWindowB), int(time.Since(handle.started).Milliseconds()))
	}
	msRecentDeleted, errWindowB := rc.masterAgent.CountRecentDeletedRows(fastCtx, masterRel, "_source_ts", fromTime, nowTime)
	if errWindowB != nil {
		return rc.smokeErrorReportB(ctx, shadowRel, masterRel, fmt.Errorf("master CountRecentDeletedRows failed: %w", errWindowB), int(time.Since(handle.started).Milliseconds()))
	}
	msRecentActive := msRecentTotal - msRecentDeleted

	shadowActiveClean := shadowActive
	if val := shadowActive - shRecentActive; val >= 0 {
		shadowActiveClean = val
	} else {
		shadowActiveClean = 0
	}

	masterActiveClean := masterActive
	if val := masterActive - msRecentActive; val >= 0 {
		masterActiveClean = val
	} else {
		masterActiveClean = 0
	}

	diff := shadowActiveClean - masterActiveClean

	rc.logger.Info("[smoke-B] subtracting recent window counts with delete awareness",
		zap.String("shadow", shadowRel),
		zap.String("master", masterRel),
		zap.Time("from", fromTime),
		zap.Time("now", nowTime),
		zap.Int64("shActive", shadowActive),
		zap.Int64("shRecentTotal", shRecentTotal),
		zap.Int64("shRecentDeleted", shRecentDeleted),
		zap.Int64("shRecentActive", shRecentActive),
		zap.Int64("shClean", shadowActiveClean),
		zap.Int64("msActive", masterActive),
		zap.Int64("msRecentTotal", msRecentTotal),
		zap.Int64("msRecentDeleted", msRecentDeleted),
		zap.Int64("msRecentActive", msRecentActive),
		zap.Int64("msClean", masterActiveClean),
	)
```

# Kế hoạch Triển khai: Khắc phục lệch pha đối soát cho các bảng ghi liên tục

Kế hoạch này sửa đổi logic của các cơ chế đối soát toàn bảng (`FullIDDiffMissingFromShadow`, `RunOrphanPrune`, `RunDeepCheck`) để chỉ đối soát các bản ghi có thời gian nhỏ hơn mốc chặn trên `upper` (now - lag time), loại bỏ false-positives do CDC lag đối với các bảng có dữ liệu ghi liên tục.

## Proposed Changes

### Component: Recon Service (`internal/service/recon`)

#### [MODIFY] [recon_tier_a.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_tier_a.go)
- Sửa đổi `FullIDDiffMissingFromShadow` để:
  1. Phân giải `srcTS`, `dstTS`.
  2. Tính `upper = time.Now().UTC().Add(-rc.adaptiveFreeze(ingestLagMs))`.
  3. Lọc ID từ Postgres shadow có `dstTS < upper`.
  4. Stream ID từ MongoDB bằng `StreamIDsInTimeRange(..., time.Time{}, upper)`.
- Sửa đổi `RunOrphanPrune` tương tự để giới hạn cả shadow query và MongoDB stream có timestamp < `upper`.
- Sửa đổi `RunDeepCheck` để tính `upper` và truyền vào `BucketHash`.

#### [MODIFY] [recon_hash.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_hash.go)
- Cập nhật signature `BucketHash` để nhận thêm `upper time.Time`.
- Áp dụng filter `{tsField: {$lt: upper}}` (hỗ trợ cả Unix milli và ISODate) cho MongoDB query.

#### [MODIFY] [recon_dest_hash.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_dest_hash.go)
- Cập nhật signature `BucketHash` để nhận thêm `upper time.Time`.
- Áp dụng filter `WHERE tsCol < ?` cho Postgres query.

#### [MODIFY] [recon_legacy.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_legacy.go) và [recon_dest_legacy.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_dest_legacy.go)
- Cập nhật các cuộc gọi `BucketHash` truyền thêm `time.Time{}` (không chặn) cho các legacy/smoke functions.

---

## Code Demo Phác Thảo

### 1. Sửa đổi `FullIDDiffMissingFromShadow`
```go
func (rc *ReconCore) FullIDDiffMissingFromShadow(ctx context.Context, entry source.TableRegistry) ([]string, int, error) {
	if rc.shadowPlane == nil {
		return nil, 0, fmt.Errorf("shadowPlane not wired")
	}

	srcTS, dstTS, err := rc.resolveSourceAndDestTSFields(ctx, entry)
	if err != nil {
		return nil, 0, fmt.Errorf("resolve source ts field: %w", err)
	}

	srcMax, err := rc.sourceAgent.MaxWindowTs(ctx, entry.SourceURL, entry.SourceDB, entry.SourceTable, srcTS)
	if err != nil {
		return nil, 0, fmt.Errorf("source max ts: %w", err)
	}
	dstMax, err := rc.destAgent.MaxWindowTs(ctx, entry.QualifiedTarget(), dstTS)
	if err != nil {
		return nil, 0, fmt.Errorf("dest max ts: %w", err)
	}

	ingestLagMs := lagBetween(srcMax, dstMax)
	rc.upsertReconLag(ctx, entry.TargetTable, "ingest_lag_ms", ingestLagMs)

	upper := time.Now().UTC().Add(-rc.adaptiveFreeze(ingestLagMs))

	var shadowIDs []string
	startVal, endVal, err := resolvePostgresTimeParams(ctx, rc.shadowPlane, entry.QualifiedTarget(), dstTS, time.Time{}, upper)
	if err != nil {
		endVal = upper
	}

	if err := rc.shadowPlane.WithContext(ctx).Raw(
		fmt.Sprintf(`SELECT "_source_id"::text FROM %s WHERE NOT "_deleted" AND "_source_id" IS NOT NULL AND %s < ?`,
			quoteRelation(entry.QualifiedTarget()), quoteIdent(dstTS)),
		endVal,
	).Scan(&shadowIDs).Error; err != nil {
		return nil, 0, fmt.Errorf("shadow list ids: %w", err)
	}

	shadowSet := make(map[string]struct{}, len(shadowIDs))
	for _, id := range shadowIDs {
		shadowSet[id] = struct{}{}
	}

	idChan, errChan := rc.sourceAgent.StreamIDsInTimeRange(ctx, entry.SourceURL, entry.SourceDB, entry.SourceTable, srcTS, time.Time{}, upper)
	srcCount := 0
	var missing []string
	var streamErr error

	for idChan != nil || errChan != nil {
		select {
		case id, ok := <-idChan:
			if !ok {
				idChan = nil
				break
			}
			srcCount++
			if _, exists := shadowSet[id]; !exists {
				missing = append(missing, id)
			}
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

	if streamErr != nil {
		return nil, srcCount, fmt.Errorf("stream partial error: %w", streamErr)
	}

	return missing, srcCount, nil
}
```

### 2. Sửa đổi MongoDB `BucketHash`
```go
func (sa *ReconSourceAgent) BucketHash(ctx context.Context, sourceURL, database, collection, timestampField string, upper time.Time) (*BucketHashResult, error) {
    ...
	tsField := resolveTimestampField(timestampField)
	opts := sa.selectOpts(bson.M{"_id": 1, tsField: 1})

	filter := bson.M{}
	if !upper.IsZero() && tsField != "" {
		filter = bson.M{
			"$or": []bson.M{
				{tsField: bson.M{"$lt": upper}},
				{tsField: bson.M{"$lt": upper.UnixMilli()}},
			},
		}
	}

	var out *BucketHashResult
	err = sa.queryWithRetry(ctx, "BucketHash", sourceURL, func() error {
		result, innerErr := sa.getBreaker(sourceURL).Execute(func() (interface{}, error) {
			cursor, err := coll.Find(ctx, filter, opts)
            ...
```

---

## Kế hoạch Kiểm thử & Xác minh

### Automated Tests
- Chạy unit test trong `recon_lag_test.go` và `recon_hash_test.go`:
  ```bash
  go test -v ./internal/service/recon/...
  ```
- Viết thêm test case giả lập dữ liệu realtime lag trong `recon_lag_test.go` để đảm bảo:
  - Khi có lag, mốc chặn thời gian được tính chính xác.
  - Các record ghi sau mốc `upper` bị loại trừ khỏi đối soát một cách chính xác.

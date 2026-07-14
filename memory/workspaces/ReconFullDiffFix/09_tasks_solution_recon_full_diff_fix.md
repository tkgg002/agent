# Solution: Sửa lỗi đối soát Full Search (Full Diff) không có kết quả

Tài liệu này đặc tả chi tiết mã nguồn sẽ thay đổi để tự động phân giải kiểu dữ liệu timestamp của Shadow DB và Postgres Source DB sang kiểu số/thời gian phù hợp.

## 1. Helper phân giải kiểu dữ liệu timestamp (`resolvePostgresTimeParams`)

Chúng ta sẽ khai báo một hàm helper package-private trong `internal/service/recon/recon_stream.go` (hoặc `recon_query.go` tùy ý) để tìm kiểu dữ liệu cột trong Postgres và cast `time.Time` sang epoch/time thích hợp.

```go
func resolvePostgresTimeParams(ctx context.Context, db *gorm.DB, tableName, columnName string, tLo, tHi time.Time) (interface{}, interface{}, error) {
	var schemaName, bareTable string
	if i := strings.IndexByte(tableName, '.'); i > 0 {
		schemaName = tableName[:i]
		bareTable = tableName[i+1:]
	} else {
		schemaName = "public"
		bareTable = tableName
	}

	var dataType string
	err := db.WithContext(ctx).Raw(`
		SELECT data_type FROM information_schema.columns 
		WHERE table_schema = ? AND table_name = ? AND column_name = ?
	`, schemaName, bareTable, columnName).Scan(&dataType).Error
	if err != nil {
		return tLo, tHi, err
	}
	dataType = strings.ToLower(dataType)

	if strings.Contains(dataType, "int") || strings.Contains(dataType, "num") || columnName == "_source_ts" {
		isEpochMillis := true
		var maxVal int64
		sqlMax := fmt.Sprintf(`SELECT COALESCE(MAX(%s), 0) FROM %s`, quoteIdent(columnName), quoteRelation(tableName))
		if err := db.WithContext(ctx).Raw(sqlMax).Scan(&maxVal).Error; err == nil {
			if maxVal > 0 && maxVal < 1e11 {
				isEpochMillis = false
			}
		}
		if isEpochMillis {
			return tLo.UnixMilli(), tHi.UnixMilli(), nil
		}
		return tLo.Unix(), tHi.Unix(), nil
	}

	return tLo, tHi, nil
}
```

## 2. Thay đổi trong `internal/service/recon/recon_tier_a.go`

Cập nhật hàm `TimeBoundedDiffMissingFromShadow`:

```go
<<<<
	// Tải ID từ Postgres Shadow DB
	var shadowIDs []string
	if err := rc.shadowPlane.WithContext(ctxPg).Raw(
		fmt.Sprintf(`SELECT "_source_id"::text FROM %s WHERE NOT "_deleted" AND "_source_id" IS NOT NULL AND %s >= ? AND %s < ?`,
			quoteRelation(entry.QualifiedTarget()), quoteIdent(dstTS), quoteIdent(dstTS)),
		startTime, endTime,
	).Scan(&shadowIDs).Error; err != nil {
		finalErr = fmt.Errorf("list shadow ids: %w", err)
		return
	}
====
	// Tải ID từ Postgres Shadow DB
	var shadowIDs []string
	startVal, endVal, err := resolvePostgresTimeParams(ctxPg, rc.shadowPlane, entry.QualifiedTarget(), dstTS, startTime, endTime)
	if err != nil {
		observability.Ctx(ctx, rc.logger).Warn("[tier2] failed to resolve postgres time params for shadow", zap.Error(err))
		startVal = startTime
		endVal = endTime
	}

	if err := rc.shadowPlane.WithContext(ctxPg).Raw(
		fmt.Sprintf(`SELECT "_source_id"::text FROM %s WHERE NOT "_deleted" AND "_source_id" IS NOT NULL AND %s >= ? AND %s < ?`,
			quoteRelation(entry.QualifiedTarget()), quoteIdent(dstTS), quoteIdent(dstTS)),
		startVal, endVal,
	).Scan(&shadowIDs).Error; err != nil {
		finalErr = fmt.Errorf("list shadow ids: %w", err)
		return
	}
>>>>
```

## 3. Thay đổi trong `internal/service/recon/recon_stream.go`

### 3.1. Cập nhật `listIDsInWindowPostgres`:

```go
<<<<
	result, err := sa.getBreaker(sourceURL).Execute(func() (interface{}, error) {
		var ids []string
		sql := fmt.Sprintf(`SELECT %s::text FROM %s WHERE %s >= ? AND %s < ?`, 
			quoteIdent(pkCol), quoteRelation(tableName), quoteIdent(tsField), quoteIdent(tsField))
		
		rows, err := db.WithContext(ctx).Raw(sql, tLo, tHi).Rows()
====
	result, err := sa.getBreaker(sourceURL).Execute(func() (interface{}, error) {
		var ids []string
		tLoVal, tHiVal, err := resolvePostgresTimeParams(ctx, db, tableName, tsField, tLo, tHi)
		if err != nil {
			tLoVal = tLo
			tHiVal = tHi
		}

		sql := fmt.Sprintf(`SELECT %s::text FROM %s WHERE %s >= ? AND %s < ?`, 
			quoteIdent(pkCol), quoteRelation(tableName), quoteIdent(tsField), quoteIdent(tsField))
		
		rows, err := db.WithContext(ctx).Raw(sql, tLoVal, tHiVal).Rows()
>>>>
```

### 3.2. Cập nhật `streamIDsPostgresInTimeRange`:

```go
<<<<
			var querySql string
			var args []interface{}
			if lastID != nil {
				querySql = fmt.Sprintf(`SELECT %s::text FROM %s WHERE %s > ? AND %s >= ? AND %s < ? ORDER BY %s LIMIT ?`,
					quoteIdent(pkCol), quoteRelation(tableName), quoteIdent(pkCol), quoteIdent(timestampField), quoteIdent(timestampField), quoteIdent(pkCol))
				args = []interface{}{lastID, startTime, endTime, batchSize}
			} else {
				querySql = fmt.Sprintf(`SELECT %s::text FROM %s WHERE %s >= ? AND %s < ? ORDER BY %s LIMIT ?`,
					quoteIdent(pkCol), quoteRelation(tableName), quoteIdent(timestampField), quoteIdent(timestampField), quoteIdent(pkCol))
				args = []interface{}{startTime, endTime, batchSize}
			}
====
			var querySql string
			var args []interface{}
			startTimeVal, endTimeVal, err := resolvePostgresTimeParams(ctx, db, tableName, timestampField, startTime, endTime)
			if err != nil {
				startTimeVal = startTime
				endTimeVal = endTime
			}

			if lastID != nil {
				querySql = fmt.Sprintf(`SELECT %s::text FROM %s WHERE %s > ? AND %s >= ? AND %s < ? ORDER BY %s LIMIT ?`,
					quoteIdent(pkCol), quoteRelation(tableName), quoteIdent(pkCol), quoteIdent(timestampField), quoteIdent(timestampField), quoteIdent(pkCol))
				args = []interface{}{lastID, startTimeVal, endTimeVal, batchSize}
			} else {
				querySql = fmt.Sprintf(`SELECT %s::text FROM %s WHERE %s >= ? AND %s < ? ORDER BY %s LIMIT ?`,
					quoteIdent(pkCol), quoteRelation(tableName), quoteIdent(timestampField), quoteIdent(timestampField), quoteIdent(pkCol))
				args = []interface{}{startTimeVal, endTimeVal, batchSize}
			}
>>>>
```

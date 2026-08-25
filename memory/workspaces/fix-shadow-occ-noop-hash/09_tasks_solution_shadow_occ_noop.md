# Technical Solution: Fix Shadow Table OCC No-Op Hash Gate

## 1. File Thay Đổi
- `centralized-data-service/internal/service/shadow/schema_adapter.go`
- `centralized-data-service/test/internal/service/schema_adapter_ordering_test.go`

## 2. Chi Tiết Thay Đổi Code (Diff Demo)

### A. Tại `centralized-data-service/internal/service/shadow/schema_adapter.go`

```go
func buildOCCWhereClause(schema *TableSchema, qualifiedTable string, hasPositiveTs bool) string {
	_, hasSourceTs := schema.Columns["_source_ts"]
	_, hasHash := schema.Columns["_hash"]
	_, hasDeleted := schema.Columns["_deleted"]

	// 1. Cổng kiểm tra thay đổi nội dung dữ liệu (Hash / Content Change Gate)
	var changeConditions []string
	if hasHash {
		changeConditions = append(changeConditions, fmt.Sprintf(`%s."_hash" IS DISTINCT FROM EXCLUDED."_hash"`, qualifiedTable))
	}
	if hasDeleted {
		changeConditions = append(changeConditions, fmt.Sprintf(`%s."_deleted" IS DISTINCT FROM EXCLUDED."_deleted"`, qualifiedTable))
	}

	var changeClause string
	if len(changeConditions) > 0 {
		changeClause = fmt.Sprintf(`(%s)`, strings.Join(changeConditions, " OR "))
	} else if _, hasRaw := schema.Columns["_raw_data"]; hasRaw {
		changeClause = fmt.Sprintf(`(%s."_raw_data" IS DISTINCT FROM EXCLUDED."_raw_data")`, qualifiedTable)
	}

	// 2. Cổng kiểm tra thứ tự thời gian OCC (OCC Time Guard)
	if hasSourceTs && hasPositiveTs {
		timeClause := fmt.Sprintf(
			`(%s."_source_ts" IS NULL `+
				`OR %s."_source_ts" < EXCLUDED."_source_ts" `+
				`OR (%s."_source_ts" = EXCLUDED."_source_ts" `+
				`    AND %s."_source" = 'snapshot:v2' `+
				`    AND EXCLUDED."_source" <> 'snapshot:v2'))`,
			qualifiedTable, qualifiedTable, qualifiedTable, qualifiedTable,
		)

		if changeClause != "" {
			return fmt.Sprintf(`WHERE %s AND %s`, changeClause, timeClause)
		}
		return fmt.Sprintf(`WHERE %s`, timeClause)
	}

	if changeClause != "" {
		return fmt.Sprintf(`WHERE %s`, changeClause)
	}
	return ""
}
```

### B. Thêm Test Case tại `centralized-data-service/test/internal/service/schema_adapter_ordering_test.go`

```go
func TestEventOrdering_SameDataSnapshot_NoOp(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	logger := zap.NewNop()
	adapter := shadow.NewSchemaAdapter(db, logger)
	schema := deleteAwareSchema()

	sourceID := "noop_user_1"

	// Step 1: Lần đầu Insert snapshot 1
	rows1 := applyUpsert(t, adapter, schema, db, "test_users", "_source_id", sourceID,
		map[string]any{"name": "alice", "_deleted": false}, "hash_identical", 1000)
	require.EqualValues(t, 1, rows1, "Lần đầu insert phải thành công")

	// Step 2: Re-snapshot với cùng dữ liệu, timestamp mới hơn (5000 > 1000)
	rows2 := applyUpsert(t, adapter, schema, db, "test_users", "_source_id", sourceID,
		map[string]any{"name": "alice", "_deleted": false}, "hash_identical", 5000)
	require.EqualValues(t, 0, rows2, "Cùng dữ liệu (hash giống hệt) phải NO-OP (0 rows affected)")

	// Step 3: Re-snapshot nhưng có UPDATE dữ liệu (name thay đổi -> hash mới)
	rows3 := applyUpsert(t, adapter, schema, db, "test_users", "_source_id", sourceID,
		map[string]any{"name": "alice_updated", "_deleted": false}, "hash_new", 6000)
	require.EqualValues(t, 1, rows3, "Dữ liệu có thay đổi phải UPDATE thành công")
	require.Equal(t, "alice_updated", readShadowValue(t, db, sourceID, "name"))
}
```

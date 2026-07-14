# Hồ sơ Giải pháp Kỹ thuật - Tối ưu hiệu năng Transmute và sửa lỗi thiếu/hỏng Index trên Shadow Tables

## 1. File `internal/service/master/transmuter.go`
Sửa hàm `ensureShadowSourceIDIndex` để kiểm tra độ hợp lệ (`indisvalid = true`) và xử lý drop index lỗi trước khi tạo.

### Nội dung cần thay đổi:
```diff
 func (t *TransmuterModule) ensureShadowSourceIDIndex(ctx context.Context, shadowDB *gorm.DB, row *masterBindingRuntime) {
 	if shadowDB.Dialector.Name() != "postgres" {
 		return
 	}
 
 	indexName := fmt.Sprintf("idx_%s_source_id", row.ShadowTable)
-	var count int64
-	errIdx := shadowDB.WithContext(ctx).Raw(`
-		SELECT COUNT(*) 
-		FROM pg_indexes 
-		WHERE schemaname = ? AND tablename = ? AND indexname = ?`,
-		row.ShadowSchema, row.ShadowTable, indexName).Scan(&count).Error
-	if errIdx == nil && count == 0 {
-		// Tạo index CONCURRENTLY bất đồng bộ dưới nền để không block transmuter
-		go func() {
-			bgCtx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
-			defer cancel()
-			sqlText := fmt.Sprintf(`CREATE INDEX CONCURRENTLY IF NOT EXISTS %s ON %s (_source_id)`,
-				quoteTransmuteIdent(indexName),
-				quoteTransmuteQualified(row.ShadowSchema, row.ShadowTable))
-			t.logger.Info("transmuter: creating missing non-partial index on _source_id concurrently",
-				zap.String("schema", row.ShadowSchema),
-				zap.String("table", row.ShadowTable),
-				zap.String("index", indexName))
-			if errCreate := shadowDB.WithContext(bgCtx).Exec(sqlText).Error; errCreate != nil {
-				t.logger.Error("transmuter: failed to create concurrent index on _source_id",
-					zap.String("table", row.ShadowTable),
-					zap.Error(errCreate))
-			} else {
-				t.logger.Info("transmuter: successfully created concurrent index on _source_id",
-					zap.String("table", row.ShadowTable))
-			}
-		}()
-	}
+	var validCount int64
+	errValid := shadowDB.WithContext(ctx).Raw(`
+		SELECT COUNT(*) 
+		FROM pg_index i
+		JOIN pg_class c ON c.oid = i.indexrelid
+		JOIN pg_class t ON t.oid = i.indrelid
+		JOIN pg_namespace n ON n.oid = t.relnamespace
+		WHERE n.nspname = ? AND t.relname = ? AND c.relname = ? AND i.indisvalid = true`,
+		row.ShadowSchema, row.ShadowTable, indexName).Scan(&validCount).Error
+
+	if errValid == nil && validCount == 0 {
+		var existCount int64
+		_ = shadowDB.WithContext(ctx).Raw(`
+			SELECT COUNT(*) 
+			FROM pg_index i
+			JOIN pg_class c ON c.oid = i.indexrelid
+			JOIN pg_class t ON t.oid = i.indrelid
+			JOIN pg_namespace n ON n.oid = t.relnamespace
+			WHERE n.nspname = ? AND t.relname = ? AND c.relname = ?`,
+			row.ShadowSchema, row.ShadowTable, indexName).Scan(&existCount).Error
+
+		// Tạo index CONCURRENTLY bất đồng bộ dưới nền để không block transmuter
+		go func() {
+			bgCtx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
+			defer cancel()
+
+			if existCount > 0 {
+				t.logger.Warn("transmuter: invalid index found, dropping it first",
+					zap.String("schema", row.ShadowSchema),
+					zap.String("table", row.ShadowTable),
+					zap.String("index", indexName))
+				
+				dropSql := fmt.Sprintf(`DROP INDEX CONCURRENTLY IF EXISTS %s.%s`,
+					quoteTransmuteIdent(row.ShadowSchema),
+					quoteTransmuteIdent(indexName))
+				if errDrop := shadowDB.WithContext(bgCtx).Exec(dropSql).Error; errDrop != nil {
+					t.logger.Error("transmuter: failed to drop invalid index",
+						zap.String("table", row.ShadowTable),
+						zap.Error(errDrop))
+					return
+				}
+			}
+
+			sqlText := fmt.Sprintf(`CREATE INDEX CONCURRENTLY IF NOT EXISTS %s ON %s (_source_id)`,
+				quoteTransmuteIdent(indexName),
+				quoteTransmuteQualified(row.ShadowSchema, row.ShadowTable))
+			t.logger.Info("transmuter: creating missing non-partial index on _source_id concurrently",
+				zap.String("schema", row.ShadowSchema),
+				zap.String("table", row.ShadowTable),
+				zap.String("index", indexName))
+			if errCreate := shadowDB.WithContext(bgCtx).Exec(sqlText).Error; errCreate != nil {
+				t.logger.Error("transmuter: failed to create concurrent index on _source_id",
+					zap.String("table", row.ShadowTable),
+					zap.Error(errCreate))
+			} else {
+				t.logger.Info("transmuter: successfully created concurrent index on _source_id",
+					zap.String("table", row.ShadowTable))
+			}
+		}()
+	}
 }
```

---

## 2. File `internal/service/shadow/schema_adapter.go`
Thêm index `idx_<tableName>_source_id` tại hàm `EnsureCDCColumnsInSchema`.

### Nội dung cần thay đổi:
```diff
 	uxName := fmt.Sprintf("ux_%s_source_id_active", tableName)
 	if err := sa.db.WithContext(ctx).Exec(fmt.Sprintf(
 		`CREATE UNIQUE INDEX IF NOT EXISTS %s ON %s.%s (_source_id) WHERE NOT _deleted`,
 		sqlutil.QuoteIdent(uxName),
 		sqlutil.QuoteIdent(schemaName), sqlutil.QuoteIdent(tableName),
 	)).Error; err != nil {
 		return err
 	}
+
+	nonUniqueIdxName := fmt.Sprintf("idx_%s_source_id", tableName)
+	if err := sa.db.WithContext(ctx).Exec(fmt.Sprintf(
+		`CREATE INDEX IF NOT EXISTS %s ON %s.%s (_source_id)`,
+		sqlutil.QuoteIdent(nonUniqueIdxName),
+		sqlutil.QuoteIdent(schemaName), sqlutil.QuoteIdent(tableName),
+	)).Error; err != nil {
+		return err
+	}
+
 	return nil
 }
```

---

## 3. File `internal/sinkworker/schema_manager.go`
Thêm index `idx_<tableName>_source_id` tại hàm `createShadowTable`.

### Nội dung cần thay đổi:
```diff
 	idx := fmt.Sprintf(
 		`CREATE UNIQUE INDEX IF NOT EXISTS %s
 		   ON %s.%s (_source_id)`,
 		quoteIdent("ux_"+table+"_source_id_active"),
 		quoteIdent(schemaName),
 		quoteIdent(table),
 	)
 	if err := s.db.WithContext(ctx).Exec(idx).Error; err != nil {
 		return fmt.Errorf("CREATE INDEX: %w", err)
 	}
 
+	nonUniqueIdx := fmt.Sprintf(
+		`CREATE INDEX IF NOT EXISTS %s
+		   ON %s.%s (_source_id)`,
+		quoteIdent("idx_"+table+"_source_id"),
+		quoteIdent(schemaName),
+		quoteIdent(table),
+	)
+	if err := s.db.WithContext(ctx).Exec(nonUniqueIdx).Error; err != nil {
+		return fmt.Errorf("CREATE INDEX idx_%s_source_id: %w", table, err)
+	}
+
 	trigName := "trg_" + table + "_fencing"
```

---

## 4. File `internal/sinkworker_bk/schema_manager.go`
Thêm index `idx_<tableName>_source_id` tại hàm `createShadowTable`.

### Nội dung cần thay đổi:
```diff
 	idx := fmt.Sprintf(
 		`CREATE UNIQUE INDEX IF NOT EXISTS %s
 		   ON %s.%s (_source_id)
 		   WHERE NOT _deleted`,
 		quoteIdent("ux_"+table+"_source_id_active"),
 		quoteIdent(schemaName),
 		quoteIdent(table),
 	)
 	if err := s.db.WithContext(ctx).Exec(idx).Error; err != nil {
 		return fmt.Errorf("CREATE INDEX: %w", err)
 	}
 
+	nonUniqueIdx := fmt.Sprintf(
+		`CREATE INDEX IF NOT EXISTS %s
+		   ON %s.%s (_source_id)`,
+		quoteIdent("idx_"+table+"_source_id"),
+		quoteIdent(schemaName),
+		quoteIdent(table),
+	)
+	if err := s.db.WithContext(ctx).Exec(nonUniqueIdx).Error; err != nil {
+		return fmt.Errorf("CREATE INDEX idx_%s_source_id: %w", table, err)
+	}
+
 	// Attach fencing trigger (T1.3).
```

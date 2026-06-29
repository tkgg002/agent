# Solution: Fallback Default Schema từ Connection Registry

## Giải pháp kỹ thuật cụ thể

### 1. File [metadata_registry_utils.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/source/metadata_registry_utils.go)
Sửa hàm `buildDSNFromFields`:
```diff
 	switch strings.ToLower(strings.TrimSpace(conn.EngineType)) {
 	case "mongodb", "mongo":
 		return fmt.Sprintf("mongodb://%s:%d/", host, port)
 	case "postgres", "postgresql":
 		if db == "" {
 			return ""
 		}
 		sslmode := "disable"
 		if len(conn.OptionsJSON) > 0 {
 			var opts map[string]interface{}
 			if err := json.Unmarshal(conn.OptionsJSON, &opts); err == nil {
 				if v, ok := opts["sslmode"].(string); ok && strings.TrimSpace(v) != "" {
 					sslmode = strings.TrimSpace(v)
 				}
 			}
 		}
-		return fmt.Sprintf("postgres://%s:%d/%s?sslmode=%s", host, port, db, sslmode)
+		searchPath := ""
+		if conn.DefaultSchema != nil && *conn.DefaultSchema != "" {
+			searchPath = fmt.Sprintf("&search_path=%s", *conn.DefaultSchema)
+		}
+		return fmt.Sprintf("postgres://%s:%d/%s?sslmode=%s%s", host, port, db, sslmode, searchPath)
 	}
```

### 2. File [snapshot_runner_handler.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/orchestration/snapshot_runner_handler.go)
Sửa 2 vị trí fallback schema:
```diff
 	var totalRows int64
 	if isMongo {
 		if estCount, err := coll.EstimatedDocumentCount(ctx); err == nil {
 			totalRows = estCount
 			r.db.WithContext(ctx).Exec("UPDATE cdc_system.snapshot_progress SET total_rows = ? WHERE id = ?", estCount, progressID)
 		}
 	} else if isPG {
 		schema := "public"
 		if so.SourceSchema != nil && *so.SourceSchema != "" {
 			schema = *so.SourceSchema
-		}
+		} else if conn.DefaultSchema != nil && *conn.DefaultSchema != "" {
+			schema = *conn.DefaultSchema
+		}
 		var estCount int64
```
và:
```diff
 		} else { // isPG
 			var rows pgx.Rows
 			var err error
 			schema := "public"
 			if so.SourceSchema != nil && *so.SourceSchema != "" {
 				schema = *so.SourceSchema
-			}
+			} else if conn.DefaultSchema != nil && *conn.DefaultSchema != "" {
+				schema = *conn.DefaultSchema
+			}
 			tableName := pgx.Identifier{schema, srcColl}.Sanitize()
```

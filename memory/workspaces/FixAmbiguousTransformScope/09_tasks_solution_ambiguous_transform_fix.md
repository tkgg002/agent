# Technical Solution - Fix Ambiguous Transform Scope (V2 Hybrid)

This document contains the detailed code changes designed to resolve target table name collisions (Ambiguous Scope) in the batch transform flow.

---

## 1. Metadata Registry Service

### Target: `metadata_registry_service.go`
We cache both the flat target table name and the fully qualified table name `schema.table` in `targetCache` and `targetRouteMap` to support unique routing lookup when collisions exist.

```diff
@@ -217,6 +217,12 @@
 		}
 
 		rs.targetCache[cfg.TargetTable] = cfg
 		rs.targetRouteMap[cfg.TargetTable] = route
+		if cfg.ShadowSchema != "" {
+			qualified := cfg.ShadowSchema + "." + cfg.TargetTable
+			rs.targetCache[qualified] = cfg
+			rs.targetRouteMap[qualified] = route
+		}
 		rs.idCache[cfg.ID] = cfg
 		routeBySourceID[src.ID] = append(routeBySourceID[src.ID], route)
```

---

## 2. Helper Functions

### Target: `helpers.go`
Update `ResolveTargetSchema` and `ResolveTargetTableConfig` to parse fully qualified names and fallback to flat names.
`ResolveTargetTableConfig` uses Go interface type assertion to check if the repository supports `GetByTargetTableAndSchema` for collision-safe querying.

```diff
@@ -24,7 +24,10 @@
 // ResolveTargetSchema phân giải và trả về tên schema của bảng đích.
 func ResolveTargetSchema(m MetadataRegistry, targetTable string) string {
+	if idx := strings.Index(targetTable, "."); idx != -1 {
+		return targetTable[:idx]
+	}
 	route := ResolveTargetRoute(m, targetTable)
 	if route != nil && route.ShadowBinding != nil {
 		if v := strings.TrimSpace(route.ShadowBinding.ShadowSchema); v != "" {
 			return v
 		}
@@ -35,6 +38,20 @@
 // ResolveTargetTableConfig lấy cấu hình đích từ tên bảng.
 func ResolveTargetTableConfig(ctx context.Context, m MetadataRegistry, r RegistryResolver, targetTable string) *source.TableRegistry {
+	pureTable := targetTable
+	schemaName := ""
+	if idx := strings.Index(targetTable, "."); idx != -1 {
+		schemaName = targetTable[:idx]
+		pureTable = targetTable[idx+1:]
+	}
 	if m != nil {
 		if item := m.GetTableConfig(targetTable); item != nil {
 			return item
 		}
+		if item := m.GetTableConfig(pureTable); item != nil {
+			return item
+		}
 	}
 	if r == nil {
 		return nil
 	}
+	if schemaName != "" {
+		if resolverWithSchema, ok := r.(interface {
+			GetByTargetTableAndSchema(ctx context.Context, targetTable string, schemaName string) (*source.TableRegistry, error)
+		}); ok {
+			if item, err := resolverWithSchema.GetByTargetTableAndSchema(ctx, pureTable, schemaName); err == nil {
+				return item
+			}
+		}
+	}
 	item, err := r.GetByTargetTable(ctx, targetTable)
 	if err != nil {
-		return nil
+		item, err = r.GetByTargetTable(ctx, pureTable)
+		if err != nil {
+			return nil
+		}
 	}
 	return item
 }
```

---

## 3. Batch Transform Handler (Worker)

### Target: `batch_transform_handler.go`
Update `runTransformJob` to:
- Parse `pureTable` from `targetTable`.
- Check database table existence and column validation using `pureTable` instead of `targetTable` (to prevent double schema qualification).
- Resolve unique mapping rules by matching the target route's `SourceObject.ID` (if route exists) using `ListActiveBySourceObjectAndBinding` instead of matching all objects by source table name flat string.

```diff
@@ -93,6 +93,10 @@
 	targetTable := payload.TargetTable
 	jobID := payload.JobID
 
+	pureTable := targetTable
+	if idx := strings.Index(targetTable, "."); idx != -1 {
+		pureTable = targetTable[idx+1:]
+	}
+
 	spanName := fmt.Sprintf("nats.HandleBatchTransform: %s", targetTable)
 	ctx, span := observability.ChildSpan(ctx, spanName)
 	defer span.End()
@@ -115,19 +119,34 @@
 	schemaName := metadata.ResolveTargetSchema(h.metadataRegistry, targetTable)
-	if !h.TableExistsInSchema(ctx, h.shadowDB, schemaName, targetTable) {
+	if !h.TableExistsInSchema(ctx, h.shadowDB, schemaName, pureTable) {
 		h.publishAndFinishJob(ctx, jobID, "skipped", 0, "table does not exist", nil)
 		return
 	}
-	if !h.HasColumnInSchema(ctx, h.shadowDB, schemaName, targetTable, "_raw_data") {
+	if !h.HasColumnInSchema(ctx, h.shadowDB, schemaName, pureTable, "_raw_data") {
 		h.publishAndFinishJob(ctx, jobID, "skipped", 0, "table has no _raw_data column yet", nil)
 		return
 	}
 
 	reg := metadata.ResolveTargetTableConfig(ctx, h.metadataRegistry, h.registryRepo, targetTable)
-	var sourceTable string
-	if reg != nil {
-		sourceTable = reg.SourceTable
-	} else {
-		sourceTable = targetTable
-	}
-
-	rules, err := h.mappingV2Repo.GetActiveRulesBySourceTable(ctx, sourceTable)
-	if err != nil || len(rules) == 0 {
-		h.publishAndFinishJob(ctx, jobID, "error", 0,
-			fmt.Sprintf("no active mapping rules for table %s (source: %s)", targetTable, sourceTable), rules)
-		return
-	}
+	var sourceObjectID int64
+	var shadowBindingID int64
+	var sourceTable string
+	route := h.metadataRegistry.ResolveTargetRoute(targetTable)
+	if route != nil {
+		if route.SourceObject != nil {
+			sourceObjectID = int64(route.SourceObject.ID)
+		}
+		if route.ShadowBinding != nil {
+			shadowBindingID = int64(route.ShadowBinding.ID)
+		}
+		if route.TableConfig != nil {
+			sourceTable = route.TableConfig.SourceTable
+		}
+	}
+	if sourceTable == "" {
+		sourceTable = pureTable
+	}
+
+	var rules []mastermodel.MappingRuleV2
+	var rulesErr error
+	if sourceObjectID > 0 && shadowBindingID > 0 {
+		rules, rulesErr = h.mappingV2Repo.ListActiveBySourceObjectAndBinding(ctx, sourceObjectID, shadowBindingID)
+	} else if sourceObjectID > 0 {
+		rules, rulesErr = h.mappingV2Repo.ListActiveBySourceObject(ctx, sourceObjectID)
+	} else {
+		rules, rulesErr = h.mappingV2Repo.GetActiveRulesBySourceTable(ctx, sourceTable)
+	}
+	if rulesErr != nil || len(rules) == 0 {
+		h.publishAndFinishJob(ctx, jobID, "error", 0,
+			fmt.Sprintf("no active mapping rules for table %s (source: %s)", targetTable, sourceTable), rules)
+		return
+	}
 
 	execDB := h.DB
@@ -161,3 +180,3 @@
 		seenCols[colKey] = struct{}{}
-		if !h.HasColumnInSchema(ctx, execDB, schemaName, targetTable, rule.TargetColumn) {
+		if !h.HasColumnInSchema(ctx, execDB, schemaName, pureTable, rule.TargetColumn) {
 			observability.Ctx(ctx, h.Logger).Warn("batch transform: target_column does not exist in db, skipping rule",
@@ -181,3 +200,3 @@
 	setClauses = append(setClauses, "_updated_at = NOW()")
-	quotedTable := sqlutil.QualifiedTable(schemaName, targetTable)
+	quotedTable := sqlutil.QualifiedTable(schemaName, pureTable)
 	whereExpr := strings.Join(whereClauses, " OR ")
 	setExpr := strings.Join(setClauses, ", ")
 
-	pkCol, pkErr := h.detectPrimaryKey(execDB, schemaName, targetTable)
+	pkCol, pkErr := h.detectPrimaryKey(execDB, schemaName, pureTable)
```

---

## 4. Table Registry Repository

### Target: `table_registry_repo.go`
Enrich GORM query results in `GetAllActive`, `GetByID`, and `GetByTargetTable`.
Implement `GetByTargetTableAndSchema` for collision-safe querying by both table and schema.

```diff
@@ -18,7 +18,47 @@
 func (r *TableRegistryRepo) GetAllActive(ctx context.Context) ([]source.TableRegistry, error) {
 	var entries []source.TableRegistry
 	err := r.db.WithContext(ctx).Where("is_active = ?", true).Find(&entries).Error
-	return entries, err
+	if err != nil {
+		return nil, err
+	}
+
+	type SchemaMap struct {
+		LegacyRegistryID uint   `gorm:"column:legacy_registry_id"`
+		ShadowSchema     string `gorm:"column:shadow_schema"`
+	}
+	var maps []SchemaMap
+	err = r.db.WithContext(ctx).Raw(`
+		SELECT so.legacy_registry_id, sb.shadow_schema
+		FROM cdc_system.shadow_binding sb
+		JOIN cdc_system.source_object_registry so ON sb.source_object_id = so.id
+		WHERE sb.is_active = true AND so.legacy_registry_id IS NOT NULL
+	`).Scan(&maps).Error
+	if err != nil {
+		return entries, nil
+	}
+
+	schemaByRegistryID := make(map[uint]string, len(maps))
+	for _, m := range maps {
+		schemaByRegistryID[m.LegacyRegistryID] = m.ShadowSchema
+	}
+	for i := range entries {
+		if schema, exists := schemaByRegistryID[entries[i].ID]; exists {
+			entries[i].ShadowSchema = schema
+		}
+	}
+	return entries, nil
 }
 
 func (r *TableRegistryRepo) GetByID(ctx context.Context, id uint) (*source.TableRegistry, error) {
 	var entry source.TableRegistry
 	err := r.db.WithContext(ctx).First(&entry, id).Error
-	return &entry, err
+	if err != nil {
+		return nil, err
+	}
+	var schema string
+	r.db.WithContext(ctx).Raw(`
+		SELECT sb.shadow_schema 
+		FROM cdc_system.shadow_binding sb
+		JOIN cdc_system.source_object_registry so ON sb.source_object_id = so.id
+		WHERE so.legacy_registry_id = ? AND sb.is_active = true
+		LIMIT 1
+	`, entry.ID).Scan(&schema)
+	entry.ShadowSchema = schema
+	return &entry, nil
 }
 
 func (r *TableRegistryRepo) GetByTargetTable(ctx context.Context, targetTable string) (*source.TableRegistry, error) {
 	var entry source.TableRegistry
 	err := r.db.WithContext(ctx).Where("target_table = ?", targetTable).First(&entry).Error
-	return &entry, err
+	if err != nil {
+		return nil, err
+	}
+	var schema string
+	r.db.WithContext(ctx).Raw(`
+		SELECT sb.shadow_schema 
+		FROM cdc_system.shadow_binding sb
+		JOIN cdc_system.source_object_registry so ON sb.source_object_id = so.id
+		WHERE so.legacy_registry_id = ? AND sb.is_active = true
+		LIMIT 1
+	`, entry.ID).Scan(&schema)
+	entry.ShadowSchema = schema
+	return &entry, nil
 }
+
+func (r *TableRegistryRepo) GetByTargetTableAndSchema(ctx context.Context, targetTable string, schemaName string) (*source.TableRegistry, error) {
+	var entry source.TableRegistry
+	err := r.db.WithContext(ctx).
+		Table("cdc_system.cdc_table_registry tr").
+		Select("tr.*").
+		Joins("JOIN cdc_system.source_object_registry so ON tr.id = so.legacy_registry_id").
+		Joins("JOIN cdc_system.shadow_binding sb ON so.id = sb.source_object_id").
+		Where("tr.target_table = ? AND sb.shadow_schema = ? AND tr.is_active = ? AND sb.is_active = ?", targetTable, schemaName, true, true).
+		First(&entry).Error
+	if err != nil {
+		return nil, err
+	}
+	entry.ShadowSchema = schemaName
+	return &entry, nil
+}
```

---

## 5. Scheduler Jobs (Worker Server)

### Target: `server_jobs.go`
Update scheduler transform dispatcher to match by both flat and qualified names, and publish qualified target table names.

```diff
@@ -60,5 +60,9 @@
 		for _, entry := range entries {
-			if targetTable != "" && entry.TargetTable != targetTable {
-				continue
-			}
-			msg := &nats.Msg{Subject: "cdc.cmd.batch-transform", Data: []byte(entry.TargetTable), Header: make(nats.Header)}
+			qualified := entry.QualifiedTarget()
+			if targetTable != "" && qualified != targetTable && entry.TargetTable != targetTable {
+				continue
+			}
+			msg := &nats.Msg{Subject: "cdc.cmd.batch-transform", Data: []byte(qualified), Header: make(nats.Header)}
 			observability.InjectNATSHeader(ctx, msg.Header)
 			if publishErr := natsClient.Conn.PublishMsg(msg); publishErr != nil {
-				observability.Ctx(ctx, logger).Error("failed to publish transform command", zap.String("table", entry.TargetTable), zap.Error(publishErr))
+				observability.Ctx(ctx, logger).Error("failed to publish transform command", zap.String("table", qualified), zap.Error(publishErr))
```

---

## 6. CMS Service Manual Trigger

### Target: `source_object_actions_handler.go` (in cdc-cms-service)
Publish the qualified target table name `schema.table` in the NATS message payload.

```diff
@@ -748,6 +748,10 @@
 	type natsPayload struct {
 		JobID       string `json:"job_id"`
 		TargetTable string `json:"target_table"`
 	}
-	payloadBytes, _ := json.Marshal(natsPayload{JobID: jobID, TargetTable: scope.TargetTable})
-	if err := h.publisher.Publish(c.UserContext(), "cdc.cmd.batch-transform", payloadBytes); err != nil {
+	targetTable := scope.TargetTable
+	if scope.ShadowSchema != "" {
+		targetTable = scope.ShadowSchema + "." + scope.TargetTable
+	}
+	payloadBytes, _ := json.Marshal(natsPayload{JobID: jobID, TargetTable: targetTable})
+	if err := h.publisher.Publish(c.UserContext(), "cdc.cmd.batch-transform", payloadBytes); err != nil {
```

---

## 7. CMS Action Handler Test

### Target: `source_object_actions_handler_test.go` (in cdc-cms-service)
Update assertions to match the new qualified target table name published in NATS.

```diff
@@ -109,5 +109,5 @@
-	if payloadObj.TargetTable != "shadow_table_123" {
-		t.Errorf("expected target_table shadow_table_123, got %s", payloadObj.TargetTable)
+	if payloadObj.TargetTable != "cdc_shadow.shadow_table_123" {
+		t.Errorf("expected target_table cdc_shadow.shadow_table_123, got %s", payloadObj.TargetTable)
 	}
 
 	if len(activityLog.logEntries) != 1 {
@@ -117,3 +117,3 @@
 		entry := activityLog.logEntries[0]
-		if entry.Operation != "transform" || entry.TargetTable != "shadow_table_123" || entry.Status != "success" {
+		if entry.Operation != "transform" || entry.TargetTable != "cdc_shadow.shadow_table_123" || entry.Status != "success" {
 			t.Errorf("unexpected activity log entry details: %+v", entry)
```

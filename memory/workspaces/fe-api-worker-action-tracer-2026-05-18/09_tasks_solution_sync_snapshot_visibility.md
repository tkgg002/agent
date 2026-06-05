# Solution — Sync Fields + Snapshot Now Visibility

## Bản chất bug
Cả 2 luồng không phải fail logic — chúng có thể đã chạy hoặc fail nhưng worker log không nói rõ. Khi user click và không thấy gì xảy ra trên shadow / Mongo, không biết:
- Sync Fields: rule load OK nhưng 0 row; rule load fail; hoặc ALTER fail từng cột?
- Snapshot Now: dispatch qua signalClient hay mongo direct insert; client nil; hay Mongo insert reject?

Fix: hardening observability — không đổi behavior, chỉ surface state ra log.

## Diff snippet

### `centralized-data-service/internal/handler/command_handler.go`

**`HandleCreateDefaultColumns` — block ALTER:**
```diff
 rules, err := h.mappingV2Repo.GetActiveRulesBySourceTable(ctx, payload.SourceTable)
-if err == nil {
+if err != nil {
+    h.logger.Error("failed to load mapping_rule_v2 for ALTER",
+        zap.String("trace_id", trace.TraceID), zap.String("action", trace.Action),
+        zap.String("source_table", payload.SourceTable), zap.Error(err))
+} else {
+    h.logger.Info("loaded mapping_rule_v2 for ALTER",
+        zap.String("trace_id", trace.TraceID), zap.String("source_table", payload.SourceTable),
+        zap.Int("rules_count", len(rules)))
+    if len(rules) == 0 {
+        h.logger.Warn("no active+approved mapping rules joined to source_object_name; check source_object_registry.source_object_name vs payload.SourceTable",
+            zap.String("trace_id", trace.TraceID), zap.String("source_table", payload.SourceTable))
+    }
+    columnsSkipped := 0
     for _, rule := range rules {
-        h.logger.Info("executing column sync", zap.String("sql", alterSQL))
+        h.logger.Info("executing column sync",
+            zap.String("trace_id", trace.TraceID),
+            zap.String("column", rule.TargetColumn), zap.String("data_type", rule.DataType))
         if err := h.shadowDB.Exec(alterSQL).Error; err != nil {
-            h.logger.Warn("failed to add column", zap.String("column", rule.TargetColumn), zap.Error(err))
+            h.logger.Warn("failed to add column",
+                zap.String("trace_id", trace.TraceID),
+                zap.String("column", rule.TargetColumn), zap.String("data_type", rule.DataType), zap.Error(err))
+            columnsSkipped++
             continue
         }
         columnsAdded++
     }
+    h.logger.Info("ALTER TABLE summary",
+        zap.String("trace_id", trace.TraceID), zap.String("table", payload.TargetTable),
+        zap.Int("rules_total", len(rules)), zap.Int("columns_added", columnsAdded),
+        zap.Int("columns_skipped", columnsSkipped))
 }
```

**`processDiscoveryRows` — surface Create err:**
```diff
 added := 0
+insertErrors := 0
 for field, dataType := range discovered {
     if !mapped[field] {
         rule := model.MappingRuleV2{ ... }
-        if err := ... Create(&rule).Error; err == nil { added++ }
+        if err := ... Create(&rule).Error; err != nil {
+            h.logger.Warn("processDiscoveryRows: failed to insert mapping_rule_v2",
+                zap.Int64("source_object_id", registryID), zap.String("source_table", sourceTable),
+                zap.String("field", field), zap.String("data_type", dataType),
+                zap.String("status", status), zap.Error(err))
+            insertErrors++
+            continue
+        }
+        added++
     }
 }
+h.logger.Info("processDiscoveryRows summary",
+    zap.Int64("source_object_id", registryID), zap.String("source_table", sourceTable),
+    zap.Int("discovered_total", len(discovered)),
+    zap.Int("already_mapped", len(discovered)-added-insertErrors),
+    zap.Int("inserted", added), zap.Int("insert_errors", insertErrors))
```

### `centralized-data-service/internal/handler/recon_handler.go`

**`HandleDebeziumSignal` — log dispatch path:**
```diff
-var signalID string
-var err error
-if h.signal != nil && h.signal.IsConfigured() {
+var dispatchPath string
+signalConfigured := h.signal != nil && h.signal.IsConfigured()
+if signalConfigured {
+    dispatchPath = "signal_client"
+    h.logger.Info("debezium signal: using SignalClient path",
+        zap.String("trace_id", trace.TraceID), zap.String("database", db), zap.String("collection", collection))
     signalID, err = h.signal.TriggerIncrementalSnapshot(ctx, db, collection, payload.Filter)
 } else {
+    dispatchPath = "mongo_direct_insert"
     if h.mongoClient == nil {
-        h.logger.Warn("action trace failed", ..., "mongodb client not configured")
+        h.logger.Warn("action trace failed", ...,
+            zap.String("reason", "mongodb client not configured"),
+            zap.Bool("signal_client_nil", h.signal == nil),
+            zap.String("hint", "worker_server.go gating reconCore=nil → handler không nhận client"))
         return
     }
+    h.logger.Info("debezium signal: using MongoClient direct-insert fallback",
+        zap.String("trace_id", trace.TraceID), zap.String("database", db),
+        zap.String("collection", collection), zap.String("signal_collection", "debezium_signal"))
     ...
 }
 if err != nil {
-    h.logger.Warn("action trace failed", ..., zap.Error(err))
+    h.logger.Warn("action trace failed", ...,
+        zap.String("dispatch_path", dispatchPath),
+        zap.String("database", db), zap.String("collection", collection), zap.Error(err))
 }
 h.logger.Info("debezium signal dispatched",
+    zap.String("dispatch_path", dispatchPath),
     zap.String("signal_id", signalID), ...)
```

## Verify
- `go build ./...` PASS.
- `go vet ./...` PASS.
- `go test ./internal/handler/... ./internal/service/...` PASS (handler 3.860s).

## Grep cheatsheet cho user
```bash
# Sync Fields end-to-end với 1 trace_id
grep "trace_id=fe-sync_fields_to_shadow-<uuid>" worker.log
# Snapshot Now end-to-end
grep "trace_id=fe-snapshot_now-<uuid>" worker.log
# Số rules ALTER per click
grep "ALTER TABLE summary" worker.log
# Insert rule err
grep "failed to insert mapping_rule_v2" worker.log
# Path nào dispatch snapshot
grep "dispatch_path" worker.log
```

## Bước tiếp theo (sau khi user click thử)
- Nếu log thấy `rules_count=0` → root cause là `source_object_registry.source_object_name` ≠ `payload.SourceTable`. Fix data hoặc fix scope resolver.
- Nếu log thấy `columns_skipped > 0` với err cụ thể → fix data_type rule (legacy seed thường rỗng/sai).
- Nếu Snapshot Now log `using MongoClient direct-insert fallback` + err `not master / no replicaset` → cần signal_client config trong worker config.yml (signal_database/signal_collection).
- Nếu Snapshot Now log `signal_client_nil=true` + `worker_server gating reconCore=nil` → worker config thiếu `mongodb.url` cho recon (khác với cdc-cms connection_registry).

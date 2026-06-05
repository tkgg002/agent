# Solution — Snapshot source-key + Sync Fields count chính xác

## Bug A — `no source route for target table "export-jobs"`

### Root cause
- FE `cdc-cms-web/src/pages/TableRegistry.tsx:490` → `/api/tools/trigger-snapshot/${record.source_table}`.
- API `cdc-cms-service/internal/api/reconciliation_handler_tools.go:36-77` pass nguyên source_table vào NATS payload `Table`.
- Worker `resolveSourceMongoDSN(payload.Table)` gọi `metadata.ResolveTargetRoute("export-jobs")` → nil (`targetRouteMap` keyed by `cfg.TargetTable = sd_export_jobs`).

### Fix `internal/handler/recon_handler.go`

```diff
-func (h *ReconHandler) resolveSourceMongoDSN(ctx context.Context, targetTable string) (string, error) {
-	if h.metadata == nil { return "", ... }
-	route := h.metadata.ResolveTargetRoute(targetTable)
-	if route == nil || route.SourceObject == nil {
-		return "", fmt.Errorf("no source route for target table %q ...", targetTable)
-	}
+func (h *ReconHandler) resolveSourceMongoDSN(ctx context.Context, table string) (string, error) {
+	if h.metadata == nil { return "", ... }
+	entry := h.resolveTargetTableConfig(table)
+	if entry == nil {
+		return "", fmt.Errorf("no table registry entry for %q ...", table)
+	}
+	route := h.metadata.ResolveTargetRoute(entry.TargetTable)
+	if route == nil || route.SourceObject == nil {
+		return "", fmt.Errorf("no source route for target %q (resolved from %q) ...", entry.TargetTable, table)
+	}
```

→ Khi FE gửi `export-jobs`:
- `resolveTargetTableConfig("export-jobs")` thử `GetTableConfig` (miss) → `GetTableConfig("sd_export-jobs")` (miss vì underscore vs hyphen) → `GetTableConfigBySource("export-jobs")` (HIT) → entry.TargetTable=`sd_export_jobs`.
- `ResolveTargetRoute("sd_export_jobs")` HIT → DSN resolve OK.

## Bug B — `rows_affected:19` nhưng 0 fields mới

### Root cause
- `ALTER TABLE ... ADD COLUMN IF NOT EXISTS` không error khi column đã tồn tại → `h.shadowDB.Exec(...)` PASS → `columnsAdded++`.
- Khi shadow đã có sẵn 19 columns (sync trước đó), 19 ALTER thành công no-op → user thấy `rows_affected:19` nhưng FE không có field mới.

### Fix `internal/handler/command_handler.go`

```diff
+existingCols := h.listShadowColumns(schemaName, payload.TargetTable)
 columnsSkipped := 0
+columnsAlreadyExist := 0
 for _, rule := range rules {
+    colLower := strings.ToLower(strings.TrimSpace(rule.TargetColumn))
+    if _, ok := existingCols[colLower]; ok {
+        columnsAlreadyExist++
+        continue
+    }
     alterSQL := fmt.Sprintf(`ALTER TABLE ... ADD COLUMN IF NOT EXISTS ...`)
     if err := h.shadowDB.Exec(alterSQL).Error; err != nil {
         columnsSkipped++; continue
     }
+    existingCols[colLower] = struct{}{}
     columnsAdded++
 }
 h.logger.Info("ALTER TABLE summary",
     ..., zap.Int("columns_added", columnsAdded),
+    zap.Int("columns_already_exist", columnsAlreadyExist),
     zap.Int("columns_skipped", columnsSkipped),
 )
```

Helper mới:

```go
func (h *CommandHandler) listShadowColumns(schemaName, tableName string) map[string]struct{} {
    if strings.TrimSpace(schemaName) == "" { schemaName = "public" }
    out := make(map[string]struct{})
    var cols []string
    h.shadowDB.Raw(
        "SELECT column_name FROM information_schema.columns WHERE table_schema = ? AND table_name = ?",
        schemaName, tableName,
    ).Scan(&cols)
    for _, c := range cols {
        out[strings.ToLower(strings.TrimSpace(c))] = struct{}{}
    }
    return out
}
```

## Verify

- `go build ./...` EXIT=0.
- `go vet ./...` EXIT=0.
- `go test -count=1 ./internal/handler/... ./internal/server/...` PASS (handler 3.928s).

## User test sau restart worker

1. Ctrl-C worker → `go run cmd/worker/main.go`.
2. Click "Snapshot Now" cho `export-jobs`:
   - Worker log expected: `dispatch_path=mongo_lazy_resolve signal_id=<ObjectID>`.
   - KHÔNG còn thấy `no source route for target table "export-jobs"`.
3. Click "Sync Fields" cho `sd_export_jobs` (shadow đã đủ field):
   - Worker log: `ALTER TABLE summary rules_total=19 columns_added=0 columns_already_exist=19 columns_skipped=0`.
   - FE response: `rows_affected: 0`.
4. Nếu mapping_rule thêm column mới sau đó, click Sync Fields:
   - `columns_added=<N> columns_already_exist=19-N`.
   - FE response: `rows_affected: N`.

## Grep cheatsheet

```bash
grep "columns_already_exist" worker.log     # confirm phép đếm mới
grep "dispatch_path=mongo_lazy_resolve" worker.log
grep "no source route for target" worker.log  # phải = 0 sau fix
grep "no table registry entry" worker.log     # signal khi cả 2 key đều miss
```

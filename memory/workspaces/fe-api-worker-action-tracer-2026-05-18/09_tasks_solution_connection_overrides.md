# Solution — Worker-side connection_overrides

## Tóm tắt

Thêm overlay map `cfg.ConnectionOverrides map[connection_code]URI`. Worker check map mỗi khi đọc 1 row `cdc_system.connection_registry` ngay TRƯỚC khi build DSN từ host/port/secret_ref. Hit → dùng override; miss → fallback nguyên behavior cũ. Empty map → diff bằng 0.

## File thay đổi

| File | Loại | Chú thích |
|---|---|---|
| `config/config.go` | EDIT | Thêm `AppConfig.ConnectionOverrides`; env scanner `CONNECTION_OVERRIDE_<CODE>` |
| `config/config-local.yml` | EDIT | Example `connectionOverrides:` block (dev preset cho `goopay`) |
| `internal/service/connection_overrides.go` | NEW | `ApplyConnectionOverride`, `NormalizeConnectionOverrides` |
| `internal/service/metadata_registry_service.go` | EDIT | `MetadataRegistryService.connectionOverrides`; ctor nhận overrides; `GetSourceDSN` check overlay |
| `internal/handler/command_handler.go` | EDIT | `CommandHandler.connectionOverrides`; `SetConnectionOverrides`; `scanFieldsMongoSource` check overlay |
| `internal/handler/recon_handler.go` | EDIT | `ReconHandler.connectionOverrides`; `WithConnectionOverrides`; `resolveSourceMongoDSN` check overlay |
| `internal/server/worker_server.go` | EDIT | Normalize overrides 1 lần; wire vào ctor/setter/builder cho cả 3 site (cả 2 nhánh ReconHandler) |

## Code diff theo Site

### Site A — `internal/service/metadata_registry_service.go::GetSourceDSN`
```diff
 conn, err := rs.connectionRepo.GetByCode(ctx, connectionCode)
 if err != nil { return "", err }
 if conn == nil { return "", fmt.Errorf("connection %q not found", connectionCode) }

+if uri, ok := ApplyConnectionOverride(conn, rs.connectionOverrides, rs.logger); ok {
+    return uri, nil
+}

 if dsn := tryPlainDSN(conn.SecretRef); dsn != "" { return dsn, nil }
 ...
```

### Site B — `internal/handler/command_handler.go::scanFieldsMongoSource`
```diff
 var conn model.ConnectionRegistry
 if dbErr := h.db.WithContext(ctx).First(&conn, registry.SourceConnectionID).Error; dbErr != nil { ... }
-hostRaw := ""
-if conn.Host != nil { hostRaw = strings.TrimSpace(*conn.Host) }
-if hostRaw == "" { return 0, 0, fmt.Errorf("connection_registry id=%d missing host", ...) }
-var dsn string
-if strings.HasPrefix(hostRaw, "mongodb://") || strings.HasPrefix(hostRaw, "mongodb+srv://") { dsn = hostRaw }
-else {
-    if conn.Port == nil || *conn.Port <= 0 { return ... }
-    dsn = fmt.Sprintf("mongodb://%s:%d/", hostRaw, *conn.Port)
-}
+var dsn string
+if uri, ok := service.ApplyConnectionOverride(&conn, h.connectionOverrides, h.logger); ok {
+    dsn = uri
+} else {
+    hostRaw := ""
+    if conn.Host != nil { hostRaw = strings.TrimSpace(*conn.Host) }
+    if hostRaw == "" { return 0, 0, fmt.Errorf("connection_registry id=%d missing host", ...) }
+    if strings.HasPrefix(hostRaw, "mongodb://") || strings.HasPrefix(hostRaw, "mongodb+srv://") { dsn = hostRaw }
+    else {
+        if conn.Port == nil || *conn.Port <= 0 { return ... }
+        dsn = fmt.Sprintf("mongodb://%s:%d/", hostRaw, *conn.Port)
+    }
+}
```

### Site C — `internal/handler/recon_handler.go::resolveSourceMongoDSN`
```diff
 var conn model.ConnectionRegistry
 if err := h.db.WithContext(ctx).First(&conn, connID).Error; err != nil { ... }
+if uri, ok := service.ApplyConnectionOverride(&conn, h.connectionOverrides, h.logger); ok {
+    return uri, nil
+}
 hostRaw := ""
 if conn.Host != nil { hostRaw = strings.TrimSpace(*conn.Host) }
 ...
```

## Helper — `internal/service/connection_overrides.go`

```go
func ApplyConnectionOverride(conn *model.ConnectionRegistry, overrides map[string]string, logger *zap.Logger) (string, bool) {
    if conn == nil || len(overrides) == 0 { return "", false }
    code := strings.ToLower(strings.TrimSpace(conn.ConnectionCode))
    if code == "" { return "", false }
    uri, ok := overrides[code]
    if !ok { return "", false }
    uri = strings.TrimSpace(uri)
    if uri == "" { return "", false }
    if logger != nil {
        logger.Info("connection override applied",
            zap.String("connection_code", conn.ConnectionCode),
            zap.String("engine", conn.EngineType),
            zap.String("origin", "config"))
    }
    return uri, true
}

func NormalizeConnectionOverrides(in map[string]string) map[string]string { ... lowercase keys ... }
```

## Config schema

```yaml
# config-local.yml (dev)
connectionOverrides:
  goopay: "mongodb://localhost:17017/?replicaSet=rs0&directConnection=true"
  # goopay1: "mongodb://localhost:17017/?replicaSet=rs0&directConnection=true"
  # default_shadow: "postgres://gpay_admin:gpay_pass@localhost:5436/cdc_shadow?sslmode=disable"
```

```bash
# Per-key env override (CI / prod selective):
export CONNECTION_OVERRIDE_GOOPAY="mongodb://override.host:27017/"
```

## Wiring (`internal/server/worker_server.go`)

```go
connectionOverrides := service.NormalizeConnectionOverrides(cfg.ConnectionOverrides)
if len(connectionOverrides) > 0 {
    codes := make([]string, 0, len(connectionOverrides))
    for code := range connectionOverrides { codes = append(codes, code) }
    logger.Info("connection overrides loaded", zap.Strings("connection_codes", codes))
}
registrySvc := service.NewMetadataRegistryService(..., cfg.MasterKey, connectionOverrides)
cmdHandler.SetConnectionOverrides(connectionOverrides)
// Both ReconHandler branches:
reconHandler := handler.NewReconHandler(...).WithConnectionOverrides(connectionOverrides)
signalOnlyHandler := handler.NewReconHandler(nil, db, nil, ...).WithConnectionOverrides(connectionOverrides)
```

## Verify

| Command | Result |
|---|---|
| `go build ./...` | EXIT=0 |
| `go vet ./...` | EXIT=0 |
| `go test -count=1 ./internal/handler/... ./internal/service/...` | PASS (handler 3.780s, service 1.369s) |
| `go test -count=1 ./config/...` | PASS (0.215s) |
| Runtime probe parse `config-local.yml` + `CONNECTION_OVERRIDE_GOOPAY1=...` | YAML hit `goopay`, env hit `goopay1` — cả 2 hiện đúng trong `cfg.ConnectionOverrides` |

## User test sau restart worker

1. Confirm `connection_registry` row có `connection_code='goopay'` (admin nhập gpay-mongo host).
2. Restart worker: Ctrl-C tty003 → `go run cmd/worker/main.go`.
3. Worker startup log expected:
   - `connection overrides loaded connection_codes=[goopay]`
4. Click Snapshot Now `export-jobs`:
   - Worker log expected:
     - `connection override applied connection_code=goopay engine=mongodb origin=config`
     - `dispatch_path=mongo_lazy_resolve signal_id=<ObjectID>`
   - KHÔNG còn `no such host: gpay-mongo` / `no source route ...`.
5. Click Sync Fields cho `sd_export_jobs` (route Mongo fallback nếu shadow empty):
   - Worker log: `connection override applied ... origin=config` (Site B hit).
   - `ALTER TABLE summary columns_added=<N>` (đã fix lần trước).

## Out-of-band (production)

- `connectionOverrides` map rỗng (mặc định prod) → behavior identical với code cũ.
- Audit log: mỗi hit print `connection override applied` 1 lần per scan/snapshot → CO compliance review dễ.

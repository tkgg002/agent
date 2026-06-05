# Plan — Worker-side connection_overrides

## Step 1 — Config layer

### `config/config.go`
1. Thêm field `ConnectionOverrides map[string]string \`mapstructure:"connectionOverrides"\`` vào `AppConfig`.
2. Trong `applyEnvOverrides`: scan `os.Environ()` cho prefix `CONNECTION_OVERRIDE_<CODE>` → upsert vào `cfg.ConnectionOverrides[strings.ToLower(code)]` (key normalize lowercase).

### `config/config-local.yml`
Thêm block ví dụ commented-out cho dev:
```yaml
connectionOverrides:
  goopay: "mongodb://localhost:17017/?replicaSet=rs0&directConnection=true"
  # goopay1: "mongodb://localhost:17017/?replicaSet=rs0&directConnection=true"
  # default_shadow: "postgres://gpay_admin:gpay_pass@localhost:5436/cdc_shadow?sslmode=disable"
```

## Step 2 — Generic helper

### `internal/service/connection_overrides.go` (NEW)
```go
package service

import (
    "strings"

    "centralized-data-service/internal/model"
    "go.uber.org/zap"
)

// ApplyConnectionOverride checks the worker-side overlay keyed by
// connection_code. Returns (overrideURI, true) when an override is
// configured for conn.ConnectionCode, ("", false) otherwise.
//
// Match rule: case-insensitive on conn.ConnectionCode. Logger is
// optional — pass nil to skip the hit log.
func ApplyConnectionOverride(conn *model.ConnectionRegistry, overrides map[string]string, logger *zap.Logger) (string, bool) {
    if conn == nil || len(overrides) == 0 {
        return "", false
    }
    code := strings.ToLower(strings.TrimSpace(conn.ConnectionCode))
    if code == "" {
        return "", false
    }
    if uri, ok := overrides[code]; ok && strings.TrimSpace(uri) != "" {
        if logger != nil {
            logger.Info("connection override applied",
                zap.String("connection_code", conn.ConnectionCode),
                zap.String("engine", conn.EngineType),
                zap.String("origin", "config"),
            )
        }
        return strings.TrimSpace(uri), true
    }
    return "", false
}

// NormalizeConnectionOverrides lowercases keys so lookups by
// strings.ToLower(connection_code) match regardless of YAML/env casing.
func NormalizeConnectionOverrides(in map[string]string) map[string]string {
    if len(in) == 0 {
        return nil
    }
    out := make(map[string]string, len(in))
    for k, v := range in {
        if strings.TrimSpace(v) == "" {
            continue
        }
        out[strings.ToLower(strings.TrimSpace(k))] = strings.TrimSpace(v)
    }
    return out
}
```

## Step 3 — Site A: `MetadataRegistryService.GetSourceDSN`

### `internal/service/metadata_registry_service.go`
- Thêm field `connectionOverrides map[string]string` vào struct.
- Constructor `NewMetadataRegistryService` nhận thêm tham số `overrides map[string]string`, normalize qua `NormalizeConnectionOverrides`.
- `GetSourceDSN`: sau khi load `conn`, GỌI `ApplyConnectionOverride(conn, rs.connectionOverrides, rs.logger)` → nếu hit, return luôn (skip toàn bộ tryPlainDSN/env/fields/AES).

```go
func (rs *MetadataRegistryService) GetSourceDSN(ctx context.Context, connectionCode string) (string, error) {
    conn, err := rs.connectionRepo.GetByCode(ctx, connectionCode)
    if err != nil { return "", err }
    if conn == nil { return "", fmt.Errorf("connection %q not found", connectionCode) }

    if uri, ok := ApplyConnectionOverride(conn, rs.connectionOverrides, rs.logger); ok {
        return uri, nil
    }

    if dsn := tryPlainDSN(conn.SecretRef); dsn != "" { return dsn, nil }
    // ... rest unchanged
}
```

## Step 4 — Site B: `CommandHandler.scanFieldsMongoSource`

### `internal/handler/command_handler.go`
- Thêm field `connectionOverrides map[string]string` + setter `SetConnectionOverrides(map[string]string)`.
- Trong `scanFieldsMongoSource`, NGAY SAU `h.db.WithContext(ctx).First(&conn, ...)` thành công:

```go
if uri, ok := service.ApplyConnectionOverride(&conn, h.connectionOverrides, h.logger); ok {
    dsn = uri
} else {
    // existing host/port assembly
    hostRaw := ""
    if conn.Host != nil { hostRaw = strings.TrimSpace(*conn.Host) }
    if hostRaw == "" { return 0, 0, fmt.Errorf("connection_registry id=%d missing host; cannot build Mongo DSN", registry.SourceConnectionID) }
    if strings.HasPrefix(hostRaw, "mongodb://") || strings.HasPrefix(hostRaw, "mongodb+srv://") {
        dsn = hostRaw
    } else {
        if conn.Port == nil || *conn.Port <= 0 {
            return 0, 0, fmt.Errorf("connection_registry id=%d missing port (host is bare %q)", registry.SourceConnectionID, hostRaw)
        }
        dsn = fmt.Sprintf("mongodb://%s:%d/", hostRaw, *conn.Port)
    }
}
```

## Step 5 — Site C: `ReconHandler.resolveSourceMongoDSN`

### `internal/handler/recon_handler.go`
- Thêm field `connectionOverrides map[string]string` + builder `WithConnectionOverrides(map[string]string) *ReconHandler`.
- Sau `h.db.WithContext(ctx).First(&conn, connID)`:

```go
if uri, ok := service.ApplyConnectionOverride(&conn, h.connectionOverrides, h.logger); ok {
    return uri, nil
}
// existing host/port logic unchanged
```

## Step 6 — Wiring `worker_server.go`

1. Normalize overrides một lần: `overrides := service.NormalizeConnectionOverrides(cfg.ConnectionOverrides)`.
2. Pass vào `NewMetadataRegistryService(..., cfg.MasterKey, overrides)`.
3. `cmdHandler.SetConnectionOverrides(overrides)` sau khi tạo CommandHandler.
4. CẢ HAI nhánh ReconHandler (reconCore enabled + signalOnly): `.WithConnectionOverrides(overrides)`.

## Step 7 — Verify (Caller-Resolver Wiring Lesson)

1. `go build ./...` → EXIT=0.
2. `go vet ./...` → EXIT=0.
3. `go test -count=1 ./internal/handler/... ./internal/service/...` → PASS.
4. Restart worker với `connectionOverrides.goopay: mongodb://localhost:17017/?directConnection=true`.
5. Trigger Snapshot Now cho `export-jobs` → expect:
   - Worker log: `connection override applied connection_code=goopay engine=mongodb origin=config`.
   - Worker log: `dispatch_path=mongo_lazy_resolve signal_id=<ObjectID>`.
   - KHÔNG còn `no such host: gpay-mongo`.
6. Click Sync Fields → quan sát path (Mongo fallback hoặc Debezium TopicConfig — Sync Fields hiện CHỦ YẾU đi qua Debezium HTTP, mongo fallback rare).

## Step 8 — Docs

- APPEND `05_progress.md` mỗi file thay đổi.
- Tạo `08_tasks_connection_overrides.md` + `09_tasks_solution_connection_overrides.md`.
- Tạo `report_connection_overrides.md` ở root workspace.
- APPEND global lesson `Worker-side overlay map keyed by stable logical code overrides admin-input URIs without DB writes`.

## Risk

- **R1** Override leak vào production: mitigated bởi NFR-1 (env var pattern rõ ràng) + log `INFO` mỗi hit để audit.
- **R2** ConnectionCode trùng giữa engine khác (mongo vs postgres cùng code): unlikely vì DB constraint UNIQUE — nhưng helper trả pass-through string nên không corrupt cross-engine.
- **R3** `Caller-Resolver Wiring Verification` lesson: nếu sót site nào thì overlay bypass tại path đó. Mitigated bởi Explore agent enumerate đã liệt kê đủ 3 site bypass + Site A canonical.

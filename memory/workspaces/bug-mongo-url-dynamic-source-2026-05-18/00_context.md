# 00_context — Bug: worker không dùng dynamic mongo source

> **Date**: 2026-05-18
> **Repo**: `centralized-data-service` (Worker plane)
> **Reporter**: User

## Error gốc

```
return 0, 0, fmt.Errorf("mongoURL not configured on worker; cannot introspect source")
```

Vị trí: `internal/handler/command_handler.go` (hiện tại line 280, đã có guard "dynamically or statically").

Trigger: NATS subject `cdc.cmd.scan-fields` (introspect Mongo source để discover fields).

## User intent

> "source đã đc update lên để user add động vào. nhưng hệ thống vẫn đang dùng url từ env. update lại chõ này."

- User đã add source mới qua UI cdc-cms-service `POST /api/v1/sources` → ghi vào `cdc_system.connection_registry`.
- Worker vẫn đi đường `cfg.MongoDB.URL` (env / YAML) → khi env không set → error trên.
- Yêu cầu: worker phải resolve DSN từ `connection_registry` row (do user add động) thay vì env tĩnh.

## Flow hiện tại (đã trace)

1. `scanFieldsMongoSource(registryID)` load `source_object_registry` row.
2. Lấy `registry.SourceConnectionID` → load `connection_registry` row.
3. Gọi `metadata.GetSourceDSN(ctx, conn.ConnectionCode)` → resolve DSN.
4. Nếu fail → fallback `h.mongoURL` (= `cfg.MongoDB.URL` từ YAML/env).
5. Nếu cả hai empty → error.

## Lỗi cốt lõi

`GetSourceDSN` trong `internal/service/metadata_registry_service.go:323-338`:

```go
func (rs *MetadataRegistryService) GetSourceDSN(ctx context.Context, connectionCode string) (string, error) {
    conn, err := rs.connectionRepo.GetByCode(ctx, connectionCode)
    if err != nil { return "", err }
    if conn == nil || conn.SecretRef == "" {
        return "", fmt.Errorf("connection %q not found or has no secret_ref", connectionCode)
    }
    dsn, err := crypto.DecryptAES(conn.SecretRef, rs.masterKey)
    if err != nil {
        return "", fmt.Errorf("decrypt DSN failed for %q: %w", connectionCode, err)
    }
    return dsn, nil
}
```

Hàm này chỉ biết DecryptAES, nhưng giá trị thực tế của `secret_ref` trong DB là URI scheme/pointer:
- `bootstrap_cdc_local.sql`: `'env://source.default'`, `'env://shadow.default'`, `'env://master.default'`
- `bootstrap_cdc_system_v2_local.sql`: `'env://mongodb.url'`, `'env://shadow.default'`, ...
- `cdc-cms-service` source create (`system_connector_repo_gorm.go:73`): `'v1:' + connector_name`
- `cdc-cms-service` shadow bootstrap (`shadow_connection.go:40`): `'env:CMS_SHADOW_DB_PASSWORD'`

KHÔNG có row nào chứa ciphertext AES → DecryptAES luôn fail → fallback env.

## Files đã đọc / verified

- `centralized-data-service/internal/handler/command_handler.go:251-311` (scanFieldsMongoSource)
- `centralized-data-service/internal/service/metadata_registry_service.go:1-340` (interface + GetSourceDSN)
- `centralized-data-service/internal/repository/connection_registry_repo.go` (GetByCode trả full row)
- `centralized-data-service/internal/model/connection_registry.go` (struct: Host*, Port*, DefaultDatabase*, EngineType, SecretRef, OptionsJSON)
- `centralized-data-service/internal/server/worker_server.go:248-256` (wiring: SetMetadataRegistry ĐÃ inject)
- `centralized-data-service/deployments/sql/cdc/bootstrap_cdc_local.sql` (seed convention `env://...`)
- `centralized-data-service/deployments/sql/bootstrap_cdc_system_v2_local.sql` (seed convention `env://...`)
- `cdc-cms-service/internal/infra/persistence/system_connector_repo_gorm.go:33-76` (UI create source: `secret_ref = 'v1:'+name`)
- `cdc-cms-service/internal/bootstrap/registry_mirror.go:30-51` (legacy mirror: `secret_ref = 'v1:'+name`)
- `cdc-cms-service/internal/bootstrap/shadow_connection.go:34-51` (shadow bootstrap: `secret_ref = 'env:CMS_SHADOW_DB_PASSWORD'`)

## Constraint

- Chỉ làm đúng yêu cầu — không refactor mở rộng (lesson P-scope-creep).
- Không cheat DB / không thay đổi YAML config để fake (rule core systems).
- Verify trước khi báo done (build + test + có thể unit test resolver).

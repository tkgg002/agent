# 02_plan — Worker resolve DSN từ connection_registry động

> **Goal**: `scanFieldsMongoSource` thành công khi user add source qua cdc-cms UI, KHÔNG cần env `MONGODB_URL`.

## Definition of Done

1. `GetSourceDSN(ctx, code)` resolve được DSN từ `connection_registry` row khi:
   - `secret_ref` là URI scheme `env://NAMESPACE.KEY` hoặc `env:VAR_NAME` → resolve env var theo convention.
   - `secret_ref` là plain DSN scheme (`mongodb://`, `mongodb+srv://`, `postgres://`, `postgresql://`, `mysql://`) → return as-is.
   - `secret_ref` là `v1:CONNECTOR_NAME` (legacy pointer) → ignore, build DSN từ fields.
   - `secret_ref` rỗng hoặc unknown scheme → build DSN từ `host:port/default_database` theo `engine_type`.
   - Last resort: thử `DecryptAES` (giữ backward-compat cho ciphertext tương lai).
2. Khi build DSN từ fields:
   - `engine_type='mongodb'` → `mongodb://host:port/` (sourceDB pass riêng vào IntrospectCollection).
   - `engine_type='postgresql'/'postgres'` → `postgres://host:port/<db>?sslmode=disable` (lấy options từ `options_json.sslmode` nếu có).
   - `engine_type='mysql'/'mariadb'` → tương tự `mysql://...` (cấp 2, không bắt buộc cho ticket này).
3. `scanFieldsMongoSource` gọi `GetSourceDSN` thành công với connection_registry row mà cdc-cms tạo (có host/port/default_database, `secret_ref='v1:'+code`).
4. Build PASS (`go build ./...`).
5. Unit test cover 5 trường hợp `secret_ref` resolver.
6. Test PASS (`go test ./internal/service/...`).

## Plan thực thi (Muscle)

### Bước 1 — Sửa `GetSourceDSN`
File: `centralized-data-service/internal/service/metadata_registry_service.go`
Hàm: `GetSourceDSN` (line 323).

Code mới (pseudo):
```go
func (rs *MetadataRegistryService) GetSourceDSN(ctx context.Context, connectionCode string) (string, error) {
    conn, err := rs.connectionRepo.GetByCode(ctx, connectionCode)
    if err != nil {
        return "", err
    }
    if conn == nil {
        return "", fmt.Errorf("connection %q not found", connectionCode)
    }

    // Layer 1: secret_ref carries a direct DSN.
    if dsn := tryPlainDSN(conn.SecretRef); dsn != "" {
        return dsn, nil
    }
    // Layer 2: env:// or env: pointer (e.g., "env://mongodb.url", "env:MONGO_URI").
    if dsn := tryEnvPointer(conn.SecretRef); dsn != "" {
        return dsn, nil
    }
    // Layer 3: build DSN from connection_registry fields.
    if dsn := buildDSNFromFields(conn); dsn != "" {
        return dsn, nil
    }
    // Layer 4: legacy AES ciphertext (backward-compat).
    if conn.SecretRef != "" {
        if dsn, err := crypto.DecryptAES(conn.SecretRef, rs.masterKey); err == nil && dsn != "" {
            return dsn, nil
        }
    }
    return "", fmt.Errorf("cannot resolve DSN for connection %q (engine=%s, secret_ref=%q): no host/port available and secret_ref unresolved",
        connectionCode, conn.EngineType, redactSecret(conn.SecretRef))
}
```

Helpers (cùng file, package-private):
- `tryPlainDSN(s string) string` — check prefix `mongodb://`, `mongodb+srv://`, `postgres://`, `postgresql://`, `mysql://`, `mariadb://`.
- `tryEnvPointer(s string) string` — match `env://X.Y` (→ env `X_Y` uppercased) hoặc `env:VAR`. Trim, uppercase, replace `.`→`_`.
- `buildDSNFromFields(conn *model.ConnectionRegistry) string` — engine-aware:
  - mongodb: cần host+port. DSN = `mongodb://host:port/`.
  - postgresql: cần host+port+default_database. DSN = `postgres://host:port/db?sslmode=<from options_json or disable>`.
  - khác: trả "".
- `redactSecret(s string) string` — log-safe (first 20 char + `...`).

### Bước 2 — Unit test
File mới: `centralized-data-service/internal/service/metadata_registry_dsn_test.go`
Test cases:
1. `secret_ref="mongodb://localhost:17017"` → returns as-is.
2. `secret_ref="env:MONGO_URI"` + `os.Setenv("MONGO_URI", "mongodb://x:1/")` → returns env value.
3. `secret_ref="env://mongodb.url"` + `os.Setenv("MONGODB_URL", "mongodb://y:2/")` → returns env value.
4. `secret_ref="v1:goopay_mongo_source"` + host="mongo-host"+port=27017+engine_type="mongodb" → returns `mongodb://mongo-host:27017/`.
5. `secret_ref=""` + host+port+engine=postgresql+default_database="goopay_dest" → returns `postgres://h:p/goopay_dest?sslmode=disable`.

Test các helper trực tiếp (không phụ thuộc gorm) — vì `GetSourceDSN` đầy đủ cần connectionRepo (DB), tách helper ra để testable.

### Bước 3 — Verify
1. `go build ./...` → EXIT=0.
2. `go test ./internal/service/...` → PASS.
3. Đọc lại `command_handler.go:scanFieldsMongoSource` để confirm logic fallback static có thể giữ làm safety net (nhưng không còn là default).
4. **Verify chéo**: check nếu sửa breakdown các caller khác của `GetSourceDSN` (provisioning_step_handlers.go:533) — chúng vẫn nhận DSN tốt hơn (engine-aware build) không phá behavior.

### Bước 4 — Document + lesson
- Tạo `report_dynamic_source_dsn_fix_2026-05-18.md` với:
  - Root cause analysis.
  - Diff summary (file:line).
  - Test output (PASS evidence).
  - Files changed list.
- APPEND `05_progress.md` (workspace).
- APPEND lesson global về "secret_ref resolver phải multi-scheme, không assume ciphertext".

## Risk / Tradeoffs

- **Risk**: Nếu hệ thống production HIỆN ĐANG dùng AES ciphertext trong `secret_ref` (chưa thấy bằng chứng) → layer 4 vẫn handle. KHÔNG break.
- **Risk**: `env://X.Y` convention — chọn `os.Getenv("X_Y")` uppercased. Convention chưa được document. Acceptable vì local seed data dùng exactly `env://mongodb.url` ⇒ env `MONGODB_URL` ⇒ khớp với `MONGODB_URL` env mà worker_server đang đọc rồi.
- **No DB cheat**: KHÔNG đụng DB, KHÔNG sửa YAML, chỉ sửa Go code resolver.
- **No scope creep**: KHÔNG sửa `provisioning_step_handlers.go`, KHÔNG remove static fallback (`h.mongoURL`). Static fallback vẫn là safety net khi dynamic fail (đúng pattern circuit-breaker lite).

## Rollback

`git diff internal/service/metadata_registry_service.go` → `git checkout`. Test file mới: `rm` nếu cần.

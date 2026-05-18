# 09_tasks_solution_cleanup — Technical Solution

## Phạm vi

Chỉ sửa **1 file**: `data-hub/centralized-data-service/config/config-local.yml`.
KHÔNG sửa `.go`, KHÔNG sửa `config-sample.yml`/`config-production.yml`.

## Diff dự kiến

### Block 1: xóa `airbyte:` (3 dòng + 1 dòng header)

```yaml
# XÓA
airbyte:
  apiUrl: http://localhost:18000
  clientId: ""
  clientSecret: ""
```

**Verify**: `grep -n "Airbyte\|airbyte" config/config.go` → 0 match cho struct field → safe.

### Block 2: xóa `sources.postgres_primary`

```yaml
# TRƯỚC
sources:
  mongodb_primary: mongodb://localhost:17017/?directConnection=true
  postgres_primary: postgres://src_user:src_pass@localhost:5435/goopay_source?sslmode=disable

# SAU
sources:
  mongodb_primary: mongodb://localhost:17017/?directConnection=true
```

**Verify**: `grep -rn '"postgres_primary"' --include="*.go"` → chỉ ở config.go env-override block, 0 reader caller → safe.

**Tradeoff**: Track D Hardening structure-only — nếu sau này wire PG source qua `cfg.SourceURL("postgres_primary")` thì add lại. Hiện tại 0 caller → noise.

### Block 3-5: xóa `worker.{fetchSize, transformInterval, scanInterval}` (3 dòng)

```yaml
# TRƯỚC
worker:
  poolSize: 10
  batchSize: 500
  batchTimeout: 2s
  fetchSize: 1000
  transformInterval: 5m
  scanInterval: 1h
  transformChunkSize: 10
  kafkaBatchFlushSize: 10

# SAU
worker:
  poolSize: 10
  batchSize: 500
  batchTimeout: 2s
  transformChunkSize: 10
  kafkaBatchFlushSize: 10
```

**Verify**: `grep -rn "FetchSize\|TransformInterval\|ScanInterval" --include="*.go"` → chỉ struct definition `config.go:189-191`, 0 caller → safe.

### Block 6: xóa `jwt.expiration` (giữ `jwt.secret`)

```yaml
# TRƯỚC
jwt:
  secret: change-me-in-production
  expiration: 24h

# SAU
jwt:
  secret: change-me-in-production
```

**Verify**: `grep -rn "JWT.Expiration\|Expiration\b" --include="*.go"` → chỉ struct definition `config.go:198`, 0 caller → safe. `jwt.secret` PHẢI giữ vì `validateConfig:447-449` require non-empty.

### Block 7: xóa `debezium.connectorName`

```yaml
# TRƯỚC
debezium:
  kafkaConnectUrl: http://127.0.0.1:18083
  connectorName: goopay-mongodb-cdc
  signalDatabase: centralized-export-service
  signalCollection: debezium_signal
  incrementalChunkSize: 1000

# SAU
debezium:
  kafkaConnectUrl: http://127.0.0.1:18083
  signalDatabase: centralized-export-service
  signalCollection: debezium_signal
  incrementalChunkSize: 1000
```

**Verify**: `command_handler.go:2018-2022` hardcode `return "goopay-mongodb-cdc"`. Env override `DEBEZIUM_CONNECTOR_NAME` set vào `cfg.Debezium.ConnectorName` nhưng KHÔNG ai đọc cfg field này.

**Lưu ý**: Sửa handler để đọc cfg là **refactor riêng** — không nằm trong scope cleanup này.

## Verification plan

1. `go build ./...` trong `centralized-data-service/` → PASS.
2. `go test ./config/...` → PASS (config_test.go cover loader behavior).
3. Smoke check: `grep -c "^[a-z]" config-local.yml` trước/sau để confirm block count giảm 1 (airbyte) + sub-keys giảm 7.

## Rollback

Nếu test fail: `git diff config/config-local.yml` → `git checkout config/config-local.yml` (file ngoài git tracking? — check repo state). Lệnh `git status` đầu để xác định.

## Risk: NONE
- Không sửa code Go → không có behavior change runtime.
- Tất cả keys xóa đã được grep verify là 0 caller.
- `validateConfig` không yêu cầu các key này.

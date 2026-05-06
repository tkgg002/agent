# Plan — B5.6.2 + B5.6.3: prod-config alignment cho centralized-data-service & cdc-cms-service

> **Date**: 2026-05-05 12:30 ICT
> **Workspace**: feature-system-refactor-2026-05
> **Pattern reference**: `02_plan_auth_prod_config.md` (B5.6.1 — đã hoàn tất)
> **Lessons reference**: `agent/memory/global/lessons.md`
>   - "Dockerfile bake `config-local.yml` only = prod ship DEV creds"
>   - "Go service `.env.example` = dead weight nếu (no godotenv) ∧ (compose có defaults)"
>   - "`.env.example` style — actionable env vars > prose comments"

## 0. Context (audit kết quả thực)

### centralized-data-service

| Vấn đề | Bằng chứng |
|---|---|
| Path resolver chỉ NAME | `config.go:197-204` — chỉ `os.Getenv("CFG_PATH")`, không support absolute path |
| applyEnvOverrides hardcoded | `config.go:273-392` — 25+ `os.Getenv(...)` blocks (vì có parsing đặc biệt) |
| Không có validateConfig | toàn bộ file 0 hit "validate" |
| Thiếu prod yml | `config/` chỉ có `config-local.yml` |
| Dockerfile.worker bake DEV | `Dockerfile.worker:12` `COPY config-local.yml only` |
| `.env.example` ACTIVE | compose có 16+ `${VAR:-default}` references → KEEP |

**Đặc biệt**: `applyEnvOverrides` có **parsing logic phức tạp** không thể chuyển sang BindEnv:
- `parseNamedURLs(v)` — JSON object hoặc semicolon list cho `CDC_SHADOW_DB_URLS` / `CDC_MASTER_DB_URLS`.
- Sources↔mongodb.url bidirectional bridge (line 364-389).
- DB_SINK_URL force `cfg.DB.SSLMode = "disable"` (side-effect line 276).

→ **Quyết định**: KEEP `applyEnvOverrides` nguyên trạng, CHỈ thêm path-resolver dual-mode + validateConfig.

### cdc-cms-service

| Vấn đề | Bằng chứng |
|---|---|
| Path resolver chỉ NAME | `config.go:87-95` — y hệt centralized |
| applyEnvOverrides hardcoded | `config.go:112-148` — 11 simple field mappings, KHÔNG có parsing đặc biệt |
| Không có validateConfig | `grep validate` 0 hit |
| Thiếu prod yml | `config/` chỉ có `config-local.yml` |
| Dockerfile bake DEV | `Dockerfile:12` `COPY config-local.yml only` |
| `.env.example` DEAD | `grep godotenv cdc-cms-service/` = 0; KHÔNG có `docker-compose.yml`; KHÔNG xuất hiện trong compose nào khác → DELETE per decision tree |
| ServerConfig thiếu Mode field | `config.go:47-50` chỉ có `Name`, `Port` |

**Đặc biệt**: applyEnvOverrides simple → có thể **REFACTOR sang BindEnv pattern** giống `cdc-auth-service` (single source of truth, no Go change khi thêm field).

→ **Quyết định**: REFACTOR applyEnvOverrides → BindEnv pattern, ADD Mode field, ADD validateConfig.

## 1. Mục tiêu (2-line)

Mỗi service dual-mode path resolver + validateConfig + 3 yml (local/production/sample) + Dockerfile copy folder. Chuẩn pattern reconcile-service đã verify từ B5.6.1.

## 2. B5.6.2 — centralized-data-service

### 2.1 Refactor `config/config.go`

**Diff plan**:

1. Thêm `defaultJWTPlaceholder = "change-me-in-production"` const.
2. `NewConfig()`:
   - Path resolver dual-mode: `cfgPath` (preferred) hoặc `CFG_PATH` (legacy), fallback `./config/config-local.yml`. Detect absolute/file-suffix `.yml/.yaml` → `SetConfigFile`; else → `SetConfigName` + `AddConfigPath` 3 paths (./config, config, .).
   - Giữ nguyên `SetEnvKeyReplacer(".", "_")` + `AutomaticEnv()` (KHÔNG set prefix vì env vars hiện hữu không có prefix nhất quán: `DB_SINK_URL`, `CDC_*`, `NATS_URL`, `KAFKA_*`, `OTEL_*`, `MONGODB_URL`, `SOURCE_DSN_*`).
3. KEEP `applyEnvOverrides` + `applyDBFallbacks` + `parseNamedURLs` + `mergeTopicPrefixAlias` nguyên trạng.
4. NEW `validateConfig(cfg)`:
   - `cfg.Server.Port` non-empty.
   - `cfg.SystemDB.URL` non-empty (đã pass applyDBFallbacks → derive từ DB.PgxDSN nếu DB.* set).
   - `cfg.ControlPlane.URL` non-empty (post-fallback derives từ SystemDB).
   - `cfg.MasterDB.URLs[default-key]` non-empty.
   - `cfg.JWT.Secret` non-empty.
   - Refuse `cfg.JWT.Secret == defaultJWTPlaceholder` khi `strings.EqualFold(cfg.Server.Mode, "production")`.
5. Call `validateConfig(cfg)` cuối `NewConfig` sau applyEnvOverrides.

**Constraint**:
- `cfg.Server.Mode` đã có sẵn trong struct (line 115). config-local hiện set `mode: worker`. Production sẽ set `mode: production`. Validation check production mode = case-insensitive equal "production".
- KHÔNG đổi flow khác — preserve existing behavior cho live worker.

### 2.2 NEW `config/config-production.yml`

```yaml
server:
  name: centralized-data-service
  port: ":8082"
  mode: production

# Empty fields — env override required at deploy time.
db:
  host: ""
  port: 5432
  username: ""
  password: ""
  database: ""
  sslMode: require
  url: ""
  maxOpenConn: 100
  maxIdleConn: 50
  connMaxLifetime: 5m

systemDb:
  url: ""

shadowDb:
  defaultKey: default
  urls: {}

masterDb:
  defaultKey: default
  urls: {}

controlPlane:
  url: ""

sources: {}

nats:
  url: ""
  name: cdc-worker
  maxReconnect: -1
  reconnectWait: 2s

kafka:
  enabled: true
  brokers: []
  groupId: cdc-worker-group
  topicPrefix: []
  schemaRegistryUrl: ""

redis:
  url: ""
  password: ""
  db: 0

worker:
  poolSize: 50
  batchSize: 1000
  batchTimeout: 5s
  fetchSize: 2000
  transformInterval: 5m
  scanInterval: 1h

otel:
  enabled: true
  serviceName: cdc-worker
  endpoint: ""
  sampleRatio: 0.1
  logs:
    sampleBySeverity:
      debug: 0.0
      info: 0.05
      warn: 1.0
      error: 1.0
      fatal: 1.0
    memoryLimitMib: 512
    fallback:
      degradedAfterErrors: 50
      recoverAfter: 10m

jwt:
  secret: ""
  expiration: 1h
```

### 2.3 NEW `config/config-sample.yml`

Clone `config-local.yml` 1:1 → dev/staging copy template.

### 2.4 EDIT `deployments/docker/Dockerfile.worker`

```dockerfile
FROM golang:1.26.1-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o cdc-worker ./cmd/worker/

FROM alpine:3.20
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY --from=builder /app/cdc-worker .
COPY --from=builder /app/config ./config
EXPOSE 8080
# Runtime prod: cfgPath=./config/config-production.yml + DSN/secret env injected.
CMD ["./cdc-worker"]
```

### 2.5 KEEP `.env.example`

Validate via decision tree:
- `grep godotenv` 0 hit. ✅ DELETE candidate.
- BUT compose `${VAR:-default}` actively uses 16+ vars. ✅ KEEP (compose contract).
- File serves as **compose substitution template** — NOT app-runtime contract.

→ KEEP. Note: file already follows actionable pattern (16 vars, 1-line headers).

## 3. B5.6.3 — cdc-cms-service

### 3.1 Refactor `config/config.go`

**Diff plan** (more aggressive — refactor applyEnvOverrides):

1. ADD `Mode string` field tới `ServerConfig`.
2. ADD `defaultJWTPlaceholder` const.
3. `NewConfig()`:
   - Path resolver dual-mode: `cfgPath` hoặc `CFG_PATH` legacy, fallback `./config/config-local.yml`. Same dual-detect logic.
   - `v.SetEnvPrefix("CMS")` + `SetEnvKeyReplacer(".", "_")` + `AutomaticEnv()`.
   - Explicit `BindEnv` map (map[key][]envNames):
     - `server.port`: `CMS_SERVER_PORT`
     - `db.host`: `CMS_DB_HOST`
     - `db.port`: `CMS_DB_PORT`
     - `db.username`: `CMS_DB_USERNAME`
     - `db.password`: `CMS_DB_PASSWORD`
     - `db.database`: `CMS_DB_DATABASE`
     - `db.sslMode`: `CMS_DB_SSL_MODE`
     - `db.maxOpenConn`: `CMS_DB_MAX_OPEN_CONN`
     - `db.maxIdleConn`: `CMS_DB_MAX_IDLE_CONN`
     - `db.connMaxLifetime`: `CMS_DB_CONN_MAX_LIFETIME`
     - `nats.url`: `CMS_NATS_URL`, `NATS_URL` (legacy)
     - `nats.user`: `CMS_NATS_USER`
     - `nats.pass`: `CMS_NATS_PASS`
     - `redis.url`: `CMS_REDIS_URL`, `REDIS_URL` (legacy)
     - `redis.password`: `CMS_REDIS_PASSWORD`
     - `jwt.secret`: `CMS_JWT_SECRET`, `JWT_SECRET` (legacy)
     - `jwt.expiration`: `CMS_JWT_EXPIRATION`
     - `otel.endpoint`: `CMS_OTEL_ENDPOINT`, `OTEL_EXPORTER_OTLP_ENDPOINT` (legacy)
     - `otel.serviceName`: `CMS_OTEL_SERVICE_NAME`
     - `system.workerUrl`: `CMS_SYSTEM_WORKER_URL`
     - `system.kafkaConnectUrl`: `CMS_SYSTEM_KAFKA_CONNECT_URL`
     - `system.natsMonitorUrl`: `CMS_SYSTEM_NATS_MONITOR_URL`
     - `system.prometheusUrl`: `CMS_SYSTEM_PROMETHEUS_URL`
     - `system.kafkaExporterUrl`: `CMS_SYSTEM_KAFKA_EXPORTER_URL`
     - `system.debeziumConnector`: `CMS_SYSTEM_DEBEZIUM_CONNECTOR`
4. DELETE `applyEnvOverrides` (24 lines `os.Getenv` đều cover bằng BindEnv).
5. DELETE import `strconv` (unused after delete).
6. NEW `validateConfig(cfg)`:
   - `cfg.Server.Port` non-empty.
   - `cfg.DB.Host` non-empty.
   - `cfg.DB.Database` non-empty.
   - `cfg.DB.UserName` non-empty.
   - `cfg.JWT.Secret` non-empty.
   - Refuse JWT placeholder khi production mode.

### 3.2 NEW `config/config-production.yml`

```yaml
server:
  name: cdc-cms-service
  port: ":8083"
  mode: production

db:
  host: ""
  port: 5432
  username: ""
  password: ""
  database: ""
  sslMode: require
  maxOpenConn: 50
  maxIdleConn: 25
  connMaxLifetime: 5m

nats:
  url: ""
  name: cdc-cms
  maxReconnect: -1
  reconnectWait: 2s

redis:
  url: ""
  password: ""
  db: 0

system:
  workerUrl: ""
  kafkaConnectUrl: ""
  natsMonitorUrl: ""
  prometheusUrl: ""
  kafkaExporterUrl: ""
  debeziumConnector: ""
  healthCacheKey: system_health:snapshot

jwt:
  secret: ""
  expiration: 1h

otel:
  enabled: true
  serviceName: cdc-cms
  endpoint: ""
  sampleRatio: 0.1
```

### 3.3 NEW `config/config-sample.yml`

Clone `config-local.yml` (kèm `mode: dev`).

### 3.4 EDIT `config/config-local.yml`

Add `mode: dev` dưới `port`.

### 3.5 EDIT `deployments/docker/Dockerfile`

```dockerfile
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o cms-service ./cmd/server/

FROM alpine:3.20
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY --from=builder /app/cms-service .
COPY --from=builder /app/config ./config
COPY --from=builder /app/docs ./docs
EXPOSE 8083
# Runtime prod: cfgPath=./config/config-production.yml + CMS_DB_*/CMS_JWT_SECRET via env.
CMD ["./cms-service"]
```

(EXPOSE 8080 → 8083 vì service thực bind :8083 per config.)

### 3.6 DELETE `.env.example`

Decision tree:
- `grep godotenv cdc-cms-service/` = 0 ✅
- KHÔNG có `docker-compose.yml` ở cdc-cms-service/ ✅
- `cdc-cms-service` KHÔNG xuất hiện như service trong centralized-data-service compose hay cdc-docker-dev compose ✅
- 0 docs/scripts reference `.env.example` ✅

→ DEAD WEIGHT. Delete safely.

## 4. Verification matrix

### 4.1 Unit/build
- `cd centralized-data-service && go build ./...` EXIT=0
- `cd cdc-cms-service && go build ./...` EXIT=0

### 4.2 Smoke per service (4 mỗi service)

**Smoke #1 — Default local config**:
- `<binary>` (no env)
- Expect: load `./config/config-local.yml`, DB connect, NATS connect, server start.

**Smoke #2 — Prod yml + no env**:
- `cfgPath=./config/config-production.yml <binary>`
- Expect: `failed to load config: validate config: <field> required`.

**Smoke #3 — Prod yml + full env override**:
- `cfgPath=./config/config-production.yml <full env vars> <binary>`
- Expect: DB connect, server start, `curl /health → 200`.

**Smoke #4 — Prod mode + JWT placeholder**:
- `cfgPath=./config/config-production.yml <full env except JWT default placeholder> <binary>`
- Expect: `validate config: jwt.secret must not use default placeholder in production mode`.

### 4.3 Regression — running services
Sau ALL fix, verify baseline pre-change:
- `curl /health` 8081 (auth host) → 200
- `curl /health` 8083 (cms host) → 200 (sẽ restart bằng binary mới sau B5.6.3)
- `curl /healthz` 8090 (admin) → 200
- `curl /` 5173 (FE) → 200
- 17 docker containers `Up ... (healthy)`
- `gpay-cdc-worker` log clean (last 1m)

## 5. Files matrix

| Path | Action | Service |
|------|--------|---------|
| `centralized-data-service/config/config.go` | Edit (path resolver + validateConfig) | centralized |
| `centralized-data-service/config/config-production.yml` | NEW | centralized |
| `centralized-data-service/config/config-sample.yml` | NEW | centralized |
| `centralized-data-service/deployments/docker/Dockerfile.worker` | Edit | centralized |
| `cdc-cms-service/config/config.go` | Rewrite (BindEnv + Mode + validateConfig) | cms |
| `cdc-cms-service/config/config-local.yml` | Edit (+mode: dev) | cms |
| `cdc-cms-service/config/config-production.yml` | NEW | cms |
| `cdc-cms-service/config/config-sample.yml` | NEW | cms |
| `cdc-cms-service/deployments/docker/Dockerfile` | Edit | cms |
| `cdc-cms-service/.env.example` | DELETE | cms |
| `agent/memory/workspaces/feature-system-refactor-2026-05/02_plan_remaining_services_prod_config.md` | NEW (file này) | docs |
| `agent/memory/workspaces/feature-system-refactor-2026-05/05_progress.md` | APPEND | docs |
| `agent/memory/workspaces/feature-system-refactor-2026-05/report_phase_b5_6_remaining_services_*.md` | NEW | docs |

## 6. Risk matrix + rollback

### Risk A — viper SetEnvPrefix("CMS") break legacy `JWT_SECRET` env

**Mitigation**: BindEnv với 2 names — cả `CMS_JWT_SECRET` (preferred) lẫn `JWT_SECRET` (legacy). User existing deploy không phá.

### Risk B — applyEnvOverrides removed cdc-cms-service → CMS_DB_PORT parsed sai

**Mitigation**: viper auto-handle `int` parsing qua mapstructure. Test smoke #3 với `CMS_DB_PORT=5433` → assert `cfg.DB.Port == 5433` (qua DB connect log).

### Risk C — centralized validateConfig refuse YAML hiện hữu

**Mitigation**: Test smoke #1 trên `config-local.yml` HIỆN TẠI (đã có DB.* + JWT.Secret = "change-me-in-production" + Mode = "worker"). Validation rule "refuse placeholder" CHỈ khi `mode==production` → mode "worker" PASS.

### Rollback

```bash
# Centralized
cd centralized-data-service
git checkout HEAD -- config/config.go deployments/docker/Dockerfile.worker
rm config/config-production.yml config/config-sample.yml
go build ./...

# CMS (also restore .env.example)
cd cdc-cms-service
git checkout HEAD -- config/config.go config/config-local.yml deployments/docker/Dockerfile
rm config/config-production.yml config/config-sample.yml
git checkout HEAD -- .env.example  # if still tracked, else recreate from progress diff
go build ./...
```

(Note: cả 2 service trong working tree chưa commit B5.6.x → rollback bằng `git stash` hoặc `git checkout HEAD --` về commit cuối — KHÔNG `reset --hard`.)

## 7. Execution order

Per CLAUDE.md §3 + §11 "ngay khi có vấn đề thì re-plan":

1. **B5.6.2 — centralized-data-service** (lower risk: chỉ thêm validateConfig + path resolver, KHÔNG đụng applyEnvOverrides).
   - Edit config.go.
   - NEW prod yml + sample yml.
   - Edit Dockerfile.worker.
   - go build PASS.
   - 4 smoke tests → ALL PASS.

2. **B5.6.3 — cdc-cms-service** (higher risk: refactor applyEnvOverrides → BindEnv).
   - Rewrite config.go.
   - Edit config-local.yml.
   - NEW prod yml + sample yml.
   - Edit Dockerfile.
   - DELETE .env.example.
   - go build PASS.
   - 4 smoke tests → ALL PASS.

3. **Final regression check** — 5 host services + 17 containers no regression.

4. **APPEND** 05_progress.md + write `report_phase_b5_6_remaining_services_*.md`.

5. Lessons: Đã có 2 lessons từ B5.6.1 cover pattern. CHỈ thêm lesson mới nếu phát hiện edge case (e.g. viper SetEnvPrefix với BindEnv multi-name behavior). Otherwise reinforce.

## 8. Out of scope

- Helm chart / K8s deployment manifest cho centralized + cms.
- Migration prod yml schema → Vault/SecretManager fetch (cần infra layer).
- centralized-data-service `cmd/admin-api`, `cmd/migrate`, `cmd/recon` — same config package, sẽ get fix tự động qua config.go refactor (test rằng admin-api boot vẫn OK trong regression check).

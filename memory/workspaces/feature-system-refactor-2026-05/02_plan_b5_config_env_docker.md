# 02 — Plan (Phase B5: Config-Env Extract + Docker Split)

> Cross-ref: `01_requirements_b5_config_env_docker.md`
> Execution order: smallest blast radius first → docker last (vì restart cluster).

---

## Step-order rationale

1. **B5.2 cdc-auth-service** trước — repo độc lập nhất, nếu boot fail không ảnh hưởng worker / cms.
2. **B5.3 cdc-cms-service** kế tiếp — nhỏ hơn worker, có airbyte secret cần xoá ngay.
3. **B5.4 centralized-data-service** — repo lớn nhất, đã có khung env override; chỉ thêm 2 key thiếu.
4. **B5.5 docker split** sau cùng — cần 3 service ở trên đã chấp nhận env override để worker đọc DSN từ env, không hardcode.
5. **B5.6 verify** — exercise-driven, 3 service stop/start lại với env file.
6. **B5.7 report** — sau khi mọi thứ pass.

---

## B5.2 — cdc-auth-service env overrides

### File 1: `cdc-system/cdc-auth-service/config/config.go`

Thêm vào `Load()` (hoặc tương đương) sau khi viper read YAML:

```go
// Track B5 — env override layer (anh trainguyen 2026-05-05).
// Nếu env var set, override giá trị YAML. Empty env = giữ YAML.
if v := os.Getenv("AUTH_DB_HOST"); v != "" { cfg.DB.Host = v }
if v := os.Getenv("AUTH_DB_PORT"); v != "" { cfg.DB.Port = parseIntDefault(v, cfg.DB.Port) }
if v := os.Getenv("AUTH_DB_USERNAME"); v != "" { cfg.DB.Username = v }
if v := os.Getenv("AUTH_DB_PASSWORD"); v != "" { cfg.DB.Password = v }
if v := os.Getenv("AUTH_DB_DATABASE"); v != "" { cfg.DB.Database = v }
if v := os.Getenv("AUTH_DB_SSL_MODE"); v != "" { cfg.DB.SSLMode = v }
if v := os.Getenv("AUTH_SERVER_PORT"); v != "" { cfg.Server.Port = v }
// JWT_SECRET đã có override.
```

### File 2: `cdc-system/cdc-auth-service/.env.example` (NEW)

```ini
# cdc-auth-service — Phase B5 ENV map (2026-05-05)
# Copy to .env, fill, do not commit.
AUTH_SERVER_PORT=:8081
AUTH_DB_HOST=localhost
AUTH_DB_PORT=5432
AUTH_DB_USERNAME=gpay_admin
AUTH_DB_PASSWORD=gpay_pass
AUTH_DB_DATABASE=cdc_auth
AUTH_DB_SSL_MODE=disable
JWT_SECRET=change-me-in-production
```

---

## B5.3 — cdc-cms-service env overrides + airbyte secret removal

### File 1: `cdc-system/cdc-cms-service/config/config.go`

Thêm sau viper read:
```go
if v := os.Getenv("CMS_DB_HOST"); v != "" { cfg.DB.Host = v }
if v := os.Getenv("CMS_DB_PORT"); v != "" { cfg.DB.Port = parseIntDefault(v, cfg.DB.Port) }
if v := os.Getenv("CMS_DB_USERNAME"); v != "" { cfg.DB.Username = v }
if v := os.Getenv("CMS_DB_PASSWORD"); v != "" { cfg.DB.Password = v }
if v := os.Getenv("CMS_DB_DATABASE"); v != "" { cfg.DB.Database = v }
if v := os.Getenv("CMS_DB_SSL_MODE"); v != "" { cfg.DB.SSLMode = v }
if v := os.Getenv("CMS_SERVER_PORT"); v != "" { cfg.Server.Port = v }
if v := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"); v != "" { cfg.Otel.Endpoint = v }
```

### File 2: `cdc-system/cdc-cms-service/config/config-local.yml`

- Xoá block `airbyte:` (4 dòng có credential).
- Xoá block `controlPlane:` + `destination:` (dead config — struct AppConfig không bind).
- Comment 1 dòng giải thích đã chuyển sang env.

### File 3: `cdc-system/cdc-cms-service/.env.example` (NEW)

```ini
# cdc-cms-service — Phase B5 ENV map (2026-05-05)
CMS_SERVER_PORT=:8083
CMS_DB_HOST=localhost
CMS_DB_PORT=5433
CMS_DB_USERNAME=gpay_admin
CMS_DB_PASSWORD=gpay_pass
CMS_DB_DATABASE=cdc_dw
CMS_DB_SSL_MODE=disable
NATS_URL=nats://cms_service:cms_secret_2026@localhost:14222
REDIS_URL=redis://localhost:16379
JWT_SECRET=change-me-in-production
OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:14318
```

---

## B5.4 — centralized-data-service SOURCE_DSN env override

### File 1: `cdc-system/centralized-data-service/config/config.go`

Tìm chỗ load `cfg.Sources` map (sau `viper.UnmarshalKey("sources", ...)`). Thêm:
```go
// Track B5 — env override cho từng key trong cfg.Sources.
// Convention: SOURCE_DSN_<KEY_UPPERCASE> override map[lowercase].
if v := os.Getenv("SOURCE_DSN_POSTGRES_PRIMARY"); v != "" {
    if cfg.Sources == nil { cfg.Sources = map[string]string{} }
    cfg.Sources["postgres_primary"] = v
}
if v := os.Getenv("SOURCE_DSN_MONGODB_PRIMARY"); v != "" {
    if cfg.Sources == nil { cfg.Sources = map[string]string{} }
    cfg.Sources["mongodb_primary"] = v
}
```

### File 2: `cdc-system/centralized-data-service/.env.example` (NEW hoặc UPDATE nếu đã có)

Append nếu thiếu:
```ini
# Phase B5 (2026-05-05) — source DSN explicit.
SOURCE_DSN_POSTGRES_PRIMARY=postgres://src_user:src_pass@localhost:5435/goopay_source?sslmode=disable
SOURCE_DSN_MONGODB_PRIMARY=mongodb://localhost:17017/?directConnection=true
```

---

## B5.5 — Docker split

### File 1: `cdc-system/centralized-data-service/docker-compose.yml` (EDIT)

Giữ chỉ 10 service core. Xoá:
- `gpay-postgres` (auth, port 5432)
- `gpay-postgres-source` (5435)
- `gpay-postgres-dest` (5434)
- `gpay-mongodb` (17017)
- `gpay-mysql` (13306)
- `gpay-mariadb` (13307)

Loại bỏ `depends_on` chéo:
- `cdc-worker.depends_on`: bỏ entry `gpay-postgres-dest`.
- `gpay-kafka-connect.depends_on`: bỏ entry `gpay-mongodb`.

Thêm cuối file:
```yaml
networks:
  cdc-bridge:
    external: true
```
Và set `networks: [cdc-bridge]` cho mọi service core.

Hardcoded password → env interpolation:
- `POSTGRES_PASSWORD: ${POSTGRES_PASSWORD:-gpay_pass}`
- NATS users → `${NATS_WORKER_PASSWORD:-worker_secret_2026}` etc.
- `.env` cho compose (không commit, chỉ `.env.example`).

### File 2: `cdc-system/cdc-docker-dev/docker-compose.yml` (NEW)

Đặt 6 service config-able (auth-pg, source-pg, dest-pg, mongo, mysql, mariadb). External `cdc-bridge` network. Healthchecks giữ. Volumes namespaced (`cdc-dev-source-data`, `cdc-dev-dest-data`).

### File 3: `cdc-system/cdc-docker-dev/.env.example` (NEW)
Mirror passwords cần override.

### File 4: `cdc-system/cdc-docker-dev/README.md` (NEW)
Hướng dẫn:
```
docker network create cdc-bridge   # 1 lần đầu
docker compose -f cdc-system/centralized-data-service/docker-compose.yml up -d
docker compose -f cdc-system/cdc-docker-dev/docker-compose.yml up -d
```

---

## B5.6 — Verify (exercise-driven, lesson 2026-04-28)

### Step 1: 3 services build clean
```bash
cd cdc-system/cdc-auth-service && go build ./...
cd cdc-system/cdc-cms-service && go build ./...
cd cdc-system/centralized-data-service && go build ./...
```
Expect: 3× exit code 0.

### Step 2: Docker stack up
```bash
docker network create cdc-bridge 2>/dev/null || true
docker compose -f cdc-system/centralized-data-service/docker-compose.yml up -d
docker compose -f cdc-system/cdc-docker-dev/docker-compose.yml up -d
sleep 15  # wait healthchecks
docker ps --format 'table {{.Names}}\t{{.Status}}' | grep -E 'gpay-|cdc-'
```
Expect: tất cả Up (healthy).

### Step 3: cdc-auth-service business endpoint
```bash
cd cdc-system/cdc-auth-service
cp .env.example .env  # giả lập dev clone
go run ./cmd/server &
sleep 3
curl -s -X POST http://localhost:8081/v1/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"admin","password":"admin"}'
# expect 200 hoặc 401 (đúng business response, KHÔNG 500/connection-refused)
```

### Step 4: cdc-cms-service business endpoint
```bash
cd cdc-system/cdc-cms-service
cp .env.example .env
go run ./cmd/server &
sleep 3
TOKEN=$(curl -s -X POST http://localhost:8081/v1/login -d '{...}' | jq -r .token)
curl -s http://localhost:8083/v2/sources -H "Authorization: Bearer $TOKEN"
# expect JSON list 200, KHÔNG fallback warning trong stderr.
```

### Step 5: centralized-data-service E2E
- Insert 1 row vào `goopay_source.public.orders`.
- Đợi 5s.
- `psql shadow_goopay_source.orders | count(*)` tăng đúng 1.

### Step 6: Startup log clean (lesson 2026-04-17)
3 service log đầu boot KHÔNG có:
- `WARN`, `WARNING`, `missing config`, `fallback`, `connection refused`, `lookup failed`.

---

## B5.7 — Report

File: `agent/memory/workspaces/feature-system-refactor-2026-05/report_phase_b5_config_env_docker_<TIMESTAMP>.md`

Sections bắt buộc:
1. Summary (1 đoạn).
2. Files changed (table path | action).
3. Diff highlights (env override snippets).
4. Verify evidence (4 step output thực).
5. Known issues / out-of-scope.

APPEND vào `05_progress.md`:
```
## 2026-05-05 <HH:MM>+07 — Phase B5 DONE
- Config env extract: 3 repos cập nhật (auth/cms/centralized).
- Docker split: 6 services moved → cdc-docker-dev/.
- Airbyte secret removed from cms-service config-local.yml.
- Verify: 3× go build PASS, 6× docker healthy, business endpoints OK.
- Report: report_phase_b5_config_env_docker_<TS>.md
```

---

## Mapping → TaskList (#115–#121)

| TaskID | Subject | Doc Section |
|---|---|---|
| 115 | Phase docs B5 | THIS FILE (in_progress) |
| 116 | cdc-auth-service env + .env.example | B5.2 |
| 117 | cdc-cms-service env + airbyte purge | B5.3 |
| 118 | centralized-data-service SOURCE_DSN env | B5.4 |
| 119 | Docker split | B5.5 |
| 120 | Verify exercise-driven | B5.6 |
| 121 | Report + APPEND 05_progress | B5.7 |

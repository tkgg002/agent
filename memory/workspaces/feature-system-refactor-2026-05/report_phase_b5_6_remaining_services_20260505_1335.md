# Report — Phase B5.6.2 + B5.6.3 — centralized-data-service & cdc-cms-service prod-config alignment

> **Date**: 2026-05-05 13:35 ICT
> **Workspace**: feature-system-refactor-2026-05
> **Plan**: 02_plan_remaining_services_prod_config.md
> **Reference pattern**: B5.6.1 cdc-auth-service (report_phase_b5_6_auth_prod_config_20260505_1158.md)
> **Owner**: Muscle (Claude Code Opus 4.7)

## Tóm tắt thực thi

Áp dụng pattern prod-ready từ B5.6.1 (đã verify trên cdc-auth-service) sang 2 service còn lại trong cdc-system:

- **B5.6.2 — centralized-data-service**: dual-mode path resolver + `validateConfig` + 3 yml (local/production/sample) + Dockerfile.worker copy folder. **GIỮ** `applyEnvOverrides` vì có parsing logic phức tạp (parseNamedURLs JSON/semicolon, sources↔mongodb bridge, DB_SINK_URL side-effect).
- **B5.6.3 — cdc-cms-service**: dual-mode path resolver + **REFACTOR** `applyEnvOverrides` → BindEnv pattern (giống cdc-auth-service) + ADD `Mode` field tới `ServerConfig` + `validateConfig` + 3 yml + Dockerfile copy folder. **DELETE** dead `.env.example` (no godotenv, no compose, no docs ref).

ALL 8 smoke tests (4 mỗi service) PASS. Existing services không regression.

## Files thay đổi

### centralized-data-service (B5.6.2)

| Path | Action | Note |
|------|--------|------|
| `centralized-data-service/config/config.go` | Edit | path resolver dual-mode, validateConfig 5 rules, sequence change `applyEnvOverrides → validateConfig → applyDBFallbacks` (validation sees pre-fallback state) |
| `centralized-data-service/config/config-production.yml` | NEW | Prod tunables (pool 100/50, sslMode require, otel sampleRatio 0.1), all DSN/secret rỗng |
| `centralized-data-service/config/config-sample.yml` | NEW | Clone local cho dev/staging template |
| `centralized-data-service/deployments/docker/Dockerfile.worker` | Edit | `COPY config ./config` (cả folder); EXPOSE :8082 (was :8080); runtime hint comment |

### cdc-cms-service (B5.6.3)

| Path | Action | Note |
|------|--------|------|
| `cdc-cms-service/config/config.go` | Rewrite (149→196 lines) | Add `Mode` field, BindEnv pattern 30 keys (CMS_*) + legacy backwards-compat (NATS_URL/REDIS_URL/JWT_SECRET/OTEL_EXPORTER_OTLP_ENDPOINT), validateConfig 6 rules |
| `cdc-cms-service/config/config-local.yml` | Edit | Add `mode: dev` |
| `cdc-cms-service/config/config-production.yml` | NEW | Prod tunables (pool 50/25, sslMode require, otel sampleRatio 0.1), DSN/secret rỗng |
| `cdc-cms-service/config/config-sample.yml` | NEW | Clone local |
| `cdc-cms-service/deployments/docker/Dockerfile` | Edit | `COPY config ./config`; EXPOSE :8083 (was :8080); runtime hint comment |
| `cdc-cms-service/.env.example` | DELETE | Dead weight (decision tree confirmed: 0 godotenv, 0 compose, 0 docs ref) |

### Workspace

| Path | Action |
|------|--------|
| `02_plan_remaining_services_prod_config.md` | NEW (plan trước thi công) |
| `report_phase_b5_6_remaining_services_20260505_1335.md` | NEW (file này) |
| `05_progress.md` | APPEND (sau report) |

## Verification thực tế (chạy thật)

### Build

```bash
$ cd centralized-data-service && go build ./...
EXIT=0
$ go build -o /tmp/cdc-worker-smoke ./cmd/worker/
50556946 bytes (50 MB) binary

$ cd cdc-cms-service && go build ./...
EXIT=0
$ go build -o /tmp/cms-smoke ./cmd/server
57519426 bytes (57 MB) binary
```

### Smoke matrix — centralized-data-service

**#1 — Default local config (no env)**:
```
2026/05/05 13:30:56 config path: ./config/config-local.yml
{"msg":"starting CDC Worker","port":":8082"}
{"msg":"PostgreSQL connected (multi-pg registry)","control_plane":"postgres://...:5433/cdc_dw","destination":"postgres://...:5434/goopay_dest"}
{"msg":"NATS JetStream connected"}
{"msg":"Redis connected"}
{"msg":"V2 metadata registry reloaded","sources":7,"connections":8,"shadow_bindings":8,"legacy_mapping_rules":27}
{"msg":"MongoDB connected"}
{"msg":"transmute scheduler started"}
{"msg":"job monitor registered"}
{"msg":"CDC Worker started","port":":8082"}
```
✅ Config load + DB connect (multi-pg: control plane + destination) + NATS + Redis + MongoDB + V2 metadata reload + scheduler + JobMonitor.

**#2 — Prod config WITHOUT env → expect FAIL**:
```
2026/05/05 13:31:09 config path: ./config/config-production.yml
2026/05/05 13:31:09 failed to load config: validate config: DB connection required (set db.host+db.database, db.url, systemDb.url, or env CDC_SYSTEM_DB_URL/DB_SINK_URL)
```
✅ validateConfig refuse rỗng đúng kỳ vọng. **Critical**: validation chạy BEFORE applyDBFallbacks → garbage DSN từ `cfg.DB.PgxDSN()` literal-non-empty không slip past.

**#3 — Prod config + full env → boot E2E**:
```
$ JWT_SECRET=$(openssl rand -hex 32) cfgPath=./config/config-production.yml \
    SERVER_PORT=":19998" \
    CDC_SYSTEM_DB_URL=postgres://...:5433/cdc_dw \
    CDC_SHADOW_DB_URL=postgres://...:5436/cdc_shadow \
    CDC_MASTER_DB_URL=postgres://...:5434/goopay_dest \
    CDC_CONTROL_PLANE_URL=postgres://...:5433/cdc_dw \
    NATS_URL=nats://...:14222 REDIS_URL=redis://...:16379 \
    KAFKA_BROKERS=localhost:19092 KAFKA_SCHEMA_REGISTRY_URL=http://...:18081 KAFKA_CONNECT_URL=http://...:18083 \
    SOURCE_DSN_POSTGRES_PRIMARY=... SOURCE_DSN_MONGODB_PRIMARY=... MONGODB_URL=... \
    OTEL_ENDPOINT=http://localhost:14318 \
    /tmp/cdc-worker-smoke

2026/05/05 13:31:38 config path: ./config/config-production.yml
{"msg":"starting CDC Worker","port":":19998"}
{"msg":"PostgreSQL connected (multi-pg registry)","control_plane":"...5433/cdc_dw","destination":"...5434/goopay_dest"}
{"msg":"NATS JetStream connected"}
{"msg":"Redis connected","addr":"localhost:16379"}
{"msg":"V2 metadata registry reloaded","sources":7,"connections":8,"shadow_bindings":8,"legacy_mapping_rules":27}
{"msg":"MongoDB connected"}
{"msg":"transmute scheduler started"}
{"msg":"job monitor registered"}
{"msg":"CDC Worker started","port":":19998"}
{"level":"error","msg":"metrics HTTP server failed","error":"listen tcp :9090: bind: address already in use"}
```
✅ Prod yml + full env → connect ALL backends + scheduler + JobMonitor. Note: `:9090 metrics` fail vì gpay-cdc-worker container đang giữ — KHÔNG phải config issue.

**#4 — Prod mode + JWT placeholder → expect REJECT**:
```
$ JWT_SECRET="change-me-in-production" cfgPath=./config/config-production.yml \
    CDC_SYSTEM_DB_URL=... CDC_MASTER_DB_URL=... /tmp/cdc-worker-smoke

2026/05/05 13:31:59 config path: ./config/config-production.yml
2026/05/05 13:31:59 failed to load config: validate config: jwt.secret must not use default placeholder in production mode
```
✅ Đóng được hole "deploy prod nhầm secret default".

### Smoke matrix — cdc-cms-service

**#1 — Default local config (no env)**:
```
2026/05/05 13:33:36 config path: ./config/config-local.yml
{"msg":"starting CMS Service","port":":8083"}
{"msg":"PostgreSQL connected"}
{"msg":"NATS JetStream connected"}
{"msg":"Redis connected"}
{"msg":"CMS Service started","port":":8083"}
{"msg":"system health collector started"}
{"msg":"audit logger started"}
```
✅ Config + DB + NATS + Redis OK. (Bind :8083 fail vì host process PID 18563 đang chiếm — KHÔNG phải config issue.)

**#2 — Prod config WITHOUT env → expect FAIL**:
```
2026/05/05 13:33:46 config path: ./config/config-production.yml
2026/05/05 13:33:46 failed to load config: validate config: db.host required (set in YAML or CMS_DB_HOST)
```
✅ Refuse rỗng đúng.

**#3 — Prod config + full env → /health 200**:
```
$ cfgPath=./config/config-production.yml \
    CMS_SERVER_PORT=:19997 \
    CMS_DB_HOST=localhost CMS_DB_PORT=5433 CMS_DB_USERNAME=gpay_admin \
    CMS_DB_PASSWORD=gpay_pass CMS_DB_DATABASE=cdc_dw CMS_DB_SSL_MODE=disable \
    CMS_NATS_URL=nats://...:14222 CMS_REDIS_URL=redis://...:16379 \
    CMS_OTEL_ENDPOINT=http://localhost:14318 \
    CMS_SYSTEM_WORKER_URL=http://localhost:8090 \
    CMS_SYSTEM_KAFKA_CONNECT_URL=http://...:18083 \
    CMS_SYSTEM_NATS_MONITOR_URL=http://...:18222 \
    CMS_SYSTEM_DEBEZIUM_CONNECTOR=goopay-mongodb-cdc \
    CMS_JWT_SECRET=$(openssl rand -hex 32) \
    /tmp/cms-smoke

{"msg":"starting CMS Service","port":":19997"}
{"msg":"PostgreSQL connected"}
{"msg":"NATS JetStream connected"}
{"msg":"Redis connected"}
{"msg":"CMS Service started","port":":19997"}

$ curl http://localhost:19997/health → 200 (1 record in stdout: "13:34:02 | 200 | 614µs | GET | /health")
```
✅ Prod yml + env → boot E2E + HTTP 200 + graceful shutdown.

**#4 — Prod mode + JWT placeholder → expect REJECT**:
```
2026/05/05 13:34:11 config path: ./config/config-production.yml
2026/05/05 13:34:11 failed to load config: validate config: jwt.secret must not use default placeholder in production mode
```
✅ Reject correct.

### Existing services không regression

```
=== HOST SERVICES (post-fix) ===
auth:8081=200 cms:8083=200 admin:8090=200 fe:5173=200

=== CONTAINERS (17 up) ===
gpay-cdc-worker        Up 2 hours
gpay-postgres-shadow   Up 3 hours (healthy)
gpay-postgres          Up 3 hours (healthy)
gpay-kafka-connect     Up 4 hours (healthy)
gpay-redpanda-console  Up 3 hours
gpay-kafka-exporter    Up 4 hours
gpay-schema-registry   Up 4 hours
gpay-postgres-cdc      Up 4 hours (healthy)
gpay-redis             Up 4 hours
gpay-kafka             Up 4 hours
gpay-nats              Up 4 hours
gpay-otel-collector    Up 4 hours
gpay-postgres-dest     Up 4 hours (healthy)
gpay-postgres-source   Up 4 hours (healthy)
gpay-mariadb           Up 4 hours (healthy)
gpay-mongo             Up 4 hours (healthy)
gpay-mysql             Up 4 hours
+ signoz stack (3 containers, healthy)

=== CDC PIPELINE LIVE (close-loop close 13:34:03) ===
SELECT id, last_status, last_run_at FROM cdc_system.transmute_schedule
 WHERE last_run_at > NOW() - INTERVAL '5 minutes';
 id | last_status |          last_run_at
----+-------------+-------------------------------
  2 | success     | 2026-05-05 06:34:03.722396+00
  3 | success     | 2026-05-05 06:34:03.722396+00
 13 | success     | 2026-05-05 06:34:03.722396+00
 14 | success     | 2026-05-05 06:34:03.722396+00
  1 | success     | 2026-05-05 06:34:03.722396+00
```
✅ 4 host services + 17 docker containers UP. Pipeline E2E close-loop tick 13:34:03 success cho 5 schedules (JobMonitor P4/D-39.A pattern alive).

## Quyết định kỹ thuật

### 1. centralized-data-service: validation BEFORE applyDBFallbacks

**Vấn đề**: `cfg.DB.PgxDSN()` luôn trả về string non-empty (từ `fmt.Sprintf("postgres://%s:%s@...")` ngay cả khi mọi field rỗng → ra `"postgres://:@:0/?sslmode="`). `applyDBFallbacks` sau đó set `cfg.SystemDB.URL = legacy` → validateConfig sẽ thấy non-empty và PASS.

**Phương án A**: validateConfig parse URL detect missing host. Phức tạp.

**Phương án B (chọn)**: Tách `applyDBFallbacks(cfg)` ra khỏi `applyEnvOverrides` (nó đã ở line cuối), chuyển vào `NewConfig`. Sequence mới: applyEnvOverrides → validateConfig (pre-fallback) → applyDBFallbacks. Validator giờ thấy đúng "user intent" — nếu user không set DB.* nào thì refuse, kể cả khi PgxDSN() có thể trả literal.

**Trade-off**: dependency order rõ ràng hơn, nhưng nếu code khác gọi `applyEnvOverrides` trực tiếp thì sẽ skip fallback — kiểm tra grep `applyEnvOverrides` 0 hit ngoài config.go → an toàn.

### 2. cdc-cms-service: REFACTOR applyEnvOverrides → BindEnv

**Tại sao khác centralized-data-service**: cdc-cms applyEnvOverrides chỉ là 11 simple `os.Getenv → cfg.X = v` mappings, KHÔNG có parsing logic đặc biệt. → safe to refactor sang BindEnv pattern (single source of truth).

**Pattern**: `v.SetEnvPrefix("CMS")` + `v.AutomaticEnv()` + `v.BindEnv(key, envName1, envName2)` map. Multi-name array cho legacy back-compat:
- `nats.url` → preferred `CMS_NATS_URL`, legacy `NATS_URL` (existing deployments)
- `redis.url` → preferred `CMS_REDIS_URL`, legacy `REDIS_URL`
- `jwt.secret` → preferred `CMS_JWT_SECRET`, legacy `JWT_SECRET`
- `otel.endpoint` → preferred `CMS_OTEL_ENDPOINT`, legacy `OTEL_EXPORTER_OTLP_ENDPOINT`

**Lợi**: Thêm field schema chỉ thêm 1 dòng vào map; viper handle Duration/int parsing tự động qua mapstructure decoder hooks.

### 3. cdc-cms-service: ADD `Mode` field tới `ServerConfig`

Centralized đã có `Mode` (line 115 cũ). cdc-cms thiếu. Phải thêm để `validateConfig` check production mode → reject JWT placeholder.

### 4. DELETE cdc-cms-service/.env.example

Decision tree (per lesson 2026-05-05):
1. `grep godotenv cdc-cms-service/` = 0 ✅
2. KHÔNG có `docker-compose.yml` ở cdc-cms-service/ ✅
3. `cdc-cms-service` KHÔNG xuất hiện như service trong centralized-data-service compose hay cdc-docker-dev compose ✅
4. 0 docs/scripts reference `.env.example` ✅

→ Dead weight, delete safely.

### 5. centralized-data-service: KEEP .env.example

Compose `docker-compose.yml` có 16+ `${VAR:-default}` references cho `CDC_SHADOW_SCHEMA_PREFIX`, DB DSNs, NATS_URL, REDIS_URL, KAFKA_*, OTEL_ENDPOINT, JWT_SECRET, SOURCE_DSN_*. File là **compose-substitution contract**, KHÔNG phải app-runtime contract. → KEEP.

### 6. EXPOSE port fix trong Dockerfile

Cũ: cả 2 Dockerfile dùng `EXPOSE 8080` (placeholder, không match runtime config).
Mới: 
- `Dockerfile.worker` → `EXPOSE 8082` (match centralized-data-service config-*.yml)
- cdc-cms `Dockerfile` → `EXPOSE 8083` (match cms config-*.yml)

EXPOSE chỉ là metadata, không enforce — nhưng đồng bộ với config tránh confuse user.

## Out of scope (chưa làm)

- Helm/K8s deployment manifest cho prod yml (ngoài repo).
- Migration prod yml secrets → Vault/SecretManager fetch (cần infra layer).
- Refactor `centralized-data-service/applyEnvOverrides` → BindEnv: phức tạp vì có `parseNamedURLs` JSON, `sources↔mongodb` bridge, side-effect `DB_SINK_URL → SSLMode=disable`. Defer hoặc keep nguyên.

## Rollback plan

```bash
# centralized-data-service
cd centralized-data-service
git checkout HEAD -- config/config.go deployments/docker/Dockerfile.worker
rm config/config-production.yml config/config-sample.yml
go build ./...

# cdc-cms-service (also restore .env.example từ git nếu đã commit, hoặc skip nếu chỉ untracked)
cd cdc-cms-service
git checkout HEAD -- config/config.go config/config-local.yml deployments/docker/Dockerfile
rm config/config-production.yml config/config-sample.yml
# .env.example: chỉ cần restore nếu đã commit; current state của repo là untracked nên bỏ
go build ./...
```

Both rollbacks return to pre-B5.6 behavior: hardcoded applyEnvOverrides, no validate, Dockerfile bake config-local.

## Pattern reinforcement (vs B5.6.1)

| Pattern | B5.6.1 (auth) | B5.6.2 (centralized) | B5.6.3 (cms) |
|---------|---------------|----------------------|--------------|
| Dual-mode path resolver (`cfgPath`/`CFG_PATH`) | ✅ | ✅ | ✅ |
| `validateConfig` 5-6 rules + production mode placeholder reject | ✅ | ✅ (5 rules) | ✅ (6 rules) |
| Dockerfile `COPY config ./config` | ✅ | ✅ | ✅ |
| Prod yml secrets rỗng | ✅ | ✅ | ✅ |
| Sample yml clone local | ✅ | ✅ | ✅ |
| BindEnv map (single source of truth) | ✅ | ❌ (kept legacy applyEnvOverrides) | ✅ |
| Validation BEFORE fallbacks | N/A | ✅ (essential — PgxDSN garbage) | N/A |
| Delete dead `.env.example` | ✅ | KEEP (compose contract) | ✅ |

## Tasks status
- #128 Refactor centralized config.go ✅
- #129 NEW prod + sample yml (centralized) ✅
- #130 Dockerfile.worker fix ✅
- #131 Build + 4 smoke (centralized) ✅
- #132 Refactor cms config.go (BindEnv pattern) ✅
- #133 NEW prod + sample yml + edit local (cms) ✅
- #134 Dockerfile fix + DELETE .env.example (cms) ✅
- #135 Build + 4 smoke (cms) ✅
- #136 Final regression check + report + progress — ĐANG GHI

Tổng: 8/8 smoke tests PASS, 0 regression trên 4 host services + 17 containers + CDC pipeline close-loop alive.

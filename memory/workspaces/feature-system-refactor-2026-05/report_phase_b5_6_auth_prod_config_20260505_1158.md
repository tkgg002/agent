# Report — Phase B5.6 — cdc-auth-service prod-config alignment

> **Date**: 2026-05-05 11:58 ICT
> **Workspace**: feature-system-refactor-2026-05
> **Plan**: 02_plan_auth_prod_config.md
> **Owner**: Muscle (Claude Code Opus 4.7)

## Tóm tắt thực thi

Align `cdc-auth-service` về pattern prod-ready của `reconcile-service`. Sửa 4 vấn đề blocker để có thể deploy lên prod sạch:

1. Path resolver chỉ accept config NAME → support cả absolute file path.
2. `applyEnvOverrides` hardcoded 8 fields → `viper.AutomaticEnv()` + explicit `BindEnv` map (single source of truth).
3. Thiếu `validateConfig()` → enforce required fields + refuse default JWT placeholder trong production mode.
4. Dockerfile copy CHỈ config-local.yml → image prod nuốt creds DEV. Đổi sang copy cả `config/`, document `cfgPath` runtime.

## Files thay đổi

| Path | Action | Note |
|------|--------|------|
| `cdc-auth-service/config/config.go` | Rewrite | Path resolver dual-mode, AutomaticEnv prefix `AUTH`, validateConfig 6 rules |
| `cdc-auth-service/config/config-local.yml` | Edit | Add `mode: dev` |
| `cdc-auth-service/config/config-production.yml` | NEW | Prod tunables: pool 50/25, sslMode require, JWT 1h/24h, fields rỗng cho secret |
| `cdc-auth-service/config/config-sample.yml` | NEW | Template clone local cho dev/staging copy |
| `cdc-auth-service/deployments/docker/Dockerfile` | Edit | `COPY config ./config` thay vì single file; comment runtime hint |
| `cdc-auth-service/.env.example` | Rewrite | Restructure theo pattern actionable (#-header + var); add `cfgPath` (commented) + `AUTH_JWT_SECRET` |
| `agent/memory/workspaces/feature-system-refactor-2026-05/02_plan_auth_prod_config.md` | NEW | Plan trước khi thi công |

## Verification thực tế (chạy thật)

### Build
```bash
$ cd cdc-auth-service && go build ./...
EXIT=0
$ go test ./...
EXIT=0  # 9 packages "no test files", không fail
```

### Smoke #1 — Local config default path
```
$ /tmp/auth-service-test
2026/05/05 11:58:00 config path: ./config/config-local.yml
{"msg":"starting Auth Service","service":"cdc-auth-service","port":":8081"}
{"msg":"PostgreSQL connected"}
{"msg":"Auth Service started","port":":8081"}
```
✅ Boot OK, DB connect, JWT load — bind ":8081" fail vì conflict (auth-service đang chạy host) — config path đã pass.

### Smoke #2 — Prod config WITHOUT env → expect FAIL
```
$ cfgPath=./config/config-production.yml /tmp/auth-service-test
2026/05/05 11:58:08 config path: ./config/config-production.yml
2026/05/05 11:58:08 failed to load config: validate config: db.host required (set in YAML or AUTH_DB_HOST)
```
✅ validateConfig refuse rỗng đúng kỳ vọng.

### Smoke #3 — Prod config + full env → boot E2E
```
$ cfgPath=./config/config-production.yml \
    AUTH_DB_HOST=localhost AUTH_DB_PORT=5432 \
    AUTH_DB_USERNAME=gpay_admin AUTH_DB_PASSWORD=gpay_pass \
    AUTH_DB_DATABASE=gpay_auth AUTH_DB_SSL_MODE=disable \
    AUTH_SERVER_PORT=:19999 \
    AUTH_JWT_SECRET=$(openssl rand -hex 32) \
    /tmp/auth-service-test

2026/05/05 11:58:37 config path: ./config/config-production.yml
{"msg":"starting Auth Service","port":":19999"}
{"msg":"PostgreSQL connected"}
{"msg":"Auth Service started","port":":19999"}

$ curl -s -o /dev/null -w "%{http_code}" http://localhost:19999/health
200
```
✅ Prod config + env override → DB connect + HTTP 200 + graceful shutdown.

### Smoke #4 — Prod mode + JWT placeholder → expect REJECT
```
$ cfgPath=./config/config-production.yml \
    AUTH_DB_HOST=localhost AUTH_DB_USERNAME=gpay_admin \
    AUTH_DB_DATABASE=gpay_auth AUTH_SERVER_PORT=:19999 \
    AUTH_JWT_SECRET=change-me-in-production \
    /tmp/auth-service-test

2026/05/05 11:58:47 failed to load config: validate config: jwt.secret must not use default placeholder in production mode
```
✅ Validation đóng được hole "deploy prod nhầm secret default".

### Existing services không bị regression
```
$ docker ps --format "..." | grep -E "auth|postgres"
gpay-postgres-shadow  Up About an hour (healthy)
gpay-postgres         Up About an hour (healthy)
gpay-postgres-cdc     Up 2 hours (healthy)
gpay-postgres-dest    Up 2 hours (healthy)
gpay-postgres-source  Up 2 hours (healthy)

$ curl -s -o /dev/null -w "%{http_code}" http://localhost:8081/health
200
```
✅ Tất cả 5 postgres + auth-service host process vẫn healthy.

## Quyết định kỹ thuật

### 1. Giữ camelCase YAML, explicit BindEnv
- Phương án A (đào sâu): rename YAML keys → snake_case → AutomaticEnv map tự nhiên.
- Phương án B (chọn): giữ camelCase (đồng bộ reconcile-service), explicit `BindEnv("db.sslMode", "AUTH_DB_SSL_MODE")` cho mỗi key.
- Lý do: YAML camelCase đẹp hơn cho Go ecosystem (đồng bộ struct field); env var snake_case là expectation user (thấy rõ trong `.env.example` hiện hữu). Không có lý do bắt user đổi mental model.

### 2. validateConfig chỉ refuse JWT placeholder khi `mode == production`
- Phương án A: refuse luôn → break dev khi YAML có `change-me-in-production`.
- Phương án B (chọn): chỉ refuse khi `server.mode == production` (case-insensitive).
- Trade-off: dev có thể quên đổi mode → false alarm. Không sao, prod CI sẽ catch.

### 3. config-production.yml để fields rỗng (KHÔNG dùng `${VAR}`)
- Phương án A: `host: ${DB_HOST}` (như reconcile-service production yml) — thực tế viper KHÔNG expand syntax này native.
- Phương án B (chọn): `host: ""` + env override qua `AUTH_DB_HOST`.
- Lý do: viper `AutomaticEnv` + `BindEnv` xử lý field rỗng đúng cách (env override điền vào). `${VAR}` chỉ hoạt động nếu có envsubst tiền xử lý — chưa có trong pipeline cdc-auth.

### 4. Dockerfile copy cả config/ folder
- Image size tăng nhỏ (~200 bytes thêm cho 2 yml files) — không đáng kể.
- Lợi: 1 image deploy được mọi env (chỉ đổi `cfgPath` runtime).
- Risk: ai cố ý hardcode secret vào yml → leak. Mitigation: prod yml để rỗng cho secret fields (rule §11 code review reject hardcoded secrets).

### 5. Backward compat env path
- `cfgPath` (reconcile-style) lẫn `CFG_PATH` (cdc-system existing) đều support — code check theo thứ tự.
- Lý do: dù muốn align reconcile, không phá deploy script hiện hữu (centralized-data + cdc-cms vẫn dùng `CFG_PATH`).

### 6. JWT_SECRET vs AUTH_JWT_SECRET
- BindEnv với 2 env names: `AUTH_JWT_SECRET` (preferred) + `JWT_SECRET` (legacy backwards-compat).
- Lý do: `.env.example` cũ dùng `JWT_SECRET` không prefix — đột ngột break nếu chỉ accept `AUTH_JWT_SECRET`.

## Out of scope (chưa làm)

- `centralized-data-service/config/config.go` & `cdc-cms-service/config/config.go`: cùng vấn đề `CFG_PATH` + có thể thiếu prod yml. Chưa user yêu cầu.
- Helm chart / K8s deployment manifest: out of repo.
- Migration của `JWT_SECRET` → deprecate hoàn toàn: cần thông báo + update mọi nơi consume → deferred.

## Rollback plan

Nếu phát hiện regression:
```bash
cd cdc-auth-service
git checkout HEAD~1 -- config/config.go config/config-local.yml deployments/docker/Dockerfile .env.example
rm config/config-production.yml config/config-sample.yml
go build ./...
```
Quay về behavior cũ: applyEnvOverrides hardcoded, không có prod yml, Dockerfile bake config-local.

## Tasks status
- #123 Refactor config.go ✅
- #124 Tạo prod + sample yml ✅
- #125 Sửa Dockerfile ✅
- #126 Build + 4 smoke tests ✅
- #127 Append progress + report (file này) — đang ghi

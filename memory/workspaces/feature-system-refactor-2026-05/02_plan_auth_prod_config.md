# 02_plan_auth_prod_config — cdc-auth-service prod-readiness alignment

> **Phase**: B5.6 (sub-phase trong feature-system-refactor-2026-05)
> **Trigger**: User flag "auth-service feels like ko lên prod được"
> **Reference**: reconcile-service đã prod-ready, dùng làm pattern.
> **Date**: 2026-05-05
> **Owner**: Muscle (CC CLI)

## Diagnosis (dựa trên thực tế file đã đọc)

### cdc-auth-service hiện trạng

| File | Issue |
|------|-------|
| `config/config.go` | Path resolver chỉ accept config-NAME (`./config/config-local`), không support absolute file path; `applyEnvOverrides` hardcoded list 8 fields → thêm field mới phải sửa code; KHÔNG có `validateConfig()` → boot zombie state nếu thiếu |
| `config/` | Chỉ có `config-local.yml` — KHÔNG có `config-production.yml`, `config-sample.yml` |
| `deployments/docker/Dockerfile:11` | `COPY --from=builder /app/config/config-local.yml ./config/config-local.yml` → image prod nuốt CREDS DEV + JWT secret `change-me-in-production` + pool 10/5 |
| `cmd/server/main.go` | OK — chỉ gọi `config.NewConfig()`, không lock path |

### reconcile-service (reference)

| File | Pattern |
|------|---------|
| `config/config.go` | Path resolver dual-mode (file path / config name + multi search paths); `v.AutomaticEnv()` + `SetEnvKeyReplacer(".", "__")` → env binding theo path key; `validateConfig()` required fields |
| `config/` | `config-local.yml`, `config-production.yml`, `config-sample.yml` đầy đủ |
| `Dockerfile:25` | `COPY --from=builder /app .` → copy cả repo, prod set `cfgPath=/path/to/config-production.yml` |

### Gap thực tế (3 cái cần đóng)

1. **Image prod không deploy được sạch** — chỉ có config-local trong image; muốn override thì user phải mount volume hoặc rebuild image cho từng env.
2. **Hardcoded env override list** — config schema mở rộng phải sửa Go code.
3. **Không validate** — boot lên với JWT_SECRET=`change-me-in-production` sẽ trông như prod runtime nhưng JWT signed bằng secret công khai.

## Plan

### B5.6.1 — Refactor `config/config.go`

- Path resolver: support cả absolute file path (`/etc/cdc-auth/config.yml`) lẫn config name (`config-production`) với multi search paths (`./config`, `config`, `.`).
- Đổi env path key từ `CFG_PATH` → `cfgPath` (match reconcile convention) — giữ tương thích bằng cách check cả 2.
- Switch `applyEnvOverrides` hardcoded → `viper.AutomaticEnv()` + `SetEnvPrefix("AUTH")` + `SetEnvKeyReplacer(".", "_")` → cover MỌI field tự động qua naming convention `AUTH_<KEY>` (e.g., `AUTH_DB_HOST`, `AUTH_JWT_SECRET`).
- `validateConfig()` enforce required: `server.port`, `db.host`, `db.database`, `jwt.secret` ≠ `"change-me-in-production"` ngoài mode `dev`.
- Bonus: thêm `wire.NewSet(NewConfig)` nếu service dùng wire (check `internal/`).

### B5.6.2 — Tạo `config/config-production.yml` + `config-sample.yml`

**`config-production.yml`** (prod-tunables, KHÔNG có secret):
```yml
server:
  name: cdc-auth-service
  port: ":8081"

db:
  host: ""           # AUTH_DB_HOST override
  port: 5432
  username: ""       # AUTH_DB_USERNAME override
  password: ""       # AUTH_DB_PASSWORD override
  database: ""       # AUTH_DB_DATABASE override
  sslMode: require
  maxOpenConn: 50
  maxIdleConn: 25
  connMaxLifetime: 5m

jwt:
  secret: ""           # AUTH_JWT_SECRET override (BẮT BUỘC, validateConfig refuse empty + default)
  accessExpiration: 1h
  refreshExpiration: 24h
```

**Lý do field rỗng (không dùng `${VAR}`)**: viper KHÔNG expand `${VAR}` syntax native. Để rỗng + dùng env override là pattern hoạt động thực sự với `AutomaticEnv`.

**`config-sample.yml`**: clone của local + comment hint cho từng field → dev/QA copy sang config-local.yml hoặc config-staging.yml.

### B5.6.3 — Sửa `deployments/docker/Dockerfile`

- `COPY --from=builder /app/config/config-local.yml ./config/config-local.yml` → `COPY --from=builder /app/config ./config`
- Comment runtime: deploy prod set env `cfgPath=./config/config-production.yml` + secret env vars.

### B5.6.4 — Verify

1. `cd cdc-auth-service && go build ./...` → 0 error.
2. `go test ./...` → existing tests PASS.
3. Smoke local: `cfgPath=./config/config-local.yml ./bin/auth-service` → boot OK với DEV creds.
4. Smoke prod-sim: `cfgPath=./config/config-production.yml AUTH_DB_HOST=localhost AUTH_DB_USERNAME=gpay_admin AUTH_DB_PASSWORD=gpay_pass AUTH_DB_DATABASE=gpay_auth AUTH_JWT_SECRET=$(openssl rand -hex 32) ./bin/auth-service` → boot OK, validate fail nếu thiếu secret.
5. Container vẫn live: `docker exec gpay-postgres psql ...` (đã verify trước, nên dùng healthcheck endpoint nếu có).

### B5.6.5 — Document

- Append `05_progress.md` (rule §11 APPEND-only).
- Tạo `report_phase_b5_6_auth_prod_config_<timestamp>.md`.

## Out of scope

- Reconcile-service (đã làm).
- centralized-data-service config (chưa user yêu cầu).
- K8s helm chart / envsubst pipeline (out of repo scope).
- Wire codegen nếu service auth chưa có (check riêng).

## Risk / Trade-off

| Risk | Mitigation |
|------|-----------|
| `AutomaticEnv` strip prefix → đổi naming env vars | Giữ prefix `AUTH_` qua `SetEnvPrefix("AUTH")` → vẫn dùng `AUTH_DB_HOST` etc., không đổi `.env.example` |
| Bake config/ vào image → secret leak nếu ai cố ý hardcode | Prod yml để rỗng cho secret fields; ai nhét hardcoded secret vào prod yml = code review reject |
| Validate refuse `"change-me-in-production"` mode prod → break dev nếu dev chạy với CFG=prod | Validate chỉ refuse khi `server.mode != dev` (thêm field `mode`) HOẶC chỉ refuse rỗng — đơn giản hơn |

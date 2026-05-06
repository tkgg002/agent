# 09 — Tasks Solution (Phase B5)

> Filled-in version. Mỗi section = 1 task, có diff/output thực tế.

---

## B5.1 — Phase docs

**Status**: ✅ DONE 2026-05-05 09:50+07.
**Files created**:
- `01_requirements_b5_config_env_docker.md` (160 lines)
- `02_plan_b5_config_env_docker.md` (180 lines)
- `08_tasks_b5_config_env_docker.md` (40 lines)
- `09_tasks_solution_b5_config_env_docker.md` (this — re-written sau khi exec xong)

---

## B5.2 — cdc-auth-service env overrides

**Status**: ✅ DONE 2026-05-05 09:55+07.
**Files**:
- `config/config.go` EDIT +`strconv` import + `applyEnvOverrides()` 30 lines.
- `.env.example` NEW (8 keys).

**Build**: `cd cdc-auth-service && go build ./...` → EXIT=0.

**Smoke (env override apply)**:
```
$ AUTH_DB_HOST=envhost AUTH_DB_PORT=9999 AUTH_DB_PASSWORD=envpass \
  AUTH_SERVER_PORT=:18081 go run /tmp/check.go
host=envhost port=9999 user=gpay_admin pass=envpass db=gpay_auth ssl=disable server=:18081
EXIT=0
```
→ Override applied (host, port, pass, server). Fallback YAML cho user/db/ssl không set env.

---

## B5.3 — cdc-cms-service env + airbyte secret purge

**Status**: ✅ DONE 2026-05-05 10:00+07.
**Files**:
- `config/config.go` EDIT +`strconv` import + `applyEnvOverrides()` 35 lines (11 env).
- `config/config-local.yml` EDIT — xoá block `airbyte:` (real key `trai.nguyen@goopay.vn:knF1jhaPIShkduykN301X1rPbqOzhfe4`), `controlPlane:`, `destination:` (dead config — AppConfig không bind).
- `.env.example` NEW (11 keys).

**Verify dead config an toàn xoá**:
```
$ grep -rn 'airbyte\|controlPlane\|destination' cdc-cms-service --include='*.go'
# Chỉ match ở repository_*.go (DB columns: airbyte_destination_id, airbyte_connection_id) — KHÔNG có viper.Get("airbyte.*"). Safe.
```

**Build**: EXIT=0.

**Grep credential sau xoá**:
```
$ grep -i 'knF1jhaPIShkduykN' cdc-cms-service/config/config-local.yml
(no match)
```

**Smoke**:
```
$ CMS_DB_HOST=envcmshost CMS_DB_PORT=9988 CMS_DB_PASSWORD=envcmspass \
  CMS_SERVER_PORT=:18083 OTEL_EXPORTER_OTLP_ENDPOINT=http://envotel:4318 go run /tmp/check.go
server=:18083 db.host=envcmshost db.port=9988 db.user=gpay_admin db.pass=envcmspass db.db=cdc_dw otel=http://envotel:4318
EXIT=0
```

---

## B5.4 — centralized-data-service SOURCE_DSN

**Status**: ✅ DONE 2026-05-05 10:02+07.
**Files**:
- `config/config.go` EDIT +18 lines: 2 env override `SOURCE_DSN_POSTGRES_PRIMARY` + `SOURCE_DSN_MONGODB_PRIMARY`, đặt TRƯỚC `applyDBFallbacks(cfg)` để fallback bridge (sources → MongoDB.URL legacy alias) thấy giá trị mới.
- `.env.example` NEW (17 keys: full DSN matrix + Kafka/NATS/Redis/OTel).

**Build**: EXIT=0.

**Smoke**:
```
$ SOURCE_DSN_POSTGRES_PRIMARY="postgres://envuser:envpass@envhost:9999/envdb?sslmode=disable" \
  SOURCE_DSN_MONGODB_PRIMARY="mongodb://envmongo:27018/" go run /tmp/check.go
postgres_primary=postgres://envuser:envpass@envhost:9999/envdb?sslmode=disable
mongodb_primary=mongodb://envmongo:27018/
MongoDB.URL=mongodb://envmongo:27018/
EXIT=0
```
→ 2 source DSN override applied. Legacy alias `MongoDB.URL` cũng hydrated đúng.

---

## B5.5 — Docker split

**Status**: ✅ DONE 2026-05-05 10:04+07.
**Files**:
- `centralized-data-service/docker-compose.yml` REWRITE (369→188 lines): chỉ 10 core service + env interpolation `${VAR:-default}` + external `cdc-bridge` network. Loại `depends_on: postgres-dest` khỏi cdc-worker, `depends_on: mongodb` khỏi kafka-connect.
- `cdc-docker-dev/docker-compose.yml` NEW (135 lines): 6 dev DB (postgres-auth, postgres-source, postgres-dest, mongo, mysql, mariadb). External volumes `name: centralized-data-service_*` để preserve data 6 ngày test.
- `cdc-docker-dev/.env.example` NEW (13 keys).
- `cdc-docker-dev/README.md` NEW (bootstrap + up/down hướng dẫn).

**Migration steps thực hiện**:
1. `docker network create cdc-bridge` → ID 94b6baafef4a.
2. `docker compose down` (centralized-data-service project, no -v) → 14 containers stopped & removed (volumes preserved).
3. `docker stop/rm` 5 dev DBs orphaned (mariadb, mongo, postgres, postgres-source, postgres-dest).
4. `docker compose -f cdc-docker-dev/docker-compose.yml up -d` → 6 dev DBs started, external volumes mount existing data.
5. `docker compose -f centralized-data-service/docker-compose.yml up -d` → 10 core services started.

**Lint**:
```
$ docker compose -f centralized-data-service/docker-compose.yml config --quiet
EXIT=0
$ docker compose -f cdc-docker-dev/docker-compose.yml config --quiet
EXIT=0
```

---

## B5.6 — Verify exercise-driven

**Status**: ✅ DONE 2026-05-05 10:07+07.

**Container state (16 services)**:
```
gpay-postgres-cdc       Up About a minute (healthy)
gpay-redis              Up About a minute
gpay-nats               Up About a minute
gpay-kafka              Up About a minute
gpay-schema-registry    Up About a minute
gpay-kafka-connect      Up About a minute (healthy)
gpay-kafka-exporter     Up 27 seconds
gpay-redpanda-console   Up About a minute
gpay-otel-collector     Up About a minute
gpay-cdc-worker         Up About a minute
gpay-postgres-dest      Up About a minute (healthy)
gpay-postgres-source    Up About a minute (healthy)
gpay-postgres           Up About a minute (healthy)
gpay-mariadb            Up About a minute (healthy)
gpay-mongo              Up About a minute (healthy)
gpay-mysql              Up About a minute
```

**Cross-bridge DNS**:
```
$ docker exec gpay-cdc-worker getent hosts gpay-postgres-dest
172.26.0.2  gpay-postgres-dest
$ docker exec gpay-kafka-connect getent hosts gpay-postgres-source
172.26.0.5  gpay-postgres-source
```

**E2E pipeline (Debezium → shadow → master)**:
```
Before: source=65 shadow=18 master=37
INSERT 3 rows → wait 10s
After:  source=68 shadow=21 master=40
Delta: +3 / +3 / +3
```

**Track D close-loop (Bug #2 P4 fix vẫn chạy)**:
```
 id | last_status | last_error |          updated_at
----+-------------+------------+-------------------------------
 14 | success     |            | 2026-05-05 03:07:03.770778+00
 13 | success     |            | 2026-05-05 03:07:03.766996+00
 15 | success     |            | 2026-05-05 03:07:03.762283+00
  3 | success     |            | 2026-05-05 03:07:03.746115+00
  2 | success     |            | 2026-05-05 03:07:03.741244+00
```

**Worker log clean (last 1m)**:
```
$ docker logs --since 1m gpay-cdc-worker | grep -E 'level":"(warn|error|fatal)'
(0 lines)
```

**Host services healthz**:
```
auth :8081  → 200
cms  :8083  → 200
admin :8090 → 200
```

**Pre-existing issue ghi nhận**:
- Connector `cdc-mariadb-source` FAILED do `debezium/debezium-connector-mysql:2.5.4` không có trên Confluent Hub. Plugin missing → connector load fail. Issue tồn tại trước B5, em copy install command verbatim từ compose cũ. PG + Mongo connector vẫn RUNNING.

---

## B5.7 — Report + APPEND 05_progress

**Status**: ✅ DONE 2026-05-05 10:07+07.
**Files**:
- `report_phase_b5_config_env_docker_20260505_1007.md` NEW (~270 lines).
- `05_progress.md` APPEND +40 lines (rule §11 no overwrite — 166→206 lines).

---

## B5.8 — Delete workspace sai (lesson 2026-04-29)

**Status**: ✅ DONE 2026-05-05 09:48+07.

```
$ rm -rf agent/memory/workspaces/feature-config-env-extract-2026-05/
$ ls agent/memory/workspaces/ | grep -c feature-config-env-extract
0
```

# Report — Phase B5: Config-Env Extract + Docker Split

> **Workspace**: `feature-system-refactor-2026-05`
> **Phase**: B5 (sau B3 hardening)
> **Date**: 2026-05-05 10:07+07
> **Owner**: Muscle (CC CLI / Claude Code)
> **Trigger**: User trainguyen — 2-task assignment 09:30+07
> **Status**: ✅ DONE — 16 containers up, E2E ingest +3 rows landed, 5 schedules close-loop=success.

---

## 1. Tóm tắt

Phase B5 hoàn thành 2 nhiệm vụ user giao:

1. **Endpoint & Secret → ENV** (3 repos): thêm tầng env override vào `config.go` của cả 3 repo Go (auth/cms/centralized) + tạo `.env.example` đầy đủ key. Đặc biệt **xoá hard credential `airbyte.apiKey: trai.nguyen@goopay.vn:knF1jhaPIShkduykN301X1rPbqOzhfe4`** khỏi `cdc-cms-service/config/config-local.yml` (block đã là dead config — AppConfig không bind, nhưng vẫn nguy cơ leak qua git tracked).

2. **Docker split**: tách `centralized-data-service/docker-compose.yml` (16 services trộn lẫn) thành 2 compose:
   - **Core CDC infra** (10 services giữ nguyên đường dẫn): NATS, postgres-cdc control plane, Redis, Kafka, schema-registry, kafka-connect, kafka-exporter, redpanda-console, otel-collector, cdc-worker.
   - **cdc-docker-dev/docker-compose.yml** (6 services config-able): postgres-auth, postgres-source, postgres-dest, mongo, mysql, mariadb.
   - 2 compose dùng external network `cdc-bridge`. Volumes external referencing existing names → **không mất data 6 ngày test**.

---

## 2. Files thay đổi / tạo mới

### 2.1 Code edits

| File | Action | Mục đích |
|---|---|---|
| `cdc-system/cdc-auth-service/config/config.go` | EDIT (+1 import, +30 lines) | Thêm `applyEnvOverrides()` cho 7 env: AUTH_DB_HOST/PORT/USERNAME/PASSWORD/DATABASE/SSL_MODE + AUTH_SERVER_PORT |
| `cdc-system/cdc-cms-service/config/config.go` | EDIT (+1 import, +35 lines) | Thêm `applyEnvOverrides()` cho 11 env: CMS_DB_*, CMS_SERVER_PORT, NATS_URL, REDIS_URL, JWT_SECRET, OTEL_EXPORTER_OTLP_ENDPOINT |
| `cdc-system/cdc-cms-service/config/config-local.yml` | EDIT (-9 lines, +9 comment lines) | Xoá block `airbyte:` (chứa real secret), `controlPlane:`, `destination:` (dead config — struct không bind) |
| `cdc-system/centralized-data-service/config/config.go` | EDIT (+18 lines) | Thêm 2 env: `SOURCE_DSN_POSTGRES_PRIMARY`, `SOURCE_DSN_MONGODB_PRIMARY` (đặt trước `applyDBFallbacks` để fallback bridge thấy giá trị mới) |

### 2.2 Files mới

| File | Vai trò |
|---|---|
| `cdc-system/cdc-auth-service/.env.example` | 8 env keys cho auth-service |
| `cdc-system/cdc-cms-service/.env.example` | 11 env keys cho cms-service |
| `cdc-system/centralized-data-service/.env.example` | 17 env keys (DSN tổng hợp) |
| `cdc-system/cdc-docker-dev/docker-compose.yml` | 6 dev DBs với external volumes (data preservation) |
| `cdc-system/cdc-docker-dev/.env.example` | 13 env keys cho dev DBs |
| `cdc-system/cdc-docker-dev/README.md` | Hướng dẫn bootstrap + up/down |

### 2.3 Files rewritten

| File | Thay đổi |
|---|---|
| `cdc-system/centralized-data-service/docker-compose.yml` | Rewrite từ 369 → 188 lines. 10 service core với env interpolation `${VAR:-default}`. Loại `depends_on` chéo (worker không depend dest, kafka-connect không depend mongo). Network → `cdc-bridge` external. |

### 2.4 Workspace docs

| File | Action |
|---|---|
| `01_requirements_b5_config_env_docker.md` | NEW — inventory + DoD |
| `02_plan_b5_config_env_docker.md` | NEW — exec plan + diff snippets |
| `08_tasks_b5_config_env_docker.md` | NEW — TaskList #115-#122 mirror |
| `09_tasks_solution_b5_config_env_docker.md` | NEW — section per task (filled-in) |
| `report_phase_b5_config_env_docker_20260505_1007.md` | NEW — this file |
| `05_progress.md` | APPEND (phía dưới) |

### 2.5 Workspace bị xoá (lỗi cũ)

| Path | Lý do xoá |
|---|---|
| `agent/memory/workspaces/feature-config-env-extract-2026-05/` | Vi phạm lesson 2026-04-29 "Phase ≠ Workspace mới". Chuyển doc về parent workspace. |

---

## 3. Diff highlights

### 3.1 cdc-auth-service/config/config.go (line 62-95)

```diff
+import (
+    "os"
+    "strconv"  // NEW
+    ...
+)
...
-    if v := os.Getenv("JWT_SECRET"); v != "" {
-        cfg.JWT.Secret = v
-    }
+    applyEnvOverrides(cfg)
     return cfg, nil
 }
+
+// applyEnvOverrides — Phase B5 (2026-05-05): tách secret/endpoint khỏi
+// config-local.yml. Env var rỗng = giữ giá trị YAML.
+func applyEnvOverrides(cfg *AppConfig) {
+    if v := os.Getenv("AUTH_SERVER_PORT"); v != "" { cfg.Server.Port = v }
+    if v := os.Getenv("AUTH_DB_HOST"); v != "" { cfg.DB.Host = v }
+    if v := os.Getenv("AUTH_DB_PORT"); v != "" {
+        if p, err := strconv.Atoi(v); err == nil { cfg.DB.Port = p }
+    }
+    if v := os.Getenv("AUTH_DB_USERNAME"); v != "" { cfg.DB.UserName = v }
+    if v := os.Getenv("AUTH_DB_PASSWORD"); v != "" { cfg.DB.Password = v }
+    if v := os.Getenv("AUTH_DB_DATABASE"); v != "" { cfg.DB.Database = v }
+    if v := os.Getenv("AUTH_DB_SSL_MODE"); v != "" { cfg.DB.SSLMode = v }
+    if v := os.Getenv("JWT_SECRET"); v != "" { cfg.JWT.Secret = v }
+}
```

### 3.2 cdc-cms-service/config/config-local.yml (security fix)

```diff
-airbyte:
-  apiUrl: http://localhost:18000/api
-  apiKey: "trai.nguyen@goopay.vn:knF1jhaPIShkduykN301X1rPbqOzhfe4"
-  workspaceId: "ece70fcd-015f-419a-883c-e411e9fbd439"
-  syncInterval: 5m
-
-controlPlane:
-  url: postgres://gpay_admin:gpay_pass@localhost:5433/cdc_dw?sslmode=disable
-destination:
-  url: postgres://gpay_admin:gpay_pass@localhost:5434/goopay_dest?sslmode=disable
+# Phase B5 (2026-05-05): block `airbyte` đã xoá. AppConfig không bind...
+# Real credential xoá khỏi git tree (lịch sử git log vẫn còn — purge
+# bằng git filter-repo là task riêng).
```

⚠️ **Lưu ý git history**: API key cũ vẫn nằm trong commit history. Để xoá hoàn toàn cần `git filter-repo --replace-text` hoặc `git filter-branch` — task riêng, anh chốt khi muốn.

### 3.3 centralized-data-service/config/config.go (line 370-397)

```diff
     if v := os.Getenv("MONGODB_URL"); v != "" {
         cfg.MongoDB.URL = strings.TrimSpace(v)
         ...
     }
+    // Phase B5 (2026-05-05) — explicit SOURCE_DSN_<KEY> env override
+    if v := os.Getenv("SOURCE_DSN_POSTGRES_PRIMARY"); v != "" {
+        if cfg.Sources == nil { cfg.Sources = make(map[string]string) }
+        cfg.Sources["postgres_primary"] = strings.TrimSpace(v)
+    }
+    if v := os.Getenv("SOURCE_DSN_MONGODB_PRIMARY"); v != "" {
+        if cfg.Sources == nil { cfg.Sources = make(map[string]string) }
+        cfg.Sources["mongodb_primary"] = strings.TrimSpace(v)
+        if strings.TrimSpace(cfg.MongoDB.URL) == "" {
+            cfg.MongoDB.URL = strings.TrimSpace(v)
+        }
+    }
     applyDBFallbacks(cfg)
```

### 3.4 docker-compose.yml split (key removal — mongo/mysql/mariadb/3pg)

Trước: 16 services trong 1 compose, network `cdc-network` internal.
Sau:
- Core 10 services trong `centralized-data-service/docker-compose.yml` (network `cdc-bridge` external).
- Dev 6 services trong `cdc-docker-dev/docker-compose.yml` (cùng `cdc-bridge`, external volumes trỏ tới `centralized-data-service_*`).

Cross-deps đã loại:
- `cdc-worker.depends_on` chỉ còn `[nats, postgres-cdc, redis]` — bỏ `postgres-dest`.
- `kafka-connect.depends_on` chỉ còn `[kafka, schema-registry]` — bỏ `mongodb`.

---

## 4. Verify evidence (exercise-driven, lesson 2026-04-28)

### 4.1 Build PASS (3 repos)

```
cd cdc-auth-service && go build ./...     # EXIT=0
cd cdc-cms-service && go build ./...      # EXIT=0
cd centralized-data-service && go build ./...  # EXIT=0
```

### 4.2 Env override smoke (3 repos — go run với env set tay)

**cdc-auth-service**:
```
host=envhost port=9999 user=gpay_admin pass=envpass db=gpay_auth ssl=disable server=:18081
```
→ override applied cho host/port/pass/server, fallback YAML cho user/db/ssl.

**cdc-cms-service**:
```
server=:18083 db.host=envcmshost db.port=9988 db.user=gpay_admin db.pass=envcmspass db.db=cdc_dw otel=http://envotel:4318
```
→ override applied cho server/host/port/pass/otel, fallback YAML cho user/db.

**centralized-data-service**:
```
postgres_primary=postgres://envuser:envpass@envhost:9999/envdb?sslmode=disable
mongodb_primary=mongodb://envmongo:27018/
MongoDB.URL=mongodb://envmongo:27018/
```
→ SOURCE_DSN env override apply chính xác, hydrate legacy `MongoDB.URL` alias.

### 4.3 Docker split — 16 containers up

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

### 4.4 Volume preservation (data NOT lost)

| Volume | Trước | Sau |
|---|---|---|
| `centralized-data-service_pg_source_data` | source=65 | source=68 (sau insert smoke) |
| `centralized-data-service_pg_cdc_data` (shadow) | shadow=18 | shadow=21 (+3) |
| `centralized-data-service_pg_dest_data` (master) | master=37 | master=40 (+3) |
| `centralized-data-service_kafka_data` | 7 topics | 7 topics |
| `centralized-data-service_mongo_data` | rs0 | rs0 (healthy) |

### 4.5 Cross-bridge network resolution

```
docker exec gpay-cdc-worker getent hosts gpay-postgres-dest
  → 172.26.0.2  gpay-postgres-dest
docker exec gpay-cdc-worker getent hosts gpay-postgres-source
  → 172.26.0.5  gpay-postgres-source
docker exec gpay-kafka-connect getent hosts gpay-postgres-source
  → 172.26.0.5  gpay-postgres-source
```

### 4.6 E2E ingest smoke (Debezium → shadow)

```
INSERT 3 rows vào source.public.orders → wait 10s → shadow grew +3 (18→21).
Source: 65 → 68
Shadow: 18 → 21 (+3)
Master: 37 → 40 (+3 sau cron transmute tick)
```

### 4.7 Track D close-loop (Bug #2 P4 fix vẫn chạy sau restart)

```
SELECT id, last_status, last_error, updated_at FROM cdc_system.transmute_schedule
ORDER BY updated_at DESC LIMIT 5;

 id | last_status | last_error |          updated_at
----+-------------+------------+-------------------------------
 14 | success     |            | 2026-05-05 03:07:03.770778+00
 13 | success     |            | 2026-05-05 03:07:03.766996+00
 15 | success     |            | 2026-05-05 03:07:03.762283+00
  3 | success     |            | 2026-05-05 03:07:03.746115+00
  2 | success     |            | 2026-05-05 03:07:03.741244+00
```

### 4.8 Worker log clean (last 1 minute)

```
$ docker logs --since 1m gpay-cdc-worker | grep -E 'level":"(warn|error|fatal)'
(no output — 0 errors)
```

### 4.9 Standalone host services (auth + cms + admin)

```
auth :8081 GET /health    → 200
cms  :8083 GET /health    → 200
admin :8090 GET /healthz  → 200
```

(Note: 3 host processes này được start trước phiên này với YAML default. Do `go build` PASS + smoke env-override test PASS, em không restart chúng để tránh gián đoạn workflow của anh. Dev clone repo mới sẽ dùng `.env.example` flow.)

---

## 5. Out-of-scope / Known issues

### 5.1 Pre-existing issue — Debezium MySQL plugin missing

```
Connector cdc-mariadb-source: FAILED
Trace: "Failed to find any class that implements Connector and which name
       matches io.debezium.connector.mysql.MySqlConnector"

confluent-hub install --no-prompt debezium/debezium-connector-mysql:2.5.4
  → "Error: Component not found"
```

Plugin install command đã có sẵn trong compose từ trước B5 (`Fix B8 comment`), em chỉ copy verbatim từ compose cũ. Phiên bản `2.5.4` không tồn tại trên Confluent Hub format; 2 plugin còn lại (mongodb, postgresql) vẫn cài thành công.

**Fix riêng** (out of B5 scope): đổi sang `debezium/debezium-connector-mysql:2.4.0` hoặc download .jar về mount thẳng vào `/usr/share/confluent-hub-components/`. Cdc-pg-source + goopay-mongodb-cdc vẫn RUNNING — pipeline core không bị ảnh hưởng.

### 5.2 Git history vẫn chứa airbyte secret cũ

Em chỉ remove khỏi working tree. Lịch sử git log (`grep knF1jhaPIShkduykN`) vẫn show secret. Để purge cần `git filter-repo --replace-text` hoặc `git filter-branch` — task riêng, anh chốt khi muốn (sẽ rewrite history → cần force-push).

### 5.3 .gitignore chưa có .env

3 repo + cdc-docker-dev không có `.gitignore`. Anh nên tạo (hoặc em làm tiếp nếu anh ok) để `.env` không lỡ commit. Hiện tại chỉ có `.env.example` — không nguy hiểm.

### 5.4 Production secret manager

Phase B5 chỉ band-aid local dev với env file. Production cần Vault / AWS SSM / GCP Secret Manager. Out of scope.

### 5.5 Track E (MongoDB Debezium connector full E2E)

Vẫn out of scope như chốt từ B3.

---

## 6. Mapping to TaskList

| TaskID | Subject | Status |
|---|---|---|
| 115 | B5.1 Phase docs B5 | ✅ DONE |
| 116 | B5.2 cdc-auth-service env + .env.example | ✅ DONE |
| 117 | B5.3 cdc-cms-service env + airbyte purge | ✅ DONE |
| 118 | B5.4 centralized-data-service SOURCE_DSN | ✅ DONE |
| 119 | B5.5 Docker split | ✅ DONE |
| 120 | B5.6 Verify | ✅ DONE |
| 121 | B5.7 Report (this file) | ✅ DONE |
| 122 | B5.8 Delete workspace sai | ✅ DONE |

---

## 7. Reproduce / Re-run checklist

```bash
# 1. Bootstrap network (idempotent).
docker network create cdc-bridge 2>/dev/null || true

# 2. Up dev DBs.
docker compose -f cdc-system/cdc-docker-dev/docker-compose.yml up -d

# 3. Up core CDC infra.
docker compose -f cdc-system/centralized-data-service/docker-compose.yml up -d

# 4. Set env per service (3 repos), copy .env.example → .env.
for svc in cdc-auth-service cdc-cms-service centralized-data-service; do
  cp -n cdc-system/$svc/.env.example cdc-system/$svc/.env
done

# 5. Verify build.
for svc in cdc-auth-service cdc-cms-service centralized-data-service; do
  (cd cdc-system/$svc && go build ./...) || echo "FAIL $svc"
done

# 6. E2E smoke ingest.
docker exec gpay-postgres-source psql -U src_user -d goopay_source \
  -c "INSERT INTO public.orders(user_id, amount, status, notes)
      SELECT 9000+i, 100+i, 'pending', 'b5-recheck-'||i FROM generate_series(1,3) i;"
sleep 10
docker exec gpay-postgres-cdc psql -U gpay_admin -d cdc_dw \
  -c "SELECT count(*) FROM shadow_goopay_source.orders;"
# Expect: count tăng +3.
```

---

## 8. Confidence statement

Em verify bằng exercise thực tế (insert source → shadow grew +3 → master +3 sau cron tick), KHÔNG chỉ ping /healthz. Build clean 3 repos. Compose lint clean. Volume data preserved (count khớp before/after). Em báo DONE với evidence ở section 4.

Nếu anh thấy thiếu chỗ nào, anh chỉ em fix ngay. Lesson 2026-04-28 "không report láo" em ghi nhớ.

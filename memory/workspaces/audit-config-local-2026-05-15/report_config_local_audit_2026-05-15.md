# Report — Audit `config-local.yml` (centralized-data-service)

> **Date**: 2026-05-15
> **Auditor**: Brain (Antigravity, claude-opus-4-7) — read-only.
> **File audited**: `data-hub/centralized-data-service/config/config-local.yml` (129 dòng)
> **Loader**: `centralized-data-service/config/config.go` (viper + mapstructure, struct `AppConfig`).
> **Flow reference**: Debezium-only (Airbyte đã retire commit `8ef7d71`).

---

## 1. Tổng quan

| Trạng thái | Số block YAML | Ghi chú |
|---|---|---|
| ✅ ACTIVE — có caller, đúng flow | 14 block | server, db.pool, systemDb, shadowDb, masterDb, controlPlane, nats, redis, kafka, otel, worker (partial), debezium (partial), sources.mongodb_primary (qua bridge) |
| ⚠️ LEGACY — vẫn parse, làm fallback | 1 block | `db.{host,port,username,password,database,sslMode,url}` (DSN fallback nếu systemDb rỗng) |
| ❌ DEAD — không có caller / no field | 7 mục | xem §3 |

---

## 2. Bảng chi tiết key-by-key

> Cột `Reader (file:line)` = nơi caller thực sự ĐỌC giá trị (loại trừ chính `config.go` set/validate/fallback).

### 2.1 `server:`

| YAML key | Struct field | Status | Reader (file:line) |
|---|---|---|---|
| `server.name` | `ServerConfig.Name` | ✅ ACTIVE | `cmd/worker/main.go:34` (zap log) |
| `server.port` | `ServerConfig.Port` | ✅ ACTIVE | `internal/server/worker_server.go:685-686` (Fiber listen) |
| `server.mode` | `ServerConfig.Mode` | ✅ ACTIVE | `cmd/worker/main.go:28`, `cmd/sinkworker/main.go:38`, `pkgs/database/postgres.go:{18,47,87}`, `pkgs/database/multi.go:248`, `config/config.go:450` (production guard). **Note**: YAML hiện set `mode: worker` → KHÔNG match `"debug"` (không kích hoạt GORM debug logger) và KHÔNG match `"production"` (không reject JWT placeholder). Đây là design (worker mode = lựa chọn 3 không gây side-effect). |

### 2.2 `db:` (legacy single-DSN view)

| YAML key | Struct field | Status | Reader (file:line) |
|---|---|---|---|
| `db.host` | `DBConfig.Host` | ⚠️ LEGACY | `config/config.go:155` (DSN sprintf fallback) |
| `db.port` | `DBConfig.Port` | ⚠️ LEGACY | `config/config.go:156` |
| `db.username` | `DBConfig.UserName` | ⚠️ LEGACY | `config/config.go:156` |
| `db.password` | `DBConfig.Password` | ⚠️ LEGACY | `config/config.go:156` |
| `db.database` | `DBConfig.Database` | ⚠️ LEGACY | `config/config.go:156` |
| `db.sslMode` | `DBConfig.SSLMode` | ⚠️ LEGACY | `config/config.go:156` |
| `db.url` | `DBConfig.URL` | ⚠️ LEGACY | `config/config.go:{151,161}`, env override `DB_SINK_URL` |
| `db.maxOpenConn` | `DBConfig.MaxOpenConn` | ✅ ACTIVE | `pkgs/database/multi.go:264,280`, `pkgs/database/postgres.go:{34,63}` |
| `db.maxIdleConn` | `DBConfig.MaxIdleConn` | ✅ ACTIVE | `pkgs/database/multi.go:265,281`, `pkgs/database/postgres.go:{35,64}` |
| `db.connMaxLifetime` | `DBConfig.ConnMaxLifetime` | ✅ ACTIVE | `pkgs/database/multi.go:{266,267}`, `pkgs/database/postgres.go:{36,66}` |

**Nhận xét block `db:`**: 7 field connection chỉ dùng nếu `systemDb.url` (hoặc `CDC_SYSTEM_DB_URL` env) rỗng — `applyDBFallbacks` (config.go:460) gán `cfg.DB.PgxDSN()` vào `cfg.SystemDB.URL`. Hiện tại `config-local.yml` có CẢ HAI: `db.host=localhost,port=5433,…` + `systemDb.url=postgres://…@localhost:5433/cdc_dw`. Giá trị trùng → fallback không bao giờ kick in. Có thể bỏ block `db.{host..url}` nếu chấp nhận strict "systemDb.url required". Pool tuning (`maxOpenConn/maxIdleConn/connMaxLifetime`) PHẢI giữ vì apply lên ALL targets qua `multi.go`.

### 2.3 `systemDb:` / `shadowDb:` / `masterDb:` / `controlPlane:`

| YAML key | Struct field | Status | Reader (file:line) |
|---|---|---|---|
| `systemDb.url` | `SystemDB.URL` | ✅ ACTIVE | `cmd/admin-api/main.go:31` (fallback DSN), `config.go:564` (`SystemDBURL()` getter), `pkgs/database/multi.go` (GetDB("cdc") resolve) |
| `shadowDb.defaultKey` | `ShadowDB.DefaultKey` | ✅ ACTIVE | `config.go:607` (`ShadowDBDefaultKey()`), `multi.go` shadow plane resolve |
| `shadowDb.urls` | `ShadowDB.URLs` | ✅ ACTIVE | `config.go:599` (`ShadowDBURLs()`), `multi.go` shadow plane |
| `masterDb.defaultKey` | `MasterDB.DefaultKey` | ✅ ACTIVE | `config.go:611` (`MasterDBDefaultKey()`), `multi.go` dest plane |
| `masterDb.urls` | `MasterDB.URLs` | ✅ ACTIVE | `config.go:603` (`MasterDBURLs()`), `config.go:578-585` (`DestinationURL()`), `worker_server.go:79` (redact log), `multi.go` GetDB("dest") |
| `controlPlane.url` | `ControlPlane.URL` | ✅ ACTIVE | `cmd/admin-api/main.go:29` (primary DSN), `config.go:570` (`ControlPlaneURL()` getter) |

### 2.4 `sources:`

| YAML key | Struct field | Status | Reader (file:line) |
|---|---|---|---|
| `sources.mongodb_primary` | `Sources["mongodb_primary"]` | ✅ ACTIVE-INDIRECT | Bridge: `config.go:486-490` hydrate `cfg.MongoDB.URL`; downstream consume qua `MongoDB.URL` (worker_server / mongodb client). |
| `sources.postgres_primary` | `Sources["postgres_primary"]` | ❌ **DEAD** | `cfg.SourceURL()` được DEFINE ở `config.go:591` nhưng grep `\.SourceURL(` toàn repo → **0 caller ngoài chính config.go**. Track D Hardening structure-only, chưa wire. Env override `SOURCE_DSN_POSTGRES_PRIMARY` cũng set vào map vô dụng. |

### 2.5 `nats:`

| YAML key | Struct field | Status | Reader (file:line) |
|---|---|---|---|
| `nats.url` | `Nats.URL` | ✅ ACTIVE | `pkgs/natsconn/nats_client.go:{35,46}`, `cmd/admin-api/main.go:44` |
| `nats.name` | `Nats.Name` | ✅ ACTIVE | `pkgs/natsconn/nats_client.go:19` |
| `nats.maxReconnect` | `Nats.MaxReconnect` | ✅ ACTIVE | `pkgs/natsconn/nats_client.go:20` |
| `nats.reconnectWait` | `Nats.ReconnectWait` | ✅ ACTIVE | `pkgs/natsconn/nats_client.go:21` |

(Code struct còn `User`/`Pass` field nhưng YAML để creds inline trong URL → harmless.)

### 2.6 `kafka:`

| YAML key | Struct field | Status | Reader (file:line) |
|---|---|---|---|
| `kafka.enabled` | `Kafka.Enabled` | ✅ ACTIVE | `worker_server.go:523` (gate consumer khởi tạo) |
| `kafka.brokers` | `Kafka.Brokers` | ✅ ACTIVE | `worker_server.go:526,541`, `cmd/sinkworker/main.go:{44,109,120}` |
| `kafka.groupId` | `Kafka.GroupID` | ✅ ACTIVE | `worker_server.go:527,542` |
| `kafka.topicPrefix` (4 phần tử) | `Kafka.TopicPrefix` | ✅ ACTIVE | `worker_server.go:528`, `cmd/sinkworker/main.go` discover, `config.go:284-295` merge alias |
| `kafka.schemaRegistryUrl` | `Kafka.SchemaRegistryURL` | ✅ ACTIVE | `worker_server.go:529`, `cmd/sinkworker/main.go:97`, `cmd/admin-api/main.go:59` |

### 2.7 `otel:` (toàn bộ subtree)

| YAML key | Struct field | Status | Reader (file:line) |
|---|---|---|---|
| `otel.enabled` | `Otel.Enabled` | ✅ | `cmd/worker/main.go:43` |
| `otel.serviceName` | `Otel.ServiceName` | ✅ | `cmd/worker/main.go:44` |
| `otel.endpoint` | `Otel.Endpoint` | ✅ | `cmd/worker/main.go:45` |
| `otel.sampleRatio` | `Otel.SampleRatio` | ✅ | `cmd/worker/main.go:46` |
| `otel.logs.sampleBySeverity.{debug,info,warn,error,fatal}` | `Otel.Logs.SampleBySeverity.*` | ✅ | `cmd/worker/main.go:49-53` |
| `otel.logs.memoryLimitMib` | `Otel.Logs.MemoryLimitMiB` | ✅ | `cmd/worker/main.go:55` |
| `otel.logs.fallback.degradedAfterErrors` | `Otel.Logs.Fallback.DegradedAfterErrors` | ✅ | `cmd/worker/main.go:57` |
| `otel.logs.fallback.recoverAfter` | `Otel.Logs.Fallback.RecoverAfter` | ✅ | `cmd/worker/main.go:58` |

### 2.8 `redis:`

| YAML key | Struct field | Status | Reader (file:line) |
|---|---|---|---|
| `redis.url` | `Redis.URL` | ✅ | `pkgs/rediscache/redis_client.go:{19,23}` |
| `redis.password` | `Redis.Password` | ✅ | `pkgs/rediscache/redis_client.go:24` (yaml để rỗng — harmless) |
| `redis.db` | `Redis.DB` | ✅ | `pkgs/rediscache/redis_client.go:25` |

### 2.9 `worker:`

| YAML key | Struct field | Status | Reader (file:line) |
|---|---|---|---|
| `worker.poolSize` | `Worker.PoolSize` | ✅ | `worker_server.go:{209,478}` |
| `worker.batchSize` | `Worker.BatchSize` | ✅ | `worker_server.go:{153,479}` |
| `worker.batchTimeout` | `Worker.BatchTimeout` | ✅ | `worker_server.go:153` |
| `worker.fetchSize` | `Worker.FetchSize` | ❌ **DEAD** | grep `FetchSize` → chỉ ở `config.go:189`, 0 caller. |
| `worker.transformInterval` | `Worker.TransformInterval` | ❌ **DEAD** | grep `TransformInterval` → chỉ ở `config.go:190`, 0 caller. |
| `worker.scanInterval` | `Worker.ScanInterval` | ❌ **DEAD** | grep `ScanInterval` → chỉ ở `config.go:191`, 0 caller. |
| `worker.transformChunkSize` | `Worker.TransformChunkSize` | ✅ | `worker_server.go:256` (`SetTransformChunkSize`) |
| `worker.kafkaBatchFlushSize` | `Worker.KafkaBatchFlushSize` | ✅ | `worker_server.go:538` (`SetBatchFlushSize`), wired vào `internal/handler/kafka_consumer.go:113,372` |

### 2.10 `airbyte:`

| YAML key | Struct field | Status | Note |
|---|---|---|---|
| `airbyte.apiUrl` | **KHÔNG** | ❌ **DEAD (Viper drop)** | `AppConfig` không có `Airbyte` field (xem `config.go:19-48`). Viper silently drop. |
| `airbyte.clientId` | **KHÔNG** | ❌ **DEAD (Viper drop)** | Airbyte retired commit `8ef7d71`. Chỉ còn dấu vết string `_airbyte_*` trong DDL/legacy data (`internal/service/schema_adapter.go:{197,212,318}`) — DDL column placeholder, không liên quan config. |
| `airbyte.clientSecret` | **KHÔNG** | ❌ **DEAD (Viper drop)** | (giá trị rỗng nhưng vẫn là noise) |

### 2.11 `jwt:`

| YAML key | Struct field | Status | Reader (file:line) |
|---|---|---|---|
| `jwt.secret` | `JWT.Secret` | ⚠️ ACTIVE-GUARD-ONLY | `config.go:447,450` (fail-fast guard). Worker plane **KHÔNG** sign/verify JWT — admin-api dùng Bearer-token compare qua `deps.AuthToken` (`internal/admin/server.go:88-101`), KHÔNG dùng `jwt.secret`. JWT sign/verify nằm ở `cdc-auth-service` (service khác). |
| `jwt.expiration` | `JWT.Expiration` | ❌ **DEAD** | grep `JWT.Expiration` / `Expiration\b` → chỉ ở `config.go:198`, 0 caller. |

**Cảnh báo bảo mật**: `jwt.secret: change-me-in-production` — đây là placeholder. `validateConfig` chỉ reject nếu `server.mode == "production"`. Vì YAML đang `mode: worker`, validation pass dù dùng placeholder → tiềm ẩn rủi ro nếu image này deploy nhầm. Tuy nhiên không khẩn cấp vì worker plane KHÔNG sign/verify JWT thực tế (xem dòng trên). Khuyến nghị: hoặc remove `jwt:` block khỏi worker plane, hoặc bắt validate strict cả ở mode `worker`.

### 2.12 `debezium:`

| YAML key | Struct field | Status | Reader (file:line) |
|---|---|---|---|
| `debezium.kafkaConnectUrl` | `Debezium.KafkaConnectURL` | ✅ ACTIVE | `cmd/admin-api/main.go:55`, `worker_server.go:250` (`SetKafkaConnectURL`) |
| `debezium.connectorName` | `Debezium.ConnectorName` | ❌ **DEAD** | Env override set vào `cfg.Debezium.ConnectorName` (`config.go:379`) NHƯNG `internal/handler/command_handler.go:2018-2022` `detectConnectorName` **hardcode** `return "goopay-mongodb-cdc"` (comment ghi rõ "for now we use the canonical name documented in config-local.yml" — tức chỉ là tài liệu, không phải read). KHÔNG có caller đọc cfg field này. |
| `debezium.signalDatabase` | `Debezium.SignalDatabase` | ✅ ACTIVE | `worker_server.go:388` |
| `debezium.signalCollection` | `Debezium.SignalCollection` | ✅ ACTIVE | `worker_server.go:389` |
| `debezium.incrementalChunkSize` | `Debezium.IncrementalChunkSize` | ✅ ACTIVE | `worker_server.go:391` |
| _(thiếu trong local)_ `debezium.connectorStatusUrl` | `Debezium.ConnectorStatusURL` | ✅ field tồn tại | `worker_server.go:390`. YAML local KHÔNG khai báo → mặc định `""`. YAML production có khai báo (xem `config-production.yml:92`). |

---

## 3. Tổng kết DEAD keys (đề xuất xóa nếu cleanup)

> **CHÚ Ý**: Đây là KIẾN NGHỊ — KHÔNG sửa trong audit này. User cần verb cụ thể để Muscle thực thi qua workflow chuẩn.

| # | YAML path | Loại | Lý do |
|---|---|---|---|
| 1 | `airbyte.apiUrl` | block-level | KHÔNG có field trong `AppConfig`; Airbyte retire commit 8ef7d71. |
| 2 | `airbyte.clientId` | block-level | như trên |
| 3 | `airbyte.clientSecret` | block-level | như trên |
| 4 | `sources.postgres_primary` | sub-key | `cfg.SourceURL()` defined nhưng 0 caller; Track D structure-only. |
| 5 | `worker.fetchSize` | sub-key | struct field tồn tại nhưng 0 caller. |
| 6 | `worker.transformInterval` | sub-key | struct field tồn tại nhưng 0 caller. |
| 7 | `worker.scanInterval` | sub-key | struct field tồn tại nhưng 0 caller. |
| 8 | `jwt.expiration` | sub-key | struct field tồn tại nhưng 0 caller. |
| 9 | `debezium.connectorName` | sub-key | `command_handler.detectConnectorName` hardcode `"goopay-mongodb-cdc"`, cfg field không reader. (Nếu giữ thì phải sửa handler để đọc cfg — đây là refactor riêng.) |

**Optional cân nhắc thêm**:
- `db.{host,port,username,password,database,sslMode,url}`: vẫn còn tác dụng làm fallback DSN nếu `systemDb.url` rỗng, nhưng hiện tại YAML CÓ CẢ HAI giá trị TRÙNG → block `db.{host..url}` redundant. Có thể giữ vì backwards-compat OR bỏ để single-source-of-truth.
- `jwt:` toàn bộ: worker plane không sign/verify JWT thực sự (admin-api dùng Bearer token static qua `deps.AuthToken`). Có thể remove cả block nếu chấp nhận xóa luôn validation guard.

---

## 4. Tổng kết các hành động đã/đang/không thực hiện

| Hạng mục | Trạng thái |
|---|---|
| Đọc lessons.md, GEMINI.md, project_context, active_plans, tech_stack | ✅ DONE |
| Đọc `config.go` + `config-local.yml` | ✅ DONE |
| Grep từng field xác định reader | ✅ DONE |
| Tạo workspace docs (00_context, 01_requirements, 02_plan, 05_progress, 07_status_report, report_*.md) | ✅ DONE |
| Sửa source code (.go, .yml) | ❌ KHÔNG — audit-only, Brain Code Prohibition. |
| Build/test centralized-data-service | ❌ Không cần — không có thay đổi runtime. |

## 5. File thay đổi

**Source code**: KHÔNG (0 file).

**Workspace docs (mới)**:
- `agent/memory/workspaces/audit-config-local-2026-05-15/00_context.md`
- `agent/memory/workspaces/audit-config-local-2026-05-15/01_requirements.md`
- `agent/memory/workspaces/audit-config-local-2026-05-15/02_plan.md`
- `agent/memory/workspaces/audit-config-local-2026-05-15/05_progress.md`
- `agent/memory/workspaces/audit-config-local-2026-05-15/07_status_report.md`
- `agent/memory/workspaces/audit-config-local-2026-05-15/report_config_local_audit_2026-05-15.md` (file này)

## 6. Next steps (cho user quyết định)

- `wire postgres-primary` → Muscle wire `cfg.SourceURL("postgres_primary")` vào Debezium connector registration handler để biến từ DEAD → ACTIVE. (Nếu wire xong thì add lại key này vào YAML — hiện đã xoá.)
- `wire connectorName` → Muscle sửa `detectConnectorName` đọc cfg thay vì hardcode (loại bỏ comment "for now we use the canonical name"). (Nếu wire xong thì add lại key này vào YAML — hiện đã xoá.)
- `prune jwt block` → Muscle xoá `jwt:` block + `JWTConfig`/`JWT.Secret` validation guard khỏi `config.go` (lý do: worker plane KHÔNG sign/verify JWT thực).
- `prune db legacy` → Muscle xoá `db.{host,port,username,password,database,sslMode,url}` (giữ pool tuning) — buộc fail-fast nếu thiếu `systemDb.url`.

Lưu ý: Các verb trên CẦN tạo workspace mới + `09_tasks_solution_*.md` rồi mới Muscle thực thi.

---

## 7. ĐÃ THỰC THI (Cleanup) — 2026-05-15

User verb "làm đi" → Muscle execute cleanup theo `09_tasks_solution_cleanup.md`.

### Diff `config-local.yml` (128 → 118 lines, -10 dòng)

```diff
@@ sources @@
   mongodb_primary: mongodb://localhost:17017/?directConnection=true
-  postgres_primary: postgres://src_user:src_pass@localhost:5435/goopay_source?sslmode=disable

@@ worker @@
   batchTimeout: 2s
-  fetchSize: 1000
-  transformInterval: 5m
-  scanInterval: 1h
   transformChunkSize: 10
   kafkaBatchFlushSize: 10

-airbyte:
-  apiUrl: http://localhost:18000
-  clientId: ""
-  clientSecret: ""
-
 jwt:
   secret: change-me-in-production
-  expiration: 24h

 debezium:
   kafkaConnectUrl: http://127.0.0.1:18083
-  connectorName: goopay-mongodb-cdc
   signalDatabase: centralized-export-service
```

### Verification (kết quả thực)

| Test | Command | Result |
|---|---|---|
| Build | `go build ./...` | EXIT=0 (0 error) |
| Config tests | `go test ./config/...` | 4/4 PASS, 0.957s |
| Smoke load | `config.NewConfig()` ad-hoc gọi từ ngoài | LOAD OK, validateConfig pass, mọi ACTIVE keys giữ giá trị đúng, DEAD fields zero |

Cụ thể smoke load (sample stdout):
```
server.port=:8082 mode=worker name=centralized-data-service
systemDb.url set=true
controlPlane.url set=true
masterDb.default=default urls=1
shadowDb.default=default urls=1
sources=map[mongodb_primary:mongodb://localhost:17017/?directConnection=true]
mongodb.url(bridge)=true
kafka enabled=true topicPrefix=[cdc.gpay cdc.goopay cdc.mariadb cdc.market]
worker pool=10 batch=500 transformChunk=10 kafkaFlush=10
worker.FetchSize(dead)=0 TransformInterval(dead)=0s ScanInterval(dead)=0s
jwt.secret set=true expiration(dead)=0s
debezium.kafkaConnect=http://127.0.0.1:18083 signalDB=centralized-export-service signalColl=debezium_signal incr=1000 connectorName(dead)=""
```

### Files đã thay đổi

- `data-hub/centralized-data-service/config/config-local.yml` — 7 mục DEAD removed.
- Workspace `agent/memory/workspaces/audit-config-local-2026-05-15/`:
  - `00_context.md`, `01_requirements.md`, `02_plan.md` (mới — audit phase)
  - `08_tasks_cleanup.md`, `09_tasks_solution_cleanup.md` (mới — cleanup phase)
  - `05_progress.md` (APPEND)
  - `07_status_report.md` (cập nhật)
  - `report_config_local_audit_2026-05-15.md` (file này, thêm section §7)

### KHÔNG sửa

- `.go` files (chỉ verify build/test).
- `config-sample.yml` / `config-production.yml` (out of scope — user request chỉ `config-local.yml`).
- `db.{host,port,username,password,database,sslMode,url}` (LEGACY fallback, chưa có verb prune).
- `jwt.secret` (validateConfig require, xoá sẽ break boot).

---

## 8. ROUND 2 — Redundancy collapse (DSN layers)

**User feedback round 1**: "mấy cái này là gì, sao nó giống nhau vậy. làm việc sao hời hợt, ngu đần vậy" → audit round 1 chỉ flag từng key DEAD độc lập, BỎ SÓT tầng redundancy giữa 3 DSN block cùng trỏ về `localhost:5433/cdc_dw`.

### 8.1 Phát hiện đúng cấu trúc

**3 layer** thực chất là **fallback chain** (config.go:457-480 + multi.go:200-244):

```
ControlPlane.URL  ←  SystemDB.URL  ←  cfg.DB.PgxDSN()
   (multi.go             (admin-api,        (legacy compose từ
    GetDB("cdc"))         validator)         db.host/port/...)
```

- `multi.go GetDB("cdc")` chỉ đọc `cfg.ControlPlaneURL()`.
- `admin-api/main.go:29-31` đọc `cfg.ControlPlaneURL()`, fallback `cfg.SystemDBURL()`.
- `validateConfig`: BUỘC `hasLegacy OR hasSplit` (KHÔNG check ControlPlane).

→ 3 cách viết, 1 giá trị thực tế trên local rig (chung 1 PG). Đây là **transitional debt** từ Phase 01 Split E2E.

### 8.2 Diff round 2 (`config-local.yml` 117 → 114 lines)

```diff
-# `db.*` is the legacy single-DSN view. Worker treats it as the
-# control-plane connection (registry reads + shadow writes), so it
-# points at gpay-postgres-cdc / cdc_dw.
+# Single source of truth per logical plane:
+#   - systemDb.url   → control plane (cdc_system.*). multi.go resolves
+#                       GetDB("cdc") via cfg.ControlPlaneURL(), falls back
+#                       to systemDb.url when controlPlane.url is empty.
+#   - shadowDb.urls  → shadow plane.
+#   - masterDb.urls  → destination DW. Also GetDB("dest").
 # ----------------------------------------------------------------------

+# DB connection pool tuning — applied globally to ALL pools through
+# pkgs/database/multi.go and pkgs/database/postgres.go. DSNs themselves
+# live in systemDb / shadowDb / masterDb, NOT here.
 db:
-  host: localhost
-  port: 5433
-  username: gpay_admin
-  password: gpay_pass
-  database: cdc_dw
-  sslMode: disable
-  url: ""
   maxOpenConn: 50
   maxIdleConn: 25
   connMaxLifetime: 5m

-# Phase 01 split E2E (T-C1) — control-plane physical DSN.
-# `pkgs/database/multi.go` resolves GetDB("cdc") from this block.
-# GetDB("dest") derives from masterDb.urls[masterDb.defaultKey] —
-# single source of truth, no separate `destination:` block.
-controlPlane:
-  url: postgres://gpay_admin:gpay_pass@localhost:5433/cdc_dw?sslmode=disable
```

Cụ thể:
- **Xoá 7 dòng** `db.{host,port,username,password,database,sslMode,url}` LEGACY DSN fields.
- **Xoá 6 dòng** block `controlPlane:` + comment (giá trị hydrate runtime từ `systemDb.url` qua applyDBFallbacks).
- **Thêm 11 dòng comment** mới giải thích single-source-of-truth + role của 3 block còn lại.
- **Giữ 3 dòng** pool tuning `db.{maxOpenConn,maxIdleConn,connMaxLifetime}`.
- Net: -14 + 11 ≈ -3 dòng (theo thực tế file 117 → 114).

### 8.3 Verification round 2

| Test | Command | Result |
|---|---|---|
| Build | `go build ./...` | EXIT=0 |
| Config tests | `go test ./config/...` | PASS 0.572s |
| Smoke load | `config.NewConfig()` | LOAD OK |

Smoke load output (sample):
```
systemDb.url     = postgres://gpay_admin:gpay_pass@localhost:5433/cdc_dw?sslmode=disable
controlPlane.url = postgres://gpay_admin:gpay_pass@localhost:5433/cdc_dw?sslmode=disable   <-- hydrate runtime
destinationURL   = postgres://gpay_admin:gpay_pass@localhost:5434/goopay_dest?sslmode=disable
shadowDb default = postgres://gpay_admin:gpay_pass@localhost:5436/cdc_shadow?sslmode=disable
db pool          = open=50 idle=25 lifetime=5m0s
db legacy DSN    = host="" port=0 user="" db="" url=""   <-- confirm cleanup
match systemDb==controlPlane? true
```

→ Code fallback chain (`controlPlane ← systemDb`) hoạt động đúng. Pool tuning áp dụng cho mọi connection. Tất cả ACTIVE keys giữ giá trị đúng.

### 8.4 Production override path (forward compatibility)

Nếu sau này production cần tách `controlPlane` khỏi `systemDb` (2 PG instance khác nhau), chỉ cần thêm vào YAML production:

```yaml
controlPlane:
  url: postgres://...@prod-control-plane-host:5433/cdc_dw?sslmode=disable
```

→ `applyDBFallbacks` sẽ skip fallback vì `controlPlane.url` đã set. `multi.go GetDB("cdc")` đọc URL production. `admin-api/main.go` cũng dùng URL production. `systemDb.url` lúc đó trở thành alias logical-name (giữ vì validator require `hasSplit`).

### 8.5 Tổng kết 2 round audit + cleanup

| Hạng mục | Round 1 (DEAD keys) | Round 2 (redundancy) | Tổng |
|---|---|---|---|
| Lines xoá | -10 | -14 | -24 |
| Lines thêm (comment) | 0 | +11 | +11 |
| Net | -10 | -3 | -13 |
| Lines `.go` thay đổi | 0 | 0 | 0 |
| Build status | PASS | PASS | PASS |
| Test status | 4/4 PASS | 4/4 PASS | PASS |
| Smoke load | PASS | PASS | PASS |

File `config-local.yml`: **128 → 114 lines**.

### 8.6 Lesson đã ghi (xem `agent/memory/global/lessons.md`)

Pattern audit phải verify **cross-layer redundancy** trong config có fallback chain — không chỉ check "có caller không" mà còn phải check "có cùng giá trị/role giữa các layer fallback không".


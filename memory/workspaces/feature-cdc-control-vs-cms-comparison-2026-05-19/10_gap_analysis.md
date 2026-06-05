# 10_gap_analysis.md — So sánh cdc-control vs cdc-cms-service + cdc-cms-web

**Workspace**: `feature-cdc-control-vs-cms-comparison-2026-05-19`
**Date**: 2026-05-19
**Phạm vi**: PURE DOCUMENTATION — không sửa source code.

---

## Mục lục

1. [Tổng quan kiến trúc](#1-tổng-quan-kiến-trúc)
2. [Domain Model / DB schema](#2-domain-model--db-schema)
3. [HTTP routes & UI](#3-http-routes--ui)
4. [Connector lifecycle](#4-connector-lifecycle)
5. [Connection / Endpoint registry](#5-connection--endpoint-registry)
6. [Shadow management](#6-shadow-management)
7. [Schema sync](#7-schema-sync)
8. [Background jobs / scheduler](#8-background-jobs--scheduler)
9. [Observability / Audit](#9-observability--audit)
10. [Security](#10-security)
11. [Operational ergonomics](#11-operational-ergonomics)
12. [Gap matrix (feature × repo)](#12-gap-matrix-feature--repo)
13. [Điểm cdc-control chi tiết hơn](#13-điểm-cdc-control-chi-tiết-hơn)
14. [Điểm cdc-cms chi tiết hơn](#14-điểm-cdc-cms-chi-tiết-hơn)

---

## 1. Tổng quan kiến trúc

| Khía cạnh | `cdc-control` | `cdc-cms-service + cdc-cms-web` |
|-----------|---------------|----------------------------------|
| **Mục đích chính** | Control plane cho Kafka Connect: tạo Mongo→Mongo CDC (source+sink pair) + JDBC Sink SMT (Mongo→MySQL/Postgres). | CMS hoàn chỉnh cho data platform: source object registry, shadow Postgres, master tables, mapping rules, schema proposals, reconciliation, alerts. Quản lý Kafka Connect chỉ là 1 mảnh. |
| **Tech stack BE** | Python 3.12 + FastAPI 0.115 + Uvicorn + Jinja2 SSR + PyMySQL + pymongo + kafka-python + cryptography (hashlib/hmac). | Go 1.26 + Fiber v2 + GORM v1.31 + Postgres pgx/v5 + NATS + Redis + OpenTelemetry + Zap + Prometheus + Viper + Sonyflake IDs. |
| **Tech stack FE** | Không có FE riêng — single Jinja2 SSR page `templates/index.html` (2613 dòng, 6 tab Bootstrap). | React 19 + Vite 8 + TypeScript 5.9 + react-router-dom v7 + TanStack Query + Axios + Ant Design v6 (SPA). |
| **DB control plane** | MySQL 8.4 `cdc_control` (6 tables, schema inline migration). | Postgres `cdc_dw` + `cdc_shadow` (≥20 tables, embedded `.sql` migrations theo subsystem). |
| **Runtime** | Single ASGI process, asyncio loops trong-process. | Multi-goroutine: Fiber HTTP server + 4 background goroutines + NATS command bus. |
| **Encryption library** | `app/security.py` — PBKDF2-HMAC-SHA256 + HMAC-XOR keystream + HMAC tag, envelope JSON+b64 với prefix `enc:v1:`. | KHÔNG có. `pkgs/crypto/` không tồn tại. Dùng `secret_ref = 'env:VAR_NAME'` lookup. |
| **Auth** | KHÔNG có (open network). | JWT HS256 + RBAC 3 tier (`ops-admin`/`admin`/`operator`), middleware stack: JWT → RBAC → RateLimit → Idempotency (Redis 1h) → Audit. Dev bypass `dev-token`. |
| **Source DB support** | **MongoDB only** (`source_type` enum chỉ chấp nhận `mongodb`). | **MongoDB + MySQL + PostgreSQL** (cả 3 trong `parseFingerprint` + form trên `SourceConnectors.tsx`). |
| **Sink DB support** | MongoDB sink (CDC replication) + JDBC sink (MySQL/MariaDB/PostgreSQL) qua Debezium JDBC Sink. | Postgres shadow (CDC events landing) qua `ShadowAutomator.EnsureShadowTable()`. Không có JDBC sink workflow. |
| **i18n** | UI label tiếng Anh. README + bình luận tiếng Việt. | Tiếng Việt hard-code trong JSX. Không có thư viện i18n. `toLocaleString('vi-VN')`. |
| **Deployment** | Single-stage Dockerfile, expose 8000. Không có `docker-compose.yml` repo. | Multi-stage Dockerfile cho service (`golang:1.24-alpine` → `alpine:3.20`, expose 8083). FE: multi-stage Node 20 builder + Nginx 1.27 + `docker-entrypoint.sh` envsubst runtime. |

---

## 2. Domain Model / DB schema

### 2.1 Bảng tổng quan các entity

| Entity / Table concept | `cdc-control` (MySQL) | `cdc-cms-service` (Postgres) |
|------------------------|-----------------------|------------------------------|
| **Connector registry** (Kafka Connect status cache) | `connector_registry` (PK `name`) — config_json + status + last_sync_at + task_count + endpoint refs | `model.Source` (chỉ fingerprint, không cache config — không có table chuyên dùng cache `config_json`) |
| **Connection / Endpoint** | `mongo_endpoints` (Mongo source/dest) + `jdbc_destinations` (MySQL/MariaDB/Postgres sink) — **2 bảng riêng** | `cdc_system.connection_registry` (mig 029) — **1 bảng thống nhất** với `role_type` enum (`source|shadow|master|system|mixed`) + `engine_type` enum (5 loại) |
| **Source object** | KHÔNG có — Mongo collection được phát hiện realtime qua `pymongo.list_collection_names()` | `cdc_system.source_object_registry` (mig 030) — full catalog: object_code, locator_json, normalized key, primary_key_field, timestamp_field, cdc_mode, sync_engine, **provisioning_state machine** (8 states, mig 047) |
| **Shadow binding** | KHÔNG có — destination Mongo URI là 1 string trong `runtime_configs` | `cdc_system.shadow_binding` (mig 031) — bind source_object × shadow_connection × schema/table + namespace_strategy + write_mode + ddl_status |
| **Master binding** | KHÔNG có concept | `cdc_system.master_binding` (mig 032) — bind source_object × master_connection + transform_type (`copy_1_to_1|filter|aggregate|group_by|join|custom_sql`) + schema review workflow |
| **Mapping rule** | KHÔNG có (Mongo→Mongo replicate 1-1; JDBC Sink dùng `fields_json` định nghĩa cột) | `cdc_mapping_rules` (V1) + `cdc_system.mapping_rule_v2` (V2, mig 033) — full CQRS với jsonpath, transform_fn, source_format enum |
| **Schema proposal** | KHÔNG có | `cdc_system.schema_proposals` (mig 025) — review workflow approve/reject |
| **Schema export file metadata** | `schema_files` (PK `path`) — Mongo schema JSON files (apply/compare status, mtime) | KHÔNG có concept tương đương |
| **JDBC SMT connector pair** | `sink_smt_connectors` — source+sink pair, source_config_json + sink_config_json + transform SMT + JDBC creds | KHÔNG có concept |
| **Schedule** | KHÔNG có scheduler entity | `cdc_system.transmute_schedule` (mig 036, cron/immediate/post_ingest) + V1 `worker_schedule` (mig 007) |
| **Reconciliation** | KHÔNG có | `cdc_reconciliation_report` + `failed_sync_logs` (V1 + V2 partitioned, mig 010) |
| **Job tracking** | KHÔNG có | `cdc_system.cdc_jobs` (mig 052) UUID PK + type/status/payload/result + stuck job reaper |
| **Activity log / audit** | `timeline` (PK `id` auto_inc) — step + status + details_json (masked), 1 bảng | `cdc_system.cdc_activity_log` (daily partition, mig 010) + `cdc_system.admin_actions` (monthly partition, mig 040) — 2 bảng audit khác mục đích |
| **Pending field detection** | KHÔNG có | `pending_fields` (schema drift) + `schema_changes_log` |
| **Alerts** | KHÔNG có (chỉ timeline+error_message) | `cdc_system.cdc_alerts` (mig 041) — UUID PK + fingerprint + status state machine + ack/silence/auto-resolve |
| **Enum types** | KHÔNG có | `cdc_system.enum_types` (mig 001) |
| **Runtime config** | `runtime_configs` (encrypted PBKDF2+HMAC-XOR) — Mongo source/dest URI + Kafka Connect host động | KHÔNG có — config qua Viper YAML + ENV |
| **ID strategy** | MySQL AUTO_INCREMENT | Postgres `BIGSERIAL` + **Sonyflake** distributed IDs (mig 003, 018) + per-schema trigger fallback |

### 2.2 Migration strategy

| | `cdc-control` | `cdc-cms-service` |
|---|---|---|
| **Mechanism** | Inline `CREATE TABLE IF NOT EXISTS` + `_ensure_column`/`_drop_column`/`_ensure_unique_key` helpers gọi mỗi lần boot trong `MySqlStore.init()` (`store_mysql.py` lines 54–216) | Embedded `.sql` files qua `internal/migrate/`, grouped theo subsystem (core, recon_dlq, partitioning, cdc_system_model, audit_security, ids, registry, worker, ops). Sequence-numbered (001 → 053+) |
| **Idempotency** | Yes (helper checks column existence) | Yes (migration runner ghi state) |
| **Rollback** | Không có | Không có rollback formal, nhưng `schema_changes_log.rollback_sql` ghi DDL ngược |
| **Versioning** | Không có version number | Có (sequence number trên tên file) |

---

## 3. HTTP routes & UI

### 3.1 Tổng số endpoint

| | `cdc-control` | `cdc-cms-service` |
|---|---|---|
| **Total route count** | ~46 route (đếm từ inventory) | ~80+ route (api + api/v1 dual mount) |
| **HTML routes (SSR)** | ~20 (form submit redirect 303 về `/`) | 0 (chỉ `/swagger/*` HTML) |
| **JSON API routes** | ~26 | Tất cả còn lại |
| **Auth middleware** | Không có | JWT + RBAC trên mọi `/api/*` route |

### 3.2 Bảng route theo nhóm chức năng

| Chức năng | `cdc-control` routes | `cdc-cms-service` routes |
|-----------|----------------------|--------------------------|
| **Connector list** | `GET /api/connectors`, `GET /api/connectors/live` | `GET /api/v1/system/connectors`, `GET /api/v1/system/connectors/:name`, `GET /api/v1/system/connector-plugins` |
| **Connector create** | `POST /connectors/register` (HTML) + `POST /api/connectors/register` (JSON, tạo source+sink pair) | `POST /api/v1/system/connectors` (chỉ 1 connector duy nhất, không pair) |
| **Connector edit config** | KHÔNG có route edit config (xóa rồi tạo lại) | `PATCH /api/v1/system/connectors/:name/config` |
| **Connector delete** | `DELETE /api/connectors/{name}` + `POST /connectors/{name}/delete` (HTML) | `DELETE /api/v1/system/connectors/:name` |
| **Connector restart** | Tự động trong `register_mongo_cdc_connectors` (chỉ chạy onlyFailed=true sau khi PUT config) | `POST /api/v1/system/connectors/:name/restart` (manual) + `POST /api/v1/system/connectors/:name/tasks/:taskId/restart` |
| **Connector pause/resume** | KHÔNG có | `POST /api/v1/system/connectors/:name/pause` + `POST /api/v1/system/connectors/:name/resume` |
| **Connector registry sync (live → DB)** | `POST /api/connector-registry/sync` (manual) + auto qua `connector_registry_sync_loop` (stub, hiện inactive) | KHÔNG có — không có table cache config_json để sync |
| **Schema export/apply/compare** | `POST /export`, `POST /api/scan`, `POST /api/apply/{database_name}`, `POST /api/retry`, `POST /api/delete`, `GET /api/files`, `GET /api/timeline` | KHÔNG có — Mongo schema không phải đối tượng quản lý chính |
| **Mongo endpoint CRUD** | Full CRUD: `POST /mongo-endpoints`, `POST /.../edit`, `/enable`, `/disable`, `/delete` + `GET /api/mongo-endpoints` + databases/collections/fields API | KHÔNG có endpoint CRUD cho `connection_registry`. Chỉ có `bootstrap.EnsureDefaultShadowConnection` insert hardcode. Endpoint `GET /api/v1/sources` chỉ list, `POST /api/v1/sources` create source (chú ý: gọi nhầm `connection_registry` là "source") |
| **JDBC destination CRUD** | Full CRUD + `POST /jdbc-destinations/check-all` (bulk connection test) + `GET /api/jdbc-destinations` | KHÔNG có concept JDBC destination |
| **Sink SMT CRUD** | Full CRUD + `POST /sink-smt`, `/edit`, `/enable`, `/disable`, `/delete` + `GET /api/sink-smt/{id}/error` (HTML error page) | KHÔNG có concept |
| **Runtime config** | `GET /api/runtime-configs`, `POST /api/runtime-configs`, `DELETE /api/runtime-configs/{key}`, `POST /runtime-config/setup` (HTML), `GET /api/runtime-config/status` | KHÔNG có (Viper YAML + ENV, không edit runtime) |
| **Source introspection** | `GET /api/source-mongo/databases`, `GET /api/source-mongo/collections`, `GET /api/source-mongo/fields` + endpoint-specific equivalents | `GET /api/introspection/mongo/databases`, `GET /api/introspection/mongo/:db/collections`, `GET /api/introspection/scan/:table`, `GET /api/introspection/scan-raw/:table` |
| **Reconciliation** | KHÔNG có | `POST /api/reconciliation/check`, `/heal`, `/heal/:table`, `/check/:table` + `GET /api/reconciliation/report`, `/report/:table` + `GET /api/failed-sync-logs` + `/:id/retry` + `/backfill-source-ts` |
| **Source objects (V2)** | KHÔNG có | `GET /api/v1/source-objects`, `/stats`, `/registry/:id`, `POST /api/v1/source-objects/register`, `/register-batch`, `PATCH /api/v1/source-objects/:id`, plus 6+ action endpoints (scan-fields, standardize, create-default-columns, detect-timestamp-field, transform) |
| **Provisioning state machine** | KHÔNG có | `GET /api/v1/cms/sources/:id/provisioning`, `/advance`, `/pause`, `/resume`, `/retry`, `/archive`, `/mode` |
| **Master registry** | KHÔNG có concept | `GET /api/v1/masters`, `POST /api/v1/masters`, `/approve`, `/reject`, `/toggle-active`, `/swap` |
| **Wizard sessions** | KHÔNG có | `POST /api/v1/wizard/sessions`, `GET /api/v1/wizard/sessions/:id`, `PATCH`, `/progress`, `/execute` |
| **Schema proposals** | KHÔNG có | `GET /api/v1/schema-proposals`, `:id`, `/approve`, `/reject` |
| **Schedules (transmute)** | KHÔNG có | `GET/POST /api/v1/schedules`, `/:id`, `/run-now`, `PATCH /:id` |
| **Worker schedule** | KHÔNG có | `GET /api/worker-schedule`, `PATCH /:id`, `POST /api/worker-schedule` |
| **Mapping rules** | KHÔNG có (cho Mongo→Mongo), `fields_json` cho JDBC SMT | `GET /api/mapping-rules`, `POST`, `PATCH /batch`, `PATCH /:id`, `POST /:id/backfill`, `POST /reload`, `POST /preview` |
| **Schema changes** | KHÔNG có | `GET /api/schema-changes/pending`, `/history`, `/:id/approve`, `/:id/reject` |
| **Alerts** | KHÔNG có | `GET /api/alerts/active`, `/silenced`, `/history`, `POST /:fingerprint/ack`, `/silence` |
| **Activity log** | `GET /api/timeline` | `GET /api/activity-log`, `/stats` |
| **Jobs** | KHÔNG có | `GET /api/jobs/:id` (polling) |
| **System health** | KHÔNG có endpoint chuyên dụng (runtime health snapshot lưu Redis nhưng không expose) | `GET /api/system/health` (cache 30s, multi-probe) + `POST /api/tools/restart-debezium` |
| **Tools (debugging)** | KHÔNG có | `POST /api/tools/reset-debezium-offset`, `/trigger-snapshot/:table`, `/restart-debezium` |
| **Prometheus metrics** | `GET /metrics` (prometheus_client) | KHÔNG expose direct (có internal Prom client nhưng chưa thấy `/metrics` route trong inventory) |

### 3.3 UI structure

| | `cdc-control` | `cdc-cms-web` |
|---|---|---|
| **Rendering** | SSR Jinja2 (1 file 2613 dòng), localStorage cho tab persist | SPA React, react-router-dom |
| **Số "page" / "tab"** | 6 tab (Monitoring / MongoDB CDC / Schema Sync / Sink SMT / Logs / Management) | 14+ page (Login, Dashboard, TableRegistry, MappingFields, MasterRegistry, SchemaProposals, TransmuteSchedules, SourceConnectors, ActivityLog, ActivityManager, DataIntegrity, SystemHealth, SchemaChanges, SourceToMasterWizard) |
| **JS interaction** | Vanilla JS + Bootstrap 5 modal, `fetch()` for dynamic dropdowns | TanStack Query polling + Ant Design components + form validation built-in |
| **Reusable component** | KHÔNG có (single template) | `ConfirmDestructiveModal`, `DispatchStatusBadge`, `MappingRuleList`, `QueryErrorBoundary`, `ReDetectButton`, `AddMappingModal` |
| **State management** | localStorage + URL query `?active=...` | TanStack Query cache + `useState` local |
| **Idempotency UX** | KHÔNG có | `Idempotency-Key` header trên mọi destructive action + reason ≥ 10 chars |
| **Tracing UX** | KHÔNG có | `X-Correlation-Id`, `X-CDC-Action`, `X-CDC-Origin: cdc-cms-web` headers |
| **Diff view** | Schema compare (Mongo source vs dest) — output JSON, không có side-by-side UI | Schema proposals hiện proposed vs current trong table row, không có side-by-side |
| **Bulk operations UI** | Bulk DB registration (multi-checkbox `database_name[]`), `check-all` JDBC | Bulk mapping rule batch status, bulk source object register. SourceConnectors KHÔNG có bulk |
| **Connector edit modal** | KHÔNG (xóa rồi tạo lại) | YES — `KEEP_SECRET_SENTINEL = '__KEEP__'` để giữ password cũ |

---

## 4. Connector lifecycle

### 4.1 Tạo connector

| Bước | `cdc-control` | `cdc-cms-service` |
|------|---------------|-------------------|
| **Validation tên** | `safe_connector_database_name` regex `[A-Za-z0-9_-]+`, blocklist `admin`/`local`/`config` | Regex `^[a-zA-Z0-9][a-zA-Z0-9_.-]{0,128}$` (max 128 chars) |
| **Pair creation (source+sink)** | YES — `register_mongo_cdc_connectors_for_database` tạo cả source + sink trong cùng 1 transaction logic | NO — chỉ 1 connector / request. FE phải gọi 2 lần để tạo pair (nhưng FE không có flow này) |
| **Topic regex tự sinh** | YES — sink dùng `topics.regex = ^souce-cdc.{profile}\..+$` để fan-out tự động | NO — sink connector không được tạo bởi cms-service |
| **Template config** | YES — `CDC_SOURCE_MONGO`, `CDC_SINK_MONGO`, `SMT_SOURCE_TEMPLATE_FALLBACK`, `SMT_SINK_TEMPLATE_FALLBACK` trong `core.py` | NO — FE `SourceConnectors.tsx` tự sinh full config JSON gửi xuống BE |
| **Kafka Connect reachable check** | `wait_for_kafka_connect` poll `/connector-plugins` tối đa 60s | KHÔNG có precheck |
| **Auto restart nếu FAILED sau deploy** | YES — `connector_is_failed` check + `restart_connector(onlyFailed=true)` (`connect_ops.py:622–689`) | NO — chỉ tạo, không restart tự động sau khi PUT |
| **Bake fingerprint** | YES — upsert `connector_registry` row với status=`registering` trước khi PUT, status `RUNNING`/`FAILED` sau khi PUT | YES — `model.Source` upsert với `Status: "created"` (hardcode, không update theo Kafka state) |
| **Cleanup legacy connector** | YES — `cleanup_legacy_mongo_sink_connectors` xóa old `mongo-souce-cdc.*` | NO |

### 4.2 Restart logic

| | `cdc-control` | `cdc-cms-service` |
|---|---------------|-------------------|
| **Manual restart endpoint** | Implicit qua re-register (xóa rồi tạo lại) | `POST /:name/restart`, `POST /:name/tasks/:taskId/restart` |
| **Auto restart trên FAILED** | Trong `register_mongo_cdc_connectors` (chỉ chạy 1 lần sau khi deploy mới) | NO — `probes.Debezium()` chỉ detect FAILED và emit alert qua `system_health_alerts.go:131`, không auto restart |
| **Rate limit** | KHÔNG có | YES — 3 lần/giờ cho `POST /api/tools/restart-debezium` |
| **Idempotency** | KHÔNG có | YES — `Idempotency-Key` header + Redis TTL 1h |
| **Reason required** | KHÔNG có | YES — min 10 chars qua `ConfirmDestructiveModal` |
| **Pause/Resume support** | KHÔNG có | YES — `Lifecycle()` method calls `PUT /:name/pause` or `/resume` |

### 4.3 Status sync (live Kafka Connect → DB)

| | `cdc-control` | `cdc-cms-service` |
|---|---------------|-------------------|
| **Sync mechanism** | `sync_live_connectors_into_registry` (`connector_sync.py:536`) — iterate all live connectors + upsert `connector_registry` table | KHÔNG có sync table — `model.Source` chỉ lưu fingerprint một lần khi create, không update theo state |
| **Manual trigger** | `POST /api/connector-registry/sync` + `POST /connector-registry/sync` (HTML) | KHÔNG có |
| **Auto trigger** | `connector_registry_sync_loop` — **CURRENTLY STUB (`return None`)** trong `runtime_service.py:273` | KHÔNG có |
| **Runtime health snapshot** | `runtime_health_snapshot()` (`runtime_service.py:399`) poll Kafka Connect plugins + lưu Redis `runtime-health:summary` mỗi 15s | `healthCollector.Run(ctx)` (`system_health_collector.go`) — ticks 15s, run all 7 probes (postgres, debezium, kafka_connect, kafka_lag, nats, redis, worker) concurrently, lưu Redis `system_health:snapshot` TTL 60s |
| **Bảng cache config** | `connector_registry.config_json` (masked) | KHÔNG có |
| **Trace truncation** | KHÔNG có | `Debezium()` probe truncate trace 500 chars |

---

## 5. Connection / Endpoint registry

### 5.1 Schema so sánh

| | `cdc-control` | `cdc-cms-service` |
|---|---------------|-------------------|
| **Bảng** | `mongo_endpoints` (cho Mongo source/dest) + `jdbc_destinations` (cho MySQL/MariaDB/Postgres sink) — **2 bảng riêng** | `cdc_system.connection_registry` — **1 bảng thống nhất** với `role_type` enum |
| **Role enum** | KHÔNG có (mongo endpoint dùng `env` field `'onprem'`/`'profile'`, JDBC dùng `db_type`) | `source` / `shadow` / `master` / `system` / `mixed` |
| **Engine enum** | mongo_endpoints: `source_type='mongodb'` (chỉ 1 giá trị). jdbc_destinations: `db_type` ∈ `mysql|mariadb|postgresql` | `postgresql` / `mariadb` / `mysql` / `mongodb` / `clickhouse` |
| **Encryption credential** | YES — `jdbc_destinations.password` + `runtime_configs.value` encrypted với `enc:v1:`. Connection URI cho Mongo endpoint LƯU PLAINTEXT (`store_mysql.py` mongo_endpoints không encrypt) | KHÔNG — `secret_ref = 'env:VAR_NAME'` pointer pattern (giá trị nằm trong ENV của pod, không trong DB) |
| **Status enum** | `enabled` / `disabled` / `error` | `active` / `paused` / `failed` / `retired` |
| **Connection test trước khi enable** | YES — `client.admin.command("ping")` cho Mongo, `test_jdbc_destination_connection` cho JDBC | KHÔNG có endpoint test |
| **CRUD endpoints công khai** | YES — full CRUD HTML + JSON cho cả 2 loại | KHÔNG — `bootstrap.EnsureDefaultShadowConnection` insert hardcode `default_shadow`. Không có PUT/PATCH/DELETE endpoint cho `connection_registry` |
| **Bulk check** | YES — `POST /jdbc-destinations/check-all` test tất cả connections | KHÔNG |
| **Notes / metadata** | YES — `mongo_endpoints.notes` text field | YES — `options_json`, `capabilities_json` JSONB |
| **Discovery API** | YES — list databases/collections/fields per endpoint (`/api/mongo-endpoints/{id}/databases|collections|fields`) | YES — nhưng endpoint không gắn với `connection_registry`, mà với Mongo connection từ source: `GET /api/introspection/mongo/...` |

### 5.2 Encryption mechanism (chỉ có ở cdc-control)

```
Plaintext → PBKDF2-HMAC-SHA256(password=KEY_ENCRYPT, salt=16 random bytes,
                               iterations=200_000, dkLen=64)
         → 64 bytes split: [0..32]=enc_key, [32..64]=mac_key
         → HMAC-SHA256-XOR keystream (counter mode): block_n = HMAC(enc_key, nonce || counter_n)
         → ciphertext = plaintext XOR keystream
         → tag = HMAC-SHA256(mac_key, salt || nonce || ciphertext)
         → envelope = {"salt": b64, "nonce": b64, "ciphertext": b64, "tag": b64}
         → "enc:v1:" + b64url(JSON(envelope))
```

Decryption: extract envelope, verify HMAC tag với `hmac.compare_digest`, regenerate keystream, XOR.

**Migration on startup**: `encrypt_existing_runtime_configs()` (`runtime_service.py:418`) tự động encrypt mọi plaintext rows trong `runtime_configs` khi container boot.

**Áp dụng cho**: `runtime_configs.value`, `jdbc_destinations.password`, `sink_smt_connectors.mysql_password`.

**Mongo endpoint URI**: KHÔNG encrypt (lưu plaintext trong `mongo_endpoints.connection_uri`).

cdc-cms-service KHÔNG có cơ chế tương đương. Tất cả credential phải nằm trong ENV của pod runtime — không lưu trong DB.

---

## 6. Shadow management

| | `cdc-control` | `cdc-cms-service` |
|---|---------------|-------------------|
| **Concept "shadow"** | Implicit — destination Mongo URI là 1 string trong `runtime_configs.destination.connection_uri`. Mongo→Mongo replicate full bằng MongoSinkConnector | Explicit — `shadow_binding` table (mig 031) bind source_object × shadow_connection_id × schema/table |
| **Multi-shadow support** | YES — qua `flow_profile_name` + `source_endpoint_id` + `target_endpoint_id`. Mỗi profile có 1 sink riêng. Source DB có thể map nhiều profile (1-N source→shadow) | NO — `resolveShadowConnectionID()` (`source_object_v2_sync.go:374`) query `WHERE role_type IN ('shadow','mixed') AND status='active' ORDER BY id ASC LIMIT 1` → chỉ dùng shadow đầu tiên (luôn là `default_shadow`) |
| **Shadow DDL automation** | YES cho JDBC sink — `sync_mysql_fields` / `sync_postgres_fields` tự CREATE/ALTER TABLE trước khi enable. Cho Mongo sink: không cần DDL vì Mongo schemaless | YES — `ShadowAutomator.EnsureShadowTable()` (`shadow_automator.go:38`). Chỉ generate Postgres DDL (10-column CDC layout: id BIGINT PK, source_id UNIQUE, _raw_data JSONB, _source, _synced_at, _version, _hash, _deleted, _created_at, _updated_at) + 3 indexes (`_synced_at`, `_source`, `_raw_data` GIN). KHÔNG có path cho MongoDB shadow |
| **Shadow schema naming** | KHÔNG có concept "schema" cho Mongo dest (database name = target name) | `shadow_<connection_slug>_<sourceDB_slug>` — connection prefix đảm bảo 2 source cùng DB name khác connection vẫn có shadow distinct |
| **Sonyflake trigger** | KHÔNG có | YES — `attachSonyflakeTrigger()` per-schema fallback (sequence + 2 functions + trigger per table) |
| **Ddl status tracking** | KHÔNG có | YES — `shadow_binding.ddl_status` enum (`pending|created|failed|drifted`) |
| **Write mode config** | KHÔNG có (Mongo sink dùng default upsert) | YES — `shadow_binding.write_mode` enum (`upsert|append|replace`) |
| **Namespace strategy** | Implicit qua topic regex | YES — `shadow_binding.namespace_strategy` enum (`preserve|prefix|flatten|custom`) |
| **physical_table_fqn** | Implicit | YES — `shadow_binding.physical_table_fqn` (VARCHAR 600) |

---

## 7. Schema sync

| | `cdc-control` (Mongo schema) | `cdc-cms-service` (Postgres shadow schema) |
|---|------------------------------|--------------------------------------------|
| **Operation: Export** | YES — `export_schema(source_uri, db, output_file)` (`schema_ops.py`). Liệt kê tất cả collections (skip `system.profile`, `system.views`), ghi indexes (skip `_id_`), views (`viewOn`/`pipeline`/`options`). Output JSON `{schemaVersion: 2, database, exportedAt, collections, views}` | KHÔNG có Export concept |
| **Operation: Apply** | YES — `apply_schema(dest_uri, schema_file)`. CREATE collections missing, CREATE indexes nếu signature khác (stable canonical JSON), CREATE/UPDATE views. Handle view dependency cycles bằng retry | YES — `ShadowAutomator.EnsureShadowTable()` (chỉ 1 layout cố định, không phải apply từ file) |
| **Operation: Compare (diff)** | YES — `compare_schema(uris, db)` returns 8 loại diff: `missing_collection_on_dest`, `extra_collection_on_dest`, `collection_options_different`, `missing_or_different_index_on_dest`, `extra_or_different_index_on_dest`, `missing_view_on_dest`, `extra_view_on_dest`, `view_definition_different` | NO — không có diff |
| **Background scan** | YES — `scanner_loop` (`runtime_service.py:259`) chạy mỗi `SCAN_INTERVAL_SECONDS` (default 30s), iterate `EXPORT_DIR/*.schema.json`, skip files đã `last_apply_status=ok AND last_compare_status=ok`, gọi `apply_and_compare` cho rest. `APPLY_ON_STARTUP=true` chạy ngay khi boot | NO — không có scanner schema. `pending_fields` (V1) detect schema drift nhưng cơ chế khác (Debezium event-driven) |
| **Storage of schema file metadata** | `schema_files` table (PK `path`) — mtime, size_bytes, last_apply_status, last_compare_status, last_error, exported_at, applied_at, compared_at, scanner_skipped_at, skip_reason | KHÔNG |
| **Retry / Delete** | `POST /retry` (re-attempt failed apply), `POST /delete` (purge schema task + file) | NO |
| **Field-level mapping** | NO (Mongo→Mongo 1-1), YES cho JDBC SMT (`fields_json` định nghĩa cột target) | YES — `cdc_mapping_rules` (V1) + `cdc_system.mapping_rule_v2` (V2) — full jsonpath + transform_fn + source_format |
| **Approval workflow** | NO | YES — `cdc_system.schema_proposals` + endpoints `/approve`, `/reject` + `cdc_system.master_binding.schema_status` (`pending_review|approved|rejected|failed|drifted`) |

---

## 8. Background jobs / scheduler

### 8.1 Loops / goroutines

| Job | `cdc-control` | `cdc-cms-service` |
|-----|---------------|-------------------|
| **Schema scanner** | `scanner_loop` — 30s, iterate `*.schema.json` apply+compare | KHÔNG có |
| **Runtime health** | `runtime_health_loop` — 15s, ghi Redis `runtime-health:summary` | `healthCollector.Run(ctx)` — 15s, ghi Redis `system_health:snapshot` TTL 60s (multi-probe 7 sources) |
| **Connector registry sync** | `connector_registry_sync_loop` — STUB (return None), INACTIVE | KHÔNG có |
| **SMT health** | `smt_health_loop` — STUB (return None), INACTIVE. `smt_health_check_once` có code nhưng KHÔNG được start | KHÔNG có |
| **Encrypt existing runtime configs** | One-shot khi boot — `encrypt_existing_runtime_configs()` | KHÔNG có |
| **Audit logger flush** | KHÔNG có (timeline write synchronously) | `auditLogger.Run(ctx)` — async batch INSERT `cdc_system.admin_actions` |
| **Alert resolver** | KHÔNG có | `alertMgr.RunBackgroundResolver(ctx)` — state machine auto-resolve cleared conditions |
| **Stuck job reaper** | KHÔNG có | `stuckJobReaper.Run(ctx)` — 30s sweep `cdc_system.cdc_jobs`, flip `running` → `failed` khi vượt per-type timeout (master.swap 60s, recon.check 10m, mapping.backfill 30m, etc.) |
| **Cron scheduler** | KHÔNG có | `robfig/cron/v3` cho `cdc_system.transmute_schedule` (cron expr per binding) |

### 8.2 Cron mode trong `transmute_schedule`

cdc-cms-service có `transmute_schedule` table (mig 036) với:
- `mode` enum: `immediate` / `cron` / `post_ingest`
- `cron_expr` (required nếu mode=cron)
- `last_run_at`, `next_run_at`, `last_status`, `last_stats` JSONB
- Partial index cho due cron jobs
- UNIQUE (`master_binding_id`, `mode`)

cdc-control KHÔNG có scheduler entity.

---

## 9. Observability / Audit

### 9.1 Audit log

| | `cdc-control` | `cdc-cms-service` |
|---|---------------|-------------------|
| **Bảng** | `timeline` (1 bảng, không partition) | `cdc_system.cdc_activity_log` (daily partition) + `cdc_system.admin_actions` (monthly partition) — 2 bảng |
| **Số sự kiện events / step** | 11 step values: `register_connector`, `connector_registry`, `export`, `apply`, `compare`, `delete`, `sink_smt`, `jdbc_destination`, `mongo_endpoint`, `runtime_config`, `runtime_health` | Tự do (`operation` text field) — không enforce enum |
| **Masking** | YES — `mask_payload()` mask password/token/secret/api_key keys + `mask_uri` mask `user:pass@` + secret query params | KHÔNG explicit mask trong audit |
| **Async write** | NO (sync) | YES — `auditLogger.Run(ctx)` batch INSERT |
| **idempotency_key** | KHÔNG có | YES — `admin_actions.idempotency_key` (indexed WHERE NOT NULL) |
| **IP + user-agent capture** | KHÔNG có | YES — `admin_actions.ip_address`, `user_agent` |
| **Reason field** | KHÔNG có | YES — `admin_actions.reason` NOT NULL |

### 9.2 Probes

| Probe | `cdc-control` | `cdc-cms-service` |
|-------|---------------|-------------------|
| Postgres | NO | YES — `Postgres()` count `cdc_table_registry` + `pg_stat_user_tables` row sum |
| Debezium | Indirect qua connector_registry status | YES — `Debezium()` GET `/connectors/:name/status` connector+task with truncated trace |
| Kafka Connect | YES — `runtime_health_snapshot()` poll `/connector-plugins` | YES — `KafkaConnect()` GET `/connectors` topic count |
| Kafka Consumer Lag | NO | YES — `KafkaLag()` scrape kafka-exporter Prometheus endpoint, aggregate `kafka_consumergroup_lag` gauge |
| NATS | NO | YES — `NATS()` monitor API |
| Redis | NO (Redis used only for cache) | YES — `Redis()` ping |
| Worker | NO | YES — `Worker()` worker API health endpoint |
| Mongo ping | YES (manual qua endpoint enable check) | KHÔNG (Mongo không phải direct dependency của CMS) |

### 9.3 Alerts

| | `cdc-control` | `cdc-cms-service` |
|---|---------------|-------------------|
| **Alert table** | KHÔNG có (chỉ timeline + `last_error` field per entity) | `cdc_system.cdc_alerts` (UUID PK, fingerprint UNIQUE, severity, status state machine) |
| **Auto-resolve** | KHÔNG có | YES — `alertMgr.Resolve(fingerprint)` cho conditions ra khỏi detected set |
| **Ack** | KHÔNG có | YES — `POST /api/alerts/:fingerprint/ack`, `ack_by`, `ack_at` columns |
| **Silence** | KHÔNG có | YES — `POST /api/alerts/:fingerprint/silence`, `silenced_until`, `silence_reason` |
| **Detection rules** | KHÔNG | 4 rules (`system_health_alerts.go:119`): `DebeziumConnectorFailed` (critical), `HighConsumerLag` (>100K critical, >10K warning), `ReconDrift` (warning), `InfrastructureDown` (critical) |
| **Occurrence count** | KHÔNG | YES — `cdc_alerts.occurrence_count`, `last_fired_at` |

### 9.4 Reconciliation (chỉ có ở cdc-cms-service)

cdc-cms-service có 1 hệ thống reconciliation hoàn chỉnh:
- `cdc_reconciliation_report` — source_count vs dest_count, diff, missing_ids, stale_ids, tier, status, duration_ms
- `failed_sync_logs` (V1 unpartitioned) + `cdc_system.failed_sync_logs` (V2 monthly partition + `next_retry_at`, `last_error`)
- Endpoints: `POST /api/reconciliation/check`, `/heal`, `/heal/:table`, `/check/:table`
- Failed log retry: `POST /api/failed-sync-logs/:id/retry`
- Backfill source_ts: `POST /api/recon/backfill-source-ts` + status endpoint

cdc-control KHÔNG có concept reconciliation.

### 9.5 Slow query / Performance

| | `cdc-control` | `cdc-cms-service` |
|---|---------------|-------------------|
| **GORM slow threshold** | N/A (no GORM) | Default GORM `200 * time.Millisecond` (chưa override trong `pkgs/database/postgres.go:35`) |
| **`pg_stat_statements`** | N/A | NO direct integration |
| **Partition pruning** | NO (timeline unpartitioned) | YES — bounded time range queries trên failed_sync_logs + cdc_activity_log để enable Postgres pruning |
| **Prometheus metrics export** | YES — `GET /metrics` (prometheus_client) | Có internal Prom client nhưng `/metrics` route chưa thấy trong inventory |

---

## 10. Security

| Aspect | `cdc-control` | `cdc-cms-service` |
|--------|---------------|-------------------|
| **Authentication** | KHÔNG có | JWT HS256 — `Authorization: Bearer <token>` |
| **Dev bypass** | N/A | YES — `tokenString == "dev-token"` → admin (`jwt.go:23`) ⚠ phải remove production |
| **RBAC** | KHÔNG có | 3 tier: `ops-admin` / `admin` / `operator`. Env fallback `ADMIN_USERS=user1,user2` |
| **Middleware stack** (destructive) | KHÔNG có | `JWTAuth → RequireOpsAdmin → [RateLimit 3/h] → Idempotency (Redis TTL 1h) → Audit (async) → handler` |
| **CSRF** | NO | NO (JWT-based, không session) |
| **Encryption at rest (credentials)** | YES (PBKDF2+HMAC-XOR `enc:v1:`) cho `runtime_configs.value`, `jdbc_destinations.password`, `sink_smt_connectors.mysql_password`. Mongo URI KHÔNG encrypt | NO — `secret_ref = 'env:VAR_NAME'` lookup |
| **Secret masking on output** | YES — `mask_payload`, `mask_uri`, `mask_value` áp dụng cho mọi API response + timeline details + connector config display | YES — `FilterSafeConfig` (`kafka_connect.go`) strip keys chứa `password`/`secret`/`token`/`credentials`/`ssl.key` khi expose connector config |
| **Input validation** | Regex name + database + table + JDBC host. Schema path resolve inside EXPORT_DIR | Regex connector name. Schema/table name `validateIdent()` lowercase `[a-z0-9_]` 1–63 chars. Reason min 10 chars |
| **Token refresh** | N/A | NO (`refresh_token` lưu localStorage nhưng không có refresh logic) |

---

## 11. Operational ergonomics

| | `cdc-control` | `cdc-cms-service + cdc-cms-web` |
|---|---------------|--------------------------------|
| **Connector edit flow** | Xóa rồi tạo lại (không có edit) | Edit modal với `KEEP_SECRET_SENTINEL = '__KEEP__'` để giữ password cũ |
| **Bulk operations** | Bulk DB register (multi-select), bulk check JDBC connections | Bulk mapping rule status, bulk source object register. Không bulk cho connector |
| **Diff view** | YES cho Mongo schema (output JSON 8 loại diff) | Schema proposal hiển thị proposed vs current trong table row, không side-by-side |
| **Error UX** | Page alert banner + error truncate 240 chars + retry button cho schema task + JDBC sink error HTML page | `ConfirmDestructiveModal` reason ≥ 10 chars + `Idempotency-Key` header + ReDetectButton cho timestamp field |
| **Polling UX** | Vanilla JS `fetch()` manual reload | TanStack Query 15s/30s polling (connectors 15s, fingerprints 30s, system health 30s) |
| **Connector retry from FAILED** | Auto trong `register_mongo_cdc_connectors` (chỉ 1 lần sau deploy) | Manual via `POST /:name/restart` button |
| **Confirm before destructive** | `onsubmit="return confirm(...)"` | `ConfirmDestructiveModal` component (reason field + warning alert) |
| **Topic + consumer group cleanup on delete** | YES — `connector_cleanup_candidates` + `KafkaAdminClient` delete topics + consumer groups | KHÔNG có cleanup khi delete connector (chỉ DELETE connector qua Kafka Connect REST) |
| **Connector test endpoint** | YES cho JDBC destination (test connect trước enable) | KHÔNG có endpoint test |
| **Time zone display** | UTC+7 hardcode | `toLocaleString('vi-VN', { hour12: false })` |

---

## 12. Gap matrix (feature × repo)

Legend: ✅ Có / ⚠ Partial / ❌ Không / `—` Không áp dụng

| # | Feature | `cdc-control` | `cdc-cms-service` | Ghi chú |
|---|---------|:------------:|:-----------------:|---------|
| 1 | Tạo connector source | ✅ | ✅ | cms-service không pair với sink |
| 2 | Tạo connector sink | ✅ | ❌ | `parseFingerprint` chỉ detect source class |
| 3 | Tạo source+sink pair atomically | ✅ | ❌ | cms-service 1 request 1 connector |
| 4 | Edit connector config | ❌ (delete+create) | ✅ | cms-service có `KEEP_SECRET_SENTINEL` |
| 5 | Pause / Resume | ❌ | ✅ | |
| 6 | Restart connector + per-task | ⚠ (chỉ onlyFailed sau deploy) | ✅ | cms-service rate-limit 3/h |
| 7 | Auto-restart background loop khi FAILED | ⚠ (chỉ 1 lần khi register) | ❌ | Chỉ emit alert, không restart |
| 8 | Status sync live → DB cache | ✅ (manual + STUB loop) | ❌ | cms-service không có table cache config |
| 9 | Topic + consumer group cleanup khi delete | ✅ | ❌ | |
| 10 | Connection registry table thống nhất | ❌ (2 bảng) | ✅ | `role_type` enum |
| 11 | CRUD endpoint cho connection | ✅ | ❌ | cms-service chỉ bootstrap default_shadow |
| 12 | Connection test trước enable | ✅ | ❌ | |
| 13 | Multi-shadow endpoint support | ✅ (qua flow_profile) | ❌ | `resolveShadowConnectionID` LIMIT 1 |
| 14 | Encryption at rest (PBKDF2+HMAC) | ✅ (`enc:v1:`) | ❌ | cms-service dùng `env:VAR_NAME` |
| 15 | Encrypt Mongo URI | ❌ (plaintext) | ❌ | cdc-control vẫn có gap này |
| 16 | Schema export Mongo (collections + indexes + views) | ✅ | ❌ | |
| 17 | Schema apply Mongo | ✅ | ❌ | |
| 18 | Schema compare Mongo (8 loại diff) | ✅ | ❌ | |
| 19 | Schema scanner background loop | ✅ (30s) | ❌ | |
| 20 | Shadow DDL automation Postgres | ❌ | ✅ | 10-column CDC layout |
| 21 | Sonyflake distributed ID | ❌ | ✅ | |
| 22 | Source object catalog + provisioning state machine | ❌ | ✅ | 8 states |
| 23 | Master table registry + transform types | ❌ | ✅ | 6 transform types |
| 24 | Mapping rule (jsonpath + transform_fn) | ❌ | ✅ | V1 + V2 |
| 25 | Schema proposal approve/reject workflow | ❌ | ✅ | |
| 26 | Transmute schedule (cron/immediate/post_ingest) | ❌ | ✅ | |
| 27 | Reconciliation (check/heal/report) | ❌ | ✅ | |
| 28 | Failed sync logs partitioned + retry | ❌ | ✅ | mig 010 |
| 29 | Alerts table + ack/silence/auto-resolve | ❌ | ✅ | mig 041 |
| 30 | Alert detection rules (4 rules) | ❌ | ✅ | |
| 31 | Stuck job reaper | ❌ | ✅ | per-type timeout |
| 32 | Pending field detection + schema drift | ❌ | ✅ | |
| 33 | Wizard sessions (source→master) | ❌ | ✅ | |
| 34 | Admin actions audit (idempotency_key + IP + UA) | ❌ | ✅ | mig 040 |
| 35 | JDBC Sink SMT connector (Mongo→MySQL/Postgres) | ✅ | ❌ | |
| 36 | JDBC sink field editor UI | ✅ | ❌ | |
| 37 | Auto CREATE/ALTER TABLE cho JDBC sink | ✅ | ❌ | |
| 38 | Mongo endpoint discovery (DB/collections/fields) | ✅ | ✅ | API similar |
| 39 | Bulk connector operations UI | ✅ (multi-DB register) | ❌ | |
| 40 | Bulk JDBC check-all | ✅ | ❌ | |
| 41 | Runtime config dynamic (Kafka host, source URI) | ✅ (UI editable) | ❌ | cms-service: Viper YAML+ENV |
| 42 | Prometheus `/metrics` endpoint | ✅ | ⚠ (internal client, route chưa thấy) | |
| 43 | Authentication (JWT) | ❌ | ✅ | HS256 |
| 44 | RBAC (3 tier) | ❌ | ✅ | ops-admin/admin/operator |
| 45 | Idempotency middleware | ❌ | ✅ | Redis TTL 1h |
| 46 | Rate limit | ❌ | ✅ | restart-debezium 3/h |
| 47 | Reason field required cho destructive | ❌ | ✅ | min 10 chars |
| 48 | NATS command bus | ❌ | ✅ | async dispatch |
| 49 | OpenTelemetry tracing | ❌ | ✅ | OTel + Zap |
| 50 | Swagger docs | ❌ | ✅ | `/swagger/*` |

**Tổng kết bảng:** 50 feature → cdc-control độc quyền 17 / cdc-cms độc quyền 28 / overlap 5 / cả hai cùng thiếu 1 (encrypt Mongo URI ở cdc-control vẫn plaintext).

---

## 13. Điểm cdc-control chi tiết hơn

Những thứ `cdc-control` làm sâu hơn so với hệ thống mới:

### 13.1 Quản lý lifecycle connector pair (Mongo CDC)

- **Pair atomic**: 1 request `POST /connectors/register` tạo cả source + sink trong cùng flow profile, với rollback khi 1 fail. cms-service phải tạo từng cái 1 — FE `SourceConnectors.tsx` không có flow này.
- **Topic regex dynamic rebuild**: khi add database mới vào profile cũ, `source_databases_from_connectors` enumerate tất cả source connector cùng prefix để rebuild `topics.regex` cho sink. cms-service không có concept này.
- **Cleanup topic + consumer group khi delete**: dùng `KafkaAdminClient` xóa kèm. cms-service chỉ DELETE qua Kafka Connect REST, để lại topic + consumer group orphan.
- **Legacy connector migration**: `cleanup_legacy_mongo_sink_connectors` xóa old naming. cms-service không có migration path.

### 13.2 Connection registry (2 entity riêng + encryption)

- **Mongo endpoint**: full CRUD + enable/disable + connection test (`client.admin.command("ping")`) trước khi enable + databases/collections/fields discovery API riêng cho từng endpoint. cms-service không có CRUD endpoint cho `connection_registry`.
- **JDBC destination**: CRUD + bulk `check-all` test toàn bộ connections + password encrypted với `enc:v1:`. cms-service không có concept JDBC destination.
- **Encryption library**: PBKDF2-HMAC-SHA256 200K iterations + HMAC-XOR keystream + HMAC tag + auto-migrate plaintext rows on boot (`encrypt_existing_runtime_configs()`). cms-service KHÔNG có `pkgs/crypto/`.

### 13.3 Schema sync (Mongo)

- Export + Apply + Compare với 8 loại diff (`missing_collection_on_dest`, `extra_collection_on_dest`, `collection_options_different`, `missing_or_different_index_on_dest`, `extra_or_different_index_on_dest`, `missing_view_on_dest`, `extra_view_on_dest`, `view_definition_different`).
- View dependency cycle handling (retry unresolved views).
- Schema file metadata bảng `schema_files` (mtime, size, last_apply_status, last_compare_status, skip_reason).
- Background scanner loop 30s tự apply.
- Skip optimization: bỏ qua files đã `ok+ok`.

cms-service KHÔNG có concept Mongo schema sync (vì target là Postgres shadow).

### 13.4 JDBC Sink SMT (Mongo → MySQL/Postgres)

- Full workflow: source connector + sink connector + field editor (two-way sync textarea ↔ table editor) + SMT filter/rename suggestions realtime.
- Auto `sync_mysql_fields` / `sync_postgres_fields` CREATE/ALTER TABLE trước enable.
- 3 dialects: MySQL, MariaDB, PostgreSQL với Hibernate dialect + JDBC URL templates.
- Field types: validate length, nullable, object detection.

cms-service KHÔNG có JDBC Sink workflow.

### 13.5 Output masking comprehensive

- `mask_payload()` + `mask_uri()` + `mask_value()` áp dụng cho **mọi** API response, timeline details, connector config display.
- Pattern detect: `credential`, `password`, `passwd`, `secret`, `token`, `api_key`, `apikey` trong key name.
- URI masking: `user:pass@` + query params matching secret patterns.

cms-service chỉ có `FilterSafeConfig` strip keys khi expose connector config — không apply globally.

### 13.6 Runtime config editable trong UI

`runtime_configs` table cho phép user UI edit:
- Mongo source URI
- Mongo dest URI
- Kafka Connect host (comma-separated failover)

Restart không cần redeploy. cms-service: tất cả qua Viper YAML + ENV → cần redeploy pod.

### 13.7 Bulk operations (cdc-control)

- Multi-database register: 1 form, multi-checkbox `database_name[]`, iterate register từng DB.
- JDBC check-all: 1 click test toàn bộ JDBC destinations.

cms-service: SourceConnectors page **không có** bulk operation. Có bulk mapping rule + bulk source object register nhưng khác phạm vi.

---

## 14. Điểm cdc-cms chi tiết hơn

Những thứ `cdc-cms-service + cdc-cms-web` làm sâu hơn `cdc-control`:

### 14.1 Domain model phong phú hơn nhiều

cms-service có:
- `source_object_registry` — full catalog với object_code, locator_json, normalized_source_key UNIQUE, primary_key_field, timestamp_field + candidates_json + detected_at + source + confidence, cdc_mode (4 modes), sync_engine (4 engines), **provisioning_state machine** với 8 states + step_log JSONB.
- `shadow_binding` — namespace_strategy + write_mode + ddl_status + physical_table_fqn.
- `master_binding` — transform_type (6 loại) + transform_spec JSONB + schema review (5 statuses) + CHECK constraint `is_active=FALSE OR schema_status='approved'`.
- `mapping_rule_v2` — jsonpath + transform_fn + source_format enum + UNIQUE (source_object_id, master_binding_id, target_column).
- `transmute_schedule` — cron expr per binding + due cron partial index.
- `cdc_jobs` — UUID + payload/result JSONB + correlation_id.
- `cdc_alerts` — UUID + fingerprint UNIQUE + state machine + occurrence_count + ack/silence.
- `admin_actions` — monthly partition + idempotency_key + IP + UA + reason.
- `pending_fields` + `schema_changes_log` — schema drift detection + audit trail with rollback_sql.

cdc-control chỉ có 6 bảng đơn giản: connector_registry, mongo_endpoints, jdbc_destinations, runtime_configs, schema_files, sink_smt_connectors, timeline.

### 14.2 Reconciliation + failed sync logs

cms-service có hệ thống reconciliation hoàn chỉnh:
- `cdc_reconciliation_report` — source_count vs dest_count, diff, missing_ids, stale_ids, tier (`critical|high|normal|low`), duration_ms, error_code.
- `failed_sync_logs` V1 + V2 partitioned monthly với `next_retry_at`, `last_error`, `retry_count`, `max_retries`.
- Endpoints: check/heal toàn bộ hoặc per-table + retry per log + backfill source_ts với status endpoint.
- Tier-based prioritization.

cdc-control KHÔNG có concept reconciliation.

### 14.3 Multi-engine source support

cms-service `parseFingerprint` + FE form support 3 engine:
- `MongoDb` → `mongodb.connection.string` parsing
- `MySql` → host/port/username/password/database/table include list/server ID
- `Postgres` → host/port/username/password/database/schema/table include list/slot name/publication name/`plugin.name: pgoutput`/`publication.autocreate.mode: filtered`

cdc-control: chỉ MongoDB source (`source_type` enum chỉ chấp nhận `mongodb`).

### 14.4 Async command bus + stuck job reaper

cms-service có NATS command bus với 16+ async subjects:
- `cdc.cmd.recon-check`, `recon-heal`, `retry-failed`, `recon-backfill-source-ts`
- `cdc.cmd.debezium-signal`, `debezium-snapshot`, `restart-debezium`
- `cdc.cmd.create-default-columns`, `standardize`, `scan-fields`, `detect-timestamp-field`
- `cdc.cmd.backfill`, `alter-column`
- `cdc.cmd.transmute`, `master-create`, `master-swap`

Stuck job reaper sweep mỗi 30s, flip `running` → `failed` khi vượt timeout per-type (`master.swap` 60s, `recon.check` 10m, `mapping.backfill` 30m, etc.).

cdc-control: chỉ asyncio in-process, không có job queue.

### 14.5 Wizard sessions + provisioning state machine

cms-service có wizard `SourceToMasterWizard.tsx` + `cdc_system.wizard_sessions` (chưa list bảng nhưng có endpoints):
- `POST /api/v1/wizard/sessions` create
- `PATCH /api/v1/wizard/sessions/:id` update step
- `GET /api/v1/wizard/sessions/:id/progress` poll progress
- `POST /api/v1/wizard/sessions/:id/execute` trigger pipeline

Source object có 8-state provisioning machine với `provisioning_step_log` JSONB + `last_step_error` + endpoints `/advance`, `/pause`, `/resume`, `/retry`, `/archive`, `/mode`.

cdc-control: không có state machine.

### 14.6 Master table workflow

`master_binding` + 6 transform types (`copy_1_to_1|filter|aggregate|group_by|join|custom_sql`) + schema review (5 statuses) + approve/reject/toggle-active/swap endpoints. Master DDL approve trước khi `is_active=true` (CHECK constraint enforced).

cdc-control: không có concept master table.

### 14.7 Alerts với state machine

- `cdc_alerts` UUID + fingerprint UNIQUE + status (`firing|resolved|silenced`) + severity + labels JSONB.
- 4 detection rules (`DebeziumConnectorFailed`, `HighConsumerLag`, `ReconDrift`, `InfrastructureDown`).
- Auto-resolve: cleared conditions tự `alertMgr.Resolve(fingerprint)`.
- Ack: `ack_by`, `ack_at` columns + endpoint.
- Silence: `silenced_until`, `silence_reason` columns + endpoint.
- Occurrence count + last_fired_at.

cdc-control: chỉ ghi vào timeline + `last_error` field per entity, không có alert machinery.

### 14.8 Security stack (JWT + RBAC + Idempotency + Rate Limit + Audit)

- JWT HS256 với 3-tier RBAC.
- Idempotency middleware Redis TTL 1h.
- Rate limit (3/h cho restart-debezium).
- Audit logger async batch INSERT `admin_actions` với idempotency_key + IP + user_agent + reason.
- Middleware stack: JWT → RBAC → RateLimit → Idempotency → Audit → handler.

cdc-control: KHÔNG có authentication. Phụ thuộc network-level access control.

### 14.9 Observability stack

- OpenTelemetry tracing + Zap structured logger.
- 7 probes concurrent: postgres, debezium, kafka_connect, kafka_lag (Prometheus scrape), nats, redis, worker.
- System health snapshot multi-section: infrastructure + cdc_pipeline + reconciliation + latency (P50/P95/P99) + failed_sync (1h/24h) + alerts + recent_events.
- Redis cache TTL 60s cho snapshot.
- Per-probe timeout 2s + errgroup concurrent.

cdc-control: 1 background loop poll `/connector-plugins` lưu Redis. Không có distributed tracing.

### 14.10 Partitioning + Sonyflake

- Postgres native partitioning: `failed_sync_logs` monthly, `cdc_activity_log` daily, `admin_actions` monthly.
- Bounded time range queries để enable partition pruning.
- Sonyflake distributed ID infrastructure trong `cdc_system` (mig 003, 018) + per-schema fallback trigger.

cdc-control: 1 bảng `timeline` không partition. MySQL AUTO_INCREMENT.

### 14.11 React SPA với UX maturity

- 14+ page với route aliases.
- TanStack Query polling 15s/30s.
- Ant Design form validation.
- `ConfirmDestructiveModal` reason ≥ 10 chars.
- `Idempotency-Key` header auto-generated.
- `X-Correlation-Id` + `X-CDC-Action` + `X-CDC-Origin: cdc-cms-web` tracing headers.
- Reconciliation error code mapping → Vietnamese messages (`reconErrorMessages.ts`).
- 6 reusable component + 6 custom hook.

cdc-control: 1 file Jinja2 2613 dòng, vanilla JS, Bootstrap modal. Không có abstraction.

### 14.12 V1 + V2 dual mount + deprecation header

cms-service router dual-mount mọi route ở `/api/*` và `/api/v1/*`. Legacy `/api/*` được stamp `Sunset: Tue, 31 Dec 2026 23:59:59 GMT` + `Deprecation` header (RFC 8594).

cdc-control: không có versioning.

---

## Tổng kết một câu

`cdc-control` và `cdc-cms-service` **giải quyết 2 problem khác nhau** dù cùng quản lý Kafka Connect:
- `cdc-control` chuyên sâu về **Kafka Connect connector lifecycle + Mongo schema sync + JDBC sink fan-out**, với UX in-process editable runtime config + encryption at rest.
- `cdc-cms-service` chuyên sâu về **data platform CMS** với source object catalog + provisioning state machine + master tables + mapping rules + reconciliation + alerts + RBAC + audit, target shadow là Postgres không phải Mongo.

Trùng lặp lớn nhất: cả hai đều quản lý Kafka Connect REST + Debezium status. Nhưng cdc-cms-service thiếu hẳn workflow JDBC sink + Mongo schema sync + connection encryption + multi-shadow routing + auto-restart loop. cdc-control thiếu hẳn reconciliation + alerts + auth + audit + state machine + multi-engine source.

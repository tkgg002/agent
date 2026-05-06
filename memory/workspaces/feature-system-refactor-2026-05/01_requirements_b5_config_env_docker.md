# 01 — Requirements (Phase B5: Config-Env Extract + Docker Split)

> **Workspace**: `feature-system-refactor-2026-05`
> **Phase**: B5 (sau B3 hardening đã DONE 2026-05-05 03:49+07)
> **Trigger**: User (anh trainguyen) explicit 2-task assignment 2026-05-05 09:30+07
> **Owner**: Muscle (CC CLI)

---

## 1. User Statement (verbatim)

> "tao cho mày 2 nhiệm vụ nhỏ.
> 1 là mang tất cả các endpoint & secret info như db name, user, pass ra config env (3 repo).
> 2 là mày tách cái docker ra những cái như db source, db desc những cái mà admincms cấu hình vào ra khoi core cdc-worker. để trong cdc-docker-dev tao vừa tạo.
> note: đọc lesson trước tất cả, nhớ từng lỗi lầm mày từng mắc phải.
> sau đó nhớ làm theo core /agent, nhớ lấy claude.md để lấy skill vào làm,
> khi report phải dựa trên kết quả tết tính thực tế, ko đc report láo.
> khi kết thúc luôn kiểm tra các service work mới báo done.
> Luôn có 1 file report_*.md ghi lại những gì thay đổi để tôi check lại"

---

## 2. Goals

### Goal 1 — Endpoint & Secret → ENV (3 repos)
Loại bỏ tất cả secret hardcode (DB host/port/user/pass/database, API keys, JWT secrets, OTEL endpoints, NATS/Redis URLs) khỏi `config-local.yml`. Toàn bộ phải resolve qua biến môi trường lúc runtime, với `.env.example` document đầy đủ key cho dev clone repo new biết cần set gì.

### Goal 2 — Docker Split
Tách `centralized-data-service/docker-compose.yml` thành 2 file:
- **Core (giữ nguyên vị trí)**: chỉ chứa hạ tầng CDC core mà cdc-worker bắt buộc cần để pipeline chạy: NATS, control-plane Postgres (`cdc_dw`), Redis, Kafka, schema-registry, kafka-connect, redpanda-console, otel-collector, kafka-exporter, cdc-worker.
- **Docker dev (mới)** — `cdc-system/cdc-docker-dev/docker-compose.yml`: chứa các thực thể "config-able qua admin/CMS" — source DBs (Postgres source, MongoDB, MySQL, MariaDB) + destination DB (`goopay_dest`) + auth-service Postgres (`postgres:5432`).
- Hai compose dùng **shared external network** `cdc-bridge`.
- Loại bỏ các `depends_on` chéo giữa core và config-able (worker không depend dest, kafka-connect không depend mongo).

---

## 3. Inventory: Hard-coded values cần extract

### 3.1 cdc-auth-service/config/config.go (67 lines)

| Field hardcode trong config-local.yml | Env var sẽ thêm | Default fallback |
|---|---|---|
| `db.host: localhost` | `AUTH_DB_HOST` | giữ nguyên YAML |
| `db.port: 5432` | `AUTH_DB_PORT` | giữ nguyên YAML |
| `db.username: gpay_admin` | `AUTH_DB_USERNAME` | giữ nguyên YAML |
| `db.password: gpay_pass` | `AUTH_DB_PASSWORD` | giữ nguyên YAML |
| `db.database: cdc_auth` | `AUTH_DB_DATABASE` | giữ nguyên YAML |
| `db.sslMode: disable` | `AUTH_DB_SSL_MODE` | giữ nguyên YAML |
| `server.port: :8081` | `AUTH_SERVER_PORT` | giữ nguyên YAML |
| `jwt.secret` | `JWT_SECRET` ✅ ĐÃ CÓ | — |

### 3.2 cdc-cms-service/config/config.go (116 lines)

| Field hardcode trong config-local.yml | Env var sẽ thêm | Ghi chú |
|---|---|---|
| `db.host` / `port` / `username` / `password` / `database` / `sslMode` | `CMS_DB_HOST/PORT/USERNAME/PASSWORD/DATABASE/SSL_MODE` | thiếu hoàn toàn |
| `server.port: :8083` | `CMS_SERVER_PORT` | thiếu |
| `otel.endpoint: http://localhost:14318` | `OTEL_EXPORTER_OTLP_ENDPOINT` | thiếu |
| `nats.url` | `NATS_URL` ✅ ĐÃ CÓ | — |
| `redis.url` | `REDIS_URL` ✅ ĐÃ CÓ | — |
| `jwt.secret` | `JWT_SECRET` ✅ ĐÃ CÓ | — |
| `airbyte.apiKey: trai.nguyen@goopay.vn:knF1jhaPIShkduykN301X1rPbqOzhfe4` | **🚨 CẦN XOÁ** | secret leak |
| `airbyte.workspaceId: ece70fcd-...` | `AIRBYTE_WORKSPACE_ID` (optional) | dead config (struct AppConfig không bind) |
| `controlPlane.url`, `destination.url` | dead config (struct không bind) | xoá khỏi YAML hoặc giữ + comment |

**Critical**: AppConfig struct trong cms-service **KHÔNG có** field Airbyte/ControlPlane/Destination. Block YAML kia đang là **dead config**, nhưng credential thật (`trai.nguyen@goopay.vn:knF1jhaPIShkduykN301X1rPbqOzhfe4`) vẫn nằm trong git → rủi ro leak.

### 3.3 centralized-data-service/config/config.go (540 lines)

Đã có viper override system cho:
`DB_SINK_URL`, `CDC_SYSTEM_DB_URL`, `CDC_CONTROL_PLANE_URL`, `CDC_DESTINATION_URL`, `CDC_SHADOW_DB_URL`, `CDC_MASTER_DB_URL`, `NATS_URL`, `REDIS_URL`, `JWT_SECRET`, `OTEL_ENDPOINT`, `KAFKA_CONNECT_URL`, `DEBEZIUM_CONNECTOR_NAME`, `KAFKA_BROKERS`, `KAFKA_SCHEMA_REGISTRY_URL`, `MONGODB_URL`.

Còn thiếu:
| Field | Env var sẽ thêm | Lý do |
|---|---|---|
| `sources.mongodb_primary` | `SOURCE_DSN_MONGODB_PRIMARY` | DSN này đang trùng `MONGODB_URL` vì legacy field, nên thêm bí danh chính thức để CMS register-source dùng |
| `sources.postgres_primary` | `SOURCE_DSN_POSTGRES_PRIMARY` | thiếu hoàn toàn — đang hardcode `src_user:src_pass@localhost:5435` |
| `airbyte.apiUrl` | `AIRBYTE_API_URL` | (nếu struct có bind — cần verify) |

### 3.4 docker-compose.yml (369 lines, 16 services)

Hardcoded passwords cần thay `${VAR:-default}` syntax:
- `POSTGRES_PASSWORD: gpay_pass` (3 instances)
- `MYSQL_ROOT_PASSWORD: gpay_pass`
- `MARIADB_ROOT_PASSWORD: gpay_pass`
- `MONGO_INITDB_ROOT_PASSWORD: gpay_pass`
- NATS users: `cdc_worker:worker_secret_2026`, `cms_service:cms_secret_2026`

---

## 4. Service Categorization (Task 2 — Docker Split)

### 4.1 KEEP in `centralized-data-service/docker-compose.yml` (Core CDC infra)

| Service | Port | Vai trò |
|---|---|---|
| `gpay-nats` | 14222 / 18222 | message broker — worker + cms cần |
| `gpay-postgres-cdc` | 5433 | control plane `cdc_dw` (cdc_system + shadow_<src>) |
| `gpay-redis` | 16379 | health cache, dedupe state |
| `gpay-kafka` | 19092 / 19093 | event log Debezium → worker |
| `gpay-schema-registry` | 18081 | Avro schemas |
| `gpay-kafka-connect` | 18083 | Debezium worker host |
| `gpay-kafka-exporter` | 9308 | Prometheus metrics |
| `gpay-redpanda-console` | 18088 | Kafka UI |
| `gpay-otel-collector` | 14318 / 14317 | OTLP receiver |
| `cdc-worker` | (no host port) | core consumer |

### 4.2 MOVE to `cdc-docker-dev/docker-compose.yml` (Config-able by CMS/admin)

| Service | Port | Vai trò |
|---|---|---|
| `gpay-postgres` (auth) | 5432 | auth-service DB (`cdc_auth`) — dev only, prod sẽ tách managed DB |
| `gpay-postgres-source` | 5435 | demo source `goopay_source` (admin register-source point at this) |
| `gpay-postgres-dest` | 5434 | demo destination `goopay_dest` (admin register-master point at this) |
| `gpay-mongodb` | 17017 | demo MongoDB source |
| `gpay-mysql` | 13306 | demo MySQL source |
| `gpay-mariadb` | 13307 | demo MariaDB source |

### 4.3 Cross-deps cần loại bỏ

- `cdc-worker.depends_on: gpay-postgres-dest` (line ~159) — worker không cần dest sống lúc boot, chỉ cần khi flush master.
- `gpay-kafka-connect.depends_on: gpay-mongodb` (line ~230) — connect không bắt buộc Mongo có sẵn.
- `gpay-postgres-cdc` migrate run-once container nếu có depends source DB → cắt.

### 4.4 Network

Tạo external network `cdc-bridge`:
- Cả 2 compose `networks: cdc-bridge: external: true`
- User chạy `docker network create cdc-bridge` 1 lần đầu (script `cdc-docker-dev/bootstrap.sh` lo).

---

## 5. Definition of Done (theo CLAUDE.md §3 Verify)

DoD = TẤT CẢ điều kiện dưới đây phải PASS, mới gọi là "Done":

1. ✅ **Code build clean**: 3 repos `go build ./...` đều PASS, không error.
2. ✅ **Config startup clean**: 3 service start với `.env` file (test override) — log không có "missing", "fallback", "empty url".
3. ✅ **Business endpoint live** (exercise-driven, không chỉ /healthz):
   - `cdc-auth-service`: POST `/v1/login` với user mock → 200.
   - `cdc-cms-service`: GET `/v2/sources` (cần JWT) → 200 + JSON list.
   - `centralized-data-service`: insert source row → wait Debezium → shadow row landed (kiểm bằng `psql`).
4. ✅ **Docker split chạy**: 2 compose up độc lập (core trước, dev sau), worker connect được source/dest qua network `cdc-bridge`.
5. ✅ **`.env.example` có đủ key** cho cả 3 repo + docker, dev clone về biết phải set gì.
6. ✅ **Airbyte real key xoá khỏi git**: `git log -p config-local.yml | grep -c knF1jhaPIShkduykN301X1rPbqOzhfe4` báo lịch sử (không xoá được history — chỉ commit removal là đủ trong scope task này).
7. ✅ **Report `report_phase_b5_*.md`** có file vật lý, gồm: changed files list, before/after diffs, verify logs.
8. ✅ **APPEND `05_progress.md`** với entry timestamp + tóm tắt B5 (rule §11 no overwrite).

---

## 6. Constraints & Anti-patterns

- **§11 Memory Protection**: tuyệt đối không overwrite `05_progress.md` cũ. Chỉ APPEND.
- **§12 Brain Code Prohibition**: KHÔNG áp dụng — em là Muscle, được phép sửa source.
- **§7 No Shadow Files**: mọi quyết định phải có file vật lý (4 doc B5 này).
- **Lesson 2026-04-29 "Phase ≠ Workspace"**: KHÔNG tạo workspace mới cho B5 — phase trong workspace cũ.
- **Lesson 2026-04-28 "PASS = exercise-driven"**: verify bằng business endpoint, KHÔNG chỉ /healthz.
- **Lesson 2026-04-17 "Startup log clean"**: phải tail log lần boot đầu, đảm bảo không có warning lạ.
- **Lesson 2026-04-28 tone**: dùng "em — anh", không "tao/mày".
- **Lesson "report láo"**: report phải có evidence thực tế (log, exit code, count). Không speculate.

---

## 7. Risks

| Risk | Mitigation |
|---|---|
| Đổi ENV key, dev quên set → service crash boot | Default fallback giữ giá trị YAML cũ (như `centralized-data-service` đã làm). Env chỉ override nếu set. |
| Docker split phá graph depends_on → boot order race | Dùng external network + healthcheck trên Postgres + retry trong app. |
| Airbyte secret history vẫn trong git log | Out of scope: cảnh báo trong report; full purge cần `git filter-repo` trên 3 repo, anh quyết riêng. |
| Cms-service AppConfig không bind block airbyte/controlPlane → xoá YAML field có nguy cơ break? | Không, vì code không đọc → an toàn. Verify bằng grep `viper.Get("airbyte.")` trong cms-service. |
| Test verify gãy do biến môi trường conflict | Test trong shell mới, `unset` các var trước. |

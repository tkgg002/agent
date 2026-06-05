# 10_gap_analysis.md — Vì sao k8s không approve `config.go` hiện tại

## TL;DR

Manifest `deployments/k8s/cdc-worker-deployment.yaml` chỉ khai báo **6 ENV vars**, trong đó **3 cái sai tên** so với struct path config.go, **1 cái không map tới field nào**, và **thiếu 3 ENV bắt buộc** mà `validateConfig` sẽ chặn → Pod chắc chắn `CrashLoopBackOff` ngay khi `NewConfig()` chạy. Thêm `image: cdc-worker:latest` (tag động) và loạt ENV generic không prefix sẽ bị admission/security policy chặn ở vòng review.

---

## A. Bằng chứng — manifest hiện tại

`deployments/k8s/cdc-worker-deployment.yaml` env block (dòng 22-43):

```yaml
env:
- name: NATS_URL                         # ✅ khớp applyEnvOverrides
  valueFrom: { secretKeyRef: { name: cdc-secrets, key: nats-url } }
- name: DB_SINK_URL                      # ❌ KHÔNG map tới field nào
  valueFrom: { secretKeyRef: { name: cdc-secrets, key: db-url } }
- name: REDIS_URL                        # ✅ khớp applyEnvOverrides
  valueFrom: { secretKeyRef: { name: cdc-secrets, key: redis-url } }
- name: WORKER_POOL_SIZE                 # ❌ AutomaticEnv expect WORKER_POOLSIZE
  value: "10"
- name: BATCH_SIZE                       # ❌ thiếu prefix WORKER_, expect WORKER_BATCHSIZE
  value: "500"
- name: BATCH_TIMEOUT                    # ❌ expect WORKER_BATCHTIMEOUT
  value: "2s"
```

Image: `cdc-worker:latest` (dòng 19) — tag động, fail security gate.

## B. Bằng chứng — config.go expectations

### B1. `validateConfig` (dòng 296-313)

| Check | Field bắt buộc | ENV cần (theo AutomaticEnv hoặc applyEnvOverrides) |
|-------|---------------|----------------------------------------------------|
| `Server.Port == ""` | `cfg.Server.Port` | `SERVER_PORT` (hoặc YAML) |
| `SystemDB.IsEmpty()` | `cfg.SystemDB.Host`, `.Database`, … | `SYSTEMDB_HOST`, `SYSTEMDB_DATABASE`, `SYSTEMDB_USER`, `SYSTEMDB_PASSWORD` |
| `MasterDB.IsEmpty()` | `cfg.MasterDB.Host`, `.Database`, … | `MASTERDB_HOST`, `MASTERDB_DATABASE`, `MASTERDB_USER`, `MASTERDB_PASSWORD` |
| `JWT.Secret == ""` | `cfg.JWT.Secret` | `JWT_SECRET` |
| `Mode==production && Secret==placeholder` | `cfg.JWT.Secret != defaultJWTPlaceholder` | non-placeholder value |

Manifest **không cung cấp bất kỳ ENV nào** trong cột thứ 3 ngoài… không có gì cả → `NewConfig()` return `errors.New("systemDb database config required")` → Pod exit code != 0 → CrashLoopBackOff sau ~10-30s.

### B2. `applyEnvOverrides` (dòng 247-294) — 9 hardcoded keys

| ENV key | Map tới | Manifest có? |
|---------|---------|:------------:|
| `NATS_URL` | `cfg.Nats.URL` | ✅ |
| `REDIS_URL` | `cfg.Redis.URL` | ✅ |
| `JWT_SECRET` | `cfg.JWT.Secret` | ❌ (bắt buộc) |
| `OTEL_ENDPOINT` | `cfg.Otel.Endpoint` | ❌ |
| `KAFKA_CONNECT_URL` | `cfg.Debezium.KafkaConnectURL` | ❌ |
| `DEBEZIUM_CONNECTOR_NAME` | `cfg.Debezium.ConnectorName` | ❌ |
| `KAFKA_BROKERS` | `cfg.Kafka.Brokers` (split `,`) | ❌ |
| `KAFKA_SCHEMA_REGISTRY_URL` | `cfg.Kafka.SchemaRegistryURL` | ❌ |
| `MASTER_KEY` | `cfg.MasterKey` | ❌ |

### B3. AutomaticEnv name derivation rule

Viper với `SetEnvKeyReplacer(".", "_")` map struct path → ENV name **theo nguyên text Go struct tag**, KHÔNG insert underscore giữa camelCase. Ví dụ:

| Struct path | ENV name auto-derived |
|-------------|----------------------|
| `worker.poolSize` | `WORKER_POOLSIZE` (KHÔNG phải `WORKER_POOL_SIZE`) |
| `worker.batchSize` | `WORKER_BATCHSIZE` |
| `worker.batchTimeout` | `WORKER_BATCHTIMEOUT` |
| `systemDb.host` | `SYSTEMDB_HOST` |
| `masterDb.database` | `MASTERDB_DATABASE` |

→ 3 ENV `WORKER_POOL_SIZE` / `BATCH_SIZE` / `BATCH_TIMEOUT` trong manifest **bị Viper bỏ qua hoàn toàn** vì không khớp với derived key.

### B4. `DB_SINK_URL` không tồn tại

Grep `DB_SINK` / `DBSink` trong config.go: **0 match**. ENV này là legacy hoặc copy-paste nhầm — không có field nào hấp thụ, không có code đọc `os.Getenv("DB_SINK_URL")`.

---

## C. Bảng tổng kết "mismatch"

| # | Vấn đề | Tác động | Mức độ |
|---|--------|----------|:------:|
| 1 | `DB_SINK_URL` không map tới field nào | Secret được mount nhưng app bỏ qua; SystemDB/MasterDB vẫn rỗng | 🔴 |
| 2 | `WORKER_POOL_SIZE` ≠ `WORKER_POOLSIZE` | Pool size dùng default, không nhận giá trị 10 | 🟠 |
| 3 | `BATCH_SIZE` thiếu prefix `WORKER_` | Batch size dùng default 500 (trùng giá trị) nhưng config không bind | 🟠 |
| 4 | `BATCH_TIMEOUT` thiếu prefix `WORKER_` | Timeout dùng default `2s` (trùng) nhưng config không bind | 🟠 |
| 5 | Thiếu `SYSTEMDB_*` (HOST/DATABASE/USER/PASSWORD) | `validateConfig` chặn → CrashLoopBackOff | 🔴 |
| 6 | Thiếu `MASTERDB_*` (HOST/DATABASE/USER/PASSWORD) | `validateConfig` chặn → CrashLoopBackOff | 🔴 |
| 7 | Thiếu `JWT_SECRET` | `validateConfig` chặn → CrashLoopBackOff | 🔴 |
| 8 | Thiếu `KAFKA_BROKERS`, `KAFKA_CONNECT_URL`, `KAFKA_SCHEMA_REGISTRY_URL` | Debezium/Kafka path chết runtime khi handler chạm tới | 🟠 |
| 9 | Thiếu `MASTER_KEY` | Encryption ConnectionOverride không hoạt động | 🟠 |
| 10 | `image: cdc-worker:latest` không pin digest/tag | Admission policy thường chặn `:latest`; rollback không deterministic | 🔴 |
| 11 | Generic ENV không prefix project (`MASTER_KEY`, `JWT_SECRET`, `NATS_URL`, `REDIS_URL`) | Trùng namespace với sidecar/platform; security review từ chối | 🟠 |
| 12 | `CONNECTION_OVERRIDE_*` quét động `os.Environ()` | Comment code ghi "dev/local only"; không deterministic, khó audit | 🟠 |
| 13 | KHÔNG có `livenessProbe`/`readinessProbe` chia tách | Health check `/health` đơn lẻ, không phân biệt deps (DB/Kafka) ready | 🟡 |
| 14 | KHÔNG có Helm/Kustomize → mỗi env phải maintain manifest riêng | Drift giữa dev/staging/prod | 🟡 |

Legend: 🔴 chặn deploy / CrashLoopBackOff / fail review · 🟠 silent bug / runtime fail · 🟡 vận hành kém.

---

## D. Nguyên nhân gốc

1. **Hai paradigm khác nhau gặp nhau**: Manifest viết theo mô hình "worker đơn DB sink" (`DB_SINK_URL`), nhưng `config.go` mới đã refactor sang mô hình "service đa cluster" (SystemDB + MasterDB + ShadowDB + ConnectionOverrides). Manifest chưa được cập nhật theo refactor config.
2. **Naming convention không match giữa 2 file**: Manifest theo SCREAMING_SNAKE phân tách "ngữ nghĩa" (POOL_SIZE), config theo Viper auto-derive từ struct camelCase (POOLSIZE).
3. **Không có nguồn truth duy nhất**: Không có Helm `values.yaml` / Kustomize overlay / ConfigMap để map 1-1 ENV name với struct path; mỗi nơi tự đặt tên.

---

## F. So sánh 2 pattern: `centralized-data-service` vs `cdc-cms-service`

### F1. Bảng đối chiếu

| Khía cạnh | `centralized-data-service/config.go` (CDS) | `cdc-cms-service/config/config.go` (CMS) |
|-----------|-------------------------------------------|------------------------------------------|
| **Prefix Viper** | ❌ KHÔNG có `SetEnvPrefix` | ✅ `v.SetEnvPrefix("CMS")` |
| **Bind cơ chế** | `AutomaticEnv` + `applyEnvOverrides` hardcoded | `AutomaticEnv` + `envBinds` map + `v.BindEnv` per key |
| **camelCase → ENV** | Raw join: `worker.poolSize` → `WORKER_POOLSIZE` ❌ confusing | Tự định nghĩa snake: `db.sslMode` → `CMS_DB_SSL_MODE` ✅ rõ ràng |
| **Multi-alias per key** | ❌ chỉ 1 ENV per field (hardcoded) | ✅ `map[string][]string` — mỗi key có thể có 2-3 alias |
| **Backward compat** | ❌ legacy `NATS_URL` chỉ tồn tại trong `applyEnvOverrides` | ✅ alias kiểu `{CMS_NATS_URL, NATS_URL}` — cả mới và cũ đều ăn |
| **Dynamic scan** | ✅ `CONNECTION_OVERRIDE_*` (comment "dev/local only") | ❌ không có (mọi key tường minh) |
| **Validation** | SystemDB + MasterDB + JWT.Secret | DB + ShadowDB + JWT.Secret |
| **K8s deployment** | 🔴 "ko ăn" — tên ENV không đoán được | 🟢 "vẫn ăn" — ENV name tường minh, có alias |

### F2. Tại sao pattern CMS "ăn" được trên k8s

1. **Prefix `CMS_` toàn cục** → không xung đột với reserved keys của platform/sidecar (`JWT_SECRET`, `NATS_URL`…).
2. **Snake-case tường minh trong `envBinds`**: `db.sslMode` → `CMS_DB_SSL_MODE` (manifest đọc vào hiểu ngay).
3. **Alias legacy** (`NATS_URL`, `REDIS_URL`, `JWT_SECRET`, `MASTER_KEY`, `OTEL_EXPORTER_OTLP_ENDPOINT`) → manifest cũ vẫn dùng được trong giai đoạn migration.
4. **Audit dễ**: 1 map `envBinds` 40+ entries → đọc 1 phát biết hết ENV name; platform team review pass.
5. **Không có dynamic prefix scan** → deterministic, ai cũng predict được pod sẽ ăn ENV nào.

### F3. Khoảng cách khi port pattern CMS sang CDS

CDS có 3 thứ CMS không có, cần xử lý đặc biệt khi port:

| Tính năng CDS | Phương án adapt |
|---------------|-----------------|
| `SystemDB` + `MasterDB` (2 cluster ngoài `ShadowDB`) | Thêm 2 block trong `envBinds`: `systemDb.*` → `CDS_SYSTEM_DB_*`, `masterDb.*` → `CDS_MASTER_DB_*` |
| `Debezium` + `Kafka` (Brokers slice, SchemaRegistry, TopicPrefix) | Thêm block `debezium.*`, `kafka.*`. Brokers slice: vẫn dùng `strings.Split(",")` sau Unmarshal (giống CDS hiện tại) |
| `ConnectionOverrides` map (dynamic `CONNECTION_OVERRIDE_*` scan) | Giữ riêng hàm `applyConnectionOverrides(cfg)` chạy sau `Unmarshal`. KHÔNG bind qua Viper. Có thể rename prefix → `CDS_CONNECTION_OVERRIDE_*` cho nhất quán |

## G. Tham chiếu

- `centralized-data-service/config/config.go:181-182` — AutomaticEnv + replacer.
- `centralized-data-service/config/config.go:247-294` — applyEnvOverrides + CONNECTION_OVERRIDE_*.
- `centralized-data-service/config/config.go:296-313` — validateConfig.
- `centralized-data-service/deployments/k8s/cdc-worker-deployment.yaml:1-82` — manifest hiện tại.
- `centralized-data-service/README.md` — section "Environment variables" đã được update phase trước.
- `cdc-cms-service/config/config.go:112-114` — `SetEnvPrefix("CMS")` + replacer + AutomaticEnv.
- `cdc-cms-service/config/config.go:116-165` — `envBinds` map 40+ entries với multi-alias.
- `cdc-cms-service/config/config.go:187-207` — validateConfig.

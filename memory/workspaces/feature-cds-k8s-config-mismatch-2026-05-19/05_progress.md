# 05_progress.md — Audit log (APPEND-ONLY)

## 2026-05-19 — Khởi tạo workspace + phân tích mismatch k8s vs config.go

- **User input**: "`config/config.go` hiện tại trên k8s ko approve kiểu này".
- **Hành động**:
  1. Re-read `deployments/k8s/cdc-worker-deployment.yaml` (82 dòng).
  2. Re-read `config/config.go` dòng 240-313 (applyEnvOverrides + validateConfig).
  3. Verify không có Helm/Kustomize: `ls deployments/helm` & `ls deployments/kustomize` → No such file or directory.
  4. Tạo workspace `feature-cds-k8s-config-mismatch-2026-05-19/`.
  5. Viết `00_context.md` (bối cảnh).
  6. Viết `10_gap_analysis.md` (14 mismatch, bảng A/B/C/D/E).
  7. Viết `09_tasks_solution_proposal.md` (3 hướng + khuyến nghị + 5 câu hỏi pending).
- **Không sửa code**: 0 thay đổi `.go` / `.yaml` / `.md` source (chỉ Write vào workspace).
- **Phát hiện chính**:
  - 🔴 3 ENV thiếu chặn `validateConfig`: SYSTEMDB_*, MASTERDB_*, JWT_SECRET.
  - 🔴 `DB_SINK_URL` không map tới field nào.
  - 🟠 `WORKER_POOL_SIZE` / `BATCH_SIZE` / `BATCH_TIMEOUT` sai naming (Viper expect `WORKER_POOLSIZE` / `WORKER_BATCHSIZE` / `WORKER_BATCHTIMEOUT`).
  - 🔴 Image `:latest` không pin.
- **Pending**: user chọn 1 trong 3 hướng (sửa manifest only / thêm prefix `CDS_` / Helm chart).

## Skills sử dụng (round 1)

- **Read** (config.go + manifest) — gather ground truth.
- **Bash ls** — verify không có helm/kustomize folder.
- **Workspace governance** (CLAUDE.md §7) — full doc set (00, 05, 07, 09, 10).
- **Brain Code Prohibition** (§12) — chỉ document, không sửa code.
- **Plan & Verify** (§3) — đề xuất 3 hướng có ưu/nhược.
- **Append-only audit log** (§11).

## 2026-05-19 (cont) — User hỏi: có nên port pattern `cdc-cms-service` không

- **User input**: "no ko ăn env kiểu này. `cdc-cms-service/config/config.go` còn kiểu này thì vẫn ăn bình thường. có nên chuyển về kiểu trên cdc-cms-service ko".
- **Hành động**:
  1. Read `cdc-cms-service/config/config.go` (208 dòng) — xác nhận pattern: `SetEnvPrefix("CMS")` + `envBinds` map 40+ entries với multi-alias.
  2. Update `10_gap_analysis.md` — thêm section F (so sánh CDS vs CMS) + section G (tham chiếu).
  3. Update `09_tasks_solution_proposal.md` — thêm Hướng 4 "Port pattern cdc-cms-service" (đánh dấu ⭐ KHUYẾN NGHỊ).
- **Phát hiện**:
  - Pattern CMS có **3 cơ chế cứu** mà CDS thiếu:
    1. Prefix `CMS_` toàn cục.
    2. `envBinds` map tường minh (snake_case dễ đoán).
    3. **Multi-alias per key** — backward compat zero-downtime.
  - CDS có **3 tính năng unique** mà CMS không có:
    1. Multi-DB cluster (SystemDB + MasterDB).
    2. Debezium + Kafka full config (Brokers slice, SchemaRegistry, TopicPrefix).
    3. `CONNECTION_OVERRIDE_*` dynamic scan.
  - Adapt: port `envBinds` cho 3 cụm DB + Debezium/Kafka block, **giữ riêng** `applyConnectionOverrides` chạy sau Unmarshal.
- **Khuyến nghị**: **Hướng 4** — port pattern + alias legacy keys. Manifest hiện tại không cần sửa nhiều (alias giữ `WORKER_POOL_SIZE` ăn được), chỉ cần bổ sung SYSTEM_DB_*, MASTER_DB_*, JWT_SECRET.

## Skills sử dụng (round 2)

- **Read** (`cdc-cms-service/config/config.go`) — verify pattern user reference.
- **Edit** (3 files trong workspace) — append section F, Hướng 4, progress log.
- **Plan & Verify** (§3) — so sánh head-to-head 2 pattern, recommend với evidence.
- **Brain Code Prohibition** (§12) — code snippet chỉ trong workspace doc, không chạm `centralized-data-service/config/config.go`.

## 2026-05-19 (cont) — Muscle thực thi Hướng 4 (prod gấp)

- **User input**: "làm đi, tao cần chạy prod bây giờ".
- **Tự chốt** 3 câu hỏi pending (do prod gấp, áp dụng "Bug Fixing Tự chủ" §2):
  - Hướng 4 only (Helm để sprint sau).
  - Giữ tên `CONNECTION_OVERRIDE_*` (không rename).
  - Giữ alias legacy lâu dài.
- **TaskCreate**: 5 task tracking (43..47).

### Bước 1 — Refactor `config/config.go` (task 43)

- Read full 313 dòng config.go gốc + config_test.go + config-production.yml + grep cmd/* usage.
- `Write` ghi đè config.go mới (399 dòng):
  - `v.SetEnvPrefix("CDS")` + replacer + `AutomaticEnv` (fix thứ tự).
  - `envBinds` map 65 entries (cấp module-level var) — bao quát tất cả field của AppConfig.
  - 4 cụm DB (SystemDB/ShadowDB/MasterDB/ReadReplica) × 6 field = 24.
  - Worker 8, Nats 6, Redis 3, JWT 2, Kafka 5, OTEL flat 4, OTEL nested logs 8, Debezium 6, MongoDB 1, Server 3, DBPool 3, MasterKey 1 = 65.
  - Alias legacy cho 12 ENV phổ biến (NATS_URL, REDIS_URL, JWT_SECRET, MASTER_KEY, WORKER_POOL_SIZE, BATCH_SIZE, BATCH_TIMEOUT, KAFKA_BROKERS, KAFKA_SCHEMA_REGISTRY_URL, KAFKA_CONNECT_URL, DEBEZIUM_CONNECTOR_NAME, MONGODB_URL, OTEL_EXPORTER_OTLP_ENDPOINT, OTEL_ENDPOINT) + 18 alias DB tách (SYSTEMDB_*, MASTERDB_*, SHADOWDB_*).
  - Tách `applyEnvOverrides` (gộp) → `applyConnectionOverrides` (chỉ giữ dynamic prefix scan `CONNECTION_OVERRIDE_*`).
  - `validateConfig` cập nhật message lỗi để gợi ý ENV mới (`CDS_SYSTEM_DB_HOST` + …).
- Verify build: `go build ./config/...` ok, `go vet ./config/...` clean.
- Verify tests: `go test -v -run 'TestUnmarshalKafka' ./config/...` → 4/4 PASS.

### Bước 2 — Cập nhật manifest k8s (task 44)

- `Write` ghi đè `deployments/k8s/cdc-worker-deployment.yaml` (mới 124 dòng):
  - Pin image `cdc-worker:v1.0.0` thay `:latest`.
  - Thêm `imagePullPolicy: IfNotPresent`.
  - Thêm `CFG_PATH=./config/config-production.yml`.
  - 6 ENV `CDS_SYSTEM_DB_*` (5 secretKeyRef + 1 sslMode value).
  - 6 ENV `CDS_MASTER_DB_*` (5 secretKeyRef + 1 sslMode value).
  - 6 ENV `CDS_SHADOW_DB_*` (5 secretKeyRef + 1 sslMode value).
  - `CDS_JWT_SECRET`, `CDS_MASTER_KEY` (secretKeyRef).
  - `CDS_NATS_URL`, `CDS_REDIS_URL` (secretKeyRef — rename từ legacy không prefix).
  - 4 ENV Kafka/Debezium (`CDS_KAFKA_BROKERS`, `CDS_KAFKA_SCHEMA_REGISTRY_URL`, `CDS_DEBEZIUM_KAFKA_CONNECT_URL`, `CDS_DEBEZIUM_CONNECTOR_NAME`).
  - `CDS_OTEL_ENDPOINT` value cố định.
  - 3 worker tuning `CDS_WORKER_POOL_SIZE/BATCH_SIZE/BATCH_TIMEOUT`.
  - **XÓA** `DB_SINK_URL` (không có field).
- Giữ nguyên: HPA, livenessProbe, readinessProbe, resources, replicas.
- **TODO devops**: bổ sung 20+ key mới vào Secret `cdc-secrets` (system-db-host, master-db-host, jwt-secret, master-key, …).

### Bước 3 — Cập nhật README (task 45)

- Edit section "Environment variables" (line 156-294 cũ) → rewrite hoàn toàn:
  - 7 sub-section (1-7).
  - Quy ước đặt tên (canonical CDS_ + snake-case + alias).
  - 8 block tables (Server, DBPool, 4 DB, MasterKey, NATS/Redis/JWT, Worker, Kafka, OTEL, MongoDB/Debezium).
  - `applyConnectionOverrides` section mô tả dynamic scan.
  - `validateConfig` 5 rule + message lỗi mới.
  - Migration table so sánh trước/sau.
  - Khẳng định "100% backward compat" — manifest cũ vẫn ăn nhờ alias.

### Bước 4 — Verify toàn repo (task 46)

- `go vet ./...` → clean (no output).
- `go build ./...` → ok (no output).
- `go test ./...` → 8 package PASS, 12 package no test files.
- **Smoke test runtime**: tạo `/tmp/cds_env_smoke.go`, set 5 ENV CDS_*, 1 legacy NATS_URL, 1 mix CDS_REDIS_URL + WORKER_POOL_SIZE, 1 CONNECTION_OVERRIDE_GOOPAY → `config.NewConfig()` chạy với `config-production.yml`. Kết quả PASS:
  - `SystemDB.Host="10.0.0.1"` (từ `CDS_SYSTEM_DB_HOST`)
  - `MasterDB.Host="10.0.0.2"` (từ `CDS_MASTER_DB_HOST`)
  - `JWT.Secret="real-prod-secret-not-placeholder"` (từ `CDS_JWT_SECRET`)
  - `Nats.URL="nats://legacy:4222"` (từ alias `NATS_URL`)
  - `Redis.URL="redis://canonical:6379"` (từ `CDS_REDIS_URL`)
  - `Worker.PoolSize=42` (từ alias `WORKER_POOL_SIZE`)
  - `ConnectionOverrides=map[goopay:mongodb://prod.internal:27017/?replicaSet=rs0]`
- Cleanup `/tmp/cds_env_smoke.go`.

### Bước 5 — Audit log + status (task 47)

- File hiện tại đang append.

## Skills sử dụng (round 3 — Muscle execution)

- **TaskCreate / TaskUpdate** × 5 (tracking 5 bước thực thi).
- **Read** (config.go, config_test.go, config-production.yml, README, manifest k8s, grep cmd/* usage).
- **Write** (config.go, cdc-worker-deployment.yaml).
- **Edit** (README.md section ENV — 138 dòng cũ thay bằng 130 dòng mới).
- **Bash** (`go vet`, `go build`, `go test`, smoke test runtime).
- **Plan & Verify §3** — verify build + test + smoke trước khi báo done.
- **Bug Fixing Tự chủ §2** — tự chốt 3 câu hỏi pending do prod gấp, không hỏi ngược.
- **Simplicity First §6** — không over-engineer (chỉ port pattern CMS, không thêm tính năng).
- **Workspace governance §7** — append vào workspace cùng chủ đề.

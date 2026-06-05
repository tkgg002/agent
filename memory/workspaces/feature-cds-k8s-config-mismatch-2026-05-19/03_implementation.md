# 03_implementation.md — Thực thi Hướng 4 (port pattern CMS)

## Quyết định tự chốt (do prod gấp)

1. **Hướng 4 only** — Helm chart để sprint sau.
2. **Giữ tên `CONNECTION_OVERRIDE_*`** — không rename, tránh phá manifest cũ.
3. **Giữ alias legacy lâu dài** — zero harm, zero-downtime migration.

## Bước 1 — Refactor `config/config.go`

### Thay đổi chính

| Thay đổi | Trước | Sau |
|----------|-------|-----|
| Prefix Viper | (không có) | `v.SetEnvPrefix("CDS")` |
| Bind ENV | `applyEnvOverrides` hardcode 9 key | `envBinds map[string][]string` 65 entry + `v.BindEnv` |
| Multi-alias | ❌ 1 ENV/field | ✅ 1-3 ENV/field (CDS prefix + legacy) |
| Dynamic scan | trong `applyEnvOverrides` | tách `applyConnectionOverrides(cfg)` riêng |
| Order Viper setup | `AutomaticEnv()` trước replacer | replacer trước `AutomaticEnv()` (fix) |

### envBinds full map (65 entries, gom theo block)

Block-level coverage:
- Server (3), DBPool (3)
- SystemDB / ShadowDB / MasterDB / ReadReplica (4 × 6 = 24)
- MasterKey (1)
- Nats (6), Redis (3), Worker (8), JWT (2)
- Kafka (5), OTEL flat (4), OTEL nested logs (8)
- MongoDB (1), Debezium (6)

Alias legacy được thêm cho 12 ENV phổ biến đã dùng ở manifest cũ:
- `NATS_URL`, `REDIS_URL`, `JWT_SECRET`, `MASTER_KEY`
- `SYSTEMDB_HOST/PORT/USER/PASSWORD/DATABASE/SSL_MODE`
- `MASTERDB_HOST/PORT/USER/PASSWORD/DATABASE/SSL_MODE`
- `SHADOWDB_HOST/PORT/USER/PASSWORD/DATABASE/SSL_MODE`
- `WORKER_POOL_SIZE`, `BATCH_SIZE`, `BATCH_TIMEOUT` (manifest hiện có)
- `KAFKA_BROKERS`, `KAFKA_SCHEMA_REGISTRY_URL`, `KAFKA_CONNECT_URL`, `DEBEZIUM_CONNECTOR_NAME`
- `OTEL_ENDPOINT`, `OTEL_EXPORTER_OTLP_ENDPOINT`
- `MONGODB_URL`

### `applyConnectionOverrides` (tách riêng)

Logic giữ nguyên 100% — chỉ rename hàm + chạy riêng sau `Unmarshal`. Vẫn dùng prefix `CONNECTION_OVERRIDE_` (không đổi để manifest cũ không phá).

## Bước 2 — Cập nhật `deployments/k8s/cdc-worker-deployment.yaml`

| Thay đổi | Lý do |
|----------|-------|
| Pin `image: cdc-worker:v1.0.0` thay `:latest` | Security gate / rollback deterministic |
| **Bỏ** `DB_SINK_URL` | Không map tới field nào |
| **Thêm** `CDS_SYSTEM_DB_*` (6 key) từ secretKeyRef | `validateConfig` bắt buộc |
| **Thêm** `CDS_MASTER_DB_*` (6 key) | `validateConfig` bắt buộc |
| **Thêm** `CDS_SHADOW_DB_*` (6 key) | Cần cho shadow database |
| **Thêm** `CDS_JWT_SECRET`, `CDS_MASTER_KEY` | bắt buộc / encryption |
| **Thêm** Kafka block (`CDS_KAFKA_BROKERS`, `CDS_KAFKA_SCHEMA_REGISTRY_URL`, `CDS_DEBEZIUM_KAFKA_CONNECT_URL`, `CDS_DEBEZIUM_CONNECTOR_NAME`) | CDC core dependencies |
| **Giữ** `NATS_URL`, `REDIS_URL` (legacy name) | Vẫn ăn qua alias, không cần đổi |
| **Giữ** `WORKER_POOL_SIZE`, `BATCH_SIZE`, `BATCH_TIMEOUT` (legacy name) | Vẫn ăn qua alias |
| **Cần** Secret `cdc-secrets` bổ sung key tương ứng (devops job) | Không thuộc scope code |

## Bước 3 — Cập nhật `README.md`

Section "Environment variables" → rewrite theo pattern mới:
- Liệt kê `envBinds` map đầy đủ
- Ghi rõ alias legacy nào còn ăn
- Ghi rõ `CONNECTION_OVERRIDE_*` vẫn giữ prefix cũ
- So sánh old vs new pattern

## Bước 4 — Verify

- `cd centralized-data-service && go build ./...` — build pass
- `go test ./config/...` — 4 existing kafka topic tests vẫn pass
- Smoke: `CDS_SYSTEM_DB_HOST=x CDS_MASTER_DB_HOST=y CDS_JWT_SECRET=z ./worker` → log "config path" + start

## Bước 5 — Append `05_progress.md`

Audit log đầy đủ steps đã thực hiện.

## Định nghĩa hoàn thành (DoD)

- [ ] `config.go` compile, không import warning, không lint.
- [ ] `envBinds` map đủ 65 entries.
- [ ] 4 test cũ vẫn pass (`TestUnmarshalKafka_*`).
- [ ] Manifest `cdc-worker-deployment.yaml` có đủ SYSTEM/MASTER/SHADOW DB + JWT + MASTER_KEY + Kafka.
- [ ] Image pinned (không `:latest`).
- [ ] README section "Environment variables" reflect pattern mới.
- [ ] `05_progress.md` append đầy đủ.

# 01_requirements — Multi-Engine Unified Pipeline

> Phase suffix: `multi_engine_unified`.
> Đối ứng: `00_context_multi_engine_unified.md`.

## R1. Functional Requirements

### R1.1 Multi-prefix Kafka topic discovery (cdc-worker)
- `KafkaConfig.TopicPrefix` đổi từ `string` → `[]string` (alias `topicPrefixes`).
- Backward-compat: parse `topicPrefix: "cdc.gpay"` (scalar YAML) thành `[]string{"cdc.gpay"}`.
- Worker discover **union** topic match bất kỳ prefix nào trong list.
- Default config (`config-local.yml`):
  ```yaml
  kafka:
    topicPrefixes:
      - cdc.gpay        # PG source rig (E2E)
      - cdc.goopay      # Mongo (production-intent)
      - cdc.mariadb     # MariaDB (sắp có, prefix giả định)
  ```

### R1.2 Multi-engine registry filter
- `RegistryService.GetDebeziumTables()` giữ semantics. Bổ sung `GetDebeziumNamespaces() []struct{Engine, DB, Schema, Object}` để filter chính xác hơn cho Mongo (db.collection vs PG schema.table vs MariaDB db.table).
- `kafka_consumer.go` filter: nếu topic `cdc.<prefix>.<db>.<object>` → lookup registry theo `(source_engine_type, source_database/source_namespace, source_object_name)` thay vì chỉ theo `tableName` (avoid collision khi 2 engine cùng tên collection/table).

### R1.3 MariaDB capture infrastructure
- `docker-compose.yml`: thêm service `gpay-mariadb` (image `mariadb:10.11`, binlog ROW format, server_id=10, gtid_strict_mode=1).
- Init SQL: tạo DB `goopay_legacy_maria` + table sample `legacy_orders` để smoke test.
- Debezium connector spec mới `cdc-mariadb-source` (file `deployments/connectors/cdc-mariadb-source.json`) dùng `io.debezium.connector.mysql.MySqlConnector` + `database.server.name: gpay-mariadb`, `topic.prefix: cdc.mariadb`.
- KHÔNG tự deploy connector trong CI; deploy thủ công sau khi code merge (giảm blast radius).

### R1.4 Toggle Auto/Manual — full stack
**cdc-worker** (đã ship, cần verify):
- `provisioning_orchestrator.SetMode` đảm bảo D1 fan-out hoạt động cho mọi engine (smoke test 1 row mỗi engine).

**cms-api** (gap):
- `GET /api/v1/cms/sources` response **MUST** expose `provisioning_mode`, `provisioning_state`, `source_engine_type`.
- `POST /api/v1/cms/sources/:id/provisioning/mode` đã có — không đổi.
- Bổ sung `GET /api/v1/cms/sources/:id/provisioning` đã có (snapshot) — verify FE consume được.
- Idempotency-Key middleware đảm bảo duplicate POST không double-fan-out.

**cms-fe** (gap chính):
- `TableRegistry.tsx` thêm 3 column:
  - **Engine** — badge color theo `source_engine_type` (postgresql=blue, mongodb=green, mysql/mariadb=orange).
  - **Mode** — Ant Switch Auto/Manual + tooltip "Auto: orchestrator tự advance state. Manual: operator click `/advance`."
  - **State** — chip color-coded (`draft|*_pending|*_active|running|paused|failed|archived`).
- Hook mới `useProvisioningMode(sourceId)`:
  - mutation: `POST /api/v1/cms/sources/:id/provisioning/mode` + header `Idempotency-Key: prov-mode-${id}-${ts}`.
  - onSuccess: invalidate query `['sources']`.
  - onError 422/409: show Ant message, KHÔNG retry tự động (CAS conflict cần refresh).
- Confirm dialog khi flip `auto → manual` lúc `provisioning_state ∈ {*_pending}`: cảnh báo "Đang có step đang chạy, switch sang Manual sẽ giữ state hiện tại, KHÔNG cancel cmd đang in-flight."
- Engine filter dropdown ở toolbar TableRegistry.

### R1.5 E2E smoke test
1. PG source `orders_e2e_d_v5` (đã có, registry id=26): flip Manual → Auto → expect orchestrator kick `Advance` → state machine drive đến `running`.
2. Mongo source mới: tạo registry row cho `payment-bill-service.payment-bills` (engine=mongodb, sync_engine=debezium). Flip Auto → expect shadow CREATE + master bind + mapping discover + schedule enable.
3. MariaDB source mới: stand up `gpay-mariadb`, deploy `cdc-mariadb-source` connector, tạo registry row cho `goopay_legacy_maria.legacy_orders`. Flip Auto → giống step 2.

## R2. Non-Functional Requirements

### R2.1 Backwards compatibility
- YAML config cũ (`topicPrefix: "cdc.gpay"` string) PHẢI parse được — không break worker đang chạy.
- Existing PG path (E2E rig) PHẢI giữ nguyên hành vi — 0 regression.

### R2.2 Observability
- Log `kafka consumer started` mở rộng: `prefixes []string`, `topics_per_prefix map[string]int`, `engines_active []string`.
- Prometheus counter `cdc_kafka_topics_discovered{prefix=...}`.

### R2.3 Security
- Toggle endpoint giữ gate `RequireOpsAdmin` — không đổi.
- FE không trust client-side `provisioning_mode` change; mọi flip qua API.

### R2.4 Idempotency
- POST `/mode` với same `Idempotency-Key` trong 60s → no-op return cached response.

## R3. Out-of-scope (phase này KHÔNG làm)

- Tự động deploy MariaDB Debezium connector qua CI/CD.
- Migrate dữ liệu legacy V1 (`legacy_payments`, `legacy_refunds`) sang V2 (xử lý ở phase riêng).
- KHÔNG đụng `prune_legacy_v1_bindings.sql` (P3 plan đã bị user reject).
- KHÔNG đổi schema `cdc_system.source_object_registry` (đã đủ với migration 047).

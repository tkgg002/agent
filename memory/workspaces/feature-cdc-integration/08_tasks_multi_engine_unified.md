# 08_tasks — Multi-Engine Unified Pipeline

> Đối ứng: `02_plan_multi_engine_unified.md`.
> Mỗi task có DoD + verify command.

## L1 — cdc-worker

### T1.1 Config struct + dual-decode TopicPrefix
- File: `centralized-data-service/config/config.go`
- Action: đổi `TopicPrefix string` → `TopicPrefix []string` (giữ tên cũ cho ít diff). Custom `UnmarshalYAML` cho `KafkaConfig`: scalar → `[]string{scalar}`, sequence → as-is. Alias `topicPrefixes` parse cùng node.
- DoD: 2 unit test pass — scalar input và list input đều ra `[]string`.
- Verify: `cd centralized-data-service && go test ./config/... -run UnmarshalKafka -v`.

### T1.2 Kafka consumer multi-prefix discovery
- File: `internal/handler/kafka_consumer.go` (function `discoverTopics`).
- Action: loop `kc.config.TopicPrefix` → gộp set topic match. Log per-prefix count. Filter qua `RegistryService.GetDebeziumNamespaces()` (xem T1.3).
- DoD: unit test với mock broker — 3 prefix → trả union; 0 collision khi 2 engine cùng object name.
- Verify: `go test ./internal/handler/... -run KafkaConsumerDiscover -v`.

### T1.3 RegistryService.GetDebeziumNamespaces
- File: `internal/service/registry_service.go`.
- Action: thêm method trả `[]struct{Engine, Database, Namespace, Object string}` cho mọi row `is_active=true AND sync_engine IN ('debezium','both')`. Cache giống `registryCache`.
- DoD: unit test seed 3 row (mongo+pg+mariadb) → trả 3 tuple đúng.
- Verify: `go test ./internal/service/... -run RegistryNamespaces -v`.

### T1.4 Update config-local.yml
- File: `config/config-local.yml`.
- Action: đổi `topicPrefix: cdc.gpay` → `topicPrefixes: [cdc.gpay, cdc.goopay, cdc.mariadb]`.
- DoD: worker boot không error.
- Verify: `go run ./cmd/worker --validate-config`.

## L4 — infra MariaDB

### T4.1 docker-compose service
- File: `centralized-data-service/docker-compose.yml`.
- Action: thêm service `gpay-mariadb` (image `mariadb:10.11`, env MYSQL_ROOT_PASSWORD/MYSQL_DATABASE=goopay_legacy_maria, command `--server-id=10 --log-bin --binlog-format=ROW --gtid-strict-mode=1 --binlog-row-image=FULL`).
- DoD: `docker compose up -d gpay-mariadb` healthy.
- Verify: `docker exec gpay-mariadb mysql -u root -p${PASS} -e "SHOW VARIABLES LIKE 'log_bin'"` → ON.

### T4.2 Init SQL seed
- File: `centralized-data-service/deployments/mariadb/init/01_seed.sql`.
- Action: `CREATE DATABASE IF NOT EXISTS goopay_legacy_maria; USE goopay_legacy_maria; CREATE TABLE legacy_orders (id BIGINT PRIMARY KEY AUTO_INCREMENT, amount INT, created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP);` + INSERT 5 row.
- DoD: container up → table có 5 row.

### T4.3 Debezium connector spec
- File: `centralized-data-service/deployments/connectors/cdc-mariadb-source.json`.
- Action: spec MySqlConnector trỏ `gpay-mariadb:3306`, `topic.prefix=cdc.mariadb`, `database.include.list=goopay_legacy_maria`, `table.include.list=goopay_legacy_maria.legacy_orders`, schema-history Kafka topic riêng.
- DoD: file commit, KHÔNG auto-deploy.
- Verify (manual): `curl -X POST -H 'Content-Type: application/json' --data @cdc-mariadb-source.json http://localhost:18083/connectors`.

### T4.4 Migration seed registry row
- File: `centralized-data-service/migrations/cdc/049_mariadb_seed_legacy_orders.sql`.
- Action: INSERT 1 row `source_object_registry` cho `legacy_orders` (engine=mysql, sync_engine=debezium, is_active=false, profile_status=draft) + idempotent guard (`ON CONFLICT (object_code) DO NOTHING`).
- DoD: psql apply 2 lần không lỗi.

## L2 — cms-api

### T2.1 List sources expose 3 field
- File: `cdc-cms-service/internal/api/source_handler.go` (xác định khi vào việc).
- Action: extend response DTO `provisioning_mode`, `provisioning_state`, `source_engine_type`.
- DoD: `curl /api/v1/cms/sources` JSON có 3 field.

### T2.2 Idempotency-Key middleware
- File: `cdc-cms-service/internal/middleware/idempotency.go` (new nếu chưa có).
- Action: in-memory + Redis fallback (TTL 60s) cache `(method+path+key) → response`. Apply lên route `/provisioning/mode`.
- DoD: 2 POST same Idempotency-Key trong 60s → 1 lần thực thi, 2 lần response giống nhau.

### T2.3 Smoke 3 row /mode
- File: doc command trong `09_tasks_solution_multi_engine_unified.md`.
- Action: curl flip mode 3 row (PG/Mongo/MariaDB), assert `200` + DB UPDATE.
- DoD: 3 lần PASS.

## L3 — cms-fe

### T3.1 Type extension
- File: `cdc-cms-web/src/types/index.ts` (xác định khi vào việc).
- Action: thêm `provisioning_mode?: 'auto'|'manual'`, `provisioning_state?: string`, `source_engine_type?: string` vào `SourceObjectRow`.

### T3.2 Hook useProvisioningMode
- File: `cdc-cms-web/src/hooks/useProvisioningMode.ts` (new).
- Action: TanStack Query mutation `(id, mode) => POST /api/v1/cms/sources/${id}/provisioning/mode` + Idempotency-Key + onSuccess invalidate `['sources']`.

### T3.3 TableRegistry columns
- File: `cdc-cms-web/src/pages/TableRegistry.tsx`.
- Action: 3 column mới (Engine badge, Mode Switch, State Tag) + filter Engine ở toolbar.
- DoD: Vite reload → page render đủ; click Switch → API gọi đúng → row refresh.

### T3.4 Confirm dialog
- File: `cdc-cms-web/src/pages/TableRegistry.tsx`.
- Action: Modal.confirm khi flip auto→manual lúc state ∈ `*_pending|failed`.

## L5 — E2E

### T5.1 PG smoke
- Row id=26 (`orders_e2e_d_v5`). Set `provisioning_mode=manual`, state đã `running`. Flip Auto → orchestrator no-op (đã terminal-ish). Pass nếu DB UPDATE thành công + log `mode_change` audit.

### T5.2 Mongo smoke
- INSERT registry row mới `mongo_payment_bills_v2` (engine=mongodb, sync_engine=debezium, source_database=`payment-bill-service`, source_namespace=`payment-bill-service`, source_object_name=`payment-bills`). Set `provisioning_mode=auto` từ ban đầu, `provisioning_state=draft`.
- Restart worker (đã có 3 prefix). Worker discover topic `cdc.goopay.payment-bill-service.payment-bills` → orchestrator advance shadow_pending → ... → running.
- DoD: `dw_*.payment_bills_*` (master table sẽ tự sinh ở P2 plan đã reject — phase này vẫn cần CREATE TABLE auto, dựa SchemaManager V2 đã có) hoặc shadow_payment-bill-service.payment-bills tồn tại + có row.

### T5.3 MariaDB smoke
- Tương tự T5.2 với `legacy_orders`.
- DoD: shadow + master table cho `legacy_orders` tự sinh, 5 row land.

## Dependency graph

```
T4.1 ──► T4.2 ──► T4.4 ──► T5.3
T1.1 ──► T1.2,T1.3 ──► T1.4 ──► T5.1,T5.2,T5.3
T2.1 ──► T2.2 ──► T2.3 ──► T3.x ──► T5.x
```

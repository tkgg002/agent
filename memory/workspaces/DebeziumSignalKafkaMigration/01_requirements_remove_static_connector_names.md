# 01_requirements_remove_static_connector_names

## 1. Trigger (user feedback nguyên văn)

> "quét cms-fe, cms api, cdc worker. bỏ luôn cái vụ goopay-mongodb-cdc này đi coi. tiện thể bỏ luôn cái cdc-pg-source, cdc-mariadb-source nếu nó còn vướng"

Cộng thêm rule:
- Đọc lesson trước tất cả ✓
- Theo core /agent, đọc GEMINI.md ✓
- Chỉ làm đúng yêu cầu
- Không cheat db hay thay đổi config để đạt kết quả
- Plan rõ ràng có solution cụ thể, có code demo
- Report dựa trên real-result, note files thay đổi
- Verify services trước khi báo done
- Có `report_*.md`

## 2. Bối cảnh (ground-truth survey)

Kafka Connect register connector ĐỘNG theo `connection_registry.connection_code` (1:1):
- DB `cdc_system.connection_registry`: `goopay-local` + `goopay-dev` (2 active source connections, engine_type=mongodb)
- Kafka Connect `/connectors` REST: `["goopay-local","goopay-dev"]` (trùng khớp 1:1)

Nhưng codebase hardcode 3 string ở 14 chỗ:
- `goopay-mongodb-cdc` (mongodb engine)
- `cdc-pg-source` (postgresql engine)
- `cdc-mariadb-source` (mariadb/mysql engine)

Survey kết quả (grep "goopay-mongodb-cdc|cdc-pg-source|cdc-mariadb-source" 4 repos):

### cdc-worker (centralized-data-service) — 7 location code/config
1. `config/config-local.yml:91` → `connectorName: goopay-mongodb-cdc` (mapstructure `cfg.Debezium.ConnectorName`)
2. `internal/handler/command_handler.go:2311-2315` → `detectConnectorName(entry) → "goopay-mongodb-cdc"` (hardcode fallback). Caller: `HandleSyncState`, `HandleRestartDebezium`.
3. `internal/admin/helpers.go:110-120` → `connectorNameFor(engineType) → switch{mongodb:"goopay-mongodb-cdc", postgresql:"cdc-pg-source", mariadb/mysql:"cdc-mariadb-source"}`. Caller: `extendDebeziumInclude` (admin RegisterSource flow).
4. `deployments/kafka/connector-mongodb.json` → `"name": "goopay-mongodb-cdc"` (sample deployment artifact).
5. `deployments/debezium/mongodb-connector.json` → same.
6. `deployments/debezium/pg-source-connector.json` → `"name": "cdc-pg-source"`.
7. `deployments/debezium/cdc-mariadb-source.json` → `"name": "cdc-mariadb-source"`.

### cdc-cms-service — 3 code + 1 sample config + 3 test files
1. `internal/infra/observability/system_health_collector.go:83,130` → `CollectorConfig.DebeziumName` field + default `"goopay-mongodb-cdc"`.
2. `internal/infra/observability/system_health_alerts.go:145` → fallback `name = c.cfg.DebeziumName`.
3. `internal/api/system_health_handler.go:70` → fallback `debeziumName = "goopay-mongodb-cdc"` cho `RestartDebezium` dispatch.
4. `config/config-sample.yml:34` → `debeziumConnector: goopay-mongodb-cdc`.
5. `config/config-local.yml:45` → `debeziumConnector: goopay` (sai trùm — không match connector nào hết).
6. `internal/infra/observability/{system_health_alerts_test.go, probes/debezium_test.go, persistence/alert_manager_test.go}` → test data.

### cdc-cms-web — 0 hit ✓
### cdc-control — 0 hit ✓

## 3. Mục tiêu / Definition of Done

- KHÔNG còn string literal `goopay-mongodb-cdc`, `cdc-pg-source`, `cdc-mariadb-source` trong runtime path (Go code + active config) của 3 repo.
- Worker resolve connector name động per-signal từ `(database, collection)` → DB lookup → `connection_code`.
- CMS-service probe debezium phải iterate ALL connectors live (auto-discover qua Kafka Connect `/connectors` REST).
- Provisioning flow (`admin/helpers.go::connectorNameFor`) resolve theo `source_connection_id` từ payload thay vì hardcode by engine.
- Build + vet + test clean ở cả 2 repos.
- Worker restart, E2E test debezium signal phải show TRUE root cause (`connector has 0 tasks ...`) thay vì HTTP 404 từ name mismatch.
- Out of scope: 4 file `deployments/**/*.json` (manual deploy reference, KHÔNG phải runtime). Sample tên-cố-định để CI test/local boot, không impact production hành vi.
- Out of scope: cdc-cms-web (0 hit).

## 4. Constraint cứng

- KHÔNG đổi DB schema (không add field `source_connection_id` vào `cdc_table_registry`).
- KHÔNG đổi payload schema NATS (caller CMS chưa truyền `connection_code` qua wire).
- KHÔNG đụng provisioning JSON deployments (sample, không runtime).
- Backwards-compat: nếu DB lookup không resolve được → optimistic skip probe (Healthy=true, Reason="connector name unresolved"), KHÔNG block publish. Lý do: lesson "visibility-vs-prevention".

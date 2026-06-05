# 02_plan.md — Implementation Plan

> **Status**: DRAFT — chờ user xác nhận H1/H2/both + cung cấp credentials/URLs prod.

## Plan A — H2 (Local POST connector → Prod Connect REST) — RECOMMEND ƯU TIÊN

Lý do recommend H2 trước H1: ít rủi ro hơn (chỉ submit config, không stream data), không cần broker external listener phức tạp, đa số dev workflow.

### Phase A0 — Pre-flight (user input cần)

| # | Hành động | Người thực hiện |
|---|---|---|
| A0.1 | User cung cấp: namespace cluster prod, tên Strimzi `KafkaConnect` resource, kubeconfig | User |
| A0.2 | User confirm có RBAC `kubectl port-forward` tới namespace prod | User |
| A0.3 | User confirm topic naming dev `cdc.dev-traingn.*` được phép | User |
| A0.4 | User confirm pattern: dùng `EnvVarConfigProvider` của Kafka Connect worker (cần worker prod set `CONNECT_CONFIG_PROVIDERS=env`) | User |

### Phase A1 — Local file refactor (Muscle execute sau user approve)

#### A1.1 — Sửa 4 connector JSON

`deployments/debezium/pg-source-connector.json`:
```json
{
  "name": "${env:CONNECTOR_NAME_PG}",
  "config": {
    "connector.class": "io.debezium.connector.postgresql.PostgresConnector",
    "tasks.max": "1",

    "database.hostname": "${env:DB_PG_HOST}",
    "database.port": "${env:DB_PG_PORT}",
    "database.user": "${env:DB_PG_USER}",
    "database.password": "${env:DB_PG_PASSWORD}",
    "database.dbname": "${env:DB_PG_DATABASE}",

    "topic.prefix": "${env:TOPIC_PREFIX_PG}",
    "plugin.name": "pgoutput",
    "slot.name": "${env:PG_SLOT_NAME}",
    "publication.name": "${env:PG_PUBLICATION_NAME}",
    "publication.autocreate.mode": "filtered",

    "schema.include.list": "public",
    "table.include.list": "${env:PG_TABLE_INCLUDE_LIST}",

    "snapshot.mode": "initial",
    "heartbeat.interval.ms": "10000",

    "schema.name.adjustment.mode": "avro",
    "field.name.adjustment.mode": "avro",
    "sanitize.field.names": "true",

    "key.converter": "io.confluent.connect.avro.AvroConverter",
    "key.converter.schema.registry.url": "${env:SCHEMA_REGISTRY_URL}",
    "key.converter.scrub.invalid.names": "true",
    "value.converter": "io.confluent.connect.avro.AvroConverter",
    "value.converter.schema.registry.url": "${env:SCHEMA_REGISTRY_URL}",
    "value.converter.scrub.invalid.names": "true",

    "max.batch.size": "2048",
    "max.queue.size": "8192"
  }
}
```

Tương tự cho `mongodb-connector.json`, `cdc-mariadb-source.json`, `connector-mongodb.json` — thay tất cả hardcode value bằng `${env:VAR}`.

#### A1.2 — Tạo `.env.example` ở `deployments/debezium/`

```bash
# Schema Registry
SCHEMA_REGISTRY_URL=http://schema-registry:8081   # local
# Prod: kubectl port-forward svc/schema-registry 18181:8081 → http://localhost:18181

# Kafka Connect REST endpoint (chỉ dùng cho script register_*.sh)
CONNECT_URL=http://localhost:18083                # local
# Prod: kubectl port-forward svc/kafka-connect-rest 28083:8083 → http://localhost:28083

# Connector tên (namespace dev khi target prod)
CONNECTOR_NAME_PG=cdc-pg-source                   # local
# Prod: CONNECTOR_NAME_PG=cdc-pg-source-dev-traingn

# Topic prefix (namespace dev khi target prod)
TOPIC_PREFIX_PG=cdc.gpay                          # local
# Prod: TOPIC_PREFIX_PG=cdc.dev-traingn

# Postgres source
DB_PG_HOST=gpay-postgres-source
DB_PG_PORT=5432
DB_PG_USER=src_user
DB_PG_PASSWORD=src_pass                           # local; prod: từ k8s secret
DB_PG_DATABASE=goopay_source
PG_SLOT_NAME=cdc_gpay_pg_source
PG_PUBLICATION_NAME=cdc_gpay_pub
PG_TABLE_INCLUDE_LIST=public.orders,public.users,public.payments

# MongoDB source
DB_MONGO_HOSTS=gpay-mongo:27017
DB_MONGO_REPL_SET=rs0
TOPIC_PREFIX_MONGO=cdc.gpay
DB_MONGO_INCLUDE_COLLS=goopay_source.orders,goopay_source.users

# MariaDB source
DB_MARIADB_HOST=gpay-mariadb
DB_MARIADB_PORT=3306
DB_MARIADB_USER=src_user
DB_MARIADB_PASSWORD=src_pass
DB_MARIADB_DATABASE=goopay_source
TOPIC_PREFIX_MARIADB=cdc.gpay
```

#### A1.3 — Sửa script `register_pg_source.sh`

Thêm:
```bash
# Đầu file: load .env.local nếu có
if [ -f "$(dirname "$0")/.env.local" ]; then
  set -a; source "$(dirname "$0")/.env.local"; set +a
fi

CONNECT_URL="${CONNECT_URL:-http://localhost:18083}"
# (giữ logic curl POST như cũ, Kafka Connect worker sẽ tự resolve ${env:VAR})
```

Tương tự script cho mongo + mariadb (nếu chưa có thì tạo `register_mongodb.sh`, `register_mariadb.sh`).

#### A1.4 — Update `.gitignore`

```
# Local dev override cho Debezium connector
deployments/debezium/.env.local
```

#### A1.5 — Sửa `docker-compose.yml` kafka-connect service

Thêm 2 ENV để worker LOCAL support `EnvVarConfigProvider`:
```yaml
kafka-connect:
  environment:
    # ... existing ...
    CONNECT_CONFIG_PROVIDERS: env
    CONNECT_CONFIG_PROVIDERS_ENV_CLASS: org.apache.kafka.common.config.provider.EnvVarConfigProvider
    # Inject các VAR mà connector JSON sẽ resolve
    SCHEMA_REGISTRY_URL: http://schema-registry:8081
    DB_PG_HOST: gpay-postgres-source
    DB_PG_PORT: "5432"
    # ... (đầy đủ list từ .env.example phía trên)
```

→ Pattern này áp dụng được cả local + prod (prod set qua Helm `env:` / Secret).

### Phase A2 — Smoke test local trước (verify pattern hoạt động)

```bash
cd data-hub/centralized-data-service

# 1. Stop kafka-connect cũ + rebuild với ENV mới
docker compose stop kafka-connect
docker compose up -d kafka-connect

# 2. Đợi REST sẵn sàng
curl -fsS http://localhost:18083/ | jq .version

# 3. Verify config provider loaded
docker logs gpay-kafka-connect 2>&1 | grep -i "config.providers"
# Expect: "config.providers = [env]"

# 4. Đăng ký connector
cd deployments/debezium
cp .env.example .env.local   # nếu chưa có
bash register_pg_source.sh

# 5. Verify connector RUNNING
curl -fsS http://localhost:18083/connectors/cdc-pg-source/status | jq .

# 6. Verify topic được Debezium tạo
docker exec gpay-kafka kafka-topics --bootstrap-server localhost:9092 --list | grep cdc.gpay
```

DoD A2: state RUNNING + ít nhất 1 topic `cdc.gpay.public.orders` xuất hiện.

### Phase A3 — Test target prod

```bash
# 1. Port-forward Connect REST prod
kubectl -n <prod-ns> port-forward svc/<connect-rest-svc> 28083:8083 &
PF_PID=$!

# 2. Verify REST reach
curl -fsS http://localhost:28083/ | jq .version

# 3. Verify worker prod đã có CONNECT_CONFIG_PROVIDERS=env
curl -fsS http://localhost:28083/connector-plugins | jq .
# Note: trong logs Strimzi KafkaConnect resource, config provider phải có

# 4. Set ENV target prod
cd deployments/debezium
cat > .env.local <<EOF
CONNECT_URL=http://localhost:28083
CONNECTOR_NAME_PG=cdc-pg-source-dev-traingn
TOPIC_PREFIX_PG=cdc.dev-traingn
SCHEMA_REGISTRY_URL=<prod-schema-registry-url>
DB_PG_HOST=<prod-db-host>
DB_PG_PORT=5432
DB_PG_USER=<prod-readonly-user>
DB_PG_PASSWORD=<prod-pass>
DB_PG_DATABASE=<prod-db>
PG_SLOT_NAME=cdc_dev_traingn
PG_PUBLICATION_NAME=cdc_dev_traingn_pub
PG_TABLE_INCLUDE_LIST=public.orders
EOF

# 5. Đăng ký connector ở prod
bash register_pg_source.sh

# 6. Verify connector RUNNING trên prod
curl -fsS http://localhost:28083/connectors/cdc-pg-source-dev-traingn/status | jq .

# 7. Teardown khi xong test
curl -fsS -X DELETE http://localhost:28083/connectors/cdc-pg-source-dev-traingn
kill $PF_PID
```

DoD A3: connector dev đăng ký + state RUNNING + sau cleanup không còn connector orphan.

⚠️ **Lưu ý prod**:
- `PG_SLOT_NAME`, `PG_PUBLICATION_NAME` phải unique cho dev session (tránh đụng slot prod)
- `PG_TABLE_INCLUDE_LIST` nên chỉ 1 table nhỏ để hạn chế load
- `TOPIC_PREFIX_PG=cdc.dev-traingn` → topic dev tách bạch
- Nếu prod broker có ACL → cần grant user `Topic:Create` + `Topic:Write` cho prefix `cdc.dev-traingn.*`

---

## Plan B — H1 (Local stack → Prod Kafka broker) — DEFER, làm sau H2 nếu cần

Phức tạp hơn vì cần:
- Bootstrap brokers external listener (Strimzi `KafkaCluster.spec.kafka.listeners[].type=route|ingress|loadbalancer`)
- mTLS cert hoặc SASL/SCRAM credentials cho client
- Local `cdc-worker` Go binary load cert/credentials qua ENV
- DNS/Ingress resolve từ máy local ra prod

### Skeleton (chi tiết hoá khi user quyết làm H1)

| # | Step | File đụng |
|---|---|---|
| B1 | User provision client credentials prod (SASL user `dev-traingn` + grant read topic `cdc.dev-*`) | k8s Secret prod (operator làm) |
| B2 | Download CA cert prod về `~/.config/cdc/prod-ca.crt` | local file |
| B3 | Sửa `docker-compose.yml` thêm ENV: `KAFKA_SECURITY_PROTOCOL`, `KAFKA_SASL_MECHANISM`, `KAFKA_SASL_USERNAME`, `KAFKA_SASL_PASSWORD`, `KAFKA_SSL_CA_LOCATION` | `data-hub/centralized-data-service/docker-compose.yml` |
| B4 | Sửa cdc-worker Go config loader hỗ trợ TLS + SASL nếu chưa có | `internal/config/kafka.go` (cần kiểm tra hiện trạng — có thể không cần đụng nếu sarama đã support qua ENV) |
| B5 | `.env.local` set `KAFKA_BROKERS=<prod-bootstrap>`, `KAFKA_SCHEMA_REGISTRY_URL=<prod-sr>` | local |
| B6 | `docker compose up cdc-worker` → log connected + consume 1 message từ topic dev | smoke verify |

DoD: tail log cdc-worker thấy 1 batch consume từ topic `cdc.dev-traingn.test` + emit OpenTelemetry trace tới collector local.

---

## Plan C — Defer to user nếu yêu cầu khác

| Khả năng | Mô tả | Action |
|---|---|---|
| C1 | Strimzi KafkaConnector CRD thay vì curl POST | Tách workspace mới sau khi user confirm pattern |
| C2 | Build Dockerfile prod thêm Debezium PostgreSQL + MySQL plugin | Sửa Dockerfile builder stage thêm `confluent-hub install`; user quyết |
| C3 | Local k8s (kind/minikube) với Strimzi → mirror prod | Quá nặng cho hybrid dev, recommend skip |

---

## Risk Register tổng

| ID | Risk | Severity | Mitigation |
|---|---|---|---|
| R-A | Local script POST nhầm vào prod cluster với connector tên trùng | HIGH | Convention `*-dev-<user>` suffix; whitelist tên trong script (refuse nếu thiếu suffix `dev-`) |
| R-B | Topic dev không cleanup → tích lũy data prod cluster | MED | Cleanup script `cleanup_dev_topics.sh` xoá `cdc.dev-<user>.*` cuối session |
| R-C | Credentials prod leak vào git | HIGH | `.env.local` gitignored; .env.example chỉ placeholder |
| R-D | `EnvVarConfigProvider` không có ở worker prod → connector fail load | MED | Phase A3 step 3 verify trước khi POST |
| R-E | Plugin Postgres/MySQL không có trong image prod → connector class not found | MED | Phase A3 verify `/connector-plugins` trước POST |
| R-F | Slot Postgres dev không cleanup → bloat WAL prod | HIGH | Trigger DELETE connector → Debezium auto drop slot (verify); thủ công: `SELECT pg_drop_replication_slot('cdc_dev_traingn')` |

---

## Approval gates

1. **Gate 1 — Plan review**: User đọc 02_plan.md + 00_context.md, xác nhận H1/H2, cung cấp URL prod + credentials.
2. **Gate 2 — Phase A1 review**: Sau khi Muscle refactor 4 JSON + script + docker-compose, user review diff trước smoke.
3. **Gate 3 — Phase A2 smoke local**: Connector local RUNNING với pattern `${env:*}` → user approve trước test prod.
4. **Gate 4 — Phase A3 smoke prod**: Báo cáo với evidence + state RUNNING + cleanup → user approve close task.

Mỗi gate fail → re-plan trong workspace, KHÔNG retry mò (CLAUDE.md §8 escalation).
